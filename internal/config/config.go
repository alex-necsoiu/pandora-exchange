// Package config provides configuration management for the User Service.
// Configuration is loaded from environment variables with sensible defaults.
// Supports multiple environments: dev, sandbox, audit, prod.
package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

const (
	// MinJWTSecretLength is the minimum required length for JWT signing keys (256 bits)
	MinJWTSecretLength = 32

	// Environment constants
	EnvDevelopment = "dev"
	EnvSandbox     = "sandbox"
	EnvAudit       = "audit"
	EnvProduction  = "prod"
)

// Config holds all configuration for the User Service
type Config struct {
	AppEnv   string         `mapstructure:"APP_ENV"`
	Server   ServerConfig   `mapstructure:",squash"`
	Database DatabaseConfig `mapstructure:",squash"`
	JWT      JWTConfig      `mapstructure:",squash"`
	Redis    RedisConfig    `mapstructure:",squash"`
	Tracing  TracingConfig  `mapstructure:",squash"`
	Audit    AuditConfig    `mapstructure:",squash"`
}

// ServerConfig holds HTTP/gRPC server configuration
type ServerConfig struct {
	Port     string `mapstructure:"SERVER_PORT"`
	Host     string `mapstructure:"SERVER_HOST"`
	GRPCPort string `mapstructure:"GRPC_PORT"`
	// AdminPort holds the HTTP port for the admin-only server
	AdminPort string `mapstructure:"ADMIN_PORT"`
}

// DatabaseConfig holds PostgreSQL connection configuration
type DatabaseConfig struct {
	Host     string `mapstructure:"DB_HOST"`
	Port     string `mapstructure:"DB_PORT"`
	User     string `mapstructure:"DB_USER"`
	Password string `mapstructure:"DB_PASSWORD"`
	Name     string `mapstructure:"DB_NAME"`
	SSLMode  string `mapstructure:"DB_SSLMODE"`
}

// JWTConfig holds JWT token configuration
type JWTConfig struct {
	Secret             string        `mapstructure:"JWT_SECRET"`
	AccessTokenExpiry  time.Duration `mapstructure:"JWT_ACCESS_TOKEN_EXPIRY"`
	RefreshTokenExpiry time.Duration `mapstructure:"JWT_REFRESH_TOKEN_EXPIRY"`
}

// RedisConfig holds Redis connection configuration
type RedisConfig struct {
	Host     string `mapstructure:"REDIS_HOST"`
	Port     string `mapstructure:"REDIS_PORT"`
	Password string `mapstructure:"REDIS_PASSWORD"`
	DB       int    `mapstructure:"REDIS_DB"`
}

// TracingConfig holds OpenTelemetry tracing configuration
type TracingConfig struct {
	Enabled      bool    `mapstructure:"OTEL_ENABLED"`
	OTLPEndpoint string  `mapstructure:"OTEL_EXPORTER_OTLP_ENDPOINT"`
	ServiceName  string  `mapstructure:"OTEL_SERVICE_NAME"`
	SampleRate   float64 `mapstructure:"OTEL_SAMPLE_RATE"`
}

// AuditConfig holds audit log retention and cleanup configuration
type AuditConfig struct {
	// RetentionDays specifies how many days to retain audit logs
	// Different retention periods per environment:
	// - dev: 30 days
	// - sandbox: 90 days
	// - audit: 2555 days (7 years for compliance)
	// - prod: 2555 days (7 years for compliance)
	RetentionDays int `mapstructure:"AUDIT_RETENTION_DAYS"`
	
	// CleanupInterval specifies how often to run the cleanup job (in hours)
	// Default: 24 hours (daily cleanup)
	CleanupInterval time.Duration `mapstructure:"AUDIT_CLEANUP_INTERVAL"`
}

// Load reads configuration from environment variables
// Returns error if required variables are missing or invalid
func Load() (*Config, error) {
	v := viper.New()

	// Set defaults
	v.SetDefault("SERVER_PORT", "8080")
	v.SetDefault("SERVER_HOST", "0.0.0.0")
	v.SetDefault("GRPC_PORT", "9090")
	v.SetDefault("ADMIN_PORT", "8081")
	v.SetDefault("DB_SSLMODE", "disable")
	v.SetDefault("JWT_ACCESS_TOKEN_EXPIRY", "15m")
	v.SetDefault("JWT_REFRESH_TOKEN_EXPIRY", "168h") // 7 days
	v.SetDefault("REDIS_HOST", "localhost")
	v.SetDefault("REDIS_PORT", "6379")
	v.SetDefault("REDIS_DB", 0)
	v.SetDefault("OTEL_ENABLED", false)
	v.SetDefault("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317")
	v.SetDefault("OTEL_SERVICE_NAME", "user-service")
	v.SetDefault("OTEL_SAMPLE_RATE", 1.0)
	v.SetDefault("AUDIT_RETENTION_DAYS", 90)
	v.SetDefault("AUDIT_CLEANUP_INTERVAL", "24h")

	// Bind environment variables explicitly
	v.AutomaticEnv()
	
	// Bind all environment variables explicitly
	envVars := []string{
		"APP_ENV",
		"SERVER_PORT", "SERVER_HOST", "GRPC_PORT",
		"ADMIN_PORT",
		"DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD", "DB_NAME", "DB_SSLMODE",
		"JWT_SECRET", "JWT_ACCESS_TOKEN_EXPIRY", "JWT_REFRESH_TOKEN_EXPIRY",
		"REDIS_HOST", "REDIS_PORT", "REDIS_PASSWORD", "REDIS_DB",
		"OTEL_ENABLED", "OTEL_EXPORTER_OTLP_ENDPOINT", "OTEL_SERVICE_NAME", "OTEL_SAMPLE_RATE",
		"AUDIT_RETENTION_DAYS", "AUDIT_CLEANUP_INTERVAL",
	}
	for _, env := range envVars {
		_ = v.BindEnv(env)
	}

	// Unmarshal into config struct
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate configuration
	if err := Validate(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Validate checks if the configuration is valid
func Validate(cfg *Config) error {
	// Validate environment
	validEnvs := map[string]bool{
		EnvDevelopment: true,
		EnvSandbox:     true,
		EnvAudit:       true,
		EnvProduction:  true,
	}
	if !validEnvs[cfg.AppEnv] {
		return fmt.Errorf("invalid environment '%s': must be one of [dev, sandbox, audit, prod]", cfg.AppEnv)
	}

	// Validate database config
	if cfg.Database.Host == "" {
		return fmt.Errorf("DB_HOST is required")
	}
	if cfg.Database.Port == "" {
		return fmt.Errorf("DB_PORT is required")
	}
	if cfg.Database.User == "" {
		return fmt.Errorf("DB_USER is required")
	}
	if cfg.Database.Password == "" {
		return fmt.Errorf("DB_PASSWORD is required")
	}
	if cfg.Database.Name == "" {
		return fmt.Errorf("DB_NAME is required")
	}

	// Validate JWT config
	if len(cfg.JWT.Secret) < MinJWTSecretLength {
		return fmt.Errorf("JWT secret must be at least %d characters long", MinJWTSecretLength)
	}
	if cfg.JWT.AccessTokenExpiry <= 0 {
		return fmt.Errorf("JWT access token expiry must be positive")
	}
	if cfg.JWT.RefreshTokenExpiry <= 0 {
		return fmt.Errorf("JWT refresh token expiry must be positive")
	}

	return nil
}

// GetDatabaseURL returns the PostgreSQL connection string
func (c *Config) GetDatabaseURL() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.Database.User,
		c.Database.Password,
		c.Database.Host,
		c.Database.Port,
		c.Database.Name,
		c.Database.SSLMode,
	)
}

// GetRedisAddr returns the Redis address in host:port format
func (c *Config) GetRedisAddr() string {
	return fmt.Sprintf("%s:%s", c.Redis.Host, c.Redis.Port)
}

// IsDevelopment returns true if running in development environment
func (c *Config) IsDevelopment() bool {
	return c.AppEnv == EnvDevelopment
}

// IsProduction returns true if running in production environment
func (c *Config) IsProduction() bool {
	return c.AppEnv == EnvProduction
}

// IsSandbox returns true if running in sandbox environment
func (c *Config) IsSandbox() bool {
	return c.AppEnv == EnvSandbox
}

// IsAudit returns true if running in audit environment
func (c *Config) IsAudit() bool {
	return c.AppEnv == EnvAudit
}
