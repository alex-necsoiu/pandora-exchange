package middleware

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/alex-necsoiu/pandora-exchange/pkg/logger"
)

const (
	// IdempotencyKeyHeader is the HTTP header name for idempotency keys
	IdempotencyKeyHeader = "Idempotency-Key"
	
	// DefaultCacheTTL is the default time-to-live for cached responses
	DefaultCacheTTL = 24 * time.Hour
	
	// MaxIdempotencyKeyLength is the maximum allowed length for idempotency keys
	MaxIdempotencyKeyLength = 255
)

// CachedResponse represents a stored HTTP response
type CachedResponse struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
	CachedAt   time.Time
	ExpiresAt  time.Time
}

// IdempotencyStore defines the interface for storing and retrieving cached responses
type IdempotencyStore interface {
	// Get retrieves a cached response by key
	Get(key string) (*CachedResponse, bool)
	
	// Set stores a response with an expiration time
	Set(key string, response *CachedResponse, ttl time.Duration)
	
	// Delete removes a cached response
	Delete(key string)
}

// InMemoryStore is a simple in-memory implementation of IdempotencyStore
// For production, consider using Redis or another distributed cache
type InMemoryStore struct {
	mu    sync.RWMutex
	store map[string]*CachedResponse
	locks sync.Map // Per-key locks for handling concurrent requests with same idempotency key
}

// NewInMemoryStore creates a new in-memory idempotency store
func NewInMemoryStore() *InMemoryStore {
	store := &InMemoryStore{
		store: make(map[string]*CachedResponse),
	}
	
	// Start background cleanup goroutine
	go store.cleanup()
	
	return store
}

// Get retrieves a cached response, checking expiration
func (s *InMemoryStore) Get(key string) (*CachedResponse, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	response, exists := s.store[key]
	if !exists {
		return nil, false
	}
	
	// Check if expired
	if time.Now().After(response.ExpiresAt) {
		return nil, false
	}
	
	return response, true
}

// Set stores a response with TTL
func (s *InMemoryStore) Set(key string, response *CachedResponse, ttl time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	response.ExpiresAt = time.Now().Add(ttl)
	s.store[key] = response
}

// Delete removes a cached response
func (s *InMemoryStore) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	delete(s.store, key)
}

// cleanup periodically removes expired entries
func (s *InMemoryStore) cleanup() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	
	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for key, response := range s.store {
			if now.After(response.ExpiresAt) {
				delete(s.store, key)
			}
		}
		s.mu.Unlock()
	}
}

// acquireLock tries to acquire a per-key lock for processing
// Returns true if lock was acquired, false if another request is processing
func (s *InMemoryStore) acquireLock(key string) bool {
	_, loaded := s.locks.LoadOrStore(key, struct{}{})
	return !loaded
}

// releaseLock releases the per-key lock
func (s *InMemoryStore) releaseLock(key string) {
	s.locks.Delete(key)
}

// IdempotencyConfig holds configuration for the idempotency middleware
type IdempotencyConfig struct {
	// Store is the backing store for cached responses
	Store IdempotencyStore
	
	// TTL is the time-to-live for cached responses
	TTL time.Duration
	
	// IncludeBody determines whether to include request body in key generation
	// Set to true for POST/PUT/PATCH requests
	IncludeBody bool
	
	// OnlyIdempotentMethods restricts middleware to only safe HTTP methods
	// If false, applies to all methods
	OnlyIdempotentMethods bool
	
	// KeyGenerator is an optional custom key generation function
	// If nil, the default generator is used
	KeyGenerator func(c *gin.Context, idempotencyKey string) string
}

// responseWriter is a custom response writer that captures the response
type responseWriter struct {
	gin.ResponseWriter
	body       *bytes.Buffer
	statusCode int
}

func (w *responseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w *responseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

// IdempotencyMiddleware creates a Gin middleware for handling idempotent requests
// It caches responses based on idempotency keys to prevent duplicate operations
func IdempotencyMiddleware(config IdempotencyConfig) gin.HandlerFunc {
	// Set defaults
	if config.Store == nil {
		config.Store = NewInMemoryStore()
	}
	if config.TTL == 0 {
		config.TTL = DefaultCacheTTL
	}
	if config.KeyGenerator == nil {
		config.KeyGenerator = defaultKeyGenerator
	}
	
	// Type assertion to access lock methods (if using InMemoryStore)
	inMemStore, hasLocks := config.Store.(*InMemoryStore)
	
	return func(c *gin.Context) {
		// Extract idempotency key from header
		idempotencyKey := c.GetHeader(IdempotencyKeyHeader)
		
		// If no idempotency key, skip middleware
		if idempotencyKey == "" {
			c.Next()
			return
		}
		
		// Validate idempotency key
		if len(idempotencyKey) > MaxIdempotencyKeyLength {
			log := logger.WithTrace(c.Request.Context(), logger.GetLogger())
			log.Warn().
				Str("key", idempotencyKey[:50]+"...").
				Msg("Idempotency key too long")
			
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "invalid_idempotency_key",
				"message": "idempotency key exceeds maximum length",
			})
			c.Abort()
			return
		}
		
		// Skip for GET/HEAD/OPTIONS if configured
		if config.OnlyIdempotentMethods && !isIdempotentMethod(c.Request.Method) {
			c.Next()
			return
		}
		
		// Generate cache key
		cacheKey := config.KeyGenerator(c, idempotencyKey)
		
		log := logger.WithTrace(c.Request.Context(), logger.GetLogger())
		
		// Check if response is cached
		if cached, found := config.Store.Get(cacheKey); found {
			log.Info().
				Str("key", idempotencyKey).
				Str("method", c.Request.Method).
				Str("path", c.Request.URL.Path).
				Msg("Returning cached idempotent response")
			
			// Replay cached response
			for key, values := range cached.Headers {
				for _, value := range values {
					c.Header(key, value)
				}
			}
			c.Header("X-Idempotency-Replay", "true")
			c.Data(cached.StatusCode, c.GetHeader("Content-Type"), cached.Body)
			c.Abort()
			return
		}
		
		// Handle concurrent requests with the same idempotency key
		if hasLocks {
			// Try to acquire lock
			if !inMemStore.acquireLock(cacheKey) {
				// Another request is processing, wait briefly and retry
				log.Info().
					Str("key", idempotencyKey).
					Msg("Concurrent request detected, waiting")
				
				// Wait for a short time and check if response is now cached
				time.Sleep(100 * time.Millisecond)
				
				if cached, found := config.Store.Get(cacheKey); found {
					log.Info().
						Str("key", idempotencyKey).
						Msg("Response became available during wait")
					
					for key, values := range cached.Headers {
						for _, value := range values {
							c.Header(key, value)
						}
					}
					c.Header("X-Idempotency-Replay", "true")
					c.Data(cached.StatusCode, c.GetHeader("Content-Type"), cached.Body)
					c.Abort()
					return
				}
				
				// Still not available, return conflict
				c.JSON(http.StatusConflict, gin.H{
					"error":   "concurrent_request",
					"message": "another request with the same idempotency key is being processed",
				})
				c.Abort()
				return
			}
			
			// Ensure lock is released when done
			defer inMemStore.releaseLock(cacheKey)
		}
		
		// Wrap response writer to capture response
		writer := &responseWriter{
			ResponseWriter: c.Writer,
			body:           &bytes.Buffer{},
			statusCode:     http.StatusOK,
		}
		c.Writer = writer
		
		// Process request
		c.Next()
		
		// Only cache successful responses (2xx status codes)
		if writer.statusCode >= 200 && writer.statusCode < 300 {
			// Capture headers
			headers := make(http.Header)
			for key, values := range c.Writer.Header() {
				headers[key] = values
			}
			
			// Create cached response
			cached := &CachedResponse{
				StatusCode: writer.statusCode,
				Headers:    headers,
				Body:       writer.body.Bytes(),
				CachedAt:   time.Now(),
			}
			
			// Store in cache
			config.Store.Set(cacheKey, cached, config.TTL)
			
			log.Info().
				Str("key", idempotencyKey).
				Str("method", c.Request.Method).
				Str("path", c.Request.URL.Path).
				Int("status", writer.statusCode).
				Msg("Cached idempotent response")
		}
	}
}

// defaultKeyGenerator creates a cache key from the idempotency key, method, path, and optionally the body
func defaultKeyGenerator(c *gin.Context, idempotencyKey string) string {
	// Include method and path for uniqueness
	base := fmt.Sprintf("%s:%s:%s", idempotencyKey, c.Request.Method, c.Request.URL.Path)
	
	// For POST/PUT/PATCH, include body hash to ensure same operation
	if c.Request.Method == "POST" || c.Request.Method == "PUT" || c.Request.Method == "PATCH" {
		// Read and restore body
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err == nil && len(bodyBytes) > 0 {
			// Restore body for handler
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			
			// Hash body
			hash := sha256.Sum256(bodyBytes)
			base += ":" + hex.EncodeToString(hash[:])
		}
	}
	
	return base
}

// isIdempotentMethod checks if an HTTP method is naturally idempotent
func isIdempotentMethod(method string) bool {
	switch method {
	case "GET", "HEAD", "OPTIONS", "PUT", "DELETE":
		return true
	default:
		return false
	}
}

// IdempotencyMiddlewareWithRedis creates an idempotency middleware using Redis
// This is a placeholder for Redis integration - implement when needed
func IdempotencyMiddlewareWithRedis(redisClient interface{}, ttl time.Duration) gin.HandlerFunc {
	// TODO: Implement Redis-backed store
	// For now, fall back to in-memory
	config := IdempotencyConfig{
		Store: NewInMemoryStore(),
		TTL:   ttl,
	}
	return IdempotencyMiddleware(config)
}

// ClearIdempotencyCache clears a specific idempotency key from the cache
// Useful for testing or manual cache invalidation
func ClearIdempotencyCache(store IdempotencyStore, idempotencyKey string, c *gin.Context) {
	keyGen := defaultKeyGenerator
	cacheKey := keyGen(c, idempotencyKey)
	store.Delete(cacheKey)
}

// SerializeResponse converts a CachedResponse to JSON (for debugging/logging)
func (cr *CachedResponse) SerializeResponse() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"status_code": cr.StatusCode,
		"headers":     cr.Headers,
		"body":        string(cr.Body),
		"cached_at":   cr.CachedAt,
		"expires_at":  cr.ExpiresAt,
	})
}
