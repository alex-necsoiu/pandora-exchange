package grpc_test

import (
	"context"
	"errors"
	"testing"

	"github.com/alex-necsoiu/pandora-exchange/internal/domain"
	"github.com/alex-necsoiu/pandora-exchange/internal/observability"
	grpcTransport "github.com/alex-necsoiu/pandora-exchange/internal/transport/grpc"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// mockUnaryHandler is a mock gRPC unary handler for testing interceptors
type mockUnaryHandler struct {
	shouldPanic bool
	panicMsg    interface{}
	returnError error
}

func (m *mockUnaryHandler) handle(ctx context.Context, req interface{}) (interface{}, error) {
	if m.shouldPanic {
		panic(m.panicMsg)
	}
	if m.returnError != nil {
		return nil, m.returnError
	}
	return "success", nil
}

func TestUnaryLoggingInterceptor(t *testing.T) {
	tests := []struct {
		name          string
		handler       *mockUnaryHandler
		expectError   bool
		expectedResp  interface{}
	}{
		{
			name:         "successful request logged",
			handler:      &mockUnaryHandler{},
			expectError:  false,
			expectedResp: "success",
		},
		{
			name: "failed request logged",
			handler: &mockUnaryHandler{
				returnError: errors.New("service error"),
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := observability.NewLogger("test", "grpc-test")
			interceptor := grpcTransport.UnaryLoggingInterceptor(logger)

			info := &grpc.UnaryServerInfo{
				FullMethod: "/pandora.user.v1.UserService/GetUser",
			}

			resp, err := interceptor(
				context.Background(),
				nil,
				info,
				tt.handler.handle,
			)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResp, resp)
			}
		})
	}
}

func TestUnaryTracingInterceptor(t *testing.T) {
	tests := []struct {
		name          string
		handler       *mockUnaryHandler
		expectError   bool
		expectedResp  interface{}
	}{
		{
			name:         "successful request with span",
			handler:      &mockUnaryHandler{},
			expectError:  false,
			expectedResp: "success",
		},
		{
			name: "failed request with error recorded in span",
			handler: &mockUnaryHandler{
				returnError: errors.New("service error"),
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interceptor := grpcTransport.UnaryTracingInterceptor()

			info := &grpc.UnaryServerInfo{
				FullMethod: "/pandora.user.v1.UserService/GetUser",
			}

			resp, err := interceptor(
				context.Background(),
				nil,
				info,
				tt.handler.handle,
			)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResp, resp)
			}
		})
	}
}

func TestUnaryRecoveryInterceptor(t *testing.T) {
	tests := []struct {
		name        string
		handler     *mockUnaryHandler
		expectPanic bool
		expectError bool
	}{
		{
			name:        "successful request no panic",
			handler:     &mockUnaryHandler{},
			expectPanic: false,
			expectError: false,
		},
		{
			name: "panic with string message",
			handler: &mockUnaryHandler{
				shouldPanic: true,
				panicMsg:    "something went wrong",
			},
			expectPanic: true,
			expectError: true,
		},
		{
			name: "panic with error",
			handler: &mockUnaryHandler{
				shouldPanic: true,
				panicMsg:    errors.New("panic error"),
			},
			expectPanic: true,
			expectError: true,
		},
		{
			name: "panic with nil",
			handler: &mockUnaryHandler{
				shouldPanic: true,
				panicMsg:    nil,
			},
			expectPanic: true,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := observability.NewLogger("test", "grpc-test")
			interceptor := grpcTransport.UnaryRecoveryInterceptor(logger)

			info := &grpc.UnaryServerInfo{
				FullMethod: "/pandora.user.v1.UserService/GetUser",
			}

			// The interceptor should catch the panic and return an error
			resp, err := interceptor(
				context.Background(),
				nil,
				info,
				tt.handler.handle,
			)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
			}
		})
	}
}

func TestInterceptorChaining(t *testing.T) {
	// Test that all three interceptors can be chained together
	logger := observability.NewLogger("test", "grpc-test")
	
	handler := &mockUnaryHandler{}
	info := &grpc.UnaryServerInfo{
		FullMethod: "/pandora.user.v1.UserService/GetUser",
	}

	// Create a chain: Recovery -> Logging -> Tracing
	recoveryInterceptor := grpcTransport.UnaryRecoveryInterceptor(logger)
	loggingInterceptor := grpcTransport.UnaryLoggingInterceptor(logger)
	tracingInterceptor := grpcTransport.UnaryTracingInterceptor()

	// Chain them together manually (simulating what grpc.ChainUnaryInterceptor does)
	chainedHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return tracingInterceptor(ctx, req, info, handler.handle)
	}

	chainedHandler2 := func(ctx context.Context, req interface{}) (interface{}, error) {
		return loggingInterceptor(ctx, req, info, chainedHandler)
	}

	resp, err := recoveryInterceptor(context.Background(), nil, info, chainedHandler2)

	assert.NoError(t, err)
	assert.Equal(t, "success", resp)
}

func TestInterceptorWithPanicInChain(t *testing.T) {
	// Test that recovery interceptor catches panics from the entire chain
	logger := observability.NewLogger("test", "grpc-test")
	
	handler := &mockUnaryHandler{
		shouldPanic: true,
		panicMsg:    "panic in handler",
	}
	
	info := &grpc.UnaryServerInfo{
		FullMethod: "/pandora.user.v1.UserService/GetUser",
	}

	recoveryInterceptor := grpcTransport.UnaryRecoveryInterceptor(logger)
	loggingInterceptor := grpcTransport.UnaryLoggingInterceptor(logger)

	chainedHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return loggingInterceptor(ctx, req, info, handler.handle)
	}

	resp, err := recoveryInterceptor(context.Background(), nil, info, chainedHandler)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "internal server error")
}

func TestErrorInterceptor(t *testing.T) {
	tests := []struct {
		name         string
		handlerErr   error
		expectedCode codes.Code
		expectedMsg  string
	}{
		{
			name:         "user_not_found",
			handlerErr:   domain.ErrUserNotFound,
			expectedCode: codes.NotFound,
			expectedMsg:  "user not found",
		},
		{
			name:         "token_not_found",
			handlerErr:   domain.ErrTokenNotFound,
			expectedCode: codes.NotFound,
			expectedMsg:  "token not found",
		},
		{
			name:         "refresh_token_not_found",
			handlerErr:   domain.ErrRefreshTokenNotFound,
			expectedCode: codes.NotFound,
			expectedMsg:  "refresh token not found",
		},
		{
			name:         "user_already_exists",
			handlerErr:   domain.ErrUserAlreadyExists,
			expectedCode: codes.AlreadyExists,
			expectedMsg:  "user already exists",
		},
		{
			name:         "invalid_credentials",
			handlerErr:   domain.ErrInvalidCredentials,
			expectedCode: codes.Unauthenticated,
			expectedMsg:  "invalid email or password",
		},
		{
			name:         "unauthorized",
			handlerErr:   domain.ErrUnauthorized,
			expectedCode: codes.Unauthenticated,
			expectedMsg:  "unauthorized",
		},
		{
			name:         "access_token_expired",
			handlerErr:   domain.ErrAccessTokenExpired,
			expectedCode: codes.Unauthenticated,
			expectedMsg:  "access token has expired",
		},
		{
			name:         "invalid_access_token",
			handlerErr:   domain.ErrInvalidAccessToken,
			expectedCode: codes.Unauthenticated,
			expectedMsg:  "invalid access token",
		},
		{
			name:         "forbidden",
			handlerErr:   domain.ErrForbidden,
			expectedCode: codes.PermissionDenied,
			expectedMsg:  "forbidden",
		},
		{
			name:         "user_deleted",
			handlerErr:   domain.ErrUserDeleted,
			expectedCode: codes.PermissionDenied,
			expectedMsg:  "user has been deleted",
		},
		{
			name:         "refresh_token_expired",
			handlerErr:   domain.ErrRefreshTokenExpired,
			expectedCode: codes.PermissionDenied,
			expectedMsg:  "refresh token has expired",
		},
		{
			name:         "refresh_token_revoked",
			handlerErr:   domain.ErrRefreshTokenRevoked,
			expectedCode: codes.PermissionDenied,
			expectedMsg:  "refresh token has been revoked",
		},
		{
			name:         "invalid_input",
			handlerErr:   domain.ErrInvalidInput,
			expectedCode: codes.InvalidArgument,
			expectedMsg:  "invalid input",
		},
		{
			name:         "invalid_email",
			handlerErr:   domain.ErrInvalidEmail,
			expectedCode: codes.InvalidArgument,
			expectedMsg:  "invalid email format",
		},
		{
			name:         "weak_password",
			handlerErr:   domain.ErrWeakPassword,
			expectedCode: codes.InvalidArgument,
			expectedMsg:  "password does not meet security requirements",
		},
		{
			name:         "invalid_kyc_status",
			handlerErr:   domain.ErrInvalidKYCStatus,
			expectedCode: codes.InvalidArgument,
			expectedMsg:  "invalid KYC status",
		},
		{
			name:         "invalid_role",
			handlerErr:   domain.ErrInvalidRole,
			expectedCode: codes.InvalidArgument,
			expectedMsg:  "invalid role",
		},
		{
			name:         "invalid_refresh_token",
			handlerErr:   domain.ErrInvalidRefreshToken,
			expectedCode: codes.InvalidArgument,
			expectedMsg:  "invalid refresh token",
		},
		{
			name:         "internal_server_error",
			handlerErr:   domain.ErrInternalServer,
			expectedCode: codes.Internal,
			expectedMsg:  "An unexpected error occurred",
		},
		{
			name:         "unknown_error_sanitized",
			handlerErr:   errors.New("database connection failed"),
			expectedCode: codes.Internal,
			expectedMsg:  "An unexpected error occurred",
		},
		{
			name:         "nil_error_returns_ok",
			handlerErr:   nil,
			expectedCode: codes.OK,
			expectedMsg:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock handler
			handler := func(ctx context.Context, req interface{}) (interface{}, error) {
				return nil, tt.handlerErr
			}

			// Create interceptor
			interceptor := grpcTransport.ErrorInterceptor()

			// Execute interceptor
			_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{}, handler)

			if tt.expectedCode == codes.OK {
				assert.NoError(t, err)
				return
			}

			// Verify error
			assert.Error(t, err)
			st, ok := status.FromError(err)
			assert.True(t, ok, "error should be a gRPC status")
			assert.Equal(t, tt.expectedCode, st.Code())
			assert.Equal(t, tt.expectedMsg, st.Message())
		})
	}
}

func TestErrorInterceptor_WithTraceID(t *testing.T) {
	// Setup OTEL tracer
	tp := trace.NewNoopTracerProvider()
	otel.SetTracerProvider(tp)
	tracer := tp.Tracer("test")

	// Create context with trace
	ctx, span := tracer.Start(context.Background(), "test-operation")
	defer span.End()

	// Create mock handler
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, domain.ErrUserNotFound
	}

	// Create interceptor
	interceptor := grpcTransport.ErrorInterceptor()

	// Execute interceptor
	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{}, handler)

	// Verify error
	assert.Error(t, err)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
	assert.Equal(t, "user not found", st.Message())
}

func TestErrorInterceptor_AppError(t *testing.T) {
	// Create AppError
	appErr := domain.NewAppError(
		domain.ErrInvalidEmail,
		"invalid email format",
		"test-trace-id",
	).WithDetails(map[string]interface{}{
		"field": "email",
		"value": "invalid-email",
	})

	// Create mock handler
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, appErr
	}

	// Create interceptor
	interceptor := grpcTransport.ErrorInterceptor()

	// Execute interceptor
	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{}, handler)

	// Verify error
	assert.Error(t, err)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Equal(t, "invalid email format", st.Message())
}
