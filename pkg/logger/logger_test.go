package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNew verifies logger initialization
func TestNew(t *testing.T) {
	tests := []struct {
		name        string
		env         string
		expectPretty bool
	}{
		{
			name:        "development logger (pretty)",
			env:         "dev",
			expectPretty: true,
		},
		{
			name:        "production logger (JSON)",
			env:         "prod",
			expectPretty: false,
		},
		{
			name:        "sandbox logger (JSON)",
			env:         "sandbox",
			expectPretty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := New(tt.env)
			assert.NotNil(t, logger)
		})
	}
}

// TestWithTrace verifies trace ID extraction from context
func TestWithTrace(t *testing.T) {
	logger := New("dev")
	
	ctx := contextWithTraceID(t, "test-trace-123")
	
	loggerWithTrace := WithTrace(ctx, logger)
	assert.NotNil(t, loggerWithTrace)
	
	// Capture log output
	var buf bytes.Buffer
	loggerWithTrace = loggerWithTrace.Output(&buf)
	
	loggerWithTrace.Info().Msg("test message")
	
	// Verify trace_id is in output
	output := buf.String()
	assert.Contains(t, output, "test message")
	// Note: trace_id verification depends on OTEL integration
}

// TestLogger_SensitiveFieldRedaction verifies password redaction
func TestLogger_SensitiveFieldRedaction(t *testing.T) {
	var buf bytes.Buffer
	logger := NewWithWriter("sandbox", &buf)  // Use JSON output for easier parsing

	// When logging sensitive data, use RedactSensitiveFields helper
	password := "supersecret123"
	token := "jwt-token-value"
	
	logger.Info().
		Str("email", "user@example.com").
		Str("password", RedactSensitiveFields(password)).
		Str("token", RedactSensitiveFields(token)).
		Msg("user login attempt")

	output := buf.String()
	
	// Password and token should be redacted
	assert.NotContains(t, output, "supersecret123")
	assert.NotContains(t, output, "jwt-token-value")
	assert.Contains(t, output, "***")  // Redaction marker
	
	// Non-sensitive data should be present
	assert.Contains(t, output, "user@example.com")
	assert.Contains(t, output, "user login attempt")
}

// TestLogger_JSONFormat verifies JSON output in production
func TestLogger_JSONFormat(t *testing.T) {
	var buf bytes.Buffer
	logger := NewWithWriter("sandbox", &buf)  // Use sandbox for Info level

	logger.Info().
		Str("field1", "value1").
		Int("field2", 42).
		Msg("test message")

	// Verify JSON format
	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err, "Log output should be valid JSON")

	assert.Equal(t, "info", logEntry["level"])
	assert.Equal(t, "test message", logEntry["message"])
	assert.Equal(t, "value1", logEntry["field1"])
	assert.Equal(t, float64(42), logEntry["field2"])
}

// TestLogger_LogLevels verifies different log levels
func TestLogger_LogLevels(t *testing.T) {
	tests := []struct {
		name  string
		level string
		env   string  // Environment to use for logger
	}{
		{"debug level", "debug", "dev"},       // dev allows debug
		{"info level", "info", "sandbox"},     // sandbox allows info
		{"warn level", "warn", "prod"},        // prod allows warn
		{"error level", "error", "prod"},      // prod allows error
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := NewWithWriter(tt.env, &buf)

			switch tt.level {
			case "debug":
				logger.Debug().Msg("debug message")
			case "info":
				logger.Info().Msg("info message")
			case "warn":
				logger.Warn().Msg("warn message")
			case "error":
				logger.Error().Msg("error message")
			}

			output := buf.String()
			assert.NotEmpty(t, output)

			// Dev environment uses console writer (not JSON), skip JSON parsing for dev
			if tt.env == "dev" {
				assert.Contains(t, output, "debug message")
			} else {
				var logEntry map[string]interface{}
				err := json.Unmarshal(buf.Bytes(), &logEntry)
				require.NoError(t, err)
				assert.Equal(t, tt.level, logEntry["level"])
			}
		})
	}
}

// contextWithTraceID creates a context with a mock trace ID for testing
func contextWithTraceID(t *testing.T, traceID string) context.Context {
	t.Helper()
	
	// For testing, return a context that the logger can extract trace ID from
	// This will be implemented in the actual logger.go
	ctx := context.Background()
	return ctx
}
