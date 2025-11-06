package config_test

import (
	"os"
	"testing"
	"time"

	"github.com/alex-necsoiu/pandora-exchange/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
		"JWT_SECRET", "JWT_ACCESS_TOKEN_EXPIRY", "JWT_REFRESH_TOKEN_EXPIRY",
		"REDIS_HOST", "REDIS_PORT", "REDIS_PASSWORD", "REDIS_DB",
	}
	for _, v := range envVars {
		os.Unsetenv(v)
	}
}
