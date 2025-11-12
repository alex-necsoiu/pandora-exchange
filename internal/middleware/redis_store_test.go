package middleware

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestRedis creates a test Redis server using miniredis
func setupTestRedis(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	// Create in-memory Redis server
	mr, err := miniredis.Run()
	require.NoError(t, err)
	
	// Create Redis client
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	
	// Verify connection
	ctx := context.Background()
	err = client.Ping(ctx).Err()
	require.NoError(t, err, "failed to connect to test Redis")
	
	return mr, client
}

func TestNewRedisStore(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()
	
	store := NewRedisStore(client, "test:")
	
	assert.NotNil(t, store)
	assert.Equal(t, "test:", store.keyPrefix)
}

func TestRedisStore_SetAndGet(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()
	
	store := NewRedisStore(client, "test:")
	
	testCases := []struct {
		name     string
		key      string
		response *CachedResponse
		ttl      time.Duration
	}{
		{
			name: "simple response",
			key:  "key1",
			response: &CachedResponse{
				StatusCode: 200,
				Body:       []byte("test body"),
				CachedAt:   time.Now(),
			},
			ttl: 1 * time.Hour,
		},
		{
			name: "response with headers",
			key:  "key2",
			response: &CachedResponse{
				StatusCode: 201,
				Headers: map[string][]string{
					"Content-Type":  {"application/json"},
					"X-Custom-Header": {"value1", "value2"},
				},
				Body:     []byte(`{"status":"created"}`),
				CachedAt: time.Now(),
			},
			ttl: 30 * time.Minute,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set response
			store.Set(tc.key, tc.response, tc.ttl)
			
			// Get response
			retrieved, found := store.Get(tc.key)
			
			assert.True(t, found, "response should be found")
			require.NotNil(t, retrieved)
			assert.Equal(t, tc.response.StatusCode, retrieved.StatusCode)
			assert.Equal(t, tc.response.Body, retrieved.Body)
			assert.Equal(t, tc.response.Headers, retrieved.Headers)
			
			// Verify expiration is set correctly
			assert.False(t, retrieved.ExpiresAt.IsZero())
			assert.True(t, retrieved.ExpiresAt.After(time.Now()))
		})
	}
}

func TestRedisStore_GetNonExistent(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()
	
	store := NewRedisStore(client, "test:")
	
	// Try to get non-existent key
	retrieved, found := store.Get("non-existent")
	
	assert.False(t, found, "should not find non-existent key")
	assert.Nil(t, retrieved)
}

func TestRedisStore_Delete(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()
	
	store := NewRedisStore(client, "test:")
	
	// Set a response
	response := &CachedResponse{
		StatusCode: 200,
		Body:       []byte("test"),
		CachedAt:   time.Now(),
	}
	store.Set("key1", response, 1*time.Hour)
	
	// Verify it exists
	_, found := store.Get("key1")
	assert.True(t, found)
	
	// Delete it
	store.Delete("key1")
	
	// Verify it's gone
	_, found = store.Get("key1")
	assert.False(t, found)
}

func TestRedisStore_Expiration(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()
	
	store := NewRedisStore(client, "test:")
	
	// Set a response with 1 second TTL
	response := &CachedResponse{
		StatusCode: 200,
		Body:       []byte("test"),
		CachedAt:   time.Now(),
	}
	store.Set("key1", response, 1*time.Second)
	
	// Should exist immediately
	_, found := store.Get("key1")
	assert.True(t, found)
	
	// Fast-forward time in miniredis
	mr.FastForward(2 * time.Second)
	
	// Should be expired
	_, found = store.Get("key1")
	assert.False(t, found, "response should be expired")
}

func TestRedisStore_KeyPrefix(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()
	
	// Create two stores with different prefixes
	store1 := NewRedisStore(client, "app1:")
	store2 := NewRedisStore(client, "app2:")
	
	// Set same key in both stores
	response1 := &CachedResponse{
		StatusCode: 200,
		Body:       []byte("app1 response"),
		CachedAt:   time.Now(),
	}
	response2 := &CachedResponse{
		StatusCode: 201,
		Body:       []byte("app2 response"),
		CachedAt:   time.Now(),
	}
	
	store1.Set("shared-key", response1, 1*time.Hour)
	store2.Set("shared-key", response2, 1*time.Hour)
	
	// Retrieve from both stores
	retrieved1, found1 := store1.Get("shared-key")
	retrieved2, found2 := store2.Get("shared-key")
	
	// Both should exist but be different
	assert.True(t, found1)
	assert.True(t, found2)
	assert.Equal(t, []byte("app1 response"), retrieved1.Body)
	assert.Equal(t, []byte("app2 response"), retrieved2.Body)
}

func TestRedisStore_AcquireAndReleaseLock(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()
	
	store := NewRedisStore(client, "test:")
	
	testCases := []struct {
		name     string
		key      string
		setup    func()
		expected bool
	}{
		{
			name:     "acquire lock successfully",
			key:      "lock1",
			setup:    func() {},
			expected: true,
		},
		{
			name: "fail to acquire locked key",
			key:  "lock2",
			setup: func() {
				// Pre-acquire the lock
				acquired := store.AcquireLock("lock2", 10*time.Second)
				require.True(t, acquired)
			},
			expected: false,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()
			
			acquired := store.AcquireLock(tc.key, 10*time.Second)
			assert.Equal(t, tc.expected, acquired)
			
			if acquired {
				// Clean up - release lock
				store.ReleaseLock(tc.key)
			}
		})
	}
}

func TestRedisStore_LockExpiration(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()
	
	store := NewRedisStore(client, "test:")
	
	// Acquire lock with 1 second TTL
	acquired := store.AcquireLock("lock1", 1*time.Second)
	assert.True(t, acquired)
	
	// Should fail to re-acquire immediately
	acquired = store.AcquireLock("lock1", 1*time.Second)
	assert.False(t, acquired)
	
	// Fast-forward past TTL
	mr.FastForward(2 * time.Second)
	
	// Should be able to acquire now
	acquired = store.AcquireLock("lock1", 1*time.Second)
	assert.True(t, acquired, "lock should have expired")
	
	// Clean up
	store.ReleaseLock("lock1")
}

func TestRedisStore_ReleaseLock(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()
	
	store := NewRedisStore(client, "test:")
	
	// Acquire lock
	acquired := store.AcquireLock("lock1", 10*time.Second)
	assert.True(t, acquired)
	
	// Release it
	store.ReleaseLock("lock1")
	
	// Should be able to re-acquire immediately
	acquired = store.AcquireLock("lock1", 10*time.Second)
	assert.True(t, acquired, "lock should have been released")
	
	// Clean up
	store.ReleaseLock("lock1")
}

func TestRedisStore_ConcurrentLocks(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()
	
	store := NewRedisStore(client, "test:")
	
	// Test concurrent lock acquisition
	key := "concurrent-lock"
	
	// First acquisition should succeed
	acquired1 := store.AcquireLock(key, 5*time.Second)
	assert.True(t, acquired1)
	
	// Second acquisition should fail (lock held)
	acquired2 := store.AcquireLock(key, 5*time.Second)
	assert.False(t, acquired2)
	
	// Release first lock
	store.ReleaseLock(key)
	
	// Now second acquisition should succeed
	acquired3 := store.AcquireLock(key, 5*time.Second)
	assert.True(t, acquired3)
	
	// Clean up
	store.ReleaseLock(key)
}

func TestRedisStore_LargePayload(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()
	
	store := NewRedisStore(client, "test:")
	
	// Create large payload (1MB)
	largeBody := make([]byte, 1024*1024)
	for i := range largeBody {
		largeBody[i] = byte(i % 256)
	}
	
	response := &CachedResponse{
		StatusCode: 200,
		Body:       largeBody,
		Headers: map[string][]string{
			"Content-Type": {"application/octet-stream"},
		},
		CachedAt: time.Now(),
	}
	
	// Set large response
	store.Set("large-key", response, 1*time.Hour)
	
	// Retrieve and verify
	retrieved, found := store.Get("large-key")
	assert.True(t, found)
	require.NotNil(t, retrieved)
	assert.Equal(t, len(largeBody), len(retrieved.Body))
	assert.Equal(t, largeBody, retrieved.Body)
}

func TestRedisStore_ConnectionError(t *testing.T) {
	// Create client with invalid address
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:9999", // Non-existent Redis
	})
	defer client.Close()
	
	store := NewRedisStore(client, "test:")
	
	// Operations should handle errors gracefully
	response := &CachedResponse{
		StatusCode: 200,
		Body:       []byte("test"),
		CachedAt:   time.Now(),
	}
	
	// Set should not panic
	assert.NotPanics(t, func() {
		store.Set("key1", response, 1*time.Hour)
	})
	
	// Get should return not found
	retrieved, found := store.Get("key1")
	assert.False(t, found)
	assert.Nil(t, retrieved)
	
	// Delete should not panic
	assert.NotPanics(t, func() {
		store.Delete("key1")
	})
}
