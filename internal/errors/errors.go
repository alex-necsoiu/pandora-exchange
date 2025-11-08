package errors

import "errors"

// Domain errors - sentinel errors for common domain failures
// These errors are used throughout the application and mapped to appropriate
// HTTP status codes and gRPC error codes.
var (
	// ErrUserNotFound is returned when a user cannot be found by ID or email
	ErrUserNotFound = errors.New("user not found")

	// ErrUserAlreadyExists is returned when attempting to create a user with an existing email
	ErrUserAlreadyExists = errors.New("user already exists")

	// ErrInvalidCredentials is returned when login credentials are incorrect
	ErrInvalidCredentials = errors.New("invalid credentials")

	// ErrInvalidInput is returned when request validation fails
	ErrInvalidInput = errors.New("invalid input")

	// ErrInvalidToken is returned when a JWT or refresh token is invalid or expired
	ErrInvalidToken = errors.New("invalid or expired token")

	// ErrTokenExpired is returned when a token has expired
	ErrTokenExpired = errors.New("token expired")

	// ErrUnauthorized is returned when authentication is required but not provided
	ErrUnauthorized = errors.New("unauthorized")

	// ErrForbidden is returned when the user lacks permission for the requested operation
	ErrForbidden = errors.New("forbidden")

	// ErrInternal is returned for internal server errors that should not expose details
	ErrInternal = errors.New("internal server error")
)
