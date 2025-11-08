package domain

import (
	"errors"
	"fmt"
	"net/http"
)

// Domain-level errors for the User Service.
// These errors represent business logic failures, not infrastructure failures.
var (
	// ErrUserNotFound is returned when a user cannot be found.
	ErrUserNotFound = errors.New("user not found")

	// ErrUserAlreadyExists is returned when attempting to create a user with an existing email.
	ErrUserAlreadyExists = errors.New("user already exists")

	// ErrUserDeleted is returned when attempting to access a soft-deleted user.
	ErrUserDeleted = errors.New("user has been deleted")

	// ErrInvalidCredentials is returned when login credentials are incorrect.
	ErrInvalidCredentials = errors.New("invalid email or password")

	// ErrInvalidKYCStatus is returned when an invalid KYC status is provided.
	ErrInvalidKYCStatus = errors.New("invalid KYC status")

	// ErrInvalidEmail is returned when email format is invalid.
	ErrInvalidEmail = errors.New("invalid email format")

	// ErrWeakPassword is returned when password doesn't meet security requirements.
	ErrWeakPassword = errors.New("password does not meet security requirements")

	// ErrRefreshTokenNotFound is returned when a refresh token cannot be found.
	ErrRefreshTokenNotFound = errors.New("refresh token not found")

	// ErrRefreshTokenExpired is returned when a refresh token has expired.
	ErrRefreshTokenExpired = errors.New("refresh token has expired")

	// ErrRefreshTokenRevoked is returned when a refresh token has been revoked.
	ErrRefreshTokenRevoked = errors.New("refresh token has been revoked")

	// ErrInvalidRefreshToken is returned when a refresh token is invalid.
	ErrInvalidRefreshToken = errors.New("invalid refresh token")

	// ErrAccessTokenExpired is returned when an access token has expired.
	ErrAccessTokenExpired = errors.New("access token has expired")

	// ErrInvalidAccessToken is returned when an access token is invalid.
	ErrInvalidAccessToken = errors.New("invalid access token")

	// ErrUnauthorized is returned when a user is not authorized to perform an action.
	ErrUnauthorized = errors.New("unauthorized")

	// ErrForbidden is returned when a user is forbidden from performing an action.
	ErrForbidden = errors.New("forbidden")

	// ErrTokenNotFound is returned when a token cannot be found.
	ErrTokenNotFound = errors.New("token not found")

	// ErrInvalidRole is returned when an invalid role is provided.
	ErrInvalidRole = errors.New("invalid role")

	// ErrInvalidInput is returned when input validation fails.
	ErrInvalidInput = errors.New("invalid input")

	// ErrInternalServer is returned for unexpected internal errors.
	ErrInternalServer = errors.New("internal server error")
)

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
//   err.WithDetails(map[string]interface{}{
//       "field": "email",
//       "reason": "invalid format",
//   })
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
	switch {
	// 404 Not Found
	case errors.Is(err, ErrUserNotFound),
		errors.Is(err, ErrTokenNotFound),
		errors.Is(err, ErrRefreshTokenNotFound):
		return http.StatusNotFound
	
	// 409 Conflict
	case errors.Is(err, ErrUserAlreadyExists):
		return http.StatusConflict
	
	// 401 Unauthorized
	case errors.Is(err, ErrInvalidCredentials),
		errors.Is(err, ErrUnauthorized),
		errors.Is(err, ErrAccessTokenExpired),
		errors.Is(err, ErrInvalidAccessToken):
		return http.StatusUnauthorized
	
	// 403 Forbidden
	case errors.Is(err, ErrForbidden),
		errors.Is(err, ErrUserDeleted),
		errors.Is(err, ErrRefreshTokenExpired),
		errors.Is(err, ErrRefreshTokenRevoked):
		return http.StatusForbidden
	
	// 400 Bad Request
	case errors.Is(err, ErrInvalidInput),
		errors.Is(err, ErrInvalidEmail),
		errors.Is(err, ErrWeakPassword),
		errors.Is(err, ErrInvalidKYCStatus),
		errors.Is(err, ErrInvalidRole),
		errors.Is(err, ErrInvalidRefreshToken):
		return http.StatusBadRequest
	
	// 500 Internal Server Error (default)
	default:
		return http.StatusInternalServerError
	}
}

// MapErrorToCode maps domain errors to machine-readable error codes.
// These codes are stable and can be used by clients for error handling.
//
// Error codes follow the pattern: RESOURCE_ACTION_REASON (e.g., "USER_NOT_FOUND")
func MapErrorToCode(err error) string {
	switch {
	case errors.Is(err, ErrUserNotFound):
		return "USER_NOT_FOUND"
	case errors.Is(err, ErrUserAlreadyExists):
		return "USER_ALREADY_EXISTS"
	case errors.Is(err, ErrUserDeleted):
		return "USER_DELETED"
	case errors.Is(err, ErrInvalidCredentials):
		return "INVALID_CREDENTIALS"
	case errors.Is(err, ErrInvalidKYCStatus):
		return "INVALID_KYC_STATUS"
	case errors.Is(err, ErrInvalidEmail):
		return "INVALID_EMAIL"
	case errors.Is(err, ErrWeakPassword):
		return "WEAK_PASSWORD"
	case errors.Is(err, ErrRefreshTokenNotFound):
		return "REFRESH_TOKEN_NOT_FOUND"
	case errors.Is(err, ErrRefreshTokenExpired):
		return "REFRESH_TOKEN_EXPIRED"
	case errors.Is(err, ErrRefreshTokenRevoked):
		return "REFRESH_TOKEN_REVOKED"
	case errors.Is(err, ErrInvalidRefreshToken):
		return "INVALID_REFRESH_TOKEN"
	case errors.Is(err, ErrAccessTokenExpired):
		return "ACCESS_TOKEN_EXPIRED"
	case errors.Is(err, ErrInvalidAccessToken):
		return "INVALID_ACCESS_TOKEN"
	case errors.Is(err, ErrUnauthorized):
		return "UNAUTHORIZED"
	case errors.Is(err, ErrForbidden):
		return "FORBIDDEN"
	case errors.Is(err, ErrTokenNotFound):
		return "TOKEN_NOT_FOUND"
	case errors.Is(err, ErrInvalidRole):
		return "INVALID_ROLE"
	case errors.Is(err, ErrInvalidInput):
		return "INVALID_INPUT"
	default:
		return "INTERNAL_SERVER_ERROR"
	}
}

// isKnownDomainError checks if an error is a known domain error.
// This is used to determine whether to sanitize error messages.
func isKnownDomainError(err error) bool {
	knownErrors := []error{
		ErrUserNotFound,
		ErrUserAlreadyExists,
		ErrUserDeleted,
		ErrInvalidCredentials,
		ErrInvalidKYCStatus,
		ErrInvalidEmail,
		ErrWeakPassword,
		ErrRefreshTokenNotFound,
		ErrRefreshTokenExpired,
		ErrRefreshTokenRevoked,
		ErrInvalidRefreshToken,
		ErrAccessTokenExpired,
		ErrInvalidAccessToken,
		ErrUnauthorized,
		ErrForbidden,
		ErrTokenNotFound,
		ErrInvalidRole,
		ErrInvalidInput,
	}
	
	for _, known := range knownErrors {
		if errors.Is(err, known) {
			return true
		}
	}
	
	return false
}
