// Package main is the entry point for the User Service.
// It initializes all dependencies and starts the HTTP server.
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

	"github.com/alex-necsoiu/pandora-exchange/internal/config"
	"github.com/alex-necsoiu/pandora-exchange/internal/domain/auth"
	"github.com/alex-necsoiu/pandora-exchange/internal/observability"
	"github.com/alex-necsoiu/pandora-exchange/internal/repository"
	"github.com/alex-necsoiu/pandora-exchange/internal/service"
	httpTransport "github.com/alex-necsoiu/pandora-exchange/internal/transport/http"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize logger
	logger := observability.NewLogger(cfg.AppEnv, "user-service")
	logger.Info("Starting User Service")
	logger.WithFields(map[string]interface{}{
		"environment": cfg.AppEnv,
		"http_port":   cfg.Server.Port,
		"grpc_port":   cfg.Server.GRPCPort,
	}).Info("Configuration loaded")

	// Initialize database connection pool
	ctx := context.Background()
	dbPool, err := initDatabase(ctx, cfg, logger)
	if err != nil {
		logger.WithField("error", err.Error()).Fatal("Failed to initialize database")
	}
	defer dbPool.Close()

	logger.Info("Database connection pool initialized")

	// Initialize JWT manager
	jwtManager, err := auth.NewJWTManager(
		cfg.JWT.Secret,
		cfg.JWT.AccessTokenExpiry,
		cfg.JWT.RefreshTokenExpiry,
	)
	if err != nil {
		logger.WithField("error", err.Error()).Fatal("Failed to initialize JWT manager")
	}

	logger.Info("JWT manager initialized")

	// Initialize repositories
	userRepo := repository.NewUserRepository(dbPool, logger)
	tokenRepo := repository.NewRefreshTokenRepository(dbPool, logger)

	logger.Info("Repositories initialized")

	// Initialize service
	userService, err := service.NewUserService(
		userRepo,
		tokenRepo,
		cfg.JWT.Secret,
		cfg.JWT.AccessTokenExpiry,
		cfg.JWT.RefreshTokenExpiry,
		logger,
	)
	if err != nil {
		logger.WithField("error", err.Error()).Fatal("Failed to initialize user service")
	}

	logger.Info("User service initialized")

	// Initialize HTTP router
	ginMode := "release"
	if cfg.IsDevelopment() {
		ginMode = "debug"
	}

	router := httpTransport.SetupRouter(userService, jwtManager, logger, ginMode)

	logger.Info("HTTP router initialized")

	// Create HTTP server
	serverAddr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	server := &http.Server{
		Addr:         serverAddr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		logger.WithField("address", serverAddr).Info("Starting HTTP server")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.WithField("error", err.Error()).Fatal("HTTP server failed")
		}
	}()

	logger.WithField("address", serverAddr).Info("User Service started successfully")

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.WithField("error", err.Error()).Error("Server forced to shutdown")
	}

	logger.Info("Server stopped gracefully")
}

// initDatabase initializes the PostgreSQL connection pool.
func initDatabase(ctx context.Context, cfg *config.Config, logger *observability.Logger) (*pgxpool.Pool, error) {
	logger.WithFields(map[string]interface{}{
		"host":     cfg.Database.Host,
		"port":     cfg.Database.Port,
		"database": cfg.Database.Name,
	}).Info("Connecting to database")

	poolConfig, err := pgxpool.ParseConfig(cfg.GetDatabaseURL())
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	// Configure connection pool
	poolConfig.MaxConns = 25
	poolConfig.MinConns = 5
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute
	poolConfig.HealthCheckPeriod = time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return pool, nil
}
