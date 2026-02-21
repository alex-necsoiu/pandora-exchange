package mocks

import (
	"context"
	"time"

	"github.com/alex-necsoiu/pandora-exchange/internal/domain/audit"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// MockAuditRepository is a mock implementation of audit.Repository
type MockAuditRepository struct {
	mock.Mock
}

// Create mocks the Create method
func (m *MockAuditRepository) Create(ctx context.Context, log *audit.Log) (*audit.Log, error) {
	args := m.Called(ctx, log)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*audit.Log), args.Error(1)
}

// GetByID mocks the GetByID method
func (m *MockAuditRepository) GetByID(ctx context.Context, id uuid.UUID) (*audit.Log, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*audit.Log), args.Error(1)
}

// ListByUser mocks the ListByUser method
func (m *MockAuditRepository) ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]*audit.Log, error) {
	args := m.Called(ctx, userID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*audit.Log), args.Error(1)
}

// ListByEventType mocks the ListByEventType method
func (m *MockAuditRepository) ListByEventType(ctx context.Context, eventType string, limit, offset int32) ([]*audit.Log, error) {
	args := m.Called(ctx, eventType, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*audit.Log), args.Error(1)
}

// ListByCategory mocks the ListByCategory method
func (m *MockAuditRepository) ListByCategory(ctx context.Context, category audit.EventCategory, limit, offset int32) ([]*audit.Log, error) {
	args := m.Called(ctx, category, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*audit.Log), args.Error(1)
}

// ListByIPAddress mocks the ListByIPAddress method
func (m *MockAuditRepository) ListByIPAddress(ctx context.Context, ipAddress string, limit, offset int32) ([]*audit.Log, error) {
	args := m.Called(ctx, ipAddress, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*audit.Log), args.Error(1)
}

// ListByResource mocks the ListByResource method
func (m *MockAuditRepository) ListByResource(ctx context.Context, resourceType, resourceID string, limit, offset int32) ([]*audit.Log, error) {
	args := m.Called(ctx, resourceType, resourceID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*audit.Log), args.Error(1)
}

// Search mocks the Search method
func (m *MockAuditRepository) Search(ctx context.Context, filter *audit.Filter) ([]*audit.Log, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*audit.Log), args.Error(1)
}

// CountByUser mocks the CountByUser method
func (m *MockAuditRepository) CountByUser(ctx context.Context, userID uuid.UUID) (int64, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(int64), args.Error(1)
}

// CountByEventType mocks the CountByEventType method
func (m *MockAuditRepository) CountByEventType(ctx context.Context, eventType string) (int64, error) {
	args := m.Called(ctx, eventType)
	return args.Get(0).(int64), args.Error(1)
}

// CountSearch mocks the CountSearch method
func (m *MockAuditRepository) CountSearch(ctx context.Context, filter *audit.Filter) (int64, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).(int64), args.Error(1)
}

// GetRecentSecurityEvents mocks the GetRecentSecurityEvents method
func (m *MockAuditRepository) GetRecentSecurityEvents(ctx context.Context) ([]*audit.Log, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*audit.Log), args.Error(1)
}

// GetFailedLoginAttempts mocks the GetFailedLoginAttempts method
func (m *MockAuditRepository) GetFailedLoginAttempts(ctx context.Context, userID uuid.UUID) ([]*audit.Log, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*audit.Log), args.Error(1)
}

// DeleteExpired mocks the DeleteExpired method
func (m *MockAuditRepository) DeleteExpired(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// Helper methods for creating test data

// CreateTestAuditLog creates a test audit log for testing
func CreateTestAuditLog(userID *uuid.UUID, eventType string) *audit.Log {
	now := time.Now()
	retentionDate := now.Add(90 * 24 * time.Hour)
	
	return &audit.Log{
		ID:            uuid.New(),
		EventType:     eventType,
		EventCategory: audit.CategoryAuthentication,
		Severity:      audit.SeverityInfo,
		ActorType:     audit.ActorUser,
		UserID:        userID,
		Action:        "test_action",
		Status:        audit.StatusSuccess,
		Metadata:      map[string]interface{}{"test": "data"},
		CreatedAt:     now,
		RetentionUntil: &retentionDate,
	}
}
