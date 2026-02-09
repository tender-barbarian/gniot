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

	"github.com/tender-barbarian/gniotek/cache"
	"github.com/tender-barbarian/gniotek/repository"
	"github.com/tender-barbarian/gniotek/repository/models"
	"github.com/tender-barbarian/gniotek/server/handlers"
	"github.com/tender-barbarian/gniotek/server/middleware"
	"github.com/tender-barbarian/gniotek/server/routes"
	"github.com/tender-barbarian/gniotek/service"
	"github.com/tender-barbarian/gniotek/web"
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
	dbPath := getEnv("DB_PATH", "./gniotek.db")
	migrationsPath := getEnv("MIGRATIONS_PATH", "file://../db/migrations")

	db, err := repository.NewDBConnection(dbPath, migrationsPath)
	if err != nil {
		return fmt.Errorf("starting new DB connection: %v", err)
	}
	defer db.Close() // nolint

	devicesCache := cache.NewCache[*models.Device]()
	devicesRepo := gocrud.NewGenericRepository(db, "devices", func() *models.Device { return &models.Device{} }).WithValidate().WithOnMutate(devicesCache.InvalidateCache)
	actionsCache := cache.NewCache[*models.Action]()
	actionsRepo := gocrud.NewGenericRepository(db, "actions", func() *models.Action { return &models.Action{} }).WithValidate().WithOnMutate(actionsCache.InvalidateCache)
	automationsRepo := gocrud.NewGenericRepository(db, "automations", func() *models.Automation { return &models.Automation{} }).WithValidate()

	queryRepo := repository.NewQueryRepo(db, []string{"devices", "actions", "automations"})

	// Initialize helpers
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Initialize service
	svc := service.NewService(service.ServiceConfig{
		DevicesRepo:     devicesRepo,
		ActionsRepo:     actionsRepo,
		AutomationsRepo: automationsRepo,
		QueryRepo:       queryRepo,
		DevicesCache:    devicesCache,
		ActionsCache:    actionsCache,
		Logger:          logger,
	})

	// Initialize handlers and routes
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServerFS(web.StaticFiles))
	errorHandler := handlers.NewErrorHandler(logger)
	customHandlers := handlers.NewCustomHandlers(logger, svc, errorHandler)
	mux = routes.RegisterCustomRoutes(mux, customHandlers)
	mux = routes.RegisterGenericRoutes(ctx, mux, errorHandler, devicesRepo)
	mux = routes.RegisterGenericRoutes(ctx, mux, errorHandler, actionsRepo)
	mux = routes.RegisterGenericRoutes(ctx, mux, errorHandler, automationsRepo)

	// Start automation runner
	automationsInterval, err := time.ParseDuration(getEnv("AUTOMATIONS_INTERVAL", "1m"))
	if err != nil {
		return fmt.Errorf("parsing AUTOMATIONS_INTERVAL: %v", err)
	}
	automationErrCh := make(chan error, 100)
	go svc.RunAutomations(ctx, automationsInterval, automationErrCh)
	go func() {
		for err := range automationErrCh {
			logger.Error("automation error", "error", err)
		}
	}()

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
