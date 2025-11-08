package errors

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/trace"
)

// AppError represents an application error with HTTP status code, error code, and trace ID.
// It is used to provide structured error responses to clients.
type AppError struct {
	// Code is a machine-readable error code (e.g., "USER_NOT_FOUND")
	Code string `json:"code"`

	// Message is a human-readable error message
	Message string `json:"message"`

	// HTTPStatus is the HTTP status code to return
	HTTPStatus int `json:"-"`

	// TraceID is the OpenTelemetry trace ID for request correlation
	TraceID string `json:"trace_id,omitempty"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// NewAppError creates a new AppError with the given parameters.
// It extracts the trace ID from the context if available.
func NewAppError(ctx context.Context, code, message string, httpStatus int) *AppError {
	traceID := extractTraceID(ctx)

	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
		TraceID:    traceID,
	}
}

// Wrap wraps an error as an AppError with the given code and HTTP status.
// It preserves the original error message and extracts trace ID from context.
func Wrap(ctx context.Context, err error, code string, httpStatus int) *AppError {
	traceID := extractTraceID(ctx)

	return &AppError{
		Code:       code,
		Message:    err.Error(),
		HTTPStatus: httpStatus,
		TraceID:    traceID,
	}
}

// WrapWithMessage wraps an error as an AppError with a custom message.
// It allows overriding the error message while preserving context and trace ID.
func WrapWithMessage(ctx context.Context, err error, code, message string, httpStatus int) *AppError {
	traceID := extractTraceID(ctx)

	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
		TraceID:    traceID,
	}
}

// IsAppError checks if an error is an AppError using type assertion.
func IsAppError(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(*AppError)
	return ok
}

// extractTraceID extracts the OpenTelemetry trace ID from the context.
// Returns an empty string if no trace ID is found.
func extractTraceID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return ""
	}

	traceID := span.SpanContext().TraceID()
	if !traceID.IsValid() {
		return ""
	}

	return traceID.String()
}
