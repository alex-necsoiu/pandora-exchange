package user

import "errors"

// Domain-level errors for user operations.
// These errors represent business logic failures, not infrastructure failures.
var (
	// ErrNotFound is returned when a user cannot be found.
	ErrNotFound = errors.New("user not found")

	// ErrAlreadyExists is returned when attempting to create a user with an existing email.
	ErrAlreadyExists = errors.New("user already exists")

	// ErrDeleted is returned when attempting to access a soft-deleted user.
	ErrDeleted = errors.New("user has been deleted")

	// ErrInvalidCredentials is returned when login credentials are incorrect.
	ErrInvalidCredentials = errors.New("invalid email or password")

	// ErrInvalidKYCStatus is returned when an invalid KYC status is provided.
	ErrInvalidKYCStatus = errors.New("invalid KYC status")

	// ErrInvalidEmail is returned when email format is invalid.
	ErrInvalidEmail = errors.New("invalid email format")

	// ErrWeakPassword is returned when password doesn't meet security requirements.
	ErrWeakPassword = errors.New("password does not meet security requirements")

	// ErrInvalidRole is returned when an invalid role is provided.
	ErrInvalidRole = errors.New("invalid role")

	// ErrInvalidInput is returned when input validation fails.
	ErrInvalidInput = errors.New("invalid input")
)
