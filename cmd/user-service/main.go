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

	// Initialize OpenTelemetry tracer if enabled
	var tracerProvider *observability.TracerProvider
	if cfg.Tracing.Enabled {
		tracerCfg := observability.TracerConfig{
			ServiceName:    cfg.Tracing.ServiceName,
			ServiceVersion: "1.0.0", // TODO: Get from build info
			Environment:    cfg.AppEnv,
			OTLPEndpoint:   cfg.Tracing.OTLPEndpoint,
			Enabled:        cfg.Tracing.Enabled,
			SampleRate:     cfg.Tracing.SampleRate,
		}

		tracerProvider, err = observability.NewTracerProvider(ctx, tracerCfg)
		if err != nil {
			logger.WithField("error", err.Error()).Warn("Failed to initialize tracer provider, continuing without tracing")
		} else {
			defer func() {
				shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := tracerProvider.Shutdown(shutdownCtx); err != nil {
					logger.WithField("error", err.Error()).Error("Failed to shutdown tracer provider")
				}
			}()
			logger.Info("OpenTelemetry tracer initialized")
		}
	} else {
		logger.Info("OpenTelemetry tracing is disabled")
	}

	// Initialize HTTP routers (user-facing and admin-facing)
	ginMode := "release"
	if cfg.IsDevelopment() {
		ginMode = "debug"
	}

	userRouter := httpTransport.SetupUserRouter(userService, jwtManager, logger, ginMode, cfg.Tracing.Enabled)
	adminRouter := httpTransport.SetupAdminRouter(userService, jwtManager, logger, ginMode, cfg.Tracing.Enabled)

	logger.Info("HTTP routers initialized")

	// Create HTTP servers: user-facing and admin-facing (separate ports)
	userAddr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	adminAddr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.AdminPort)

	userServer := &http.Server{
		Addr:         userAddr,
		Handler:      userRouter,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	adminServer := &http.Server{
		Addr:         adminAddr,
		Handler:      adminRouter,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start user server
	go func() {
		logger.WithField("address", userAddr).Info("Starting user HTTP server")
		if err := userServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.WithField("error", err.Error()).Fatal("User HTTP server failed")
		}
	}()

	// Start admin server
	go func() {
		logger.WithField("address", adminAddr).Info("Starting admin HTTP server")
		if err := adminServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.WithField("error", err.Error()).Fatal("Admin HTTP server failed")
		}
	}()

	logger.WithFields(map[string]interface{}{
		"user_address":  userAddr,
		"admin_address": adminAddr,
	}).Info("User Service started successfully")

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown with timeout for both servers
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := userServer.Shutdown(ctx); err != nil {
		logger.WithField("error", err.Error()).Error("User server forced to shutdown")
	}
	if err := adminServer.Shutdown(ctx); err != nil {
		logger.WithField("error", err.Error()).Error("Admin server forced to shutdown")
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
