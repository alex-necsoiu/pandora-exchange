package http_test

import (
	"context"

	userDomain "github.com/alex-necsoiu/pandora-exchange/internal/domain/user"
	"github.com/alex-necsoiu/pandora-exchange/internal/domain/auth"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// MockUserService is a mock implementation of user.Service for testing HTTP handlers
type MockUserService struct {
	mock.Mock
}

// Register mocks the Register method
func (m *MockUserService) Register(ctx context.Context, email, password, firstName, lastName string) (*userDomain.User, error) {
	args := m.Called(ctx, email, password, firstName, lastName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*userDomain.User), args.Error(1)
}

// Login mocks the Login method
func (m *MockUserService) Login(ctx context.Context, email, password, ipAddress, userAgent string) (*userDomain.TokenPair, error) {
	args := m.Called(ctx, email, password, ipAddress, userAgent)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*userDomain.TokenPair), args.Error(1)
}

// AdminLogin mocks the AdminLogin method
func (m *MockUserService) AdminLogin(ctx context.Context, email, password, ipAddress, userAgent string) (*userDomain.TokenPair, error) {
	args := m.Called(ctx, email, password, ipAddress, userAgent)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*userDomain.TokenPair), args.Error(1)
}

// RefreshToken mocks the RefreshToken method
func (m *MockUserService) RefreshToken(ctx context.Context, refreshToken, ipAddress, userAgent string) (*userDomain.TokenPair, error) {
	args := m.Called(ctx, refreshToken, ipAddress, userAgent)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*userDomain.TokenPair), args.Error(1)
}

// Logout mocks the Logout method
func (m *MockUserService) Logout(ctx context.Context, refreshToken string) error {
	args := m.Called(ctx, refreshToken)
	return args.Error(0)
}

// LogoutAll mocks the LogoutAll method
func (m *MockUserService) LogoutAll(ctx context.Context, userID uuid.UUID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

// GetByID mocks the GetByID method
func (m *MockUserService) GetByID(ctx context.Context, id uuid.UUID) (*userDomain.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*userDomain.User), args.Error(1)
}

// GetByEmail mocks the GetByEmail method
func (m *MockUserService) GetByEmail(ctx context.Context, email string) (*userDomain.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*userDomain.User), args.Error(1)
}

// UpdateKYC mocks the UpdateKYC method
func (m *MockUserService) UpdateKYC(ctx context.Context, userID uuid.UUID, status userDomain.KYCStatus) (*userDomain.User, error) {
	args := m.Called(ctx, userID, status)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*userDomain.User), args.Error(1)
}

// UpdateProfile mocks the UpdateProfile method
func (m *MockUserService) UpdateProfile(ctx context.Context, userID uuid.UUID, firstName, lastName string) (*userDomain.User, error) {
	args := m.Called(ctx, userID, firstName, lastName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*userDomain.User), args.Error(1)
}

// DeleteAccount mocks the DeleteAccount method
func (m *MockUserService) DeleteAccount(ctx context.Context, userID uuid.UUID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

// GetActiveSessions mocks the GetActiveSessions method
func (m *MockUserService) GetActiveSessions(ctx context.Context, userID uuid.UUID) ([]*auth.RefreshToken, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*auth.RefreshToken), args.Error(1)
}

// ListUsers mocks the ListUsers method
func (m *MockUserService) ListUsers(ctx context.Context, limit, offset int) ([]*userDomain.User, int64, error) {
	args := m.Called(ctx, limit, offset)
	return args.Get(0).([]*userDomain.User), args.Get(1).(int64), args.Error(2)
}

// SearchUsers mocks the SearchUsers method
func (m *MockUserService) SearchUsers(ctx context.Context, query string, limit, offset int) ([]*userDomain.User, error) {
	args := m.Called(ctx, query, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*userDomain.User), args.Error(1)
}

// GetUserByIDAdmin mocks the GetUserByIDAdmin method
func (m *MockUserService) GetUserByIDAdmin(ctx context.Context, id uuid.UUID) (*userDomain.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*userDomain.User), args.Error(1)
}

// UpdateUserRole mocks the UpdateUserRole method
func (m *MockUserService) UpdateUserRole(ctx context.Context, userID uuid.UUID, role userDomain.Role) (*userDomain.User, error) {
	args := m.Called(ctx, userID, role)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*userDomain.User), args.Error(1)
}

// GetAllActiveSessions mocks the GetAllActiveSessions method
func (m *MockUserService) GetAllActiveSessions(ctx context.Context, limit, offset int) ([]*auth.RefreshToken, int64, error) {
	args := m.Called(ctx, limit, offset)
	return args.Get(0).([]*auth.RefreshToken), args.Get(1).(int64), args.Error(2)
}

// ForceLogout mocks the ForceLogout method
func (m *MockUserService) ForceLogout(ctx context.Context, token string) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

// GetSystemStats mocks the GetSystemStats method
func (m *MockUserService) GetSystemStats(ctx context.Context) (map[string]interface{}, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]interface{}), args.Error(1)
}
