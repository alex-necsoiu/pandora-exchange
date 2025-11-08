package grpc

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/alex-necsoiu/pandora-exchange/internal/domain"
	"github.com/alex-necsoiu/pandora-exchange/internal/observability"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	grpc_codes "google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UnaryLoggingInterceptor logs all gRPC calls with request/response information
func UnaryLoggingInterceptor(logger *observability.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		start := time.Now()

		logger.WithFields(map[string]interface{}{
			"method": info.FullMethod,
		}).Info("gRPC request started")

		// Call the handler
		resp, err := handler(ctx, req)

		duration := time.Since(start)
		code := status.Code(err)

		logFields := map[string]interface{}{
			"method":   info.FullMethod,
			"duration": duration.String(),
			"code":     code.String(),
		}

		if err != nil {
			logFields["error"] = err.Error()
			logger.WithFields(logFields).Error("gRPC request failed")
		} else {
			logger.WithFields(logFields).Info("gRPC request completed")
		}

		return resp, err
	}
}

// UnaryTracingInterceptor creates OpenTelemetry spans for each gRPC call
func UnaryTracingInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		tracer := otel.Tracer(observability.TracerName)

		// Start a new span
		ctx, span := tracer.Start(ctx, info.FullMethod,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				attribute.String("rpc.system", "grpc"),
				attribute.String("rpc.service", "UserService"),
				attribute.String("rpc.method", info.FullMethod),
			),
		)
		defer span.End()

		// Call the handler
		resp, err := handler(ctx, req)

		// Record error if present
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			span.SetAttributes(
				attribute.String("rpc.grpc.status_code", status.Code(err).String()),
			)
		} else {
			span.SetStatus(codes.Ok, "success")
			span.SetAttributes(
				attribute.String("rpc.grpc.status_code", grpc_codes.OK.String()),
			)
		}

		return resp, err
	}
}

// UnaryRecoveryInterceptor recovers from panics in gRPC handlers
func UnaryRecoveryInterceptor(logger *observability.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				// Log the panic
				logger.WithFields(map[string]interface{}{
					"method": info.FullMethod,
					"panic":  fmt.Sprintf("%v", r),
					"stack":  string(debug.Stack()),
				}).Error("gRPC handler panicked")

				// Return internal error
				err = status.Error(grpc_codes.Internal, "internal server error")
			}
		}()

		return handler(ctx, req)
	}
}

// ErrorInterceptor maps domain errors to gRPC status codes.
// It attaches OpenTelemetry trace IDs for request correlation.
//
// Returns:
//   - grpc.UnaryServerInterceptor: Interceptor function for error handling
//
// Security: Internal errors are sanitized to prevent information leakage.
func ErrorInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Execute the handler
		resp, err := handler(ctx, req)
		
		// If no error, return immediately
		if err == nil {
			return resp, nil
		}

		// Extract trace ID from context
		traceID := extractTraceIDFromContext(ctx)

		// Convert to AppError
		var appErr *domain.AppError
		if domainErr, ok := err.(*domain.AppError); ok {
			// Already an AppError, use it directly
			appErr = domainErr
			// Update trace ID if not set
			if appErr.TraceID == "" {
				appErr.TraceID = traceID
			}
		} else {
			// Convert domain error to AppError
			appErr = domain.NewAppError(err, err.Error(), traceID)
		}

		// Map to gRPC status code
		grpcCode := mapErrorToGRPCCode(appErr.Err)
		
		// Return gRPC status error
		return nil, status.Error(grpcCode, appErr.Message)
	}
}

// mapErrorToGRPCCode maps domain errors to gRPC status codes.
//
// Parameters:
//   - err: Domain error to map
//
// Returns:
//   - grpc_codes.Code: Corresponding gRPC status code
func mapErrorToGRPCCode(err error) grpc_codes.Code {
	switch {
	case errors.Is(err, domain.ErrUserNotFound):
		return grpc_codes.NotFound
	case errors.Is(err, domain.ErrTokenNotFound):
		return grpc_codes.NotFound
	case errors.Is(err, domain.ErrRefreshTokenNotFound):
		return grpc_codes.NotFound
		
	case errors.Is(err, domain.ErrUserAlreadyExists):
		return grpc_codes.AlreadyExists
		
	case errors.Is(err, domain.ErrInvalidCredentials):
		return grpc_codes.Unauthenticated
	case errors.Is(err, domain.ErrUnauthorized):
		return grpc_codes.Unauthenticated
	case errors.Is(err, domain.ErrAccessTokenExpired):
		return grpc_codes.Unauthenticated
	case errors.Is(err, domain.ErrInvalidAccessToken):
		return grpc_codes.Unauthenticated
		
	case errors.Is(err, domain.ErrForbidden):
		return grpc_codes.PermissionDenied
	case errors.Is(err, domain.ErrUserDeleted):
		return grpc_codes.PermissionDenied
	case errors.Is(err, domain.ErrRefreshTokenExpired):
		return grpc_codes.PermissionDenied
	case errors.Is(err, domain.ErrRefreshTokenRevoked):
		return grpc_codes.PermissionDenied
		
	case errors.Is(err, domain.ErrInvalidInput):
		return grpc_codes.InvalidArgument
	case errors.Is(err, domain.ErrInvalidEmail):
		return grpc_codes.InvalidArgument
	case errors.Is(err, domain.ErrWeakPassword):
		return grpc_codes.InvalidArgument
	case errors.Is(err, domain.ErrInvalidKYCStatus):
		return grpc_codes.InvalidArgument
	case errors.Is(err, domain.ErrInvalidRole):
		return grpc_codes.InvalidArgument
	case errors.Is(err, domain.ErrInvalidRefreshToken):
		return grpc_codes.InvalidArgument
		
	default:
		// Unknown errors are treated as internal server errors
		return grpc_codes.Internal
	}
}

// extractTraceIDFromContext extracts the OpenTelemetry trace ID from the context.
//
// Parameters:
//   - ctx: Request context containing trace information
//
// Returns:
//   - string: Trace ID in hexadecimal format, or empty string if not found
func extractTraceIDFromContext(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span == nil {
		return ""
	}

	spanCtx := span.SpanContext()
	if !spanCtx.IsValid() {
		return ""
	}

	return spanCtx.TraceID().String()
}
