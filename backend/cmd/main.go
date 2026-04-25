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

	"go.uber.org/zap"
)

func main() {
	// Load config
	cfg, err := pkg.LoadConfig("./config")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize logger
	logger, err := pkg.NewLogger(cfg.Server.Env)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		if err := logger.Sync(); err != nil {
			logger.Error("Error syncing logger", zap.Error(err))
		}
	}()

	logger.Info("Starting application", zap.String("env", cfg.Server.Env), zap.Int("port", cfg.Server.Port))

	// Initialize database
	db, err := database.NewDB(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	db.Logger = logger
	defer db.Close()

	// Initialize redis
	redisClient, err := pkg.NewRedis(cfg.Redis)
	if err != nil {
		log.Fatalf("Failed to initialize redis: %v", err)
	}
	defer func() {
		if err := redisClient.Close(); err != nil {
			log.Printf("Error closing redis: %v", err)
		}
	}()

	// Initialize JWT
	secret, err := os.ReadFile("./secret/jwt-secret.txt")
	if err != nil {
		logger.Fatal("error reading jwt secret", zap.Error(err))
	}
	jwt := pkg.NewJWTService(secret)

	// Create App
	app := &model.App{
		DB:     db,
		JWT:    jwt,
		Redis:  redisClient,
		Logger: logger,
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
		logger.Info("Server starting", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil {
			logger.Fatal("Server error", zap.Error(err))
		}
	}()

	// Wait for shutdown signal
	<-quit
	logger.Info("Shutdown signal received")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server shutdown error", zap.Error(err))
	}

	logger.Info("Server stopped gracefully")
}
