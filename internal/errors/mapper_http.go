package errors

import (
	"context"
	stderrors "errors"
	"net/http"
)

// ErrorResponse represents the JSON error response structure sent to clients.
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contains the detailed error information.
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	TraceID string `json:"trace_id"`
}

// ToHTTPError converts a domain error or AppError to an HTTP status code and ErrorResponse.
// It maps domain errors to appropriate HTTP status codes and constructs a JSON response.
// All internal error details are sanitized to prevent information leakage.
func ToHTTPError(ctx context.Context, err error) (int, ErrorResponse) {
	// If it's already an AppError, use it directly
	if appErr, ok := err.(*AppError); ok {
		return appErr.HTTPStatus, ErrorResponse{
			Error: ErrorDetail{
				Code:    appErr.Code,
				Message: appErr.Message,
				TraceID: appErr.TraceID,
			},
		}
	}

	// Extract trace ID for the response
	traceID := extractTraceID(ctx)

	// Map domain errors to HTTP status codes and error codes
	switch {
	case stderrors.Is(err, ErrUserNotFound):
		return http.StatusNotFound, ErrorResponse{
			Error: ErrorDetail{
				Code:    "USER_NOT_FOUND",
				Message: "User not found",
				TraceID: traceID,
			},
		}

	case stderrors.Is(err, ErrUserAlreadyExists):
		return http.StatusConflict, ErrorResponse{
			Error: ErrorDetail{
				Code:    "USER_ALREADY_EXISTS",
				Message: "User already exists",
				TraceID: traceID,
			},
		}

	case stderrors.Is(err, ErrInvalidCredentials):
		return http.StatusUnauthorized, ErrorResponse{
			Error: ErrorDetail{
				Code:    "INVALID_CREDENTIALS",
				Message: "Invalid credentials",
				TraceID: traceID,
			},
		}

	case stderrors.Is(err, ErrInvalidInput):
		return http.StatusBadRequest, ErrorResponse{
			Error: ErrorDetail{
				Code:    "INVALID_INPUT",
				Message: "Invalid input",
				TraceID: traceID,
			},
		}

	case stderrors.Is(err, ErrInvalidToken):
		return http.StatusUnauthorized, ErrorResponse{
			Error: ErrorDetail{
				Code:    "INVALID_TOKEN",
				Message: "Invalid or expired token",
				TraceID: traceID,
			},
		}

	case stderrors.Is(err, ErrTokenExpired):
		return http.StatusUnauthorized, ErrorResponse{
			Error: ErrorDetail{
				Code:    "TOKEN_EXPIRED",
				Message: "Token has expired",
				TraceID: traceID,
			},
		}

	case stderrors.Is(err, ErrUnauthorized):
		return http.StatusUnauthorized, ErrorResponse{
			Error: ErrorDetail{
				Code:    "UNAUTHORIZED",
				Message: "Unauthorized",
				TraceID: traceID,
			},
		}

	case stderrors.Is(err, ErrForbidden):
		return http.StatusForbidden, ErrorResponse{
			Error: ErrorDetail{
				Code:    "FORBIDDEN",
				Message: "Forbidden",
				TraceID: traceID,
			},
		}

	case stderrors.Is(err, ErrInternal):
		return http.StatusInternalServerError, ErrorResponse{
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Internal server error",
				TraceID: traceID,
			},
		}

	default:
		// For unknown errors, return generic internal server error
		// Never expose internal error details to clients
		return http.StatusInternalServerError, ErrorResponse{
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Internal server error",
				TraceID: traceID,
			},
		}
	}
}
