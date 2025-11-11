package middleware

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/alex-necsoiu/pandora-exchange/internal/observability"
)

// PrometheusMiddleware returns a Gin middleware that records HTTP metrics
func PrometheusMiddleware(metrics *observability.MetricsCollector) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Record start time
		start := time.Now()

		// Increment active connections
		metrics.IncrementHTTPActiveConnections()
		defer metrics.DecrementHTTPActiveConnections()

		// Get request size
		requestSize := 0
		if c.Request.ContentLength > 0 {
			requestSize = int(c.Request.ContentLength)
		}

		// Process request
		c.Next()

		// Calculate duration
		duration := time.Since(start)

		// Get response details
		status := c.Writer.Status()
		responseSize := c.Writer.Size()
		if responseSize < 0 {
			responseSize = 0
		}

		// Normalize path (replace UUIDs and IDs with placeholders)
		normalizedPath := normalizePath(c.FullPath())
		if normalizedPath == "" {
			normalizedPath = c.Request.URL.Path
		}

		// Record metrics
		metrics.RecordHTTPRequest(
			c.Request.Method,
			normalizedPath,
			statusCodeToString(status),
			duration,
			requestSize,
			responseSize,
		)

		// Check for idempotency cache hit
		if c.GetHeader("X-Idempotency-Replay") == "true" {
			metrics.RecordIdempotencyCacheHit(normalizedPath, c.Request.Method)
		}
	}
}

// normalizePath normalizes the path by using the route pattern instead of actual values
func normalizePath(fullPath string) string {
	// Gin's FullPath() already returns the route pattern like "/users/:id"
	return fullPath
}

// statusCodeToString converts HTTP status code to a string category
func statusCodeToString(code int) string {
	switch {
	case code >= 200 && code < 300:
		return "2xx"
	case code >= 300 && code < 400:
		return "3xx"
	case code >= 400 && code < 500:
		return "4xx"
	case code >= 500 && code < 600:
		return "5xx"
	default:
		return "unknown"
	}
}
