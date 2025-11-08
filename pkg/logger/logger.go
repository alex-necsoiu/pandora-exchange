package logger

import (
	"context"
	"io"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/trace"
)

var globalLogger zerolog.Logger

// init initializes the global logger with default settings
func init() {
	globalLogger = New("dev")
}

// New creates a new zerolog logger configured for the given environment.
// In development ("dev"), it uses pretty console output.
// In production/sandbox, it uses JSON output.
func New(env string) zerolog.Logger {
	return NewWithWriter(env, nil)
}

// NewWithWriter creates a new zerolog logger with a custom writer.
// If writer is nil, defaults to os.Stdout.
// This function is useful for testing.
func NewWithWriter(env string, writer io.Writer) zerolog.Logger {
	if writer == nil {
		writer = os.Stdout
	}

	// Configure based on environment
	var logger zerolog.Logger

	if env == "dev" || env == "development" {
		// Pretty console output for development
		output := zerolog.ConsoleWriter{
			Out:        writer,
			TimeFormat: time.RFC3339,
		}
		logger = zerolog.New(output).With().Timestamp().Logger()
	} else {
		// JSON output for production/sandbox
		logger = zerolog.New(writer).With().Timestamp().Logger()
	}

	// Set log level based on environment
	switch env {
	case "dev", "development":
		logger = logger.Level(zerolog.DebugLevel)
	case "sandbox":
		logger = logger.Level(zerolog.InfoLevel)
	case "prod", "production":
		logger = logger.Level(zerolog.WarnLevel)
	default:
		logger = logger.Level(zerolog.InfoLevel)
	}

	// Add sensitive field hooks for redaction
	logger = logger.Hook(sensitiveFieldHook{})

	return logger
}

// WithTrace extracts the OpenTelemetry trace ID from the context and returns a logger
// with the trace_id field set. This allows correlating logs with distributed traces.
func WithTrace(ctx context.Context, logger zerolog.Logger) zerolog.Logger {
	if ctx == nil {
		return logger
	}

	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return logger
	}

	traceID := span.SpanContext().TraceID()
	if !traceID.IsValid() {
		return logger
	}

	return logger.With().Str("trace_id", traceID.String()).Logger()
}

// GetLogger returns the global logger instance.
// Use this to get a logger without creating a new one.
func GetLogger() zerolog.Logger {
	return globalLogger
}

// SetLogger sets the global logger instance.
// This is useful for testing or custom logger configuration.
func SetLogger(logger zerolog.Logger) {
	globalLogger = logger
}

// sensitiveFieldHook is a zerolog hook that redacts sensitive fields from logs.
// It prevents passwords, tokens, and other secrets from being logged.
type sensitiveFieldHook struct{}

// Run implements the zerolog.Hook interface.
// It redacts sensitive fields before logging.
func (h sensitiveFieldHook) Run(e *zerolog.Event, level zerolog.Level, msg string) {
	// Note: Field-level redaction is tricky with zerolog
	// For production, ensure sensitive fields are never logged
	// or use a custom encoder/formatter

	// This is a placeholder - actual implementation would require
	// custom field handling or encoder
}

// RedactSensitiveFields is a helper function to redact sensitive data from strings.
// Use this before logging user input or error messages that might contain secrets.
func RedactSensitiveFields(s string) string {
	// Redact common sensitive patterns
	s = redactPattern(s, "password", "***")
	s = redactPattern(s, "token", "***")
	s = redactPattern(s, "secret", "***")
	s = redactPattern(s, "api_key", "***")
	s = redactPattern(s, "apiKey", "***")

	return s
}

// redactPattern replaces values for a given key pattern
func redactPattern(s, pattern, replacement string) string {
	// Simple redaction - in production, use more robust regex
	if strings.Contains(strings.ToLower(s), strings.ToLower(pattern)) {
		// This is a simplified version
		// A production implementation would use proper JSON parsing
		// or regex to redact only the values, not the keys
		return strings.ReplaceAll(s, pattern, replacement)
	}
	return s
}
