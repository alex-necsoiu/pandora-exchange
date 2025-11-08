package domain

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSentinelErrors verifies all sentinel errors are defined
func TestSentinelErrors(t *testing.T) {
	sentinelErrors := []error{
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
		ErrInternalServer,
	}

	for _, err := range sentinelErrors {
		assert.NotNil(t, err)
		assert.NotEmpty(t, err.Error())
	}
}

// TestNewAppError creates AppError instances
func TestNewAppError(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		message        string
		traceID        string
		expectedCode   string
		expectedStatus int
	}{
		{
			name:           "user not found",
			err:            ErrUserNotFound,
			message:        "The requested user was not found",
			traceID:        "trace-123",
			expectedCode:   "USER_NOT_FOUND",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "user already exists",
			err:            ErrUserAlreadyExists,
			message:        "A user with this email already exists",
			traceID:        "trace-456",
			expectedCode:   "USER_ALREADY_EXISTS",
			expectedStatus: http.StatusConflict,
		},
		{
			name:           "invalid credentials",
			err:            ErrInvalidCredentials,
			message:        "Invalid email or password",
			traceID:        "trace-789",
			expectedCode:   "INVALID_CREDENTIALS",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "unauthorized",
			err:            ErrUnauthorized,
			message:        "You are not authorized to access this resource",
			traceID:        "trace-abc",
			expectedCode:   "UNAUTHORIZED",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "forbidden",
			err:            ErrForbidden,
			message:        "You do not have permission to perform this action",
			traceID:        "trace-def",
			expectedCode:   "FORBIDDEN",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "invalid input",
			err:            ErrInvalidInput,
			message:        "The provided input is invalid",
			traceID:        "trace-ghi",
			expectedCode:   "INVALID_INPUT",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "internal server error",
			err:            ErrInternalServer,
			message:        "An unexpected error occurred",
			traceID:        "trace-jkl",
			expectedCode:   "INTERNAL_SERVER_ERROR",
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			appErr := NewAppError(tt.err, tt.message, tt.traceID)

			require.NotNil(t, appErr)
			assert.Equal(t, tt.expectedCode, appErr.Code)
			assert.Equal(t, tt.message, appErr.Message)
			assert.Equal(t, tt.traceID, appErr.TraceID)
			assert.Equal(t, tt.expectedStatus, appErr.HTTPStatus)
			assert.True(t, errors.Is(appErr.Err, tt.err))
		})
	}
}

// TestAppError_Error implements error interface
func TestAppError_Error(t *testing.T) {
	appErr := NewAppError(ErrUserNotFound, "User not found", "trace-123")
	
	errorString := appErr.Error()
	assert.Contains(t, errorString, "USER_NOT_FOUND")
	assert.Contains(t, errorString, "User not found")
}

// TestAppError_Unwrap supports errors.Is and errors.As
func TestAppError_Unwrap(t *testing.T) {
	appErr := NewAppError(ErrInvalidCredentials, "Bad credentials", "trace-456")
	
	// Test errors.Is
	assert.True(t, errors.Is(appErr, ErrInvalidCredentials))
	assert.False(t, errors.Is(appErr, ErrUserNotFound))
	
	// Test errors.As
	var targetErr *AppError
	assert.True(t, errors.As(appErr, &targetErr))
	assert.Equal(t, "INVALID_CREDENTIALS", targetErr.Code)
}

// TestMapErrorToHTTPStatus maps domain errors to HTTP status codes
func TestMapErrorToHTTPStatus(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		expectedStatus int
	}{
		// 404 Not Found
		{"user not found", ErrUserNotFound, http.StatusNotFound},
		{"token not found", ErrTokenNotFound, http.StatusNotFound},
		{"refresh token not found", ErrRefreshTokenNotFound, http.StatusNotFound},
		
		// 409 Conflict
		{"user already exists", ErrUserAlreadyExists, http.StatusConflict},
		
		// 401 Unauthorized
		{"invalid credentials", ErrInvalidCredentials, http.StatusUnauthorized},
		{"unauthorized", ErrUnauthorized, http.StatusUnauthorized},
		{"access token expired", ErrAccessTokenExpired, http.StatusUnauthorized},
		{"invalid access token", ErrInvalidAccessToken, http.StatusUnauthorized},
		
		// 403 Forbidden
		{"forbidden", ErrForbidden, http.StatusForbidden},
		{"user deleted", ErrUserDeleted, http.StatusForbidden},
		{"refresh token expired", ErrRefreshTokenExpired, http.StatusForbidden},
		{"refresh token revoked", ErrRefreshTokenRevoked, http.StatusForbidden},
		
		// 400 Bad Request
		{"invalid input", ErrInvalidInput, http.StatusBadRequest},
		{"invalid email", ErrInvalidEmail, http.StatusBadRequest},
		{"weak password", ErrWeakPassword, http.StatusBadRequest},
		{"invalid KYC status", ErrInvalidKYCStatus, http.StatusBadRequest},
		{"invalid role", ErrInvalidRole, http.StatusBadRequest},
		{"invalid refresh token", ErrInvalidRefreshToken, http.StatusBadRequest},
		
		// 500 Internal Server Error
		{"internal server error", ErrInternalServer, http.StatusInternalServerError},
		{"unknown error", errors.New("unknown"), http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := MapErrorToHTTPStatus(tt.err)
			assert.Equal(t, tt.expectedStatus, status)
		})
	}
}

// TestMapErrorToCode maps domain errors to error codes
func TestMapErrorToCode(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		expectedCode string
	}{
		{"user not found", ErrUserNotFound, "USER_NOT_FOUND"},
		{"user already exists", ErrUserAlreadyExists, "USER_ALREADY_EXISTS"},
		{"invalid credentials", ErrInvalidCredentials, "INVALID_CREDENTIALS"},
		{"unauthorized", ErrUnauthorized, "UNAUTHORIZED"},
		{"forbidden", ErrForbidden, "FORBIDDEN"},
		{"invalid input", ErrInvalidInput, "INVALID_INPUT"},
		{"invalid email", ErrInvalidEmail, "INVALID_EMAIL"},
		{"weak password", ErrWeakPassword, "WEAK_PASSWORD"},
		{"token expired", ErrAccessTokenExpired, "ACCESS_TOKEN_EXPIRED"},
		{"internal error", ErrInternalServer, "INTERNAL_SERVER_ERROR"},
		{"unknown error", errors.New("unknown"), "INTERNAL_SERVER_ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := MapErrorToCode(tt.err)
			assert.Equal(t, tt.expectedCode, code)
		})
	}
}

// TestAppError_WithDetails adds additional context
func TestAppError_WithDetails(t *testing.T) {
	appErr := NewAppError(ErrInvalidInput, "Validation failed", "trace-123")
	
	details := map[string]interface{}{
		"field": "email",
		"reason": "invalid format",
	}
	
	appErr = appErr.WithDetails(details)
	
	require.NotNil(t, appErr.Details)
	assert.Equal(t, "email", appErr.Details["field"])
	assert.Equal(t, "invalid format", appErr.Details["reason"])
}

// TestAppError_ToJSON serializes to JSON response
func TestAppError_ToJSON(t *testing.T) {
	appErr := NewAppError(ErrUserNotFound, "User not found", "trace-123")
	appErr = appErr.WithDetails(map[string]interface{}{
		"user_id": "123",
	})
	
	json := appErr.ToJSON()
	
	assert.Equal(t, "USER_NOT_FOUND", json["error"])
	assert.Equal(t, "User not found", json["message"])
	assert.Equal(t, "trace-123", json["trace_id"])
	assert.NotNil(t, json["details"])
	
	details, ok := json["details"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "123", details["user_id"])
}

// TestSanitizeInternalError ensures internal errors don't leak
func TestSanitizeInternalError(t *testing.T) {
	tests := []struct {
		name            string
		err             error
		shouldSanitize  bool
		expectedMessage string
	}{
		{
			name:            "domain error not sanitized",
			err:             ErrUserNotFound,
			shouldSanitize:  false,
			expectedMessage: "user not found",
		},
		{
			name:            "internal error sanitized",
			err:             errors.New("database connection failed: password=secret123"),
			shouldSanitize:  true,
			expectedMessage: "an unexpected error occurred",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			appErr := NewAppError(tt.err, tt.err.Error(), "trace-123")
			
			if tt.shouldSanitize {
				assert.NotContains(t, appErr.Message, "password")
				assert.NotContains(t, appErr.Message, "secret")
			} else {
				assert.Equal(t, tt.expectedMessage, appErr.Message)
			}
		})
	}
}
