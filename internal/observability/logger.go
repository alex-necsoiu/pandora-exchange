// Package observability provides structured logging, tracing, and metrics for the User Service.
// Uses zerolog for high-performance structured logging with security-first design.
package observability

import (
	"io"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// Logger wraps zerolog.Logger with additional functionality
type Logger struct {
	logger  zerolog.Logger
	service string
}

// AuditLogger handles audit-specific logging for compliance
type AuditLogger struct {
	logger *Logger
}

// NewLogger creates a new logger instance configured for the environment
// env: dev, sandbox, audit, prod
// serviceName: name of the service (e.g., "user-service")
func NewLogger(env, serviceName string) *Logger {
	return NewLoggerWithWriter(env, serviceName, os.Stdout)
}

// NewLoggerWithWriter creates a logger with a custom writer (useful for testing)
func NewLoggerWithWriter(env, serviceName string, w io.Writer) *Logger {
	// Configure based on environment
	var logger zerolog.Logger
	var logLevel zerolog.Level

	// Set log level based on environment
	switch env {
	case "dev":
		logLevel = zerolog.DebugLevel
	case "sandbox", "audit":
		logLevel = zerolog.InfoLevel
	case "prod":
		logLevel = zerolog.WarnLevel
	default:
		logLevel = zerolog.InfoLevel
	}

	if env == "dev" {
		// Development: human-readable console output with colors
		output := zerolog.ConsoleWriter{
			Out:        w,
			TimeFormat: time.RFC3339,
			NoColor:    false,
		}
		logger = zerolog.New(output).Level(logLevel).With().Timestamp().Logger()
	} else {
		// Production/Sandbox/Audit: JSON output for log aggregation
		logger = zerolog.New(w).Level(logLevel).With().Timestamp().Logger()
	}

	// Add service name to all logs
	logger = logger.With().Str("service", serviceName).Logger()

	return &Logger{
		logger:  logger,
		service: serviceName,
	}
}

// Debug logs a debug message
func (l *Logger) Debug(msg string) {
	l.logger.Debug().Msg(msg)
}

// Info logs an info message
func (l *Logger) Info(msg string) {
	l.logger.Info().Msg(msg)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string) {
	l.logger.Warn().Msg(msg)
}

// Error logs an error message
func (l *Logger) Error(msg string) {
	l.logger.Error().Msg(msg)
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(msg string) {
	l.logger.Fatal().Msg(msg)
}

// WithField adds a single field to the logger context
func (l *Logger) WithField(key string, value interface{}) *Logger {
	return &Logger{
		logger:  l.logger.With().Interface(key, value).Logger(),
		service: l.service,
	}
}

// WithFields adds multiple fields to the logger context
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	// Sanitize fields to prevent sensitive data leakage
	sanitized := SanitizeFields(fields)

	ctx := l.logger.With()
	for k, v := range sanitized {
		ctx = ctx.Interface(k, v)
	}

	return &Logger{
		logger:  ctx.Logger(),
		service: l.service,
	}
}

// WithError adds an error to the logger context
func (l *Logger) WithError(err error) *Logger {
	return &Logger{
		logger:  l.logger.With().Err(err).Logger(),
		service: l.service,
	}
}

// SanitizeFields removes or redacts sensitive information from log fields
// Prevents passwords, tokens, secrets, and other sensitive data from being logged
func SanitizeFields(fields map[string]interface{}) map[string]interface{} {
	sensitiveKeys := []string{
		"password", "passwd", "pwd",
		"secret", "api_key", "apikey",
		"token", "access_token", "refresh_token",
		"jwt", "bearer",
		"authorization", "auth",
		"credit_card", "cvv", "ssn",
		"private_key", "key",
	}

	sanitized := make(map[string]interface{})
	for k, v := range fields {
		keyLower := strings.ToLower(k)
		redacted := false

		for _, sensitive := range sensitiveKeys {
			if strings.Contains(keyLower, sensitive) {
				sanitized[k] = "[REDACTED]"
				redacted = true
				break
			}
		}

		if !redacted {
			sanitized[k] = v
		}
	}

	return sanitized
}

// NewAuditLogger creates a new audit logger for compliance logging
// Audit logs are immutable and include additional metadata
func NewAuditLogger(logger *Logger) *AuditLogger {
	return &AuditLogger{logger: logger}
}

// LogEvent logs an audit event with structured data
// eventType: semantic event identifier (e.g., "user.registered", "user.kyc_verified")
// data: event-specific structured data
func (a *AuditLogger) LogEvent(eventType string, data map[string]interface{}) {
	// Sanitize data to prevent sensitive information in audit logs
	sanitized := SanitizeFields(data)

	// Add audit marker and event type
	fields := make(map[string]interface{})
	fields["audit"] = "audit"
	fields["event"] = eventType

	// Merge sanitized data
	for k, v := range sanitized {
		fields[k] = v
	}

	a.logger.WithFields(fields).Info("audit event")
}

// LogUserAction logs a user-initiated action for audit trail
func (a *AuditLogger) LogUserAction(userID, action string, metadata map[string]interface{}) {
	data := make(map[string]interface{})
	data["user_id"] = userID
	data["action"] = action

	for k, v := range metadata {
		data[k] = v
	}

	a.LogEvent("user.action", data)
}

// LogSecurityEvent logs security-related events
func (a *AuditLogger) LogSecurityEvent(eventType, severity string, data map[string]interface{}) {
	fields := make(map[string]interface{})
	fields["security_event"] = eventType
	fields["severity"] = severity

	for k, v := range data {
		fields[k] = v
	}

	a.LogEvent("security.event", fields)
}
