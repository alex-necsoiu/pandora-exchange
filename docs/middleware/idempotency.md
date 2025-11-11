# Idempotency Middleware

## Overview

The idempotency middleware provides protection against duplicate operations in HTTP endpoints. It caches successful responses based on client-provided idempotency keys, ensuring that retried requests return the same result without re-executing the operation.

## Features

- **Request Deduplication**: Prevents duplicate processing of identical requests
- **Configurable TTL**: Control how long responses are cached
- **Body Hash Validation**: Ensures request body matches for POST/PUT/PATCH operations
- **Concurrent Request Handling**: Safely handles multiple concurrent requests with same idempotency key
- **Custom Key Generation**: Support for custom cache key generation strategies
- **Header Preservation**: Maintains all response headers in cached responses
- **Selective Caching**: Only caches successful (2xx) responses

## Usage

### Basic Setup

```go
import (
    "time"
    "github.com/alex-necsoiu/pandora-exchange/internal/middleware"
    "github.com/gin-gonic/gin"
)

// Create in-memory store (for development/testing)
store := middleware.NewInMemoryStore()

// Configure middleware
config := middleware.IdempotencyConfig{
    Store: store,
    TTL:   24 * time.Hour,
}

// Apply to router
router := gin.New()
router.Use(middleware.IdempotencyMiddleware(config))
```

### Production Setup with Redis

For production environments with multiple instances, use a distributed cache like Redis:

```go
// TODO: Implement Redis-backed store
router.Use(middleware.IdempotencyMiddlewareWithRedis(redisClient, 24*time.Hour))
```

### Making Idempotent Requests

Clients include an `Idempotency-Key` header with a unique value:

```bash
curl -X POST https://api.pandora.com/api/v1/users \
  -H "Idempotency-Key: 550e8400-e29b-41d4-a716-446655440000" \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"secure123"}'
```

Subsequent requests with the same key return the cached response:

```bash
# Same request - returns cached response
curl -X POST https://api.pandora.com/api/v1/users \
  -H "Idempotency-Key: 550e8400-e29b-41d4-a716-446655440000" \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"secure123"}'

# Response includes: X-Idempotency-Replay: true
```

## Configuration

### IdempotencyConfig Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `Store` | `IdempotencyStore` | `NewInMemoryStore()` | Backing store for cached responses |
| `TTL` | `time.Duration` | `24h` | Time-to-live for cached responses |
| `IncludeBody` | `bool` | `true` (for POST/PUT/PATCH) | Include request body in cache key |
| `OnlyIdempotentMethods` | `bool` | `false` | Restrict to GET/HEAD/OPTIONS/PUT/DELETE only |
| `KeyGenerator` | `func` | `defaultKeyGenerator` | Custom key generation function |

### Custom Key Generator

Create custom cache keys based on user context or other factors:

```go
customGen := func(c *gin.Context, idempotencyKey string) string {
    userID := c.GetHeader("X-User-ID")
    return fmt.Sprintf("%s:user:%s", idempotencyKey, userID)
}

config := middleware.IdempotencyConfig{
    Store:        store,
    TTL:          24 * time.Hour,
    KeyGenerator: customGen,
}
```

## How It Works

### Cache Key Generation

The default key generator creates a unique cache key from:

1. **Idempotency Key**: Client-provided unique identifier
2. **HTTP Method**: POST, PUT, PATCH, etc.
3. **Request Path**: `/api/v1/users`
4. **Body Hash**: SHA-256 hash of request body (for mutating methods)

Example cache key:
```
550e8400-e29b-41d4-a716-446655440000:POST:/api/v1/users:a3c7f8e9...
```

### Request Flow

```
Client Request with Idempotency-Key
         ↓
Validate Key (length, format)
         ↓
Generate Cache Key
         ↓
Check Cache ────→ Cache Hit ────→ Return Cached Response
         ↓                         (X-Idempotency-Replay: true)
    Cache Miss
         ↓
Acquire Lock (prevent concurrent processing)
         ↓
Process Request
         ↓
Capture Response
         ↓
Cache Response (if 2xx status)
         ↓
Release Lock
         ↓
Return Response
```

### Concurrent Request Handling

When multiple requests arrive with the same idempotency key:

1. **First Request**: Acquires lock, processes normally
2. **Concurrent Requests**: 
   - Wait briefly (100ms)
   - Check if response is now cached
   - If cached: Return cached response
   - If not cached: Return 409 Conflict

This prevents duplicate processing while handling network retries gracefully.

## Best Practices

### Idempotency Key Guidelines

1. **Use UUIDs**: `550e8400-e29b-41d4-a716-446655440000`
2. **Client-Generated**: Key must be unique per operation
3. **Consistent**: Same key for retries of same operation
4. **Scoped**: Different operations need different keys

### When to Use

**Recommended for:**
- User registration
- Payment processing
- Order creation
- KYC submissions
- Any state-changing operation that should not be duplicated

**Not needed for:**
- Read operations (GET)
- Health checks
- Metrics endpoints
- Idempotent-by-nature operations (pure PUT, DELETE)

### TTL Selection

Choose TTL based on operation characteristics:

- **Critical Financial**: 7 days (allow for delayed retries)
- **User Registration**: 24 hours (typical retry window)
- **Session Operations**: 1 hour (short-lived context)
- **Background Jobs**: 15 minutes (quick retry cycles)

## Error Handling

### Validation Errors

**Key Too Long** (>255 characters):
```json
{
  "error": "invalid_idempotency_key",
  "message": "idempotency key exceeds maximum length"
}
```
HTTP Status: `400 Bad Request`

**Concurrent Processing**:
```json
{
  "error": "concurrent_request",
  "message": "another request with the same idempotency key is being processed"
}
```
HTTP Status: `409 Conflict`

### Error Response Caching

The middleware **only caches successful (2xx) responses**. Error responses are never cached, allowing clients to retry after fixing issues.

## Integration Examples

### Apply to Specific Routes

```go
// Apply only to user creation endpoint
router.POST("/api/v1/users", 
    middleware.IdempotencyMiddleware(config),
    handler.CreateUser,
)
```

### Global Application with Exclusions

```go
// Apply globally
router.Use(middleware.IdempotencyMiddleware(config))

// Skip for specific endpoints (no Idempotency-Key header required)
router.GET("/health", handler.HealthCheck)
router.GET("/metrics", handler.Metrics)
```

### Per-Endpoint Configuration

```go
// Short TTL for session operations
sessionConfig := middleware.IdempotencyConfig{
    Store: store,
    TTL:   1 * time.Hour,
}
router.POST("/auth/login", 
    middleware.IdempotencyMiddleware(sessionConfig),
    handler.Login,
)

// Long TTL for financial operations
paymentConfig := middleware.IdempotencyConfig{
    Store: store,
    TTL:   7 * 24 * time.Hour,
}
router.POST("/payments", 
    middleware.IdempotencyMiddleware(paymentConfig),
    handler.ProcessPayment,
)
```

## Monitoring

### Cache Hit Rate

Monitor the `X-Idempotency-Replay` header to track cache effectiveness:

```go
// In logging middleware
if c.GetHeader("X-Idempotency-Replay") == "true" {
    metrics.IdempotencyCacheHits.Inc()
} else if c.GetHeader(middleware.IdempotencyKeyHeader) != "" {
    metrics.IdempotencyCacheMisses.Inc()
}
```

### Cache Size

For in-memory stores, monitor memory usage:

```go
store := middleware.NewInMemoryStore()

// Expose metrics endpoint
router.GET("/debug/cache/size", func(c *gin.Context) {
    store.mu.RLock()
    size := len(store.store)
    store.mu.RUnlock()
    
    c.JSON(200, gin.H{"cache_entries": size})
})
```

## Testing

### Unit Testing

```go
func TestIdempotentEndpoint(t *testing.T) {
    store := middleware.NewInMemoryStore()
    config := middleware.IdempotencyConfig{
        Store: store,
        TTL:   1 * time.Hour,
    }

    router := gin.New()
    router.Use(middleware.IdempotencyMiddleware(config))
    router.POST("/test", handler.CreateResource)

    // First request
    req1 := httptest.NewRequest("POST", "/test", body)
    req1.Header.Set("Idempotency-Key", "test-key")
    w1 := httptest.NewRecorder()
    router.ServeHTTP(w1, req1)

    // Verify response
    assert.Equal(t, http.StatusOK, w1.Code)

    // Retry with same key
    req2 := httptest.NewRequest("POST", "/test", body)
    req2.Header.Set("Idempotency-Key", "test-key")
    w2 := httptest.NewRecorder()
    router.ServeHTTP(w2, req2)

    // Verify cached response
    assert.Equal(t, "true", w2.Header().Get("X-Idempotency-Replay"))
    assert.Equal(t, w1.Body.String(), w2.Body.String())
}
```

### Cache Clearing (for testing)

```go
middleware.ClearIdempotencyCache(store, "test-key", c)
```

## Performance Considerations

### In-Memory Store

**Pros:**
- Fast access (no network overhead)
- Simple setup

**Cons:**
- Not shared across instances
- Memory usage grows with cache size
- Lost on restart

**Use for:**
- Development
- Single-instance deployments
- Testing

### Redis Store (Future)

**Pros:**
- Shared across instances
- Persistent storage
- Configurable eviction policies

**Cons:**
- Network latency
- Additional infrastructure

**Use for:**
- Production with multiple instances
- Long TTL requirements
- High availability needs

## Security Considerations

1. **Key Uniqueness**: Clients must generate cryptographically random keys
2. **Key Length Limit**: Maximum 255 characters prevents abuse
3. **User Scoping**: Consider including user ID in custom key generator
4. **Cache Poisoning**: Only successful responses cached
5. **Memory Limits**: In-memory store has no built-in size limits (use Redis for production)

## Migration Path

### Phase 1: Development (Current)
- In-memory store
- Applied to critical endpoints only

### Phase 2: Production
- Implement Redis-backed store
- Apply globally to all mutating operations
- Add monitoring and alerting

### Phase 3: Enhancement
- Distributed locking with Redis
- Configurable eviction policies
- Advanced cache analytics

## References

- [Stripe API Idempotency](https://stripe.com/docs/api/idempotent_requests)
- [RFC 7231 - HTTP/1.1 Semantics and Content](https://tools.ietf.org/html/rfc7231)
- [IETF Draft - The Idempotency-Key HTTP Header Field](https://datatracker.ietf.org/doc/html/draft-ietf-httpapi-idempotency-key-header)
