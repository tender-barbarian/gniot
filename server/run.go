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

	"github.com/tender-barbarian/gniot/repository/env"
	"github.com/tender-barbarian/gniot/repository/models"
	"github.com/tender-barbarian/gniot/server/handlers"
	"github.com/tender-barbarian/gniot/server/middleware"
	"github.com/tender-barbarian/gniot/server/routes"
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

	db, err := env.NewDBConnection(dbPath, migrationsPath)
	if err != nil {
		return fmt.Errorf("starting new DB connection: %v", err)
	}
	defer db.Close() // nolint

	devicesRepo := gocrud.NewGenericRepository(db, "devices", func() *models.Device { return &models.Device{} })
	actionsRepo:= gocrud.NewGenericRepository(db, "actions", func() *models.Action { return &models.Action{} })

	// Initialize helpers
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Initialize handlers and routes
	mux := http.NewServeMux()
	mux.Handle("/", http.NotFoundHandler())
	h := handlers.NewHandlers(logger)
	mux = routes.RegisterCustomRoutes(mux, h)
	mux = routes.RegisterGenericRoutes(ctx, devicesRepo, mux, h)
	mux = routes.RegisterGenericRoutes(ctx, actionsRepo, mux, h)

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
