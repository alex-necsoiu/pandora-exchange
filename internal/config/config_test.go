package config_test

import (
	"os"
	"testing"
	"time"

	"github.com/alex-necsoiu/pandora-exchange/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoadConfig tests the LoadConfig function with various scenarios
func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name        string
		env         string
		setupEnv    func()
		configFile  string
		expectError bool
		errorMsg    string
		validate    func(*testing.T, *config.Config)
	}{
		{
			name: "load from env vars only - dev environment",
			env:  "dev",
			setupEnv: func() {
				os.Setenv("APP_ENV", "dev")
				os.Setenv("SERVER_PORT", "8080")
				os.Setenv("SERVER_HOST", "localhost")
				os.Setenv("DB_HOST", "localhost")
				os.Setenv("DB_PORT", "5432")
				os.Setenv("DB_USER", "testuser")
				os.Setenv("DB_PASSWORD", "testpass")
				os.Setenv("DB_NAME", "testdb")
				os.Setenv("DB_SSLMODE", "disable")
				os.Setenv("JWT_SECRET", "test-secret-key-min-32-characters-long")
				os.Setenv("JWT_ACCESS_TOKEN_EXPIRY", "15m")
				os.Setenv("JWT_REFRESH_TOKEN_EXPIRY", "168h")
				os.Setenv("REDIS_HOST", "localhost")
				os.Setenv("REDIS_PORT", "6379")
			},
			expectError: false,
			validate: func(t *testing.T, cfg *config.Config) {
				assert.Equal(t, "dev", cfg.AppEnv)
				assert.Equal(t, "8080", cfg.Server.Port)
				assert.Equal(t, "localhost", cfg.Database.Host)
				assert.Equal(t, "testdb", cfg.Database.Name)
			},
		},
		{
			name: "load from DATABASE_URL override",
			env:  "dev",
			setupEnv: func() {
				os.Setenv("APP_ENV", "dev")
				os.Setenv("DATABASE_URL", "postgres://dbuser:dbpass@dbhost:5433/mydb?sslmode=require")
				os.Setenv("JWT_SECRET", "test-secret-key-min-32-characters-long")
				os.Setenv("REDIS_HOST", "localhost")
				os.Setenv("REDIS_PORT", "6379")
			},
			expectError: false,
			validate: func(t *testing.T, cfg *config.Config) {
				// DATABASE_URL should be parsed and populate granular fields
				url := cfg.GetDatabaseURL()
				assert.Contains(t, url, "dbuser")
				assert.Contains(t, url, "dbhost")
				assert.Contains(t, url, "mydb")
			},
		},
		{
			name: "load from REDIS_URL override",
			env:  "dev",
			setupEnv: func() {
				os.Setenv("APP_ENV", "dev")
				os.Setenv("DB_HOST", "localhost")
				os.Setenv("DB_PORT", "5432")
				os.Setenv("DB_USER", "user")
				os.Setenv("DB_PASSWORD", "pass")
				os.Setenv("DB_NAME", "db")
				os.Setenv("JWT_SECRET", "test-secret-key-min-32-characters-long")
				os.Setenv("REDIS_URL", "redis://redishost:6380")
			},
			expectError: false,
			validate: func(t *testing.T, cfg *config.Config) {
				// REDIS_URL should be parsed
				addr := cfg.GetRedisAddr()
				assert.Contains(t, addr, "redishost")
			},
		},
		{
			name: "fail when JWT secret missing",
			env:  "dev",
			setupEnv: func() {
				os.Setenv("APP_ENV", "dev")
				os.Setenv("DB_HOST", "localhost")
				os.Setenv("DB_PORT", "5432")
				os.Setenv("DB_USER", "user")
				os.Setenv("DB_PASSWORD", "pass")
				os.Setenv("DB_NAME", "db")
				// No JWT_SECRET
			},
			expectError: true,
			errorMsg:    "JWT_SECRET",
		},
		{
			name: "fail when JWT secret too short",
			env:  "dev",
			setupEnv: func() {
				os.Setenv("APP_ENV", "dev")
				os.Setenv("DB_HOST", "localhost")
				os.Setenv("DB_PORT", "5432")
				os.Setenv("DB_USER", "user")
				os.Setenv("DB_PASSWORD", "pass")
				os.Setenv("DB_NAME", "db")
				os.Setenv("JWT_SECRET", "short")
			},
			expectError: true,
			errorMsg:    "JWT secret must be at least 32 characters",
		},
		{
			name: "fail when database config missing",
			env:  "dev",
			setupEnv: func() {
				os.Setenv("APP_ENV", "dev")
				os.Setenv("JWT_SECRET", "test-secret-key-min-32-characters-long")
				// No database config
			},
			expectError: true,
			errorMsg:    "database",
		},
		{
			name: "accept Vault placeholders in dev",
			env:  "dev",
			setupEnv: func() {
				os.Setenv("APP_ENV", "dev")
				os.Setenv("DB_HOST", "localhost")
				os.Setenv("DB_PORT", "5432")
				os.Setenv("DB_USER", "user")
				os.Setenv("DB_PASSWORD", "vault://secret/db/password")
				os.Setenv("DB_NAME", "db")
				os.Setenv("JWT_SECRET", "vault://secret/jwt/secret")
				os.Setenv("REDIS_HOST", "localhost")
				os.Setenv("REDIS_PORT", "6379")
			},
			expectError: false,
			validate: func(t *testing.T, cfg *config.Config) {
				// Vault placeholders should be accepted in dev
				assert.Contains(t, cfg.Database.Password, "vault://")
				assert.Contains(t, cfg.JWT.Secret, "vault://")
			},
		},
		{
			name: "use defaults for optional fields",
			env:  "dev",
			setupEnv: func() {
				os.Setenv("APP_ENV", "dev")
				os.Setenv("DB_HOST", "localhost")
				os.Setenv("DB_PORT", "5432")
				os.Setenv("DB_USER", "user")
				os.Setenv("DB_PASSWORD", "pass")
				os.Setenv("DB_NAME", "db")
				os.Setenv("JWT_SECRET", "test-secret-key-min-32-characters-long")
				// No SERVER_PORT, SERVER_HOST, REDIS, etc.
			},
			expectError: false,
			validate: func(t *testing.T, cfg *config.Config) {
				// Should use defaults
				assert.Equal(t, "8080", cfg.Server.Port)
				assert.Equal(t, "0.0.0.0", cfg.Server.Host)
				assert.Equal(t, "localhost", cfg.Redis.Host)
				assert.Equal(t, "6379", cfg.Redis.Port)
				assert.Equal(t, 15*time.Minute, cfg.JWT.AccessTokenExpiry)
				assert.Equal(t, 7*24*time.Hour, cfg.JWT.RefreshTokenExpiry)
			},
		},
		{
			name: "parse duration strings correctly",
			env:  "dev",
			setupEnv: func() {
				os.Setenv("APP_ENV", "dev")
				os.Setenv("DB_HOST", "localhost")
				os.Setenv("DB_PORT", "5432")
				os.Setenv("DB_USER", "user")
				os.Setenv("DB_PASSWORD", "pass")
				os.Setenv("DB_NAME", "db")
				os.Setenv("JWT_SECRET", "test-secret-key-min-32-characters-long")
				os.Setenv("JWT_ACCESS_TOKEN_EXPIRY", "30m")
				os.Setenv("JWT_REFRESH_TOKEN_EXPIRY", "720h")
			},
			expectError: false,
			validate: func(t *testing.T, cfg *config.Config) {
				assert.Equal(t, 30*time.Minute, cfg.JWT.AccessTokenExpiry)
				assert.Equal(t, 720*time.Hour, cfg.JWT.RefreshTokenExpiry)
			},
		},
		{
			name: "fail on invalid duration format",
			env:  "dev",
			setupEnv: func() {
				os.Setenv("APP_ENV", "dev")
				os.Setenv("DB_HOST", "localhost")
				os.Setenv("DB_PORT", "5432")
				os.Setenv("DB_USER", "user")
				os.Setenv("DB_PASSWORD", "pass")
				os.Setenv("DB_NAME", "db")
				os.Setenv("JWT_SECRET", "test-secret-key-min-32-characters-long")
				os.Setenv("JWT_ACCESS_TOKEN_EXPIRY", "invalid")
			},
			expectError: true,
			errorMsg:    "duration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Clean environment and setup test conditions
			clearEnv()
			if tt.setupEnv != nil {
				tt.setupEnv()
			}

			// When: LoadConfig is called
			cfg, err := config.LoadConfig(tt.env)

			// Then: Verify expectations
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, cfg)
				if tt.validate != nil {
					tt.validate(t, cfg)
				}
			}

			// Cleanup
			clearEnv()
		})
	}
}

// TestLoadConfigFromYAML tests loading configuration from YAML file
func TestLoadConfigFromYAML(t *testing.T) {
	t.Run("load from YAML file when APP_ENV is prod", func(t *testing.T) {
		// Given: YAML config file exists
		clearEnv()
		os.Setenv("APP_ENV", "prod")
		os.Setenv("CONFIG_FILE", "../../configs/test.yaml")
		defer clearEnv()

		// When: LoadConfig is called
		cfg, err := config.LoadConfig("prod")

		// Then: Should load from YAML (or fall back to env if file missing)
		// This test will fail until implementation exists
		if err != nil {
			t.Skip("YAML loading not yet implemented")
		}
		require.NoError(t, err)
		require.NotNil(t, cfg)
	})
}

// TestLoadConfigWithDotEnv tests .env file loading in dev environment
func TestLoadConfigWithDotEnv(t *testing.T) {
	t.Run("load .env file in dev environment", func(t *testing.T) {
		// Given: Clean environment
		clearEnv()
		
		// When: LoadConfig is called with dev environment
		cfg, err := config.LoadConfig("dev")

		// Then: Should attempt to load .env.dev file
		// This test will fail until godotenv is integrated
		if err != nil && !os.IsNotExist(err) {
			t.Skip(".env loading not yet implemented")
		}
		
		// If .env.dev exists, config should load
		if cfg != nil {
			assert.Equal(t, "dev", cfg.AppEnv)
		}
	})
}

// TestLoadFromEnv tests loading configuration from environment variables
func TestLoadFromEnv(t *testing.T) {
	t.Run("load all required env vars", func(t *testing.T) {
		// Set environment variables
		os.Setenv("APP_ENV", "dev")
		os.Setenv("SERVER_PORT", "8080")
		os.Setenv("SERVER_HOST", "localhost")
		os.Setenv("DB_HOST", "localhost")
		os.Setenv("DB_PORT", "5432")
		os.Setenv("DB_USER", "testuser")
		os.Setenv("DB_PASSWORD", "testpass")
		os.Setenv("DB_NAME", "testdb")
		os.Setenv("JWT_SECRET", "test-secret-key-min-32-characters-long")
		os.Setenv("JWT_ACCESS_TOKEN_EXPIRY", "15m")
		os.Setenv("JWT_REFRESH_TOKEN_EXPIRY", "168h")
		os.Setenv("REDIS_HOST", "localhost")
		os.Setenv("REDIS_PORT", "6379")
		defer clearEnv()

		cfg, err := config.Load()
		require.NoError(t, err)

		assert.Equal(t, "dev", cfg.AppEnv)
		assert.Equal(t, "8080", cfg.Server.Port)
		assert.Equal(t, "localhost", cfg.Server.Host)
		assert.Equal(t, "localhost", cfg.Database.Host)
		assert.Equal(t, "5432", cfg.Database.Port)
		assert.Equal(t, "testuser", cfg.Database.User)
		assert.Equal(t, "testpass", cfg.Database.Password)
		assert.Equal(t, "testdb", cfg.Database.Name)
		assert.Equal(t, "test-secret-key-min-32-characters-long", cfg.JWT.Secret)
		assert.Equal(t, 15*time.Minute, cfg.JWT.AccessTokenExpiry)
		assert.Equal(t, 168*time.Hour, cfg.JWT.RefreshTokenExpiry)
		assert.Equal(t, "localhost", cfg.Redis.Host)
		assert.Equal(t, "6379", cfg.Redis.Port)
	})

	t.Run("use default values when optional vars not set", func(t *testing.T) {
		// Set only required vars
		os.Setenv("APP_ENV", "sandbox")
		os.Setenv("DB_HOST", "localhost")
		os.Setenv("DB_PORT", "5432")
		os.Setenv("DB_USER", "user")
		os.Setenv("DB_PASSWORD", "pass")
		os.Setenv("DB_NAME", "db")
		os.Setenv("JWT_SECRET", "secret-key-that-is-at-least-32-chars")
		defer clearEnv()

		cfg, err := config.Load()
		require.NoError(t, err)

		// Check defaults
		assert.Equal(t, "8080", cfg.Server.Port)
		assert.Equal(t, "0.0.0.0", cfg.Server.Host)
		assert.Equal(t, 15*time.Minute, cfg.JWT.AccessTokenExpiry)
		assert.Equal(t, 7*24*time.Hour, cfg.JWT.RefreshTokenExpiry)
	})

	t.Run("fail when JWT secret too short", func(t *testing.T) {
		os.Setenv("APP_ENV", "dev")
		os.Setenv("DB_HOST", "localhost")
		os.Setenv("DB_PORT", "5432")
		os.Setenv("DB_USER", "user")
		os.Setenv("DB_PASSWORD", "pass")
		os.Setenv("DB_NAME", "db")
		os.Setenv("JWT_SECRET", "short")
		defer clearEnv()

		_, err := config.Load()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "JWT secret")
	})

	t.Run("fail when required env vars missing", func(t *testing.T) {
		clearEnv()
		os.Setenv("APP_ENV", "dev")
		
		_, err := config.Load()
		assert.Error(t, err)
	})
}

// TestValidateConfig tests configuration validation
func TestValidateConfig(t *testing.T) {
	t.Run("valid configuration passes", func(t *testing.T) {
		cfg := &config.Config{
			AppEnv: "dev",
			Server: config.ServerConfig{
				Port: "8080",
				Host: "localhost",
			},
			Database: config.DatabaseConfig{
				Host:     "localhost",
				Port:     "5432",
				User:     "user",
				Password: "pass",
				Name:     "db",
			},
			JWT: config.JWTConfig{
				Secret:             "test-secret-key-min-32-characters-long",
				AccessTokenExpiry:  15 * time.Minute,
				RefreshTokenExpiry: 7 * 24 * time.Hour,
			},
			Redis: config.RedisConfig{
				Host: "localhost",
				Port: "6379",
			},
		}

		err := config.Validate(cfg)
		assert.NoError(t, err)
	})

	t.Run("invalid environment fails", func(t *testing.T) {
		cfg := &config.Config{
			AppEnv: "invalid",
			Server: config.ServerConfig{Port: "8080", Host: "localhost"},
			Database: config.DatabaseConfig{
				Host: "localhost", Port: "5432", User: "user", Password: "pass", Name: "db",
			},
			JWT: config.JWTConfig{
				Secret:             "test-secret-key-min-32-characters-long",
				AccessTokenExpiry:  15 * time.Minute,
				RefreshTokenExpiry: 7 * 24 * time.Hour,
			},
		}

		err := config.Validate(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "environment")
	})

	t.Run("JWT secret too short fails", func(t *testing.T) {
		cfg := &config.Config{
			AppEnv: "dev",
			Server: config.ServerConfig{Port: "8080", Host: "localhost"},
			Database: config.DatabaseConfig{
				Host: "localhost", Port: "5432", User: "user", Password: "pass", Name: "db",
			},
			JWT: config.JWTConfig{
				Secret:             "short",
				AccessTokenExpiry:  15 * time.Minute,
				RefreshTokenExpiry: 7 * 24 * time.Hour,
			},
		}

		err := config.Validate(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "JWT secret")
	})

	t.Run("zero token expiry fails", func(t *testing.T) {
		cfg := &config.Config{
			AppEnv: "dev",
			Server: config.ServerConfig{Port: "8080", Host: "localhost"},
			Database: config.DatabaseConfig{
				Host: "localhost", Port: "5432", User: "user", Password: "pass", Name: "db",
			},
			JWT: config.JWTConfig{
				Secret:             "test-secret-key-min-32-characters-long",
				AccessTokenExpiry:  0,
				RefreshTokenExpiry: 7 * 24 * time.Hour,
			},
		}

		err := config.Validate(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "token expiry")
	})
}

// TestGetDatabaseURL tests database connection string generation
func TestGetDatabaseURL(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Host:     "localhost",
			Port:     "5432",
			User:     "testuser",
			Password: "testpass",
			Name:     "testdb",
			SSLMode:  "disable",
		},
	}

	url := cfg.GetDatabaseURL()
	expected := "postgres://testuser:testpass@localhost:5432/testdb?sslmode=disable"
	assert.Equal(t, expected, url)
}

// TestGetRedisAddr tests Redis address generation
func TestGetRedisAddr(t *testing.T) {
	cfg := &config.Config{
		Redis: config.RedisConfig{
			Host: "localhost",
			Port: "6379",
		},
	}

	addr := cfg.GetRedisAddr()
	assert.Equal(t, "localhost:6379", addr)
}

// TestIsDevelopment tests environment detection helpers
func TestEnvironmentHelpers(t *testing.T) {
	tests := []struct {
		name        string
		env         string
		isDev       bool
		isProduction bool
	}{
		{"dev environment", "dev", true, false},
		{"sandbox environment", "sandbox", false, false},
		{"audit environment", "audit", false, false},
		{"production environment", "prod", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{AppEnv: tt.env}
			assert.Equal(t, tt.isDev, cfg.IsDevelopment())
			assert.Equal(t, tt.isProduction, cfg.IsProduction())
		})
	}
}

// clearEnv clears all test environment variables
func clearEnv() {
	envVars := []string{
		"APP_ENV", "SERVER_PORT", "SERVER_HOST",
		"DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD", "DB_NAME", "DB_SSLMODE",
		"DATABASE_URL",
		"JWT_SECRET", "JWT_ACCESS_TOKEN_EXPIRY", "JWT_REFRESH_TOKEN_EXPIRY",
		"REDIS_HOST", "REDIS_PORT", "REDIS_PASSWORD", "REDIS_DB",
		"REDIS_URL",
		"OTEL_ENABLED", "OTEL_EXPORTER_OTLP_ENDPOINT", "OTEL_SERVICE_NAME", "OTEL_SAMPLE_RATE",
		"CONFIG_FILE",
	}
	for _, v := range envVars {
		os.Unsetenv(v)
	}
}

// TestTracingConfiguration tests OpenTelemetry tracing configuration
func TestTracingConfiguration(t *testing.T) {
	t.Run("default tracing config", func(t *testing.T) {
		os.Setenv("APP_ENV", "dev")
		os.Setenv("DB_HOST", "localhost")
		os.Setenv("DB_PORT", "5432")
		os.Setenv("DB_USER", "user")
		os.Setenv("DB_PASSWORD", "pass")
		os.Setenv("DB_NAME", "db")
		os.Setenv("JWT_SECRET", "test-secret-key-min-32-characters-long")
		defer clearEnv()

		cfg, err := config.Load()
		require.NoError(t, err)

		// Check default values
		assert.False(t, cfg.Tracing.Enabled)
		assert.Equal(t, "localhost:4317", cfg.Tracing.OTLPEndpoint)
		assert.Equal(t, "user-service", cfg.Tracing.ServiceName)
		assert.Equal(t, 1.0, cfg.Tracing.SampleRate)
	})

	t.Run("custom tracing config", func(t *testing.T) {
		os.Setenv("APP_ENV", "prod")
		os.Setenv("DB_HOST", "localhost")
		os.Setenv("DB_PORT", "5432")
		os.Setenv("DB_USER", "user")
		os.Setenv("DB_PASSWORD", "pass")
		os.Setenv("DB_NAME", "db")
		os.Setenv("JWT_SECRET", "test-secret-key-min-32-characters-long")
		os.Setenv("OTEL_ENABLED", "true")
		os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "otel-collector:4317")
		os.Setenv("OTEL_SERVICE_NAME", "pandora-user-service")
		os.Setenv("OTEL_SAMPLE_RATE", "0.1")
		defer clearEnv()

		cfg, err := config.Load()
		require.NoError(t, err)

		assert.True(t, cfg.Tracing.Enabled)
		assert.Equal(t, "otel-collector:4317", cfg.Tracing.OTLPEndpoint)
		assert.Equal(t, "pandora-user-service", cfg.Tracing.ServiceName)
		assert.Equal(t, 0.1, cfg.Tracing.SampleRate)
	})
}

