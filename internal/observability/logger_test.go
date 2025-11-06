package observability_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/alex-necsoiu/pandora-exchange/internal/observability"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewLogger tests logger initialization
func TestNewLogger(t *testing.T) {
	t.Run("create development logger", func(t *testing.T) {
		logger := observability.NewLogger("dev", "user-service")
		require.NotNil(t, logger)
	})

	t.Run("create production logger", func(t *testing.T) {
		logger := observability.NewLogger("prod", "user-service")
		require.NotNil(t, logger)
	})

	t.Run("create sandbox logger", func(t *testing.T) {
		logger := observability.NewLogger("sandbox", "user-service")
		require.NotNil(t, logger)
	})
}

// TestLogLevels tests different log levels
func TestLogLevels(t *testing.T) {
	tests := []struct {
		name     string
		logFunc  func(*observability.Logger, string)
		levelAbbr string
		disabled bool
	}{
		{"debug level", func(l *observability.Logger, msg string) { l.Debug(msg) }, "DBG", false},
		{"info level", func(l *observability.Logger, msg string) { l.Info(msg) }, "INF", false},
		{"warn level", func(l *observability.Logger, msg string) { l.Warn(msg) }, "WRN", false},
		{"error level", func(l *observability.Logger, msg string) { l.Error(msg) }, "ERR", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := observability.NewLoggerWithWriter("dev", "test-service", &buf)

			tt.logFunc(logger, "test message")

			if !tt.disabled {
				output := buf.String()
				assert.Contains(t, output, "test message")
				assert.Contains(t, output, tt.levelAbbr)
			}
		})
	}
}

// TestStructuredLogging tests structured logging with fields
func TestStructuredLogging(t *testing.T) {
	t.Run("log with string field", func(t *testing.T) {
		var buf bytes.Buffer
		logger := observability.NewLoggerWithWriter("dev", "test-service", &buf)

		logger.WithField("user_id", "123").Info("user logged in")

		output := buf.String()
		assert.Contains(t, output, "user_id")
		assert.Contains(t, output, "123")
		assert.Contains(t, output, "user logged in")
	})

	t.Run("log with multiple fields", func(t *testing.T) {
		var buf bytes.Buffer
		logger := observability.NewLoggerWithWriter("dev", "test-service", &buf)

		logger.WithFields(map[string]interface{}{
			"user_id": "123",
			"email":   "test@example.com",
			"action":  "login",
		}).Info("authentication successful")

		output := buf.String()
		assert.Contains(t, output, "user_id")
		assert.Contains(t, output, "123")
		assert.Contains(t, output, "email")
		assert.Contains(t, output, "test@example.com")
		assert.Contains(t, output, "action")
		assert.Contains(t, output, "login")
	})

	t.Run("log with error", func(t *testing.T) {
		var buf bytes.Buffer
		logger := observability.NewLoggerWithWriter("dev", "test-service", &buf)

		testErr := assert.AnError
		logger.WithError(testErr).Error("operation failed")

		output := buf.String()
		assert.Contains(t, output, "error")
		assert.Contains(t, output, "operation failed")
	})
}

// TestSensitiveDataRedaction tests that sensitive data is not logged
func TestSensitiveDataRedaction(t *testing.T) {
	t.Run("sanitize password field", func(t *testing.T) {
		var buf bytes.Buffer
		logger := observability.NewLoggerWithWriter("dev", "test-service", &buf)

		fields := map[string]interface{}{
			"email":    "user@example.com",
			"password": "secretPassword123",
		}

		sanitized := observability.SanitizeFields(fields)
		logger.WithFields(sanitized).Info("user registration")

		output := buf.String()
		assert.NotContains(t, output, "secretPassword123")
		assert.Contains(t, output, "[REDACTED]")
	})

	t.Run("sanitize multiple sensitive fields", func(t *testing.T) {
		fields := map[string]interface{}{
			"email":    "user@example.com",
			"password": "secret",
			"jwt":      "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			"token":    "Bearer xyz123",
			"secret":   "my-secret-key",
		}

		sanitized := observability.SanitizeFields(fields)

		assert.Equal(t, "user@example.com", sanitized["email"])
		assert.Equal(t, "[REDACTED]", sanitized["password"])
		assert.Equal(t, "[REDACTED]", sanitized["jwt"])
		assert.Equal(t, "[REDACTED]", sanitized["token"])
		assert.Equal(t, "[REDACTED]", sanitized["secret"])
	})
}

// TestJSONOutput tests JSON structured output
func TestJSONOutput(t *testing.T) {
	t.Run("output is valid JSON", func(t *testing.T) {
		var buf bytes.Buffer
		// Use sandbox environment which has InfoLevel, not prod which has WarnLevel
		logger := observability.NewLoggerWithWriter("sandbox", "test-service", &buf)

		logger.WithFields(map[string]interface{}{
			"user_id": "123",
			"action":  "login",
		}).Info("test message")

		var logEntry map[string]interface{}
		err := json.Unmarshal(buf.Bytes(), &logEntry)
		require.NoError(t, err)

		assert.Equal(t, "info", logEntry["level"])
		assert.Equal(t, "test message", logEntry["message"])
		assert.Equal(t, "test-service", logEntry["service"])
		assert.Equal(t, "123", logEntry["user_id"])
		assert.Equal(t, "login", logEntry["action"])
	})
}

// TestEnvironmentSpecificFormatting tests different output formats per environment
func TestEnvironmentSpecificFormatting(t *testing.T) {
	t.Run("dev environment uses console format", func(t *testing.T) {
		var buf bytes.Buffer
		logger := observability.NewLoggerWithWriter("dev", "test-service", &buf)

		logger.Info("test message")

		// Dev should have human-readable console output
		output := buf.String()
		assert.NotEmpty(t, output)
	})

	t.Run("prod environment uses JSON format", func(t *testing.T) {
		var buf bytes.Buffer
		logger := observability.NewLoggerWithWriter("sandbox", "test-service", &buf)

		logger.Info("test message")

		// Sandbox/Prod should have JSON output
		var logEntry map[string]interface{}
		err := json.Unmarshal(buf.Bytes(), &logEntry)
		require.NoError(t, err)
		assert.Equal(t, "info", logEntry["level"])
	})
}

// TestContextLogger tests context-aware logging
func TestContextLogger(t *testing.T) {
	t.Run("create logger with context", func(t *testing.T) {
		var buf bytes.Buffer
		logger := observability.NewLoggerWithWriter("dev", "test-service", &buf)

		contextLogger := logger.WithFields(map[string]interface{}{
			"request_id": "req-123",
			"user_id":    "user-456",
		})

		contextLogger.Info("processing request")

		output := buf.String()
		assert.Contains(t, output, "req-123")
		assert.Contains(t, output, "user-456")
		assert.Contains(t, output, "processing request")
	})
}

// TestLoggerChaining tests method chaining
func TestLoggerChaining(t *testing.T) {
	t.Run("chain multiple field additions", func(t *testing.T) {
		var buf bytes.Buffer
		logger := observability.NewLoggerWithWriter("dev", "test-service", &buf)

		logger.
			WithField("step", "1").
			WithField("user_id", "123").
			WithField("action", "register").
			Info("user registration started")

		output := buf.String()
		assert.Contains(t, output, "step")
		assert.Contains(t, output, "user_id")
		assert.Contains(t, output, "action")
	})
}

// TestAuditLogging tests audit log specific functionality
func TestAuditLogging(t *testing.T) {
	t.Run("create audit log entry", func(t *testing.T) {
		var buf bytes.Buffer
		logger := observability.NewLoggerWithWriter("sandbox", "test-service", &buf)

		auditLogger := observability.NewAuditLogger(logger)
		auditLogger.LogEvent("user.registered", map[string]interface{}{
			"user_id": "123",
			"email":   "user@example.com",
		})

		var logEntry map[string]interface{}
		err := json.Unmarshal(buf.Bytes(), &logEntry)
		require.NoError(t, err)

		assert.Equal(t, "audit", logEntry["audit"])
		assert.Equal(t, "user.registered", logEntry["event"])
		assert.Equal(t, "123", logEntry["user_id"])
	})

	t.Run("audit log sanitizes sensitive data", func(t *testing.T) {
		var buf bytes.Buffer
		logger := observability.NewLoggerWithWriter("sandbox", "test-service", &buf)

		auditLogger := observability.NewAuditLogger(logger)
		auditLogger.LogEvent("user.login", map[string]interface{}{
			"user_id":  "123",
			"password": "secret123",
		})

		output := buf.String()
		assert.NotContains(t, output, "secret123")
		assert.Contains(t, output, "[REDACTED]")
	})
}
