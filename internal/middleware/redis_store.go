package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisStore is a Redis-backed implementation of IdempotencyStore
// It supports distributed caching and locking for multi-instance deployments
type RedisStore struct {
	client    *redis.Client
	keyPrefix string
	ctx       context.Context
}

// NewRedisStore creates a new Redis-backed idempotency store
// keyPrefix is prepended to all keys to avoid collisions (e.g., "idempotency:")
func NewRedisStore(client *redis.Client, keyPrefix string) *RedisStore {
	return &RedisStore{
		client:    client,
		keyPrefix: keyPrefix,
		ctx:       context.Background(),
	}
}

// Get retrieves a cached response from Redis
func (s *RedisStore) Get(key string) (*CachedResponse, bool) {
	fullKey := s.keyPrefix + key
	
	// Get JSON data from Redis
	data, err := s.client.Get(s.ctx, fullKey).Bytes()
	if err == redis.Nil {
		// Key doesn't exist
		return nil, false
	}
	if err != nil {
		// Redis error - log and return not found
		// In production, you'd log this error
		return nil, false
	}
	
	// Deserialize cached response
	var response CachedResponse
	if err := json.Unmarshal(data, &response); err != nil {
		// Corrupted data - return not found
		return nil, false
	}
	
	// Check if expired (Redis TTL handles this, but double-check)
	if time.Now().After(response.ExpiresAt) {
		return nil, false
	}
	
	return &response, true
}

// Set stores a response in Redis with TTL
func (s *RedisStore) Set(key string, response *CachedResponse, ttl time.Duration) {
	fullKey := s.keyPrefix + key
	
	// Set expiration
	response.ExpiresAt = time.Now().Add(ttl)
	
	// Serialize response to JSON
	data, err := json.Marshal(response)
	if err != nil {
		// Failed to serialize - log error in production
		return
	}
	
	// Store in Redis with TTL
	err = s.client.Set(s.ctx, fullKey, data, ttl).Err()
	if err != nil {
		// Redis error - log in production
		return
	}
}

// Delete removes a cached response from Redis
func (s *RedisStore) Delete(key string) {
	fullKey := s.keyPrefix + key
	
	err := s.client.Del(s.ctx, fullKey).Err()
	if err != nil {
		// Redis error - log in production
		return
	}
}

// AcquireLock attempts to acquire a distributed lock for the given key
// Returns true if lock was acquired, false if already held by another process
// The lock expires after the specified TTL to prevent deadlocks
func (s *RedisStore) AcquireLock(key string, ttl time.Duration) bool {
	lockKey := s.keyPrefix + "lock:" + key
	
	// Use SET NX (set if not exists) with expiration
	// This is atomic in Redis
	success, err := s.client.SetNX(s.ctx, lockKey, "locked", ttl).Result()
	if err != nil {
		// Redis error - fail closed (don't acquire lock)
		return false
	}
	
	return success
}

// ReleaseLock releases a distributed lock
func (s *RedisStore) ReleaseLock(key string) {
	lockKey := s.keyPrefix + "lock:" + key
	
	err := s.client.Del(s.ctx, lockKey).Err()
	if err != nil {
		// Redis error - log in production
		// Lock will expire anyway due to TTL
		return
	}
}

// Ping checks if Redis is available
func (s *RedisStore) Ping() error {
	return s.client.Ping(s.ctx).Err()
}

// Close closes the Redis connection
func (s *RedisStore) Close() error {
	return s.client.Close()
}

// Stats returns Redis store statistics (for monitoring)
type RedisStoreStats struct {
	TotalKeys       int64
	IdempotencyKeys int64
	LockKeys        int64
	MemoryUsageKB   int64
}

// GetStats retrieves store statistics
func (s *RedisStore) GetStats() (*RedisStoreStats, error) {
	stats := &RedisStoreStats{}
	
	// Count total keys with our prefix
	pattern := s.keyPrefix + "*"
	keys, err := s.client.Keys(s.ctx, pattern).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get keys: %w", err)
	}
	stats.TotalKeys = int64(len(keys))
	
	// Count lock keys specifically
	lockPattern := s.keyPrefix + "lock:*"
	lockKeys, err := s.client.Keys(s.ctx, lockPattern).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get lock keys: %w", err)
	}
	stats.LockKeys = int64(len(lockKeys))
	
	// Idempotency keys = total - locks
	stats.IdempotencyKeys = stats.TotalKeys - stats.LockKeys
	
	// Get memory usage
	_, err = s.client.Info(s.ctx, "memory").Result()
	if err == nil {
		// Parse used_memory from INFO output
		// This is approximate - you might want more sophisticated parsing
		stats.MemoryUsageKB = 0 // Placeholder
	}
	
	return stats, nil
}
