package mocks

import (
	"github.com/alex-necsoiu/pandora-exchange/internal/domain"
	"github.com/stretchr/testify/mock"
)

// MockEventPublisher is a mock implementation of domain.EventPublisher
type MockEventPublisher struct {
	mock.Mock
}

// Publish mocks the Publish method
func (m *MockEventPublisher) Publish(event *domain.Event) error {
	args := m.Called(event)
	return args.Error(0)
}

// PublishBatch mocks the PublishBatch method
func (m *MockEventPublisher) PublishBatch(events []*domain.Event) error {
	args := m.Called(events)
	return args.Error(0)
}

// Close mocks the Close method
func (m *MockEventPublisher) Close() error {
	args := m.Called()
	return args.Error(0)
}
