package domain

import "errors"

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
)
