# Error Handling System

This document describes the comprehensive error handling system implemented in the Pandora Exchange backend (Task 15).

## Overview

The error handling system provides:
- **Domain-level sentinel errors** for business logic failures
- **AppError struct** for HTTP context and serialization
- **HTTP middleware** for automatic error response mapping
- **gRPC interceptor** for automatic status code mapping
- **OpenTelemetry integration** for trace ID correlation
- **Security** through internal error sanitization

## Architecture

### Domain Layer (`internal/domain/errors.go`)

#### Sentinel Errors

19 predefined domain errors representing business logic failures:

```go
// Not Found (404 / NotFound)
ErrUserNotFound
ErrTokenNotFound
ErrRefreshTokenNotFound

// Already Exists (409 / AlreadyExists)
ErrUserAlreadyExists

// Unauthorized (401 / Unauthenticated)
ErrInvalidCredentials
ErrUnauthorized
ErrAccessTokenExpired
ErrInvalidAccessToken

// Forbidden (403 / PermissionDenied)
ErrForbidden
ErrUserDeleted
ErrRefreshTokenExpired
ErrRefreshTokenRevoked

// Bad Request (400 / InvalidArgument)
ErrInvalidInput
ErrInvalidEmail
ErrWeakPassword
ErrInvalidKYCStatus
ErrInvalidRole
ErrInvalidRefreshToken

// Internal Server Error (500 / Internal)
ErrInternalServer
```

#### AppError Struct

Wraps domain errors with HTTP context:

```go
type AppError struct {
    Err        error                  // Underlying domain error
    Code       string                 // Machine-readable error code (e.g., "USER_NOT_FOUND")
    Message    string                 // Human-readable message
    TraceID    string                 // OpenTelemetry trace ID
    HTTPStatus int                    // HTTP status code (404, 409, 401, 403, 400, 500)
    Details    map[string]interface{} // Additional context (optional)
}
```

#### Key Functions

**NewAppError(err error, message string, traceID string) *AppError**
- Creates AppError from domain error
- Auto-maps to HTTP status and error code
- **Security**: Sanitizes internal/unknown errors to "An unexpected error occurred"

**MapErrorToHTTPStatus(err error) int**
- Maps 19 domain errors to 6 HTTP status codes
- Unknown errors → 500 Internal Server Error

**MapErrorToCode(err error) string**
- Maps domain errors to stable error codes (RESOURCE_ACTION_REASON pattern)
- Unknown errors → "INTERNAL_SERVER_ERROR"

**ToJSON() map[string]interface{}**
- Serializes AppError to HTTP response format:
```json
{
  "error": "USER_NOT_FOUND",
  "message": "user not found",
  "trace_id": "abc123...",
  "details": {
    "field": "email",
    "value": "test@example.com"
  }
}
```

**WithDetails(details map[string]interface{}) *AppError**
- Adds field-specific context to error
- Useful for validation errors

### HTTP Transport (`internal/transport/http/middleware.go`)

#### ErrorMiddleware()

Gin middleware that:
1. Executes the handler (via `c.Next()`)
2. Checks for errors in Gin context
3. Extracts OpenTelemetry trace ID from request context
4. Converts domain error to AppError
5. Returns JSON error response with appropriate HTTP status

**Usage:**
```go
router := gin.New()
router.Use(ErrorMiddleware())

router.GET("/users/:id", func(c *gin.Context) {
    user, err := userService.GetUser(ctx, userID)
    if err != nil {
        _ = c.Error(err) // ErrorMiddleware handles conversion
        return
    }
    c.JSON(200, user)
})
```

**Features:**
- Automatic trace ID extraction from OTEL spans
- Supports both domain errors and AppError
- Sanitizes internal errors for security
- Returns first error if multiple errors exist

### gRPC Transport (`internal/transport/grpc/interceptors.go`)

#### ErrorInterceptor()

gRPC unary interceptor that:
1. Executes the handler
2. Extracts OpenTelemetry trace ID from request context
3. Converts domain error to AppError
4. Maps to gRPC status code
5. Returns gRPC status error

**Error Code Mapping:**
```
Domain Error              → gRPC Status Code
-----------------           ------------------
ErrUserNotFound           → codes.NotFound
ErrUserAlreadyExists      → codes.AlreadyExists
ErrInvalidCredentials     → codes.Unauthenticated
ErrForbidden              → codes.PermissionDenied
ErrInvalidInput           → codes.InvalidArgument
ErrInternalServer         → codes.Internal
(unknown errors)          → codes.Internal
```

**Usage:**
```go
server := grpc.NewServer(
    grpc.ChainUnaryInterceptor(
        ErrorInterceptor(),
        UnaryLoggingInterceptor(logger),
        UnaryTracingInterceptor(),
    ),
)
```

## Usage Examples

### Service Layer Error Handling

```go
func (s *UserService) GetUser(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
    user, err := s.repo.GetByID(ctx, userID)
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            // Return domain error - middleware will handle conversion
            return nil, domain.ErrUserNotFound
        }
        // Unknown database error - will be sanitized
        return nil, fmt.Errorf("database error: %w", err)
    }
    return user, nil
}
```

### HTTP Handler with Field Details

```go
func (h *Handler) CreateUser(c *gin.Context) {
    var req CreateUserRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        // Add validation details
        appErr := domain.NewAppError(
            domain.ErrInvalidInput,
            "Invalid request body",
            extractTraceID(c.Request.Context()),
        ).WithDetails(map[string]interface{}{
            "field": "email",
            "error": "invalid email format",
        })
        _ = c.Error(appErr)
        return
    }
    
    user, err := h.service.CreateUser(c.Request.Context(), req.Email, req.Password)
    if err != nil {
        _ = c.Error(err) // ErrorMiddleware handles conversion
        return
    }
    
    c.JSON(201, user)
}
```

### gRPC Handler

```go
func (s *Server) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.User, error) {
    userID, err := uuid.Parse(req.UserId)
    if err != nil {
        // Return domain error - interceptor will map to InvalidArgument
        return nil, domain.ErrInvalidInput
    }
    
    user, err := s.service.GetUser(ctx, userID)
    if err != nil {
        // ErrorInterceptor maps domain error to gRPC status
        return nil, err
    }
    
    return toProtoUser(user), nil
}
```

## Error Response Format

### HTTP Response

**Status:** 404 Not Found
```json
{
  "error": "USER_NOT_FOUND",
  "message": "user not found",
  "trace_id": "d5b2c6e8f1a3b7c9d4e6f8a1b2c3d4e5"
}
```

**Status:** 400 Bad Request (with details)
```json
{
  "error": "INVALID_EMAIL",
  "message": "invalid email format",
  "trace_id": "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6",
  "details": {
    "field": "email",
    "value": "not-an-email"
  }
}
```

**Status:** 500 Internal Server Error (sanitized)
```json
{
  "error": "INTERNAL_SERVER_ERROR",
  "message": "An unexpected error occurred",
  "trace_id": "f6e5d4c3b2a1f0e9d8c7b6a5f4e3d2c1"
}
```

### gRPC Response

```
Status: NOT_FOUND
Message: "user not found"
```

## Security Considerations

### Information Leakage Prevention

The system prevents sensitive information leakage through:

1. **Internal Error Sanitization**
   - Unknown errors → "An unexpected error occurred"
   - Database errors → "An unexpected error occurred"
   - System errors → "An unexpected error occurred"

2. **Known Domain Errors**
   - Only predefined domain errors pass through
   - Messages are carefully crafted to avoid leaking implementation details
   - Example: "user not found" (not "no row in database table users")

3. **Implementation**
   ```go
   func NewAppError(err error, message string, traceID string) *AppError {
       code := MapErrorToCode(err)
       httpStatus := MapErrorToHTTPStatus(err)
       
       // Sanitize message for unknown/internal errors
       if errors.Is(err, ErrInternalServer) || !isKnownDomainError(err) {
           message = "An unexpected error occurred"
       }
       
       return &AppError{...}
   }
   ```

### Trace ID Correlation

- OpenTelemetry trace IDs included in all error responses
- Allows correlation between client errors and server logs
- Enables debugging without exposing sensitive error details
- Trace IDs are extracted from OTEL span context

## Testing

### Test Coverage

**Domain Layer (8 tests, 293 lines):**
- TestSentinelErrors: Validates all 19 errors exist
- TestNewAppError: 7 scenarios (user_not_found, already_exists, etc.)
- TestAppError_Error: Implements error interface
- TestAppError_Unwrap: Supports errors.Is/As
- TestMapErrorToHTTPStatus: 21 scenarios
- TestMapErrorToCode: 12 scenarios
- TestAppError_WithDetails: Context addition
- TestAppError_ToJSON: Serialization
- TestSanitizeInternalError: Security validation

**HTTP Middleware (3 tests, 227 lines):**
- TestErrorMiddleware: 10 scenarios (domain errors, sanitization, etc.)
- TestErrorMiddleware_WithTraceID: Trace ID extraction
- TestErrorMiddleware_LogsErrors: Logging verification

**gRPC Interceptor (3 tests):**
- TestErrorInterceptor: 21 scenarios (all domain errors + unknown)
- TestErrorInterceptor_WithTraceID: Trace ID extraction
- TestErrorInterceptor_AppError: AppError handling

### Running Tests

```bash
# All error tests
go test -v ./internal/domain ./internal/transport/http ./internal/transport/grpc -run "Error|error"

# Domain error tests only
go test -v ./internal/domain -run "TestError|TestSentinel|TestMap|TestApp"

# HTTP middleware tests
go test -v ./internal/transport/http -run TestErrorMiddleware

# gRPC interceptor tests
go test -v ./internal/transport/grpc -run TestErrorInterceptor
```

## Error Code Reference

| Domain Error | HTTP Status | gRPC Code | Error Code |
|--------------|-------------|-----------|------------|
| ErrUserNotFound | 404 | NotFound | USER_NOT_FOUND |
| ErrTokenNotFound | 404 | NotFound | TOKEN_NOT_FOUND |
| ErrRefreshTokenNotFound | 404 | NotFound | REFRESH_TOKEN_NOT_FOUND |
| ErrUserAlreadyExists | 409 | AlreadyExists | USER_ALREADY_EXISTS |
| ErrInvalidCredentials | 401 | Unauthenticated | INVALID_CREDENTIALS |
| ErrUnauthorized | 401 | Unauthenticated | UNAUTHORIZED |
| ErrAccessTokenExpired | 401 | Unauthenticated | ACCESS_TOKEN_EXPIRED |
| ErrInvalidAccessToken | 401 | Unauthenticated | INVALID_ACCESS_TOKEN |
| ErrForbidden | 403 | PermissionDenied | FORBIDDEN |
| ErrUserDeleted | 403 | PermissionDenied | USER_DELETED |
| ErrRefreshTokenExpired | 403 | PermissionDenied | REFRESH_TOKEN_EXPIRED |
| ErrRefreshTokenRevoked | 403 | PermissionDenied | REFRESH_TOKEN_REVOKED |
| ErrInvalidInput | 400 | InvalidArgument | INVALID_INPUT |
| ErrInvalidEmail | 400 | InvalidArgument | INVALID_EMAIL |
| ErrWeakPassword | 400 | InvalidArgument | WEAK_PASSWORD |
| ErrInvalidKYCStatus | 400 | InvalidArgument | INVALID_KYC_STATUS |
| ErrInvalidRole | 400 | InvalidArgument | INVALID_ROLE |
| ErrInvalidRefreshToken | 400 | InvalidArgument | INVALID_REFRESH_TOKEN |
| ErrInternalServer | 500 | Internal | INTERNAL_SERVER_ERROR |
| (unknown errors) | 500 | Internal | INTERNAL_SERVER_ERROR |

## Best Practices

1. **Use Sentinel Errors**: Always return predefined domain errors from services
2. **Add Context with WithDetails**: Include field-specific information for validation errors
3. **Let Middleware Handle Conversion**: Don't manually construct error responses
4. **Preserve Trace IDs**: Pass context through all layers for trace correlation
5. **Never Expose Internal Errors**: Unknown errors are automatically sanitized
6. **Test Error Paths**: Ensure all error scenarios are covered in tests

## Integration Checklist

- [x] Domain sentinel errors defined
- [x] AppError struct implemented
- [x] HTTP ErrorMiddleware added to router
- [x] gRPC ErrorInterceptor added to server
- [x] Tests cover all error scenarios
- [x] Security sanitization verified
- [x] Trace ID extraction working
- [x] Documentation complete

## Future Enhancements

- [ ] Add structured logging of errors with trace IDs
- [ ] Implement error metrics (Prometheus counters by error type)
- [ ] Add custom error details for specific business logic failures
- [ ] Support internationalization (i18n) for error messages
- [ ] Add error aggregation for batch operations
