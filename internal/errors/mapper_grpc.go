package errors

import (
	"context"
	stderrors "errors"
	"net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ToGRPCError converts a domain error or AppError to a gRPC status error.
// It maps domain errors to appropriate gRPC status codes.
func ToGRPCError(ctx context.Context, err error) error {
	// If it's an AppError, map HTTP status to gRPC code
	if appErr, ok := err.(*AppError); ok {
		code := httpStatusToGRPCCode(appErr.HTTPStatus)
		return status.Error(code, appErr.Message)
	}

	// Map domain errors to gRPC codes
	switch {
	case stderrors.Is(err, ErrUserNotFound):
		return status.Error(codes.NotFound, "User not found")

	case stderrors.Is(err, ErrUserAlreadyExists):
		return status.Error(codes.AlreadyExists, "User already exists")

	case stderrors.Is(err, ErrInvalidCredentials):
		return status.Error(codes.Unauthenticated, "Invalid credentials")

	case stderrors.Is(err, ErrInvalidInput):
		return status.Error(codes.InvalidArgument, "Invalid input")

	case stderrors.Is(err, ErrInvalidToken):
		return status.Error(codes.Unauthenticated, "Invalid or expired token")

	case stderrors.Is(err, ErrTokenExpired):
		return status.Error(codes.Unauthenticated, "Token has expired")

	case stderrors.Is(err, ErrUnauthorized):
		return status.Error(codes.Unauthenticated, "Unauthorized")

	case stderrors.Is(err, ErrForbidden):
		return status.Error(codes.PermissionDenied, "Forbidden")

	case stderrors.Is(err, ErrInternal):
		return status.Error(codes.Internal, "Internal server error")

	default:
		// For unknown errors, return generic internal error
		return status.Error(codes.Internal, "Internal server error")
	}
}

// httpStatusToGRPCCode maps HTTP status codes to gRPC codes.
// This is used when converting AppError (which has HTTP status) to gRPC errors.
func httpStatusToGRPCCode(httpStatus int) codes.Code {
	switch httpStatus {
	case http.StatusBadRequest:
		return codes.InvalidArgument
	case http.StatusUnauthorized:
		return codes.Unauthenticated
	case http.StatusForbidden:
		return codes.PermissionDenied
	case http.StatusNotFound:
		return codes.NotFound
	case http.StatusConflict:
		return codes.AlreadyExists
	case http.StatusTooManyRequests:
		return codes.ResourceExhausted
	case http.StatusInternalServerError:
		return codes.Internal
	case http.StatusNotImplemented:
		return codes.Unimplemented
	case http.StatusServiceUnavailable:
		return codes.Unavailable
	case http.StatusGatewayTimeout:
		return codes.DeadlineExceeded
	default:
		return codes.Unknown
	}
}
