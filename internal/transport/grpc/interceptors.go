package grpc

import (
	"context"
	"fmt"
	"runtime/debug"
	"time"

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
