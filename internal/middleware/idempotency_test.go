package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestIdempotencyMiddleware_WithoutKey(t *testing.T) {
	// Setup
	store := NewInMemoryStore()
	config := IdempotencyConfig{
		Store: store,
		TTL:   1 * time.Hour,
	}

	router := gin.New()
	router.Use(IdempotencyMiddleware(config))
	router.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Execute - Request without idempotency key
	req := httptest.NewRequest("POST", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Verify - Should process normally
	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotContains(t, w.Header().Get("X-Idempotency-Replay"), "true")
}

func TestIdempotencyMiddleware_CachesSuccessfulResponse(t *testing.T) {
	// Setup
	store := NewInMemoryStore()
	config := IdempotencyConfig{
		Store: store,
		TTL:   1 * time.Hour,
	}

	callCount := 0
	router := gin.New()
	router.Use(IdempotencyMiddleware(config))
	router.POST("/test", func(c *gin.Context) {
		callCount++
		c.JSON(http.StatusOK, gin.H{"message": "success", "count": callCount})
	})

	// First request
	req1 := httptest.NewRequest("POST", "/test", nil)
	req1.Header.Set(IdempotencyKeyHeader, "test-key-123")
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	// Verify first request
	assert.Equal(t, http.StatusOK, w1.Code)
	assert.Equal(t, 1, callCount)
	assert.Empty(t, w1.Header().Get("X-Idempotency-Replay"))

	var body1 map[string]interface{}
	err := json.Unmarshal(w1.Body.Bytes(), &body1)
	require.NoError(t, err)
	assert.Equal(t, float64(1), body1["count"])

	// Second request with same key
	req2 := httptest.NewRequest("POST", "/test", nil)
	req2.Header.Set(IdempotencyKeyHeader, "test-key-123")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	// Verify second request used cache
	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Equal(t, 1, callCount) // Handler not called again
	assert.Equal(t, "true", w2.Header().Get("X-Idempotency-Replay"))

	var body2 map[string]interface{}
	err = json.Unmarshal(w2.Body.Bytes(), &body2)
	require.NoError(t, err)
	assert.Equal(t, float64(1), body2["count"]) // Same response as first
}

func TestIdempotencyMiddleware_DifferentKeysNotCached(t *testing.T) {
	// Setup
	store := NewInMemoryStore()
	config := IdempotencyConfig{
		Store: store,
		TTL:   1 * time.Hour,
	}

	callCount := 0
	router := gin.New()
	router.Use(IdempotencyMiddleware(config))
	router.POST("/test", func(c *gin.Context) {
		callCount++
		c.JSON(http.StatusOK, gin.H{"count": callCount})
	})

	// First request
	req1 := httptest.NewRequest("POST", "/test", nil)
	req1.Header.Set(IdempotencyKeyHeader, "key-1")
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	// Second request with different key
	req2 := httptest.NewRequest("POST", "/test", nil)
	req2.Header.Set(IdempotencyKeyHeader, "key-2")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	// Verify both processed
	assert.Equal(t, 2, callCount)
	assert.Empty(t, w1.Header().Get("X-Idempotency-Replay"))
	assert.Empty(t, w2.Header().Get("X-Idempotency-Replay"))
}

func TestIdempotencyMiddleware_DoesNotCacheErrors(t *testing.T) {
	// Setup
	store := NewInMemoryStore()
	config := IdempotencyConfig{
		Store: store,
		TTL:   1 * time.Hour,
	}

	callCount := 0
	router := gin.New()
	router.Use(IdempotencyMiddleware(config))
	router.POST("/test", func(c *gin.Context) {
		callCount++
		if callCount == 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
		} else {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		}
	})

	// First request (fails)
	req1 := httptest.NewRequest("POST", "/test", nil)
	req1.Header.Set(IdempotencyKeyHeader, "test-key")
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	assert.Equal(t, http.StatusBadRequest, w1.Code)

	// Second request (should process again, not cached)
	req2 := httptest.NewRequest("POST", "/test", nil)
	req2.Header.Set(IdempotencyKeyHeader, "test-key")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	// Verify second request was processed
	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Equal(t, 2, callCount)
	assert.Empty(t, w2.Header().Get("X-Idempotency-Replay"))
}

func TestIdempotencyMiddleware_KeyTooLong(t *testing.T) {
	// Setup
	store := NewInMemoryStore()
	config := IdempotencyConfig{
		Store: store,
		TTL:   1 * time.Hour,
	}

	router := gin.New()
	router.Use(IdempotencyMiddleware(config))
	router.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Request with very long key
	longKey := string(make([]byte, MaxIdempotencyKeyLength+1))
	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set(IdempotencyKeyHeader, longKey)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Verify rejection
	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "invalid_idempotency_key", response["error"])
}

func TestIdempotencyMiddleware_DifferentBodyHashDifferentCache(t *testing.T) {
	// Setup
	store := NewInMemoryStore()
	config := IdempotencyConfig{
		Store:       store,
		TTL:         1 * time.Hour,
		IncludeBody: true,
	}

	callCount := 0
	router := gin.New()
	router.Use(IdempotencyMiddleware(config))
	router.POST("/test", func(c *gin.Context) {
		callCount++
		c.JSON(http.StatusOK, gin.H{"count": callCount})
	})

	// First request with body
	body1 := bytes.NewBufferString(`{"data":"value1"}`)
	req1 := httptest.NewRequest("POST", "/test", body1)
	req1.Header.Set(IdempotencyKeyHeader, "test-key")
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	// Second request with different body (same key)
	body2 := bytes.NewBufferString(`{"data":"value2"}`)
	req2 := httptest.NewRequest("POST", "/test", body2)
	req2.Header.Set(IdempotencyKeyHeader, "test-key")
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	// Verify both processed (different body hashes)
	assert.Equal(t, 2, callCount)
}

func TestIdempotencyMiddleware_SameBodyHashCached(t *testing.T) {
	// Setup
	store := NewInMemoryStore()
	config := IdempotencyConfig{
		Store:       store,
		TTL:         1 * time.Hour,
		IncludeBody: true,
	}

	callCount := 0
	router := gin.New()
	router.Use(IdempotencyMiddleware(config))
	router.POST("/test", func(c *gin.Context) {
		callCount++
		c.JSON(http.StatusOK, gin.H{"count": callCount})
	})

	// First request
	body1 := bytes.NewBufferString(`{"data":"same"}`)
	req1 := httptest.NewRequest("POST", "/test", body1)
	req1.Header.Set(IdempotencyKeyHeader, "test-key")
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	// Second request with same body
	body2 := bytes.NewBufferString(`{"data":"same"}`)
	req2 := httptest.NewRequest("POST", "/test", body2)
	req2.Header.Set(IdempotencyKeyHeader, "test-key")
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	// Verify cache hit
	assert.Equal(t, 1, callCount)
	assert.Equal(t, "true", w2.Header().Get("X-Idempotency-Replay"))
}

func TestIdempotencyMiddleware_ConcurrentRequestsWithSameKey(t *testing.T) {
	// Setup
	store := NewInMemoryStore()
	config := IdempotencyConfig{
		Store: store,
		TTL:   1 * time.Hour,
	}

	callCount := 0
	var mu sync.Mutex
	router := gin.New()
	router.Use(IdempotencyMiddleware(config))
	router.POST("/test", func(c *gin.Context) {
		// Simulate slow operation
		time.Sleep(200 * time.Millisecond)
		mu.Lock()
		callCount++
		mu.Unlock()
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Make concurrent requests
	var wg sync.WaitGroup
	results := make([]int, 3)
	
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			req := httptest.NewRequest("POST", "/test", nil)
			req.Header.Set(IdempotencyKeyHeader, "concurrent-key")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			results[idx] = w.Code
		}(i)
	}

	wg.Wait()

	// Verify: first request processes, others either wait and get cached response or get conflict
	mu.Lock()
	defer mu.Unlock()
	assert.LessOrEqual(t, callCount, 1, "Handler should be called at most once")
	
	// Check that we got either OK or Conflict responses
	for _, code := range results {
		assert.True(t, code == http.StatusOK || code == http.StatusConflict)
	}
}

func TestIdempotencyMiddleware_CustomKeyGenerator(t *testing.T) {
	// Setup
	store := NewInMemoryStore()
	customGen := func(c *gin.Context, idempotencyKey string) string {
		// Custom generator that includes user ID
		userID := c.GetHeader("X-User-ID")
		return idempotencyKey + ":" + userID
	}

	config := IdempotencyConfig{
		Store:        store,
		TTL:          1 * time.Hour,
		KeyGenerator: customGen,
	}

	callCount := 0
	router := gin.New()
	router.Use(IdempotencyMiddleware(config))
	router.POST("/test", func(c *gin.Context) {
		callCount++
		c.JSON(http.StatusOK, gin.H{"count": callCount})
	})

	// First request from user 1
	req1 := httptest.NewRequest("POST", "/test", nil)
	req1.Header.Set(IdempotencyKeyHeader, "test-key")
	req1.Header.Set("X-User-ID", "user-1")
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	// Second request from user 2 (same idempotency key, different user)
	req2 := httptest.NewRequest("POST", "/test", nil)
	req2.Header.Set(IdempotencyKeyHeader, "test-key")
	req2.Header.Set("X-User-ID", "user-2")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	// Verify both processed (different cache keys due to user ID)
	assert.Equal(t, 2, callCount)
	assert.Empty(t, w1.Header().Get("X-Idempotency-Replay"))
	assert.Empty(t, w2.Header().Get("X-Idempotency-Replay"))
}

func TestIdempotencyMiddleware_TTLExpiration(t *testing.T) {
	// Setup with very short TTL
	store := NewInMemoryStore()
	config := IdempotencyConfig{
		Store: store,
		TTL:   100 * time.Millisecond,
	}

	callCount := 0
	router := gin.New()
	router.Use(IdempotencyMiddleware(config))
	router.POST("/test", func(c *gin.Context) {
		callCount++
		c.JSON(http.StatusOK, gin.H{"count": callCount})
	})

	// First request
	req1 := httptest.NewRequest("POST", "/test", nil)
	req1.Header.Set(IdempotencyKeyHeader, "test-key")
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	assert.Equal(t, 1, callCount)

	// Wait for TTL to expire
	time.Sleep(150 * time.Millisecond)

	// Second request after expiration
	req2 := httptest.NewRequest("POST", "/test", nil)
	req2.Header.Set(IdempotencyKeyHeader, "test-key")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	// Verify cache expired, handler called again
	assert.Equal(t, 2, callCount)
	assert.Empty(t, w2.Header().Get("X-Idempotency-Replay"))
}

func TestIdempotencyMiddleware_PreservesHeaders(t *testing.T) {
	// Setup
	store := NewInMemoryStore()
	config := IdempotencyConfig{
		Store: store,
		TTL:   1 * time.Hour,
	}

	router := gin.New()
	router.Use(IdempotencyMiddleware(config))
	router.POST("/test", func(c *gin.Context) {
		c.Header("X-Custom-Header", "custom-value")
		c.Header("X-Request-ID", "12345")
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// First request
	req1 := httptest.NewRequest("POST", "/test", nil)
	req1.Header.Set(IdempotencyKeyHeader, "test-key")
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	// Second request (cached)
	req2 := httptest.NewRequest("POST", "/test", nil)
	req2.Header.Set(IdempotencyKeyHeader, "test-key")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	// Verify headers preserved
	assert.Equal(t, "custom-value", w2.Header().Get("X-Custom-Header"))
	assert.Equal(t, "12345", w2.Header().Get("X-Request-ID"))
	assert.Equal(t, "true", w2.Header().Get("X-Idempotency-Replay"))
}

func TestIdempotencyMiddleware_DifferentPaths(t *testing.T) {
	// Setup
	store := NewInMemoryStore()
	config := IdempotencyConfig{
		Store: store,
		TTL:   1 * time.Hour,
	}

	callCount := 0
	router := gin.New()
	router.Use(IdempotencyMiddleware(config))
	router.POST("/path1", func(c *gin.Context) {
		callCount++
		c.JSON(http.StatusOK, gin.H{"path": "1", "count": callCount})
	})
	router.POST("/path2", func(c *gin.Context) {
		callCount++
		c.JSON(http.StatusOK, gin.H{"path": "2", "count": callCount})
	})

	// Request to path1
	req1 := httptest.NewRequest("POST", "/path1", nil)
	req1.Header.Set(IdempotencyKeyHeader, "test-key")
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	// Request to path2 with same key (should not be cached)
	req2 := httptest.NewRequest("POST", "/path2", nil)
	req2.Header.Set(IdempotencyKeyHeader, "test-key")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	// Verify both processed
	assert.Equal(t, 2, callCount)
	assert.Empty(t, w1.Header().Get("X-Idempotency-Replay"))
	assert.Empty(t, w2.Header().Get("X-Idempotency-Replay"))
}

func TestInMemoryStore_Cleanup(t *testing.T) {
	// Setup store with manual expiration check
	store := NewInMemoryStore()

	// Add expired entry by manually setting it in the map with past expiration
	expired := &CachedResponse{
		StatusCode: 200,
		Body:       []byte("test"),
		CachedAt:   time.Now().Add(-2 * time.Hour),
		ExpiresAt:  time.Now().Add(-1 * time.Hour), // Already expired
	}
	
	// Manually insert expired entry (bypassing Set which would update ExpiresAt)
	store.mu.Lock()
	store.store["expired-key"] = expired
	store.mu.Unlock()

	// Verify it exists in map but Get returns false due to expiration
	store.mu.RLock()
	_, exists := store.store["expired-key"]
	store.mu.RUnlock()
	assert.True(t, exists, "Entry should exist in map")

	_, found := store.Get("expired-key")
	assert.False(t, found, "Get should return false for expired entry")
}

func TestClearIdempotencyCache(t *testing.T) {
	// Setup
	store := NewInMemoryStore()
	config := IdempotencyConfig{
		Store: store,
		TTL:   1 * time.Hour,
	}

	router := gin.New()
	router.Use(IdempotencyMiddleware(config))
	router.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Create cached response
	req1 := httptest.NewRequest("POST", "/test", nil)
	req1.Header.Set(IdempotencyKeyHeader, "test-key")
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	// Verify cached
	req2 := httptest.NewRequest("POST", "/test", nil)
	req2.Header.Set(IdempotencyKeyHeader, "test-key")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	assert.Equal(t, "true", w2.Header().Get("X-Idempotency-Replay"))

	// Clear cache
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = req2
	ClearIdempotencyCache(store, "test-key", c)

	// Verify cache cleared
	req3 := httptest.NewRequest("POST", "/test", nil)
	req3.Header.Set(IdempotencyKeyHeader, "test-key")
	w3 := httptest.NewRecorder()
	router.ServeHTTP(w3, req3)
	assert.Empty(t, w3.Header().Get("X-Idempotency-Replay"))
}

func TestCachedResponse_Serialize(t *testing.T) {
	// Setup
	cached := &CachedResponse{
		StatusCode: 200,
		Headers: http.Header{
			"Content-Type": []string{"application/json"},
			"X-Custom":     []string{"value"},
		},
		Body:      []byte(`{"test":"data"}`),
		CachedAt:  time.Now(),
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	// Serialize
	data, err := cached.SerializeResponse()
	require.NoError(t, err)

	// Verify
	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)
	assert.Equal(t, float64(200), result["status_code"])
	assert.Contains(t, result, "headers")
	assert.Contains(t, result, "cached_at")
}
