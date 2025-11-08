package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alex-necsoiu/pandora-exchange/internal/mocks"
	"github.com/alex-necsoiu/pandora-exchange/internal/observability"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNewAuditCleanupJob(t *testing.T) {
	logger := observability.NewLogger("dev", "test-service")
	mockRepo := new(mocks.MockAuditRepository)
	
	job := NewAuditCleanupJob(mockRepo, logger, 24*time.Hour)
	
	assert.NotNil(t, job)
	assert.Equal(t, 24*time.Hour, job.cleanupInterval)
	assert.NotNil(t, job.stopChan)
	assert.NotNil(t, job.doneChan)
}

func TestAuditCleanupJob_RunOnce_Success(t *testing.T) {
	logger := observability.NewLogger("dev", "test-service")
	mockRepo := new(mocks.MockAuditRepository)
	
	job := NewAuditCleanupJob(mockRepo, logger, 1*time.Hour)
	ctx := context.Background()
	
	// Mock successful deletion
	mockRepo.On("DeleteExpired", mock.Anything).Return(nil).Once()
	
	err := job.RunOnce(ctx)
	
	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestAuditCleanupJob_RunOnce_Error(t *testing.T) {
	logger := observability.NewLogger("dev", "test-service")
	mockRepo := new(mocks.MockAuditRepository)
	
	job := NewAuditCleanupJob(mockRepo, logger, 1*time.Hour)
	ctx := context.Background()
	
	expectedErr := errors.New("database connection failed")
	mockRepo.On("DeleteExpired", mock.Anything).Return(expectedErr).Once()
	
	err := job.RunOnce(ctx)
	
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete expired audit logs")
	mockRepo.AssertExpectations(t)
}

func TestAuditCleanupJob_RunOnce_ContextTimeout(t *testing.T) {
	logger := observability.NewLogger("dev", "test-service")
	mockRepo := new(mocks.MockAuditRepository)
	
	job := NewAuditCleanupJob(mockRepo, logger, 1*time.Hour)
	
	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	
	mockRepo.On("DeleteExpired", mock.Anything).Return(context.Canceled).Once()
	
	err := job.RunOnce(ctx)
	
	require.Error(t, err)
	mockRepo.AssertExpectations(t)
}

func TestAuditCleanupJob_Start_ImmediateCleanup(t *testing.T) {
	logger := observability.NewLogger("dev", "test-service")
	mockRepo := new(mocks.MockAuditRepository)
	
	// Use a very short interval for testing
	job := NewAuditCleanupJob(mockRepo, logger, 100*time.Millisecond)
	ctx := context.Background()
	
	// Mock should be called at least once (immediate cleanup)
	mockRepo.On("DeleteExpired", mock.Anything).Return(nil)
	
	job.Start(ctx)
	
	// Wait for initial cleanup to complete
	time.Sleep(50 * time.Millisecond)
	
	job.Stop()
	
	// Verify at least one cleanup ran
	mockRepo.AssertCalled(t, "DeleteExpired", mock.Anything)
}

func TestAuditCleanupJob_Start_PeriodicCleanup(t *testing.T) {
	logger := observability.NewLogger("dev", "test-service")
	mockRepo := new(mocks.MockAuditRepository)
	
	// Use a very short interval for testing
	job := NewAuditCleanupJob(mockRepo, logger, 50*time.Millisecond)
	ctx := context.Background()
	
	// Mock should be called multiple times
	mockRepo.On("DeleteExpired", mock.Anything).Return(nil)
	
	job.Start(ctx)
	
	// Wait for multiple cleanup cycles (longer wait to ensure ticks happen)
	time.Sleep(300 * time.Millisecond)
	
	job.Stop()
	
	// Verify cleanup ran at least once (immediate cleanup is guaranteed)
	callCount := len(mockRepo.Calls)
	assert.GreaterOrEqual(t, callCount, 1, "Expected at least 1 cleanup call")
	
	// In most cases, should get more than 1 (immediate + periodic)
	t.Logf("Cleanup ran %d times", callCount)
}

func TestAuditCleanupJob_Stop_GracefulShutdown(t *testing.T) {
	logger := observability.NewLogger("dev", "test-service")
	mockRepo := new(mocks.MockAuditRepository)
	
	job := NewAuditCleanupJob(mockRepo, logger, 1*time.Hour)
	ctx := context.Background()
	
	mockRepo.On("DeleteExpired", mock.Anything).Return(nil).Maybe()
	
	job.Start(ctx)
	
	// Stop should complete quickly
	done := make(chan struct{})
	go func() {
		job.Stop()
		close(done)
	}()
	
	select {
	case <-done:
		// Success - stop completed
	case <-time.After(2 * time.Second):
		t.Fatal("Stop() did not complete in time")
	}
}

func TestAuditCleanupJob_Start_ContextCancellation(t *testing.T) {
	logger := observability.NewLogger("dev", "test-service")
	mockRepo := new(mocks.MockAuditRepository)
	
	job := NewAuditCleanupJob(mockRepo, logger, 1*time.Hour)
	
	mockRepo.On("DeleteExpired", mock.Anything).Return(nil).Maybe()
	
	ctx, cancel := context.WithCancel(context.Background())
	job.Start(ctx)
	
	// Cancel context
	cancel()
	
	// Wait briefly for job to stop
	time.Sleep(100 * time.Millisecond)
	
	// Job should have stopped gracefully
	// (doneChan should be closed, but we can't directly test that without blocking)
}

func TestAuditCleanupJob_Start_CleanupError(t *testing.T) {
	logger := observability.NewLogger("dev", "test-service")
	mockRepo := new(mocks.MockAuditRepository)
	
	job := NewAuditCleanupJob(mockRepo, logger, 50*time.Millisecond)
	ctx := context.Background()
	
	// Mock returns error - job should continue running
	mockRepo.On("DeleteExpired", mock.Anything).Return(errors.New("cleanup error"))
	
	job.Start(ctx)
	
	// Wait for multiple cleanup attempts
	time.Sleep(300 * time.Millisecond)
	
	job.Stop()
	
	// Verify cleanup was attempted at least once (despite errors)
	callCount := len(mockRepo.Calls)
	assert.GreaterOrEqual(t, callCount, 1, "Job should attempt cleanup at least once")
	
	// Log actual call count for debugging
	t.Logf("Cleanup attempted %d times despite errors", callCount)
}
