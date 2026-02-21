package user

import (
	"time"

	"github.com/google/uuid"
)

// EventType represents different types of user domain events
type EventType string

const (
	// User lifecycle events
	EventTypeUserRegistered      EventType = "user.registered"
	EventTypeUserKYCUpdated      EventType = "user.kyc.updated"
	EventTypeUserProfileUpdated  EventType = "user.profile.updated"
	EventTypeUserDeleted         EventType = "user.deleted"
	EventTypeUserLoggedIn        EventType = "user.logged_in"
	EventTypeUserPasswordChanged EventType = "user.password.changed"
)

// Event represents a domain event that occurred in the user domain
type Event struct {
	ID        string                 `json:"id"`        // Unique event ID
	Type      EventType              `json:"type"`      // Event type
	Timestamp time.Time              `json:"timestamp"` // When the event occurred
	UserID    uuid.UUID              `json:"user_id"`   // User associated with the event
	Payload   map[string]interface{} `json:"payload"`   // Event-specific data
	Metadata  map[string]string      `json:"metadata"`  // Additional metadata (IP, user agent, etc.)
}

// NewEvent creates a new user domain event
func NewEvent(eventType EventType, userID uuid.UUID, payload map[string]interface{}) *Event {
	return &Event{
		ID:        uuid.New().String(),
		Type:      eventType,
		Timestamp: time.Now().UTC(),
		UserID:    userID,
		Payload:   payload,
		Metadata:  make(map[string]string),
	}
}

// WithMetadata adds metadata to the event
func (e *Event) WithMetadata(key, value string) *Event {
	if e.Metadata == nil {
		e.Metadata = make(map[string]string)
	}
	e.Metadata[key] = value
	return e
}
