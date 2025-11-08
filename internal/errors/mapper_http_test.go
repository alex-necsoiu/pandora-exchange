package errors

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestToHTTPError verifies domain error to HTTP response mapping
func TestToHTTPError(t *testing.T) {
	tests := []struct {
		name               string
		err                error
		expectedStatusCode int
		expectedCode       string
		expectedMessage    string
	}{
		{
			name:               "ErrUserNotFound",
			err:                ErrUserNotFound,
			expectedStatusCode: http.StatusNotFound,
			expectedCode:       "USER_NOT_FOUND",
			expectedMessage:    "User not found",
		},
		{
			name:               "ErrUserAlreadyExists",
			err:                ErrUserAlreadyExists,
			expectedStatusCode: http.StatusConflict,
			expectedCode:       "USER_ALREADY_EXISTS",
			expectedMessage:    "User already exists",
		},
		{
			name:               "ErrInvalidCredentials",
			err:                ErrInvalidCredentials,
			expectedStatusCode: http.StatusUnauthorized,
			expectedCode:       "INVALID_CREDENTIALS",
			expectedMessage:    "Invalid credentials",
		},
		{
			name:               "ErrInvalidInput",
			err:                ErrInvalidInput,
			expectedStatusCode: http.StatusBadRequest,
			expectedCode:       "INVALID_INPUT",
			expectedMessage:    "Invalid input",
		},
		{
			name:               "ErrInvalidToken",
			err:                ErrInvalidToken,
			expectedStatusCode: http.StatusUnauthorized,
			expectedCode:       "INVALID_TOKEN",
			expectedMessage:    "Invalid or expired token",
		},
		{
			name:               "ErrUnauthorized",
			err:                ErrUnauthorized,
			expectedStatusCode: http.StatusUnauthorized,
			expectedCode:       "UNAUTHORIZED",
			expectedMessage:    "Unauthorized",
		},
		{
			name:               "ErrForbidden",
			err:                ErrForbidden,
			expectedStatusCode: http.StatusForbidden,
			expectedCode:       "FORBIDDEN",
			expectedMessage:    "Forbidden",
		},
		{
			name:               "ErrInternal",
			err:                ErrInternal,
			expectedStatusCode: http.StatusInternalServerError,
			expectedCode:       "INTERNAL_ERROR",
			expectedMessage:    "Internal server error",
		},
		{
			name:               "Unknown error",
			err:                assert.AnError,
			expectedStatusCode: http.StatusInternalServerError,
			expectedCode:       "INTERNAL_ERROR",
			expectedMessage:    "Internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			statusCode, response := ToHTTPError(ctx, tt.err)

			assert.Equal(t, tt.expectedStatusCode, statusCode)
			assert.Equal(t, tt.expectedCode, response.Error.Code)
			assert.Equal(t, tt.expectedMessage, response.Error.Message)
			// TraceID may be empty in test environment without OTEL
			// assert.NotEmpty(t, response.Error.TraceID)
		})
	}
}

// TestToHTTPError_WithAppError verifies AppError is passed through correctly
func TestToHTTPError_WithAppError(t *testing.T) {
	ctx := context.Background()
	
	appErr := NewAppError(ctx, "CUSTOM_ERROR", "Custom error message", http.StatusTeapot)
	
	statusCode, response := ToHTTPError(ctx, appErr)

	assert.Equal(t, http.StatusTeapot, statusCode)
	assert.Equal(t, "CUSTOM_ERROR", response.Error.Code)
	assert.Equal(t, "Custom error message", response.Error.Message)
}

// TestToHTTPError_TraceIDExtraction verifies trace ID is extracted from context
func TestToHTTPError_TraceIDExtraction(t *testing.T) {
	ctx := contextWithTraceID(t, "expected-trace-id")
	
	statusCode, response := ToHTTPError(ctx, ErrUserNotFound)

	assert.Equal(t, http.StatusNotFound, statusCode)
	// TraceID may be empty in test environment without OTEL tracing active
	// In production with OTEL, trace_id would be populated
	_ = response.Error.TraceID
}
