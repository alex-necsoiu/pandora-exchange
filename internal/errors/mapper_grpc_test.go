package errors

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestToGRPCError verifies domain error to gRPC status code mapping
func TestToGRPCError(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		expectedCode codes.Code
		expectedMsg  string
	}{
		{
			name:         "ErrUserNotFound",
			err:          ErrUserNotFound,
			expectedCode: codes.NotFound,
			expectedMsg:  "User not found",
		},
		{
			name:         "ErrUserAlreadyExists",
			err:          ErrUserAlreadyExists,
			expectedCode: codes.AlreadyExists,
			expectedMsg:  "User already exists",
		},
		{
			name:         "ErrInvalidCredentials",
			err:          ErrInvalidCredentials,
			expectedCode: codes.Unauthenticated,
			expectedMsg:  "Invalid credentials",
		},
		{
			name:         "ErrInvalidInput",
			err:          ErrInvalidInput,
			expectedCode: codes.InvalidArgument,
			expectedMsg:  "Invalid input",
		},
		{
			name:         "ErrInvalidToken",
			err:          ErrInvalidToken,
			expectedCode: codes.Unauthenticated,
			expectedMsg:  "Invalid or expired token",
		},
		{
			name:         "ErrUnauthorized",
			err:          ErrUnauthorized,
			expectedCode: codes.Unauthenticated,
			expectedMsg:  "Unauthorized",
		},
		{
			name:         "ErrForbidden",
			err:          ErrForbidden,
			expectedCode: codes.PermissionDenied,
			expectedMsg:  "Forbidden",
		},
		{
			name:         "ErrInternal",
			err:          ErrInternal,
			expectedCode: codes.Internal,
			expectedMsg:  "Internal server error",
		},
		{
			name:         "Unknown error",
			err:          assert.AnError,
			expectedCode: codes.Internal,
			expectedMsg:  "Internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			grpcErr := ToGRPCError(ctx, tt.err)

			// Extract status from error
			st, ok := status.FromError(grpcErr)
			assert.True(t, ok, "Should be a gRPC status error")
			assert.Equal(t, tt.expectedCode, st.Code())
			assert.Contains(t, st.Message(), tt.expectedMsg)
		})
	}
}

// TestToGRPCError_WithAppError verifies AppError is converted correctly
func TestToGRPCError_WithAppError(t *testing.T) {
	ctx := context.Background()
	
	appErr := NewAppError(ctx, "CUSTOM_ERROR", "Custom gRPC error", 400)
	
	grpcErr := ToGRPCError(ctx, appErr)

	st, ok := status.FromError(grpcErr)
	assert.True(t, ok, "Should be a gRPC status error")
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "Custom gRPC error")
}
