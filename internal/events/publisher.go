package events

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/alex-necsoiu/pandora-exchange/internal/domain"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const (
	// Default stream name for user service events
	DefaultStreamName = "user-service:events"
	// Maximum number of events to retain in the stream (trimming)
	MaxStreamLength = 10000
)

// RedisEventPublisher publishes domain events to Redis Streams
type RedisEventPublisher struct {
	client     *redis.Client
	streamName string
	logger     *zap.Logger
}

// NewRedisEventPublisher creates a new Redis-based event publisher
func NewRedisEventPublisher(client *redis.Client, logger *zap.Logger) *RedisEventPublisher {
	return &RedisEventPublisher{
		client:     client,
		streamName: DefaultStreamName,
		logger:     logger,
	}
}

// WithStreamName sets a custom stream name
func (p *RedisEventPublisher) WithStreamName(streamName string) *RedisEventPublisher {
	p.streamName = streamName
	return p
}

// Publish publishes a single event to Redis Streams
func (p *RedisEventPublisher) Publish(event *domain.Event) error {
	if event == nil {
		return fmt.Errorf("event cannot be nil")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Serialize the event payload to JSON
	payloadJSON, err := json.Marshal(event.Payload)
	if err != nil {
		p.logger.Error("Failed to marshal event payload",
			zap.String("event_id", event.ID),
			zap.String("event_type", string(event.Type)),
			zap.Error(err))
		return fmt.Errorf("failed to marshal event payload: %w", err)
	}

	// Serialize metadata to JSON
	metadataJSON, err := json.Marshal(event.Metadata)
	if err != nil {
		p.logger.Error("Failed to marshal event metadata",
			zap.String("event_id", event.ID),
			zap.String("event_type", string(event.Type)),
			zap.Error(err))
		return fmt.Errorf("failed to marshal event metadata: %w", err)
	}

	// Prepare Redis Stream entry
	// Each field is stored as a key-value pair in the stream entry
	values := map[string]interface{}{
		"event_id":   event.ID,
		"event_type": string(event.Type),
		"user_id":    event.UserID.String(),
		"timestamp":  event.Timestamp.Format(time.RFC3339Nano),
		"payload":    string(payloadJSON),
		"metadata":   string(metadataJSON),
	}

	// Add event to Redis Stream with automatic ID generation (*)
	// MAXLEN ~ caps the stream size (approximate trimming for performance)
	result := p.client.XAdd(ctx, &redis.XAddArgs{
		Stream: p.streamName,
		MaxLen: MaxStreamLength, // Trim to prevent unbounded growth
		Approx: true,            // Approximate trimming for better performance
		Values: values,
	})

	if err := result.Err(); err != nil {
		p.logger.Error("Failed to publish event to Redis Stream",
			zap.String("event_id", event.ID),
			zap.String("event_type", string(event.Type)),
			zap.String("stream", p.streamName),
			zap.Error(err))
		return fmt.Errorf("failed to publish event to Redis: %w", err)
	}

	streamID := result.Val()
	p.logger.Info("Event published successfully",
		zap.String("event_id", event.ID),
		zap.String("event_type", string(event.Type)),
		zap.String("user_id", event.UserID.String()),
		zap.String("stream", p.streamName),
		zap.String("stream_id", streamID))

	return nil
}

// PublishBatch publishes multiple events in a pipeline for better performance
func (p *RedisEventPublisher) PublishBatch(events []*domain.Event) error {
	if len(events) == 0 {
		return nil // Nothing to publish
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Use Redis pipeline for batch operations
	pipe := p.client.Pipeline()

	for _, event := range events {
		if event == nil {
			p.logger.Warn("Skipping nil event in batch")
			continue
		}

		// Serialize payload
		payloadJSON, err := json.Marshal(event.Payload)
		if err != nil {
			p.logger.Error("Failed to marshal event payload in batch",
				zap.String("event_id", event.ID),
				zap.String("event_type", string(event.Type)),
				zap.Error(err))
			return fmt.Errorf("failed to marshal event payload: %w", err)
		}

		// Serialize metadata
		metadataJSON, err := json.Marshal(event.Metadata)
		if err != nil {
			p.logger.Error("Failed to marshal event metadata in batch",
				zap.String("event_id", event.ID),
				zap.String("event_type", string(event.Type)),
				zap.Error(err))
			return fmt.Errorf("failed to marshal event metadata: %w", err)
		}

		values := map[string]interface{}{
			"event_id":   event.ID,
			"event_type": string(event.Type),
			"user_id":    event.UserID.String(),
			"timestamp":  event.Timestamp.Format(time.RFC3339Nano),
			"payload":    string(payloadJSON),
			"metadata":   string(metadataJSON),
		}

		pipe.XAdd(ctx, &redis.XAddArgs{
			Stream: p.streamName,
			MaxLen: MaxStreamLength,
			Approx: true,
			Values: values,
		})
	}

	// Execute all commands in the pipeline
	_, err := pipe.Exec(ctx)
	if err != nil {
		p.logger.Error("Failed to publish batch events to Redis Stream",
			zap.Int("batch_size", len(events)),
			zap.String("stream", p.streamName),
			zap.Error(err))
		return fmt.Errorf("failed to publish batch events: %w", err)
	}

	p.logger.Info("Batch events published successfully",
		zap.Int("batch_size", len(events)),
		zap.String("stream", p.streamName))

	return nil
}

// Close closes the Redis connection
func (p *RedisEventPublisher) Close() error {
	if p.client != nil {
		return p.client.Close()
	}
	return nil
}
