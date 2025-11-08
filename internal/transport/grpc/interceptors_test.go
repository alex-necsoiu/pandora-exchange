package grpc_test

import (
	"context"
	"errors"
	"testing"

	"github.com/alex-necsoiu/pandora-exchange/internal/observability"
	grpcTransport "github.com/alex-necsoiu/pandora-exchange/internal/transport/grpc"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
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
