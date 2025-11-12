package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/alex-necsoiu/pandora-exchange/internal/observability"
	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIdempotencyMiddlewareWithRedis_Integration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	// Setup Redis
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()
	
	logger := observability.NewLogger("test", "test-service")
	
	// Create middleware with Redis
	middleware := IdempotencyMiddlewareWithRedis(client, "test:idempotency:", 1*time.Hour, logger)
	
	router := gin.New()
	router.Use(middleware)
	
	callCount := 0
	router.POST("/api/test", func(c *gin.Context) {
		callCount++
		c.JSON(http.StatusOK, gin.H{
			"message": "created",
			"count":   callCount,
		})
	})
	
	// First request
	req1 := httptest.NewRequest("POST", "/api/test", bytes.NewBuffer([]byte(`{"data":"value"}`)))
	req1.Header.Set("Idempotency-Key", "test-key-123")
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)
	
	assert.Equal(t, http.StatusOK, w1.Code)
	assert.Equal(t, 1, callCount, "handler should be called once")
	assert.NotContains(t, w1.Header().Get("X-Idempotency-Replay"), "true")
	
	// Second request with same key - should be cached
	req2 := httptest.NewRequest("POST", "/api/test", bytes.NewBuffer([]byte(`{"data":"value"}`)))
	req2.Header.Set("Idempotency-Key", "test-key-123")
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	
	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Equal(t, 1, callCount, "handler should NOT be called again")
	assert.Equal(t, "true", w2.Header().Get("X-Idempotency-Replay"))
	assert.Equal(t, w1.Body.String(), w2.Body.String(), "responses should match")
}

func TestIdempotencyMiddlewareWithRedis_ConcurrentRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	// Setup Redis
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()
	
	logger := observability.NewLogger("test", "test-service")
	
	// Create middleware with Redis
	middleware := IdempotencyMiddlewareWithRedis(client, "test:idempotency:", 1*time.Hour, logger)
	
	router := gin.New()
	router.Use(middleware)
	
	var callCount int
	var mu sync.Mutex
	
	router.POST("/api/slow", func(c *gin.Context) {
		mu.Lock()
		callCount++
		mu.Unlock()
		
		// Simulate slow processing
		time.Sleep(200 * time.Millisecond)
		
		c.JSON(http.StatusOK, gin.H{
			"message": "processed",
		})
	})
	
	// Send concurrent requests with same idempotency key
	var wg sync.WaitGroup
	results := make(chan int, 5)
	
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			
			req := httptest.NewRequest("POST", "/api/slow", bytes.NewBuffer([]byte(`{"data":"test"}`)))
			req.Header.Set("Idempotency-Key", "concurrent-key")
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			results <- w.Code
		}()
	}
	
	wg.Wait()
	close(results)
	
	// Collect results
	statusCodes := make(map[int]int)
	for code := range results {
		statusCodes[code]++
	}
	
	// Either we get 1 success + 4 conflicts, or 1 success + 4 cached responses
	// Depends on timing
	mu.Lock()
	finalCallCount := callCount
	mu.Unlock()
	
	// Handler should be called at most once
	assert.LessOrEqual(t, finalCallCount, 1, "handler should be called at most once for concurrent requests")
}

func TestIdempotencyMiddlewareWithRedis_DifferentInstances(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	// Setup Redis
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()
	
	logger := observability.NewLogger("test", "test-service")
	
	// Create two separate router instances (simulating different app instances)
	middleware1 := IdempotencyMiddlewareWithRedis(client, "test:idempotency:", 1*time.Hour, logger)
	middleware2 := IdempotencyMiddlewareWithRedis(client, "test:idempotency:", 1*time.Hour, logger)
	
	router1 := gin.New()
	router1.Use(middleware1)
	
	router2 := gin.New()
	router2.Use(middleware2)
	
	callCount := 0
	handler := func(c *gin.Context) {
		callCount++
		c.JSON(http.StatusOK, gin.H{"count": callCount})
	}
	
	router1.POST("/api/test", handler)
	router2.POST("/api/test", handler)
	
	// Request to instance 1
	req1 := httptest.NewRequest("POST", "/api/test", bytes.NewBuffer([]byte(`{"data":"value"}`)))
	req1.Header.Set("Idempotency-Key", "shared-key")
	w1 := httptest.NewRecorder()
	router1.ServeHTTP(w1, req1)
	
	assert.Equal(t, http.StatusOK, w1.Code)
	assert.Equal(t, 1, callCount)
	
	// Same request to instance 2 - should return cached response
	req2 := httptest.NewRequest("POST", "/api/test", bytes.NewBuffer([]byte(`{"data":"value"}`)))
	req2.Header.Set("Idempotency-Key", "shared-key")
	w2 := httptest.NewRecorder()
	router2.ServeHTTP(w2, req2)
	
	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Equal(t, 1, callCount, "handler should NOT be called on second instance")
	assert.Equal(t, "true", w2.Header().Get("X-Idempotency-Replay"))
}

func TestIdempotencyMiddlewareWithRedis_Expiration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	// Setup Redis
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()
	
	logger := observability.NewLogger("test", "test-service")
	
	// Create middleware with short TTL
	middleware := IdempotencyMiddlewareWithRedis(client, "test:idempotency:", 1*time.Second, logger)
	
	router := gin.New()
	router.Use(middleware)
	
	callCount := 0
	router.POST("/api/test", func(c *gin.Context) {
		callCount++
		c.JSON(http.StatusOK, gin.H{"count": callCount})
	})
	
	// First request
	req1 := httptest.NewRequest("POST", "/api/test", bytes.NewBuffer([]byte(`{"data":"value"}`)))
	req1.Header.Set("Idempotency-Key", "expiring-key")
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)
	
	assert.Equal(t, http.StatusOK, w1.Code)
	assert.Equal(t, 1, callCount)
	
	// Fast-forward past TTL
	mr.FastForward(2 * time.Second)
	
	// Request after expiration - should NOT be cached
	req2 := httptest.NewRequest("POST", "/api/test", bytes.NewBuffer([]byte(`{"data":"value"}`)))
	req2.Header.Set("Idempotency-Key", "expiring-key")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	
	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Equal(t, 2, callCount, "handler should be called again after expiration")
	assert.NotEqual(t, "true", w2.Header().Get("X-Idempotency-Replay"))
}

func TestIdempotencyMiddlewareWithRedis_DifferentPrefixes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	// Setup Redis
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()
	
	logger := observability.NewLogger("test", "test-service")
	
	// Create two middlewares with different prefixes
	middleware1 := IdempotencyMiddlewareWithRedis(client, "app1:", 1*time.Hour, logger)
	middleware2 := IdempotencyMiddlewareWithRedis(client, "app2:", 1*time.Hour, logger)
	
	router1 := gin.New()
	router1.Use(middleware1)
	
	router2 := gin.New()
	router2.Use(middleware2)
	
	callCount1 := 0
	callCount2 := 0
	
	router1.POST("/api/test", func(c *gin.Context) {
		callCount1++
		c.JSON(http.StatusOK, gin.H{"app": "app1", "count": callCount1})
	})
	
	router2.POST("/api/test", func(c *gin.Context) {
		callCount2++
		c.JSON(http.StatusOK, gin.H{"app": "app2", "count": callCount2})
	})
	
	// Same idempotency key to both apps
	req1 := httptest.NewRequest("POST", "/api/test", bytes.NewBuffer([]byte(`{"data":"value"}`)))
	req1.Header.Set("Idempotency-Key", "shared-key")
	w1 := httptest.NewRecorder()
	router1.ServeHTTP(w1, req1)
	
	req2 := httptest.NewRequest("POST", "/api/test", bytes.NewBuffer([]byte(`{"data":"value"}`)))
	req2.Header.Set("Idempotency-Key", "shared-key")
	w2 := httptest.NewRecorder()
	router2.ServeHTTP(w2, req2)
	
	// Both should process (different prefixes = different cache keys)
	assert.Equal(t, 1, callCount1)
	assert.Equal(t, 1, callCount2)
	assert.Contains(t, w1.Body.String(), "app1")
	assert.Contains(t, w2.Body.String(), "app2")
}

func BenchmarkIdempotencyMiddlewareWithRedis(b *testing.B) {
	gin.SetMode(gin.ReleaseMode)
	
	// Setup Redis
	mr, err := miniredis.Run()
	require.NoError(b, err)
	defer mr.Close()
	
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer client.Close()
	
	logger := observability.NewLogger("test", "test-service")
	
	middleware := IdempotencyMiddlewareWithRedis(client, "bench:", 1*time.Hour, logger)
	
	router := gin.New()
	router.Use(middleware)
	router.POST("/api/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/api/test", bytes.NewBuffer([]byte(`{"data":"test"}`)))
		req.Header.Set("Idempotency-Key", "bench-key")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}
