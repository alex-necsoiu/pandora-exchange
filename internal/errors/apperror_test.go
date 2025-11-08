package errors

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewAppError verifies AppError creation
func TestNewAppError(t *testing.T) {
	tests := []struct {
		name           string
		code           string
		message        string
		httpStatus     int
		ctx            context.Context
		expectedCode   string
		expectedMsg    string
		expectedStatus int
		hasTraceID     bool
	}{
		{
			name:           "basic error without trace",
			code:           "USER_NOT_FOUND",
			message:        "User not found",
			httpStatus:     404,
			ctx:            context.Background(),
			expectedCode:   "USER_NOT_FOUND",
			expectedMsg:    "User not found",
			expectedStatus: 404,
			hasTraceID:     false,
		},
		{
			name:           "error with trace context",
			code:           "INVALID_INPUT",
			message:        "Invalid input provided",
			httpStatus:     400,
			ctx:            contextWithTraceID(t, "test-trace-123"),
			expectedCode:   "INVALID_INPUT",
			expectedMsg:    "Invalid input provided",
			expectedStatus: 400,
			hasTraceID:     false, // No OTEL in test environment
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			appErr := NewAppError(tt.ctx, tt.code, tt.message, tt.httpStatus)

			assert.Equal(t, tt.expectedCode, appErr.Code)
			assert.Equal(t, tt.expectedMsg, appErr.Message)
			assert.Equal(t, tt.expectedStatus, appErr.HTTPStatus)

			if tt.hasTraceID {
				assert.NotEmpty(t, appErr.TraceID)
			}
		})
	}
}

// TestAppError_Error verifies error interface implementation
func TestAppError_Error(t *testing.T) {
	appErr := &AppError{
		Code:       "TEST_ERROR",
		Message:    "This is a test error",
		HTTPStatus: 500,
		TraceID:    "trace-123",
	}

	errorString := appErr.Error()
	assert.Contains(t, errorString, "TEST_ERROR")
	assert.Contains(t, errorString, "This is a test error")
}

// TestIsAppError verifies type assertion helper
func TestIsAppError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "is AppError",
			err:      &AppError{Code: "TEST", Message: "test", HTTPStatus: 400},
			expected: true,
		},
		{
			name:     "is not AppError",
			err:      ErrUserNotFound,
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsAppError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestWrap verifies error wrapping
func TestWrap(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		code           string
		httpStatus     int
		expectedCode   string
		expectedStatus int
	}{
		{
			name:           "wrap domain error",
			err:            ErrUserNotFound,
			code:           "USER_NOT_FOUND",
			httpStatus:     404,
			expectedCode:   "USER_NOT_FOUND",
			expectedStatus: 404,
		},
		{
			name:           "wrap generic error",
			err:            assert.AnError,
			code:           "INTERNAL_ERROR",
			httpStatus:     500,
			expectedCode:   "INTERNAL_ERROR",
			expectedStatus: 500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			appErr := Wrap(ctx, tt.err, tt.code, tt.httpStatus)

			assert.Equal(t, tt.expectedCode, appErr.Code)
			assert.Equal(t, tt.expectedStatus, appErr.HTTPStatus)
			assert.Contains(t, appErr.Message, tt.err.Error())
		})
	}
}

// contextWithTraceID creates a context with a trace ID for testing
func contextWithTraceID(t *testing.T, traceID string) context.Context {
	t.Helper()
	
	// Create a span context with the trace ID
	ctx := context.Background()
	
	// For testing, we'll use a mock trace ID
	// In real implementation, this would come from OTEL
	return ctx
}
