package server

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/tender-barbarian/gniot/repository"
	"github.com/tender-barbarian/gniot/repository/models"
	"github.com/tender-barbarian/gniot/server/handlers"
	"github.com/tender-barbarian/gniot/server/middleware"
	"github.com/tender-barbarian/gniot/server/routes"
	"github.com/tender-barbarian/gniot/service"
	gocrud "github.com/tender-barbarian/go-crud"
)

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func Run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	// Start DB
	dbPath := getEnv("DB_PATH", "./gniot.db")
	migrationsPath := getEnv("MIGRATIONS_PATH", "file://../db/migrations")

	db, err := repository.NewDBConnection(dbPath, migrationsPath)
	if err != nil {
		return fmt.Errorf("starting new DB connection: %v", err)
	}
	defer db.Close() // nolint

	devicesRepo := gocrud.NewGenericRepository(db, "devices", func() *models.Device { return &models.Device{} })
	actionsRepo := gocrud.NewGenericRepository(db, "actions", func() *models.Action { return &models.Action{} })
	jobsRepo := gocrud.NewGenericRepository(db, "jobs", func() *models.Job { return &models.Job{} })

	// Initialize helpers
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Initialize service
	service := service.NewService(devicesRepo, actionsRepo, jobsRepo, logger)

	// Initialize handlers and routes
	mux := http.NewServeMux()
	mux.Handle("/", http.NotFoundHandler())
	errorHandler := handlers.NewErrorHandler(logger)
	customHandlers := handlers.NewCustomHandlers(logger, service, errorHandler)
	mux = routes.RegisterCustomRoutes(mux, customHandlers)
	mux = routes.RegisterGenericRoutes(ctx, mux, errorHandler, devicesRepo)
	mux = routes.RegisterGenericRoutes(ctx, mux, errorHandler, actionsRepo)
	mux = routes.RegisterGenericRoutes(ctx, mux, errorHandler, jobsRepo)

	// Initialize middleware
	var wrappedMux http.Handler = mux
	wrappedMux = middleware.NewLoggingMiddleware(wrappedMux, logger)
	wrappedMux = middleware.NewRecoverMiddleware(wrappedMux, logger)

	// Start server
	httpServer := &http.Server{
		Addr:    net.JoinHostPort("127.0.0.1", "8080"),
		Handler: wrappedMux,
	}

	go func() {
		log.Printf("listening on %s\n", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "listening and serving requests: %s\n", err)
		}
	}()

	// Start job runner
	jobErrCh := make(chan error, 10)
	jobsInterval, err := time.ParseDuration(getEnv("JOBS_INTERVAL", "1m"))
	if err != nil {
		return fmt.Errorf("parsing JOBS_INTERVAL: %w", err)
	}
	go service.RunJobs(ctx, jobsInterval, jobErrCh)

	// Handle job errors
	go func() {
		for err := range jobErrCh {
			logger.Error("job runner error", "error", err)
		}
	}()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			fmt.Fprintf(os.Stderr, "error shutting down http server: %s\n", err)
		}
	}()

	wg.Wait()
	return nil
}
