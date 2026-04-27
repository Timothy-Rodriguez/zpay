package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"zpay/internal/database"
	"zpay/internal/model"
	"zpay/internal/pkg"
	"zpay/internal/router"

	"go.opentelemetry.io/otel"
)

func main() {
	// Load config
	cfg, err := pkg.LoadConfig("./config")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize logger
	logger := pkg.NewStructuredLogger("zpay-backend")
	logger.Info("application_starting",
		"env", cfg.Server.Env,
		"port", cfg.Server.Port,
	)

	// Initialize tracing
	ctx := context.Background()
	trackerProvider, err := pkg.InitTracerProvider(ctx, "zpay-backend")
	if err != nil {
		logger.Error("tracer_init_failed", "error", err.Error())
		os.Exit(1)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := trackerProvider.Shutdown(shutdownCtx); err != nil {
			logger.Error("tracer_shutdown_failed", "error", err.Error())
		}
	}()

	// Initialize database
	db, err := database.NewDB(&cfg.Database)
	if err != nil {
		logger.Error("database_init_failed", "error", err.Error())
		os.Exit(1)
	}
	db.Logger = logger
	defer db.Close()

	// Initialize redis
	redisClient, err := pkg.NewRedis(cfg.Redis)
	if err != nil {
		logger.Error("redis_init_failed", "error", err.Error())
		os.Exit(1)
	}
	defer func() {
		if err := redisClient.Close(); err != nil {
			logger.Error("redis_close_failed", "error", err.Error())
		}
	}()

	// Initialize JWT
	secret, err := os.ReadFile("./secret/jwt-secret.txt")
	if err != nil {
		logger.Error("jwt_secret_read_failed", "error", err.Error())
		os.Exit(1)
	}
	jwt := pkg.NewJWTService(secret)

	// Create App
	app := &model.App{
		DB:     db,
		JWT:    jwt,
		Redis:  redisClient,
		Logger: logger,
		Tracer: otel.Tracer("zpay-backend"),
	}

	// Setup router with app
	router := router.SetupRouter(app)

	// Create HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	// Graceful shutdown channel
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Start server in goroutine
	go func() {
		logger.Info("server_starting", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil {
			if err == http.ErrServerClosed {
				return
			}
			logger.Error("server_stopped_with_error", "error", err.Error())
			os.Exit(1)
		}
	}()

	// Wait for shutdown signal
	<-quit
	logger.Info("shutdown_signal_received")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("server_shutdown_failed", "error", err.Error())
		os.Exit(1)
	}

	logger.Info("server_stopped_gracefully")
}
