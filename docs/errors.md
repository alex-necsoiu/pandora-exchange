# Error Catalog

> **Complete error reference for Pandora Exchange User Service**  
> **Last Updated:** November 8, 2025

---

## Error Handling Philosophy

Pandora Exchange follows a **domain-driven error handling** approach:

1. **Domain errors** are sentinel errors defined in `/internal/domain/errors.go`
2. **Transport layer** maps domain errors to HTTP/gRPC status codes
3. **No internal errors leaked** to clients (security consideration)
4. **Trace IDs attached** to all error responses (OpenTelemetry)
5. **Structured logging** of all errors (Zerolog)

See [ERROR_HANDLING.md](../ERROR_HANDLING.md) for complete implementation details.

---

## Domain Errors

### Sentinel Errors

Domain errors are pre-defined constant errors used throughout the application:

```go
// internal/domain/errors.go
var (
    ErrUserNotFound       = errors.New("user not found")
    ErrUserAlreadyExists  = errors.New("user already exists")
    ErrInvalidCredentials = errors.New("invalid credentials")
    ErrInvalidInput       = errors.New("invalid input")
    ErrUnauthorized       = errors.New("unauthorized")
    ErrForbidden          = errors.New("forbidden")
    ErrInvalidToken       = errors.New("invalid token")
    ErrTokenExpired       = errors.New("token expired")
    ErrInternalError      = errors.New("internal server error")
)
```

---

## HTTP Error Mapping

| Domain Error | HTTP Status | Error Code | Message | Retry? |
|--------------|-------------|------------|---------|--------|
| `ErrUserNotFound` | 404 | `USER_NOT_FOUND` | "User not found" | No |
| `ErrUserAlreadyExists` | 409 | `USER_ALREADY_EXISTS` | "User already exists with this email" | No |
| `ErrInvalidCredentials` | 401 | `INVALID_CREDENTIALS` | "Invalid email or password" | No |
| `ErrInvalidInput` | 400 | `INVALID_INPUT` | "Invalid input provided" | No |
| `ErrUnauthorized` | 401 | `UNAUTHORIZED` | "Authentication required" | No |
| `ErrForbidden` | 403 | `FORBIDDEN` | "Insufficient permissions" | No |
| `ErrInvalidToken` | 401 | `INVALID_TOKEN` | "Invalid authentication token" | No |
| `ErrTokenExpired` | 401 | `TOKEN_EXPIRED` | "Authentication token expired" | Yes* |
| `ErrInternalError` | 500 | `INTERNAL_ERROR` | "Internal server error" | Yes |

**\*Retry after refreshing token**

---

## HTTP Error Response Format

### Success Response (200-299)
```json
{
  "id": "user-uuid",
  "email": "user@example.com",
  "first_name": "John",
  "last_name": "Doe"
}
```

### Error Response (400-599)
```json
{
  "error": {
    "code": "USER_NOT_FOUND",
    "message": "User not found",
    "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736"
  }
}
```

### Validation Error Response (400)
```json
{
  "error": {
    "code": "INVALID_INPUT",
    "message": "Validation failed",
    "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736",
    "details": {
      "email": "Invalid email format",
      "password": "Password must be at least 8 characters"
    }
  }
}
```

---

## Error Codes by Endpoint

### POST `/auth/register`

| Error Code | HTTP Status | Cause | Solution |
|------------|-------------|-------|----------|
| `INVALID_INPUT` | 400 | Invalid email/password format | Validate input client-side |
| `USER_ALREADY_EXISTS` | 409 | Email already registered | Use different email or login |
| `INTERNAL_ERROR` | 500 | Database/service error | Retry or contact support |

**Example:**
```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"invalid","password":"weak"}'

# Response: 400 Bad Request
{
  "error": {
    "code": "INVALID_INPUT",
    "message": "Invalid input provided",
    "trace_id": "...",
    "details": {
      "email": "Invalid email format",
      "password": "Password must be at least 8 characters"
    }
  }
}
```

---

### POST `/auth/login`

| Error Code | HTTP Status | Cause | Solution |
|------------|-------------|-------|----------|
| `INVALID_INPUT` | 400 | Missing email/password | Provide credentials |
| `INVALID_CREDENTIALS` | 401 | Wrong email/password | Check credentials |
| `USER_NOT_FOUND` | 404 | Email not registered | Register first |
| `INTERNAL_ERROR` | 500 | Database/service error | Retry or contact support |

**Example:**
```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"wrongpass"}'

# Response: 401 Unauthorized
{
  "error": {
    "code": "INVALID_CREDENTIALS",
    "message": "Invalid email or password",
    "trace_id": "..."
  }
}
```

---

### POST `/auth/refresh`

| Error Code | HTTP Status | Cause | Solution |
|------------|-------------|-------|----------|
| `INVALID_INPUT` | 400 | Missing refresh token | Provide refresh_token |
| `INVALID_TOKEN` | 401 | Invalid refresh token | Login again |
| `TOKEN_EXPIRED` | 401 | Refresh token expired | Login again |
| `INTERNAL_ERROR` | 500 | Database/service error | Retry |

---

### GET `/users/me` (Protected)

| Error Code | HTTP Status | Cause | Solution |
|------------|-------------|-------|----------|
| `UNAUTHORIZED` | 401 | Missing Authorization header | Add `Authorization: Bearer <token>` |
| `INVALID_TOKEN` | 401 | Invalid JWT token | Get new token |
| `TOKEN_EXPIRED` | 401 | Access token expired | Refresh token |
| `USER_NOT_FOUND` | 404 | User deleted | Account no longer exists |
| `INTERNAL_ERROR` | 500 | Database/service error | Retry |

---

### PUT `/users/me` (Protected)

| Error Code | HTTP Status | Cause | Solution |
|------------|-------------|-------|----------|
| `UNAUTHORIZED` | 401 | Missing/invalid token | Authenticate |
| `INVALID_INPUT` | 400 | Invalid name format | Validate input |
| `INTERNAL_ERROR` | 500 | Database/service error | Retry |

---

### PATCH `/users/me/kyc` (Protected)

| Error Code | HTTP Status | Cause | Solution |
|------------|-------------|-------|----------|
| `UNAUTHORIZED` | 401 | Missing/invalid token | Authenticate |
| `FORBIDDEN` | 403 | Cannot set status to 'verified' | Only admins can verify |
| `INVALID_INPUT` | 400 | Invalid KYC status | Use: pending, verified, rejected |
| `INTERNAL_ERROR` | 500 | Database/service error | Retry |

---

### DELETE `/users/me` (Protected)

| Error Code | HTTP Status | Cause | Solution |
|------------|-------------|-------|----------|
| `UNAUTHORIZED` | 401 | Missing/invalid token | Authenticate |
| `INTERNAL_ERROR` | 500 | Database/service error | Retry |

---

### GET `/admin/users` (Admin Only)

| Error Code | HTTP Status | Cause | Solution |
|------------|-------------|-------|----------|
| `UNAUTHORIZED` | 401 | Missing/invalid token | Authenticate |
| `FORBIDDEN` | 403 | Not an admin | Only admins can access |
| `INVALID_INPUT` | 400 | Invalid query parameters | Check page/limit values |
| `INTERNAL_ERROR` | 500 | Database/service error | Retry |

---

### GET `/admin/users/:id` (Admin Only)

| Error Code | HTTP Status | Cause | Solution |
|------------|-------------|-------|----------|
| `UNAUTHORIZED` | 401 | Missing/invalid token | Authenticate |
| `FORBIDDEN` | 403 | Not an admin | Only admins can access |
| `USER_NOT_FOUND` | 404 | Invalid user ID | Check ID |
| `INTERNAL_ERROR` | 500 | Database/service error | Retry |

---

### PUT `/admin/users/:id/kyc` (Admin Only)

| Error Code | HTTP Status | Cause | Solution |
|------------|-------------|-------|----------|
| `UNAUTHORIZED` | 401 | Missing/invalid token | Authenticate |
| `FORBIDDEN` | 403 | Not an admin | Only admins can access |
| `USER_NOT_FOUND` | 404 | Invalid user ID | Check ID |
| `INVALID_INPUT` | 400 | Invalid KYC status | Use: pending, verified, rejected |
| `INTERNAL_ERROR` | 500 | Database/service error | Retry |

---

### DELETE `/admin/users/:id` (Admin Only)

| Error Code | HTTP Status | Cause | Solution |
|------------|-------------|-------|----------|
| `UNAUTHORIZED` | 401 | Missing/invalid token | Authenticate |
| `FORBIDDEN` | 403 | Not an admin | Only admins can access |
| `USER_NOT_FOUND` | 404 | Invalid user ID | Check ID |
| `INTERNAL_ERROR` | 500 | Database/service error | Retry |

---

## gRPC Error Mapping (Planned)

When gRPC is implemented, domain errors will map to gRPC status codes:

| Domain Error | gRPC Code | Description |
|--------------|-----------|-------------|
| `ErrUserNotFound` | `NOT_FOUND` | Resource not found |
| `ErrUserAlreadyExists` | `ALREADY_EXISTS` | Resource already exists |
| `ErrInvalidCredentials` | `UNAUTHENTICATED` | Authentication failed |
| `ErrInvalidInput` | `INVALID_ARGUMENT` | Invalid request |
| `ErrUnauthorized` | `UNAUTHENTICATED` | Missing authentication |
| `ErrForbidden` | `PERMISSION_DENIED` | Insufficient permissions |
| `ErrInvalidToken` | `UNAUTHENTICATED` | Invalid credentials |
| `ErrTokenExpired` | `UNAUTHENTICATED` | Credentials expired |
| `ErrInternalError` | `INTERNAL` | Server error |

---

## Error Handling Implementation

### Middleware Pattern

All HTTP errors are handled by the **Error Middleware**:

```go
// internal/middleware/error.go
func ErrorMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Next()
        
        if len(c.Errors) > 0 {
            err := c.Errors.Last().Err
            
            // Map domain error to HTTP error
            appErr := mapErrorToAppError(err, c)
            
            c.JSON(appErr.HTTPStatus, gin.H{
                "error": gin.H{
                    "code":     appErr.Code,
                    "message":  appErr.Message,
                    "trace_id": appErr.TraceID,
                },
            })
        }
    }
}
```

### Error Mapping Function

```go
func mapErrorToAppError(err error, c *gin.Context) *domain.AppError {
    traceID := trace.SpanFromContext(c.Request.Context()).SpanContext().TraceID().String()
    
    switch {
    case errors.Is(err, domain.ErrUserNotFound):
        return domain.NewAppError(err, "USER_NOT_FOUND", "User not found", http.StatusNotFound, traceID)
    case errors.Is(err, domain.ErrUserAlreadyExists):
        return domain.NewAppError(err, "USER_ALREADY_EXISTS", "User already exists", http.StatusConflict, traceID)
    // ... more mappings
    default:
        return domain.NewAppError(err, "INTERNAL_ERROR", "Internal server error", http.StatusInternalServerError, traceID)
    }
}
```

---

## Error Testing

### Test Coverage

All error scenarios are tested:

| Error Type | Test Count | Coverage |
|------------|-----------|----------|
| Domain errors | 10+ tests | 100% |
| Error middleware | 15+ tests | 100% |
| HTTP error mapping | 20+ tests | 100% |
| Validation errors | 15+ tests | 100% |

### Example Test

```go
func TestErrorMiddleware_UserNotFound(t *testing.T) {
    w := httptest.NewRecorder()
    c, _ := gin.CreateTestContext(w)
    
    // Trigger error
    c.Error(domain.ErrUserNotFound)
    
    // Apply middleware
    ErrorMiddleware()(c)
    
    // Assert response
    assert.Equal(t, 404, w.Code)
    
    var response map[string]interface{}
    json.Unmarshal(w.Body.Bytes(), &response)
    
    assert.Equal(t, "USER_NOT_FOUND", response["error"].(map[string]interface{})["code"])
}
```

---

## Error Logging

All errors are logged with structured context:

```json
{
  "level": "error",
  "time": "2025-11-08T10:30:00Z",
  "message": "user not found",
  "error": "user not found",
  "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736",
  "user_id": "user-uuid",
  "endpoint": "/api/v1/users/me",
  "method": "GET",
  "ip": "192.168.1.1"
}
```

**PII Redaction:**
- Passwords never logged
- Email addresses redacted in production
- IP addresses anonymized in production

---

## Client Error Handling Best Practices

### 1. Always Check HTTP Status

```typescript
// TypeScript example
const response = await fetch('/api/v1/auth/login', {
  method: 'POST',
  body: JSON.stringify({ email, password })
});

if (!response.ok) {
  const error = await response.json();
  console.error(error.error.code, error.error.message);
  // Handle error based on code
}
```

### 2. Use Trace IDs for Support

```typescript
if (error.error.trace_id) {
  // Show to user: "Error ID: {trace_id} - Contact support"
  console.log('Trace ID for support:', error.error.trace_id);
}
```

### 3. Retry on 5xx Errors

```typescript
if (response.status >= 500) {
  // Retry with exponential backoff
  await retryWithBackoff(() => makeRequest());
}
```

### 4. Don't Retry on 4xx Errors

```typescript
if (response.status >= 400 && response.status < 500) {
  // Client error - fix input, don't retry
  showValidationError(error.error.message);
}
```

### 5. Handle Token Expiration

```typescript
if (error.error.code === 'TOKEN_EXPIRED') {
  // Refresh token and retry
  await refreshAccessToken();
  return retryRequest();
}
```

---

## Security Considerations

### Never Leak Internal Details

**❌ Bad:**
```json
{
  "error": "sql: no rows in result set - query: SELECT * FROM users WHERE id = $1"
}
```

**✅ Good:**
```json
{
  "error": {
    "code": "USER_NOT_FOUND",
    "message": "User not found",
    "trace_id": "..."
  }
}
```

### Rate Limit Error Responses

Prevent enumeration attacks by rate limiting authentication errors:

- Max 5 failed login attempts per minute per IP
- Return generic error messages ("Invalid credentials" not "Email not found")
- Log suspicious patterns for security monitoring

---

## References

- [ERROR_HANDLING.md](../ERROR_HANDLING.md) - Complete error handling implementation
- [domain/errors.go](../internal/domain/errors.go) - Domain error definitions
- [middleware/error.go](../internal/middleware/error.go) - Error middleware
- [OpenTelemetry Trace IDs](https://opentelemetry.io/docs/concepts/signals/traces/)

---

**Last Updated:** November 8, 2025  
**Maintained By:** Pandora Engineering Team
