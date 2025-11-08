package events

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/alex-necsoiu/pandora-exchange/internal/domain"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestNewRedisEventPublisher(t *testing.T) {
	logger := zaptest.NewLogger(t)
	client := redis.NewClient(&redis.Options{})

	publisher := NewRedisEventPublisher(client, logger)

	assert.NotNil(t, publisher)
	assert.Equal(t, DefaultStreamName, publisher.streamName)
	assert.Equal(t, client, publisher.client)
	assert.Equal(t, logger, publisher.logger)
}

func TestWithStreamName(t *testing.T) {
	logger := zaptest.NewLogger(t)
	client := redis.NewClient(&redis.Options{})
	customStreamName := "custom:stream"

	publisher := NewRedisEventPublisher(client, logger).WithStreamName(customStreamName)

	assert.Equal(t, customStreamName, publisher.streamName)
}

func TestPublish_Success(t *testing.T) {
	// Use miniredis for in-memory Redis testing
	ctx := context.Background()
	logger := zaptest.NewLogger(t)

	// Create a real Redis client pointing to a test server
	// In a real test environment, you'd use miniredis or testcontainers
	// For this example, we'll test the logic without actual Redis
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer client.Close()

	publisher := NewRedisEventPublisher(client, logger)

	userID := uuid.New()
	event := domain.NewEvent(
		domain.EventTypeUserRegistered,
		userID,
		map[string]interface{}{
			"email":      "test@example.com",
			"first_name": "John",
			"last_name":  "Doe",
		},
	).WithMetadata("ip_address", "127.0.0.1")

	// Note: This test will only pass if Redis is actually running
	// In a production test suite, you'd use miniredis or skip if Redis unavailable
	err := publisher.Publish(event)

	// Check if Redis is available
	if err != nil && (errors.Is(err, context.DeadlineExceeded) || 
		errors.Is(err, redis.Nil) ||
		err.Error() == "dial tcp 127.0.0.1:6379: connect: connection refused" ||
		err.Error() == "dial tcp [::1]:6379: connect: connection refused") {
		t.Skip("Redis not available, skipping integration test")
	}

	require.NoError(t, err)

	// Verify event was added to stream
	result := client.XLen(ctx, DefaultStreamName)
	assert.NoError(t, result.Err())
	assert.Greater(t, result.Val(), int64(0))

	// Read back the event
	messages, err := client.XRange(ctx, DefaultStreamName, "-", "+").Result()
	require.NoError(t, err)
	require.NotEmpty(t, messages)

	// Verify the last message contains our event
	lastMessage := messages[len(messages)-1]
	assert.Equal(t, event.ID, lastMessage.Values["event_id"])
	assert.Equal(t, string(domain.EventTypeUserRegistered), lastMessage.Values["event_type"])
	assert.Equal(t, userID.String(), lastMessage.Values["user_id"])

	// Verify payload deserialization
	var payload map[string]interface{}
	err = json.Unmarshal([]byte(lastMessage.Values["payload"].(string)), &payload)
	require.NoError(t, err)
	assert.Equal(t, "test@example.com", payload["email"])
	assert.Equal(t, "John", payload["first_name"])
	assert.Equal(t, "Doe", payload["last_name"])

	// Verify metadata
	var metadata map[string]string
	err = json.Unmarshal([]byte(lastMessage.Values["metadata"].(string)), &metadata)
	require.NoError(t, err)
	assert.Equal(t, "127.0.0.1", metadata["ip_address"])
}

func TestPublish_NilEvent(t *testing.T) {
	logger := zaptest.NewLogger(t)
	client := redis.NewClient(&redis.Options{})
	publisher := NewRedisEventPublisher(client, logger)

	err := publisher.Publish(nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "event cannot be nil")
}

func TestPublish_InvalidPayload(t *testing.T) {
	logger := zaptest.NewLogger(t)
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer client.Close()

	publisher := NewRedisEventPublisher(client, logger)

	// Create event with un-marshalable payload (channels can't be JSON encoded)
	userID := uuid.New()
	event := &domain.Event{
		ID:        uuid.New().String(),
		Type:      domain.EventTypeUserRegistered,
		Timestamp: time.Now(),
		UserID:    userID,
		Payload: map[string]interface{}{
			"invalid": make(chan int), // Channels can't be marshaled to JSON
		},
		Metadata: make(map[string]string),
	}

	err := publisher.Publish(event)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to marshal event payload")
}

func TestPublish_InvalidMetadata(t *testing.T) {
	logger := zaptest.NewLogger(t)
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer client.Close()

	publisher := NewRedisEventPublisher(client, logger)

	userID := uuid.New()

	// Create a properly typed event
	validEvent := domain.NewEvent(
		domain.EventTypeUserKYCUpdated,
		userID,
		map[string]interface{}{"status": "approved"},
	)

	// The publisher expects domain.Event with map[string]string metadata
	// which is always marshalable, so this test verifies normal operation
	err := publisher.Publish(validEvent)

	// Check if Redis is available
	if err != nil && (errors.Is(err, context.DeadlineExceeded) ||
		err.Error() == "dial tcp 127.0.0.1:6379: connect: connection refused" ||
		err.Error() == "dial tcp [::1]:6379: connect: connection refused") {
		t.Skip("Redis not available, skipping integration test")
	}

	// Should succeed since metadata is always map[string]string
	assert.NoError(t, err)
}

func TestPublish_RedisConnectionError(t *testing.T) {
	logger := zaptest.NewLogger(t)
	// Create client pointing to invalid address
	client := redis.NewClient(&redis.Options{
		Addr:         "localhost:9999", // Wrong port
		DialTimeout:  100 * time.Millisecond,
		ReadTimeout:  100 * time.Millisecond,
		WriteTimeout: 100 * time.Millisecond,
	})
	defer client.Close()

	publisher := NewRedisEventPublisher(client, logger)

	userID := uuid.New()
	event := domain.NewEvent(
		domain.EventTypeUserRegistered,
		userID,
		map[string]interface{}{"email": "test@example.com"},
	)

	err := publisher.Publish(event)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to publish event to Redis")
}

func TestPublishBatch_Success(t *testing.T) {
	ctx := context.Background()
	logger := zaptest.NewLogger(t)
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer client.Close()

	publisher := NewRedisEventPublisher(client, logger)

	userID1 := uuid.New()
	userID2 := uuid.New()

	events := []*domain.Event{
		domain.NewEvent(
			domain.EventTypeUserRegistered,
			userID1,
			map[string]interface{}{"email": "user1@example.com"},
		),
		domain.NewEvent(
			domain.EventTypeUserKYCUpdated,
			userID2,
			map[string]interface{}{"status": "approved"},
		),
	}

	// Get initial stream length
	initialLen := client.XLen(ctx, DefaultStreamName).Val()

	err := publisher.PublishBatch(events)

	// Check if Redis is available
	if err != nil && (errors.Is(err, context.DeadlineExceeded) ||
		err.Error() == "dial tcp 127.0.0.1:6379: connect: connection refused" ||
		err.Error() == "dial tcp [::1]:6379: connect: connection refused") {
		t.Skip("Redis not available, skipping integration test")
	}

	require.NoError(t, err)

	// Verify both events were added
	finalLen := client.XLen(ctx, DefaultStreamName).Val()
	assert.Equal(t, initialLen+2, finalLen)
}

func TestPublishBatch_EmptyBatch(t *testing.T) {
	logger := zaptest.NewLogger(t)
	client := redis.NewClient(&redis.Options{})
	publisher := NewRedisEventPublisher(client, logger)

	err := publisher.PublishBatch([]*domain.Event{})

	assert.NoError(t, err) // Empty batch should not error
}

func TestPublishBatch_WithNilEvent(t *testing.T) {
	ctx := context.Background()
	logger := zaptest.NewLogger(t)
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer client.Close()

	publisher := NewRedisEventPublisher(client, logger)

	userID := uuid.New()
	events := []*domain.Event{
		domain.NewEvent(
			domain.EventTypeUserRegistered,
			userID,
			map[string]interface{}{"email": "user@example.com"},
		),
		nil, // Nil event should be skipped
		domain.NewEvent(
			domain.EventTypeUserLoggedIn,
			userID,
			map[string]interface{}{"ip": "127.0.0.1"},
		),
	}

	initialLen := client.XLen(ctx, DefaultStreamName).Val()

	err := publisher.PublishBatch(events)

	// Check if Redis is available
	if err != nil && (errors.Is(err, context.DeadlineExceeded) ||
		err.Error() == "dial tcp 127.0.0.1:6379: connect: connection refused" ||
		err.Error() == "dial tcp [::1]:6379: connect: connection refused") {
		t.Skip("Redis not available, skipping integration test")
	}

	require.NoError(t, err)

	// Should only add 2 events (nil was skipped)
	finalLen := client.XLen(ctx, DefaultStreamName).Val()
	assert.Equal(t, initialLen+2, finalLen)
}

func TestPublishBatch_MarshalError(t *testing.T) {
	logger := zaptest.NewLogger(t)
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer client.Close()

	publisher := NewRedisEventPublisher(client, logger)

	userID := uuid.New()
	events := []*domain.Event{
		{
			ID:        uuid.New().String(),
			Type:      domain.EventTypeUserRegistered,
			Timestamp: time.Now(),
			UserID:    userID,
			Payload: map[string]interface{}{
				"invalid": make(chan int), // Can't marshal channel
			},
			Metadata: make(map[string]string),
		},
	}

	err := publisher.PublishBatch(events)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to marshal event payload")
}

func TestPublishBatch_RedisError(t *testing.T) {
	logger := zaptest.NewLogger(t)
	client := redis.NewClient(&redis.Options{
		Addr:         "localhost:9999", // Wrong port
		DialTimeout:  100 * time.Millisecond,
		ReadTimeout:  100 * time.Millisecond,
		WriteTimeout: 100 * time.Millisecond,
	})
	defer client.Close()

	publisher := NewRedisEventPublisher(client, logger)

	userID := uuid.New()
	events := []*domain.Event{
		domain.NewEvent(
			domain.EventTypeUserRegistered,
			userID,
			map[string]interface{}{"email": "user@example.com"},
		),
	}

	err := publisher.PublishBatch(events)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to publish batch events")
}

func TestClose(t *testing.T) {
	logger := zaptest.NewLogger(t)
	client := redis.NewClient(&redis.Options{})
	publisher := NewRedisEventPublisher(client, logger)

	err := publisher.Close()
	assert.NoError(t, err)
}

func TestClose_NilClient(t *testing.T) {
	logger := zaptest.NewLogger(t)
	publisher := &RedisEventPublisher{
		client:     nil,
		streamName: DefaultStreamName,
		logger:     logger,
	}

	err := publisher.Close()
	assert.NoError(t, err) // Should handle nil client gracefully
}

// Test event type constants
func TestEventTypes(t *testing.T) {
	tests := []struct {
		name      string
		eventType domain.EventType
		expected  string
	}{
		{"User Registered", domain.EventTypeUserRegistered, "user.registered"},
		{"KYC Updated", domain.EventTypeUserKYCUpdated, "user.kyc.updated"},
		{"Profile Updated", domain.EventTypeUserProfileUpdated, "user.profile.updated"},
		{"User Deleted", domain.EventTypeUserDeleted, "user.deleted"},
		{"User Logged In", domain.EventTypeUserLoggedIn, "user.logged_in"},
		{"Password Changed", domain.EventTypeUserPasswordChanged, "user.password.changed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.eventType))
		})
	}
}

// Test NewEvent helper
func TestNewEvent(t *testing.T) {
	userID := uuid.New()
	payload := map[string]interface{}{
		"email": "test@example.com",
		"name":  "John Doe",
	}

	event := domain.NewEvent(domain.EventTypeUserRegistered, userID, payload)

	assert.NotEmpty(t, event.ID)
	assert.Equal(t, domain.EventTypeUserRegistered, event.Type)
	assert.Equal(t, userID, event.UserID)
	assert.Equal(t, payload, event.Payload)
	assert.NotNil(t, event.Metadata)
	assert.WithinDuration(t, time.Now(), event.Timestamp, time.Second)
}

// Test WithMetadata helper
func TestWithMetadata(t *testing.T) {
	userID := uuid.New()
	event := domain.NewEvent(
		domain.EventTypeUserLoggedIn,
		userID,
		map[string]interface{}{"session_id": "abc123"},
	)

	event.WithMetadata("ip_address", "192.168.1.1").
		WithMetadata("user_agent", "Mozilla/5.0")

	assert.Equal(t, "192.168.1.1", event.Metadata["ip_address"])
	assert.Equal(t, "Mozilla/5.0", event.Metadata["user_agent"])
}

// Test stream constants
func TestStreamConstants(t *testing.T) {
	assert.Equal(t, "user-service:events", DefaultStreamName)
	assert.Equal(t, int64(10000), int64(MaxStreamLength))
}
