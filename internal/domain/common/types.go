// Package common contains shared domain types and value objects used across multiple domain packages.
package common

import (
	"errors"
	"fmt"
	"net/http"
)

// ErrInternalServer is returned for unexpected internal errors.
var ErrInternalServer = errors.New("internal server error")

// AppError represents a structured application error with HTTP context.
// It wraps domain errors and adds metadata for API responses and tracing.
type AppError struct {
	// Err is the underlying domain error
	Err error `json:"-"`

	// Code is the machine-readable error code (e.g., "USER_NOT_FOUND")
	Code string `json:"error"`

	// Message is the human-readable error message
	Message string `json:"message"`

	// TraceID is the OpenTelemetry trace ID for request correlation
	TraceID string `json:"trace_id,omitempty"`

	// HTTPStatus is the HTTP status code to return
	HTTPStatus int `json:"-"`

	// Details contains additional error context (optional)
	Details map[string]interface{} `json:"details,omitempty"`
}

// NewAppError creates an AppError from a domain error.
//
// Parameters:
//   - err: The underlying domain error (should be a sentinel error)
//   - message: Human-readable error message (will be sanitized if err is internal)
//   - traceID: OpenTelemetry trace ID for request correlation
//
// Returns:
//   - *AppError: Structured error with HTTP context
//
// Security: Internal errors are sanitized to prevent information leakage.
func NewAppError(err error, message string, traceID string) *AppError {
	code := MapErrorToCode(err)
	httpStatus := MapErrorToHTTPStatus(err)

	// Sanitize message for unknown/internal errors
	if errors.Is(err, ErrInternalServer) || !isKnownDomainError(err) {
		message = "An unexpected error occurred"
	}

	return &AppError{
		Err:        err,
		Code:       code,
		Message:    message,
		TraceID:    traceID,
		HTTPStatus: httpStatus,
	}
}

// Error implements the error interface.
func (e *AppError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap implements the errors.Unwrap interface for errors.Is and errors.As support.
func (e *AppError) Unwrap() error {
	return e.Err
}

// WithDetails adds additional context to the error.
// This is useful for validation errors or providing field-specific information.
//
// Example:
//
//	err.WithDetails(map[string]interface{}{
//	    "field": "email",
//	    "reason": "invalid format",
//	})
func (e *AppError) WithDetails(details map[string]interface{}) *AppError {
	e.Details = details
	return e
}

// ToJSON returns a JSON-serializable representation of the error.
// This is used by HTTP handlers to format error responses.
func (e *AppError) ToJSON() map[string]interface{} {
	response := map[string]interface{}{
		"error":   e.Code,
		"message": e.Message,
	}

	if e.TraceID != "" {
		response["trace_id"] = e.TraceID
	}

	if e.Details != nil {
		response["details"] = e.Details
	}

	return response
}

// MapErrorToHTTPStatus maps domain errors to HTTP status codes.
// Unknown errors default to 500 Internal Server Error.
//
// Mapping:
//   - 404 Not Found: Resource not found errors
//   - 409 Conflict: Resource already exists
//   - 401 Unauthorized: Authentication failures
//   - 403 Forbidden: Authorization failures, expired/revoked tokens
//   - 400 Bad Request: Validation errors
//   - 500 Internal Server Error: Unknown/internal errors
func MapErrorToHTTPStatus(err error) int {
	// Import domain errors for mapping
	// This will be updated after we reorganize the imports
	// For now, we'll check error messages
	errMsg := err.Error()

	switch {
	// 404 Not Found
	case containsAny(errMsg, "not found"):
		return http.StatusNotFound

	// 409 Conflict
	case containsAny(errMsg, "already exists"):
		return http.StatusConflict

	// 401 Unauthorized
	case containsAny(errMsg, "invalid email or password", "unauthorized", "access token expired", "invalid access token"):
		return http.StatusUnauthorized

	// 403 Forbidden
	case containsAny(errMsg, "forbidden", "deleted", "token expired", "token revoked"):
		return http.StatusForbidden

	// 400 Bad Request
	case containsAny(errMsg, "invalid", "weak password"):
		return http.StatusBadRequest

	// 500 Internal Server Error (default)
	default:
		return http.StatusInternalServerError
	}
}

// MapErrorToCode maps domain errors to machine-readable error codes.
// These codes are stable and can be used by clients for error handling.
func MapErrorToCode(err error) string {
	errMsg := err.Error()

	switch {
	case errMsg == "user not found":
		return "USER_NOT_FOUND"
	case errMsg == "user already exists":
		return "USER_ALREADY_EXISTS"
	case errMsg == "user has been deleted":
		return "USER_DELETED"
	case errMsg == "invalid email or password":
		return "INVALID_CREDENTIALS"
	case errMsg == "invalid KYC status":
		return "INVALID_KYC_STATUS"
	case errMsg == "invalid email format":
		return "INVALID_EMAIL"
	case errMsg == "password does not meet security requirements":
		return "WEAK_PASSWORD"
	case errMsg == "refresh token not found":
		return "REFRESH_TOKEN_NOT_FOUND"
	case errMsg == "refresh token has expired":
		return "REFRESH_TOKEN_EXPIRED"
	case errMsg == "refresh token has been revoked":
		return "REFRESH_TOKEN_REVOKED"
	case errMsg == "invalid refresh token":
		return "INVALID_REFRESH_TOKEN"
	case errMsg == "access token has expired":
		return "ACCESS_TOKEN_EXPIRED"
	case errMsg == "invalid access token":
		return "INVALID_ACCESS_TOKEN"
	case errMsg == "unauthorized":
		return "UNAUTHORIZED"
	case errMsg == "forbidden":
		return "FORBIDDEN"
	case errMsg == "token not found":
		return "TOKEN_NOT_FOUND"
	case errMsg == "invalid role":
		return "INVALID_ROLE"
	case errMsg == "invalid input":
		return "INVALID_INPUT"
	default:
		return "INTERNAL_SERVER_ERROR"
	}
}

// isKnownDomainError checks if an error is a known domain error.
func isKnownDomainError(err error) bool {
	// Check if the error matches any known domain error patterns
	errMsg := err.Error()
	knownErrors := []string{
		"user not found",
		"user already exists",
		"user has been deleted",
		"invalid email or password",
		"invalid KYC status",
		"invalid email format",
		"password does not meet security requirements",
		"refresh token not found",
		"refresh token has expired",
		"refresh token has been revoked",
		"invalid refresh token",
		"access token has expired",
		"invalid access token",
		"unauthorized",
		"forbidden",
		"token not found",
		"invalid role",
		"invalid input",
	}

	for _, known := range knownErrors {
		if errMsg == known {
			return true
		}
	}
	return false
}

// containsAny checks if a string contains any of the given substrings
func containsAny(s string, substrs ...string) bool {
	for _, substr := range substrs {
		if contains(s, substr) {
			return true
		}
	}
	return false
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || string(s[0:len(substr)]) == substr || containsInner(s[1:], substr))
}

func containsInner(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// EventPublisher defines the interface for publishing domain events.
// This interface is implemented by the infrastructure layer (Redis Streams).
type EventPublisher interface {
	// Publish publishes a single event
	Publish(event interface{}) error

	// PublishBatch publishes multiple events in a batch
	PublishBatch(events []interface{}) error

	// Close closes the publisher and releases resources
	Close() error
}
