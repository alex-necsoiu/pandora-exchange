// Package config provides configuration management for the User Service.
// Configuration is loaded from environment variables with sensible defaults.
// Supports multiple environments: dev, sandbox, audit, prod.
// In dev/test: loads .env files via godotenv
// In prod/staging: can load from YAML files
// Priority: env vars > YAML > defaults
package config

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
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
	Vault    VaultConfig    `mapstructure:",squash"`
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

// VaultConfig holds HashiCorp Vault configuration for secret management
type VaultConfig struct {
	// Enabled determines if Vault integration is active
	// In dev, typically false (use ENV vars)
	// In prod, should be true
	Enabled bool `mapstructure:"VAULT_ENABLED"`
	
	// Addr is the Vault server address
	// Example: "http://vault.default.svc.cluster.local:8200"
	Addr string `mapstructure:"VAULT_ADDR"`
	
	// Token is the Vault authentication token
	// In Kubernetes, this is typically injected by Vault Agent
	Token string `mapstructure:"VAULT_TOKEN"`
	
	// SecretPath is the base path for secrets in Vault
	// Example: "secret/data/pandora/user-service"
	SecretPath string `mapstructure:"VAULT_SECRET_PATH"`
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
	v.SetDefault("VAULT_ENABLED", false)
	v.SetDefault("VAULT_ADDR", "http://localhost:8200")
	v.SetDefault("VAULT_SECRET_PATH", "secret/data/pandora/user-service")

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
		"VAULT_ENABLED", "VAULT_ADDR", "VAULT_TOKEN", "VAULT_SECRET_PATH",
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

// LoadConfig loads configuration with support for .env files and YAML
// Priority: environment variables > YAML file > defaults
//
// In dev/test environments:
//   - Attempts to load .env.{env} file (e.g., .env.dev)
//   - Falls back to environment variables
//
// In prod/staging environments:
//   - Can load from YAML file if CONFIG_FILE is set
//   - Falls back to environment variables
//
// Supports DATABASE_URL and REDIS_URL for simplified configuration
func LoadConfig(env string) (*Config, error) {
	// Step 1: Load .env file in dev/test environments
	if env == EnvDevelopment || env == "test" {
		envFile := fmt.Sprintf(".env.%s", env)
		if _, err := os.Stat(envFile); err == nil {
			if err := godotenv.Load(envFile); err != nil {
				// .env file exists but failed to load - log but continue
				fmt.Fprintf(os.Stderr, "Warning: failed to load %s: %v\n", envFile, err)
			}
		}
		// Also try loading .env as fallback
		_ = godotenv.Load()
	}

	// Step 2: Try to load from YAML if CONFIG_FILE is set
	configFile := os.Getenv("CONFIG_FILE")
	if configFile != "" {
		cfg, err := loadFromYAML(configFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to load YAML config from %s, falling back to env vars\n", configFile)
		} else {
			return cfg, nil
		}
	}

	// Step 3: Handle DATABASE_URL before calling Load()
	// Parse and set individual env vars so Load() picks them up
	var dbURLParsed bool
	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		parsedURL, err := url.Parse(dbURL)
		if err != nil {
			return nil, fmt.Errorf("invalid DATABASE_URL format: %w", err)
		}

		// Set individual env vars from DATABASE_URL
		if parsedURL.User != nil {
			_ = os.Setenv("DB_USER", parsedURL.User.Username()) // #nosec G104 -- error is always nil
			if password, ok := parsedURL.User.Password(); ok {
				_ = os.Setenv("DB_PASSWORD", password) // #nosec G104 -- error is always nil
			}
		}
		if parsedURL.Hostname() != "" {
			_ = os.Setenv("DB_HOST", parsedURL.Hostname()) // #nosec G104 -- error is always nil
		}
		if parsedURL.Port() != "" {
			_ = os.Setenv("DB_PORT", parsedURL.Port()) // #nosec G104 -- error is always nil
		}
		if len(parsedURL.Path) > 1 {
			_ = os.Setenv("DB_NAME", parsedURL.Path[1:]) // #nosec G104 -- error is always nil
		}
		if sslmode := parsedURL.Query().Get("sslmode"); sslmode != "" {
			_ = os.Setenv("DB_SSLMODE", sslmode) // #nosec G104 -- error is always nil
		}
		dbURLParsed = true
	}

	// Step 4: Handle REDIS_URL before calling Load()
	if redisURL := os.Getenv("REDIS_URL"); redisURL != "" {
		parsedURL, err := url.Parse(redisURL)
		if err != nil {
			return nil, fmt.Errorf("invalid REDIS_URL format: %w", err)
		}

		// Set individual env vars from REDIS_URL
		if parsedURL.Hostname() != "" {
			_ = os.Setenv("REDIS_HOST", parsedURL.Hostname()) // #nosec G104 -- error is always nil
		}
		if parsedURL.Port() != "" {
			_ = os.Setenv("REDIS_PORT", parsedURL.Port()) // #nosec G104 -- error is always nil
		}
		if parsedURL.User != nil {
			if password, ok := parsedURL.User.Password(); ok {
				_ = os.Setenv("REDIS_PASSWORD", password) // #nosec G104 -- error is always nil
			}
		}
		if len(parsedURL.Path) > 1 {
			if db, err := strconv.Atoi(parsedURL.Path[1:]); err == nil {
				_ = os.Setenv("REDIS_DB", strconv.Itoa(db)) // #nosec G104 -- error is always nil
			}
		}
	}

	// Step 5: Load from environment variables
	cfg, err := Load()
	if err != nil {
		return nil, err
	}

	// Step 6: Validate Vault placeholders in dev/test
	if env == EnvDevelopment || env == "test" {
		if err := validateVaultPlaceholders(cfg); err != nil {
			return nil, err
		}
	}

	// For debugging: log if DATABASE_URL was used
	_ = dbURLParsed

	return cfg, nil
}

// loadFromYAML loads configuration from a YAML file
func loadFromYAML(filename string) (*Config, error) {
	// Validate that the filename is not attempting path traversal
	if strings.Contains(filename, "..") {
		return nil, fmt.Errorf("invalid config file path: path traversal detected")
	}
	
	data, err := os.ReadFile(filename) // #nosec G304 -- filename is from CONFIG_FILE env var, validated above
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse YAML config: %w", err)
	}

	// Set environment from YAML
	if cfg.AppEnv != "" {
		_ = os.Setenv("APP_ENV", cfg.AppEnv) // #nosec G104 -- error is always nil
	}

	// Validate configuration
	if err := Validate(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// validateVaultPlaceholders validates that Vault placeholders follow the expected format
// Format: vault://secret/path/to/key
func validateVaultPlaceholders(cfg *Config) error {
	checkPlaceholder := func(value, fieldName string) error {
		if !strings.HasPrefix(value, "vault://") {
			return nil // Not a placeholder, skip
		}

		// Validate format
		parts := strings.Split(value, "://")
		if len(parts) != 2 || parts[1] == "" {
			return fmt.Errorf("%s has invalid Vault placeholder format (expected vault://secret/path/to/key)", fieldName)
		}

		return nil
	}

	// Check DB password
	if err := checkPlaceholder(cfg.Database.Password, "DB_PASSWORD"); err != nil {
		return err
	}

	// Check JWT secret
	if err := checkPlaceholder(cfg.JWT.Secret, "JWT_SECRET"); err != nil {
		return err
	}

	// Check Redis password
	if err := checkPlaceholder(cfg.Redis.Password, "REDIS_PASSWORD"); err != nil {
		return err
	}

	return nil
}

// Validate checks if the configuration is valid
func Validate(cfg *Config) error {
	// Validate environment
	validEnvs := map[string]bool{
		EnvDevelopment: true,
		EnvSandbox:     true,
		EnvAudit:       true,
		EnvProduction:  true,
		"test":         true, // Allow "test" environment
	}
	if !validEnvs[cfg.AppEnv] {
		return fmt.Errorf("invalid environment '%s': must be one of [dev, sandbox, audit, prod, test]", cfg.AppEnv)
	}

	// Validate database config
	if cfg.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}
	if cfg.Database.Port == "" {
		return fmt.Errorf("database port is required")
	}
	if cfg.Database.User == "" {
		return fmt.Errorf("database user is required")
	}
	if cfg.Database.Password == "" {
		return fmt.Errorf("database password is required")
	}
	if cfg.Database.Name == "" {
		return fmt.Errorf("database name is required")
	}

	// Validate JWT config
	// Allow Vault placeholders in dev/test environments
	isVaultPlaceholder := strings.HasPrefix(cfg.JWT.Secret, "vault://")
	isDev := cfg.AppEnv == EnvDevelopment || cfg.AppEnv == "test"
	
	// Check if JWT_SECRET is set
	if cfg.JWT.Secret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}
	
	if !isVaultPlaceholder && len(cfg.JWT.Secret) < MinJWTSecretLength {
		return fmt.Errorf("JWT secret must be at least %d characters long", MinJWTSecretLength)
	}
	
	// In production, Vault placeholders must be resolved before validation
	if isVaultPlaceholder && !isDev {
		return fmt.Errorf("JWT_SECRET contains unresolved Vault placeholder in %s environment", cfg.AppEnv)
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

// LoadSecretsFromVault loads sensitive configuration from HashiCorp Vault
// This method should be called after Load() to override ENV-based secrets with Vault values
//
// Parameters:
//   - ctx: Context for Vault operations
//   - vaultClient: Configured Vault client (can be nil if Vault disabled)
//
// Returns:
//   - error: Returns error if Vault enabled but secret fetch fails
//
// Secrets loaded from Vault:
//   - JWT_SECRET: JWT signing key
//   - DB_PASSWORD: PostgreSQL password
//   - REDIS_PASSWORD: Redis password
//
// In development (Vault disabled): Falls back to environment variables
// In production (Vault enabled): Fetches from Vault, fails if unavailable
func (c *Config) LoadSecretsFromVault(ctx context.Context, vaultClient interface{}) error {
	// Type assertion to avoid circular import
	// The vaultClient should implement GetSecret(ctx, path, key, envFallback) (string, error)
	type SecretGetter interface {
		GetSecret(ctx context.Context, path, key, envFallback string) (string, error)
		Enabled() bool
	}
	
	// If vault client is nil or disabled, keep ENV-based config
	if vaultClient == nil {
		return nil
	}
	
	client, ok := vaultClient.(SecretGetter)
	if !ok {
		return fmt.Errorf("invalid vault client type")
	}
	
	if !client.Enabled() {
		// Vault disabled - ENV vars already loaded
		return nil
	}
	
	// Build full secret paths
	basePath := c.Vault.SecretPath
	
	// Fetch JWT secret
	jwtSecret, err := client.GetSecret(ctx, basePath+"/jwt", "secret", "JWT_SECRET")
	if err != nil {
		return fmt.Errorf("failed to load JWT secret from vault: %w", err)
	}
	c.JWT.Secret = jwtSecret
	
	// Fetch database password
	dbPassword, err := client.GetSecret(ctx, basePath+"/database", "password", "DB_PASSWORD")
	if err != nil {
		return fmt.Errorf("failed to load database password from vault: %w", err)
	}
	c.Database.Password = dbPassword
	
	// Fetch Redis password (optional - can be empty)
	redisPassword, err := client.GetSecret(ctx, basePath+"/redis", "password", "REDIS_PASSWORD")
	if err != nil {
		// Redis password is optional, log but don't fail
		// Keep ENV value or empty string
	} else {
		c.Redis.Password = redisPassword
	}
	
	// Re-validate config after loading secrets
	if err := Validate(c); err != nil {
		return fmt.Errorf("config validation failed after loading vault secrets: %w", err)
	}
	
	return nil
}
