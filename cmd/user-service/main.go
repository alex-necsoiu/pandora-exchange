// Package main is the entry point for the User Service.
// It initializes all dependencies and starts the HTTP server.
package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alex-necsoiu/pandora-exchange/internal/config"
	"github.com/alex-necsoiu/pandora-exchange/internal/domain/auth"
	"github.com/alex-necsoiu/pandora-exchange/internal/events"
	"github.com/alex-necsoiu/pandora-exchange/internal/observability"
	"github.com/alex-necsoiu/pandora-exchange/internal/repository"
	"github.com/alex-necsoiu/pandora-exchange/internal/service"
	grpcTransport "github.com/alex-necsoiu/pandora-exchange/internal/transport/grpc"
	pb "github.com/alex-necsoiu/pandora-exchange/internal/transport/grpc/proto"
	httpTransport "github.com/alex-necsoiu/pandora-exchange/internal/transport/http"
	"github.com/alex-necsoiu/pandora-exchange/internal/vault"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"google.golang.org/grpc"
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

	// Initialize Vault client for secret management
	ctx := context.Background()
	var vaultClient *vault.Client
	
	if cfg.Vault.Enabled {
		logger.WithFields(map[string]interface{}{
			"vault_addr": cfg.Vault.Addr,
			"secret_path": cfg.Vault.SecretPath,
		}).Info("Initializing Vault client")
		
		vaultClient, err = vault.NewClient(cfg.Vault.Addr, cfg.Vault.Token)
		if err != nil {
			logger.WithField("error", err.Error()).Fatal("Failed to initialize Vault client")
		}
		
		// Check Vault availability
		if !vaultClient.IsAvailable(ctx) {
			logger.Warn("Vault is configured but not available, falling back to environment variables")
		} else {
			logger.Info("Vault client initialized successfully")
		}
	} else {
		logger.Info("Vault integration disabled, using environment variables for secrets")
		vaultClient = vault.NewDisabledClient()
	}

	// Load secrets from Vault (or fall back to ENV if Vault disabled/unavailable)
	if err := cfg.LoadSecretsFromVault(ctx, vaultClient); err != nil {
		logger.WithField("error", err.Error()).Fatal("Failed to load secrets from Vault")
	}
	
	if cfg.Vault.Enabled && vaultClient.IsAvailable(ctx) {
		logger.Info("Secrets loaded from Vault successfully")
	} else {
		logger.Info("Using environment variable secrets (dev mode)")
	}

	// Initialize database connection pool
	dbPool, err := initDatabase(ctx, cfg, logger)
	if err != nil {
		logger.WithField("error", err.Error()).Fatal("Failed to initialize database")
	}
	defer dbPool.Close()

	logger.Info("Database connection pool initialized")

	// Initialize Redis client for event publishing
	redisAddr := fmt.Sprintf("%s:%s", cfg.Redis.Host, cfg.Redis.Port)
	redisClient := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	// Test Redis connection
	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.WithField("error", err.Error()).Warn("Failed to connect to Redis, event publishing will be disabled")
		redisClient = nil // Disable event publishing if Redis is unavailable
	} else {
		logger.WithField("redis_addr", redisAddr).Info("Redis connection established")
	}

	// Initialize event publisher
	var eventPublisher *events.RedisEventPublisher
	if redisClient != nil {
		// Create a zap logger for the event publisher
		var zapLogger *zap.Logger
		if cfg.AppEnv == "prod" {
			zapLogger, err = zap.NewProduction()
		} else {
			zapLogger, err = zap.NewDevelopment()
		}
		if err != nil {
			logger.WithField("error", err.Error()).Warn("Failed to create zap logger for event publisher")
		} else {
			eventPublisher = events.NewRedisEventPublisher(redisClient, zapLogger)
			logger.Info("Event publisher initialized")
		}
	}

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
	auditRepo := repository.NewAuditRepository(dbPool, logger)

	logger.Info("Repositories initialized")

	// Initialize service
	userService, err := service.NewUserService(
		userRepo,
		tokenRepo,
		cfg.JWT.Secret,
		cfg.JWT.AccessTokenExpiry,
		cfg.JWT.RefreshTokenExpiry,
		logger,
		eventPublisher, // Event publisher (can be nil if Redis is unavailable)
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

	// Initialize gRPC server with interceptors
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			grpcTransport.UnaryRecoveryInterceptor(logger),
			grpcTransport.UnaryLoggingInterceptor(logger),
			grpcTransport.UnaryTracingInterceptor(),
		),
	)

	// Initialize ServiceRegistry with reflection enabled for dev/sandbox only
	enableReflection := cfg.IsDevelopment() || cfg.AppEnv == config.EnvSandbox
	registry := grpcTransport.NewServiceRegistry(grpcServer, grpcTransport.WithReflection(enableReflection))
	
	logger.WithField("reflection_enabled", enableReflection).Info("Service registry initialized")

	// Register gRPC service
	userGRPCService := grpcTransport.NewServer(userService, logger)
	pb.RegisterUserServiceServer(grpcServer, userGRPCService)

	// Register UserService metadata in registry
	serviceInfo := &grpcTransport.ServiceInfo{
		Name:        "pandora.user.v1.UserService",
		Version:     "v1",
		Description: "User authentication and management service",
		Methods: []string{
			"Register",
			"Login",
			"GetUser",
			"UpdateUser",
			"DeleteUser",
			"RefreshToken",
			"Logout",
		},
		ProtoFile: "internal/transport/grpc/proto/user_service.proto",
		Metadata: map[string]string{
			"service_version": "1.0.0",
			"environment":     cfg.AppEnv,
		},
	}
	
	if err := registry.RegisterService(serviceInfo); err != nil {
		logger.WithField("error", err.Error()).Fatal("Failed to register service in registry")
	}

	logger.Info("gRPC server initialized")

	// Initialize HTTP routers (user-facing and admin-facing)
	ginMode := "release"
	if cfg.IsDevelopment() {
		ginMode = "debug"
	}

	userRouter := httpTransport.SetupUserRouter(userService, jwtManager, auditRepo, cfg, logger, ginMode, cfg.Tracing.Enabled)
	adminRouter := httpTransport.SetupAdminRouter(userService, jwtManager, auditRepo, cfg, logger, ginMode, cfg.Tracing.Enabled, registry)

	logger.Info("HTTP routers initialized")

	// Create HTTP servers: user-facing and admin-facing (separate ports)
	userAddr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	adminAddr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.AdminPort)
	grpcAddr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.GRPCPort)

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

	// Start gRPC server
	go func() {
		listener, err := net.Listen("tcp", grpcAddr)
		if err != nil {
			logger.WithField("error", err.Error()).Fatal("Failed to create gRPC listener")
		}

		logger.WithField("address", grpcAddr).Info("Starting gRPC server")
		if err := grpcServer.Serve(listener); err != nil {
			logger.WithField("error", err.Error()).Fatal("gRPC server failed")
		}
	}()

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
		"grpc_address":  grpcAddr,
	}).Info("User Service started successfully")

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown with timeout for all servers
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Shutdown HTTP servers
	if err := userServer.Shutdown(ctx); err != nil {
		logger.WithField("error", err.Error()).Error("User server forced to shutdown")
	}
	if err := adminServer.Shutdown(ctx); err != nil {
		logger.WithField("error", err.Error()).Error("Admin server forced to shutdown")
	}

	// Gracefully stop gRPC server and service registry
	logger.Info("Stopping gRPC server and service registry...")
	if err := registry.Shutdown(ctx); err != nil {
		logger.WithField("error", err.Error()).Error("Service registry forced to shutdown")
	}

	// Close event publisher and Redis connection
	if eventPublisher != nil {
		logger.Info("Closing event publisher...")
		if err := eventPublisher.Close(); err != nil {
			logger.WithField("error", err.Error()).Error("Failed to close event publisher")
		}
	}

	logger.Info("All servers stopped gracefully")
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
