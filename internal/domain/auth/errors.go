package auth

import "errors"

// Domain-level errors for authentication operations.
var (
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
)
