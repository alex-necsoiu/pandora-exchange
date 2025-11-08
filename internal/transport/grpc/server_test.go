package grpc_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alex-necsoiu/pandora-exchange/internal/domain"
	"github.com/alex-necsoiu/pandora-exchange/internal/observability"
	grpcTransport "github.com/alex-necsoiu/pandora-exchange/internal/transport/grpc"
	pb "github.com/alex-necsoiu/pandora-exchange/internal/transport/grpc/proto"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// MockUserService is a mock implementation of domain.UserService for testing
type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) Register(ctx context.Context, email, password, firstName, lastName string) (*domain.User, error) {
	args := m.Called(ctx, email, password, firstName, lastName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserService) Login(ctx context.Context, email, password, ipAddress, userAgent string) (*domain.TokenPair, error) {
	args := m.Called(ctx, email, password, ipAddress, userAgent)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.TokenPair), args.Error(1)
}

func (m *MockUserService) AdminLogin(ctx context.Context, email, password, ipAddress, userAgent string) (*domain.TokenPair, error) {
	args := m.Called(ctx, email, password, ipAddress, userAgent)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.TokenPair), args.Error(1)
}

func (m *MockUserService) RefreshToken(ctx context.Context, refreshToken, ipAddress, userAgent string) (*domain.TokenPair, error) {
	args := m.Called(ctx, refreshToken, ipAddress, userAgent)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.TokenPair), args.Error(1)
}

func (m *MockUserService) Logout(ctx context.Context, refreshToken string) error {
	args := m.Called(ctx, refreshToken)
	return args.Error(0)
}

func (m *MockUserService) LogoutAll(ctx context.Context, userID uuid.UUID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockUserService) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserService) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserService) UpdateKYC(ctx context.Context, userID uuid.UUID, kycStatus domain.KYCStatus) (*domain.User, error) {
	args := m.Called(ctx, userID, kycStatus)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserService) UpdateProfile(ctx context.Context, userID uuid.UUID, firstName, lastName string) (*domain.User, error) {
	args := m.Called(ctx, userID, firstName, lastName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserService) DeleteAccount(ctx context.Context, userID uuid.UUID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockUserService) GetActiveSessions(ctx context.Context, userID uuid.UUID) ([]*domain.RefreshToken, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.RefreshToken), args.Error(1)
}

func (m *MockUserService) ListUsers(ctx context.Context, limit, offset int) ([]*domain.User, int64, error) {
	args := m.Called(ctx, limit, offset)
	return args.Get(0).([]*domain.User), args.Get(1).(int64), args.Error(2)
}

func (m *MockUserService) SearchUsers(ctx context.Context, query string, limit, offset int) ([]*domain.User, error) {
	args := m.Called(ctx, query, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.User), args.Error(1)
}

func (m *MockUserService) GetUserByIDAdmin(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserService) UpdateUserRole(ctx context.Context, userID uuid.UUID, role domain.Role) (*domain.User, error) {
	args := m.Called(ctx, userID, role)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserService) GetAllActiveSessions(ctx context.Context, limit, offset int) ([]*domain.RefreshToken, int64, error) {
	args := m.Called(ctx, limit, offset)
	return args.Get(0).([]*domain.RefreshToken), args.Get(1).(int64), args.Error(2)
}

func (m *MockUserService) ForceLogout(ctx context.Context, token string) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

func (m *MockUserService) GetSystemStats(ctx context.Context) (map[string]interface{}, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

// Helper to create test user
func createTestUser() *domain.User {
	now := time.Now()
	return &domain.User{
		ID:        uuid.New(),
		Email:     "test@example.com",
		FirstName: "John",
		LastName:  "Doe",
		KYCStatus: domain.KYCStatusPending,
		Role:      domain.RoleUser,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func TestGetUser(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		mockSetup      func(*MockUserService)
		expectedError  codes.Code
		expectedUserID string
	}{
		{
			name:   "successfully get user",
			userID: uuid.New().String(),
			mockSetup: func(m *MockUserService) {
				user := createTestUser()
				m.On("GetByID", mock.Anything, mock.Anything).Return(user, nil)
			},
			expectedError: codes.OK,
		},
		{
			name:          "invalid user ID format",
			userID:        "invalid-uuid",
			mockSetup:     func(m *MockUserService) {},
			expectedError: codes.InvalidArgument,
		},
		{
			name:   "user not found",
			userID: uuid.New().String(),
			mockSetup: func(m *MockUserService) {
				m.On("GetByID", mock.Anything, mock.Anything).Return(nil, domain.ErrUserNotFound)
			},
			expectedError: codes.NotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockUserService)
			tt.mockSetup(mockService)
			logger := observability.NewLogger("test", "grpc-test")
			server := grpcTransport.NewServer(mockService, logger)

			req := &pb.GetUserRequest{UserId: tt.userID}
			resp, err := server.GetUser(context.Background(), req)

			if tt.expectedError == codes.OK {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.NotNil(t, resp.User)
			} else {
				assert.Error(t, err)
				st, ok := status.FromError(err)
				assert.True(t, ok)
				assert.Equal(t, tt.expectedError, st.Code())
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestGetUserByEmail(t *testing.T) {
	tests := []struct {
		name          string
		email         string
		mockSetup     func(*MockUserService)
		expectedError codes.Code
	}{
		{
			name:  "successfully get user by email",
			email: "test@example.com",
			mockSetup: func(m *MockUserService) {
				user := createTestUser()
				m.On("GetByEmail", mock.Anything, "test@example.com").Return(user, nil)
			},
			expectedError: codes.OK,
		},
		{
			name:          "empty email",
			email:         "",
			mockSetup:     func(m *MockUserService) {},
			expectedError: codes.InvalidArgument,
		},
		{
			name:  "user not found",
			email: "notfound@example.com",
			mockSetup: func(m *MockUserService) {
				m.On("GetByEmail", mock.Anything, "notfound@example.com").Return(nil, domain.ErrUserNotFound)
			},
			expectedError: codes.NotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockUserService)
			tt.mockSetup(mockService)
			logger := observability.NewLogger("test", "grpc-test")
			server := grpcTransport.NewServer(mockService, logger)

			req := &pb.GetUserByEmailRequest{Email: tt.email}
			resp, err := server.GetUserByEmail(context.Background(), req)

			if tt.expectedError == codes.OK {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.NotNil(t, resp.User)
				assert.Equal(t, tt.email, resp.User.Email)
			} else {
				assert.Error(t, err)
				st, ok := status.FromError(err)
				assert.True(t, ok)
				assert.Equal(t, tt.expectedError, st.Code())
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestUpdateKYCStatus(t *testing.T) {
	tests := []struct {
		name          string
		userID        string
		kycStatus     string
		mockSetup     func(*MockUserService)
		expectedError codes.Code
	}{
		{
			name:      "successfully update KYC status",
			userID:    uuid.New().String(),
			kycStatus: "verified",
			mockSetup: func(m *MockUserService) {
				user := createTestUser()
				user.KYCStatus = domain.KYCStatusVerified
				m.On("UpdateKYC", mock.Anything, mock.Anything, domain.KYCStatusVerified).Return(user, nil)
			},
			expectedError: codes.OK,
		},
		{
			name:          "invalid user ID format",
			userID:        "invalid-uuid",
			kycStatus:     "verified",
			mockSetup:     func(m *MockUserService) {},
			expectedError: codes.InvalidArgument,
		},
		{
			name:      "user not found",
			userID:    uuid.New().String(),
			kycStatus: "verified",
			mockSetup: func(m *MockUserService) {
				m.On("UpdateKYC", mock.Anything, mock.Anything, mock.Anything).Return(nil, domain.ErrUserNotFound)
			},
			expectedError: codes.NotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockUserService)
			tt.mockSetup(mockService)
			logger := observability.NewLogger("test", "grpc-test")
			server := grpcTransport.NewServer(mockService, logger)

			req := &pb.UpdateKYCRequest{
				UserId:    tt.userID,
				KycStatus: tt.kycStatus,
				UpdatedBy: "admin@example.com",
			}
			resp, err := server.UpdateKYCStatus(context.Background(), req)

			if tt.expectedError == codes.OK {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.NotNil(t, resp.User)
				assert.Equal(t, tt.kycStatus, resp.User.KycStatus)
			} else {
				assert.Error(t, err)
				st, ok := status.FromError(err)
				assert.True(t, ok)
				assert.Equal(t, tt.expectedError, st.Code())
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestValidateUser(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		mockSetup      func(*MockUserService)
		expectedValid  bool
		expectedActive bool
		expectedError  codes.Code
	}{
		{
			name:   "user exists and is active",
			userID: uuid.New().String(),
			mockSetup: func(m *MockUserService) {
				user := createTestUser()
				m.On("GetByID", mock.Anything, mock.Anything).Return(user, nil)
			},
			expectedValid:  true,
			expectedActive: true,
			expectedError:  codes.OK,
		},
		{
			name:   "user exists but is deleted",
			userID: uuid.New().String(),
			mockSetup: func(m *MockUserService) {
				user := createTestUser()
				now := time.Now()
				user.DeletedAt = &now
				m.On("GetByID", mock.Anything, mock.Anything).Return(user, nil)
			},
			expectedValid:  true,
			expectedActive: false,
			expectedError:  codes.OK,
		},
		{
			name:   "user not found",
			userID: uuid.New().String(),
			mockSetup: func(m *MockUserService) {
				m.On("GetByID", mock.Anything, mock.Anything).Return(nil, domain.ErrUserNotFound)
			},
			expectedValid:  false,
			expectedActive: false,
			expectedError:  codes.OK,
		},
		{
			name:          "invalid user ID format",
			userID:        "invalid-uuid",
			mockSetup:     func(m *MockUserService) {},
			expectedError: codes.InvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockUserService)
			tt.mockSetup(mockService)
			logger := observability.NewLogger("test", "grpc-test")
			server := grpcTransport.NewServer(mockService, logger)

			req := &pb.ValidateUserRequest{UserId: tt.userID}
			resp, err := server.ValidateUser(context.Background(), req)

			if tt.expectedError == codes.OK {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Equal(t, tt.expectedValid, resp.IsValid)
				assert.Equal(t, tt.expectedActive, resp.IsActive)
			} else {
				assert.Error(t, err)
				st, ok := status.FromError(err)
				assert.True(t, ok)
				assert.Equal(t, tt.expectedError, st.Code())
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestListUsers(t *testing.T) {
	tests := []struct {
		name          string
		limit         int32
		offset        int32
		mockSetup     func(*MockUserService)
		expectedError codes.Code
		expectedCount int
	}{
		{
			name:   "successfully list users",
			limit:  10,
			offset: 0,
			mockSetup: func(m *MockUserService) {
				users := []*domain.User{createTestUser(), createTestUser()}
				m.On("ListUsers", mock.Anything, 10, 0).Return(users, int64(2), nil)
			},
			expectedError: codes.OK,
			expectedCount: 2,
		},
		{
			name:   "use default limit when 0",
			limit:  0,
			offset: 0,
			mockSetup: func(m *MockUserService) {
				users := []*domain.User{createTestUser()}
				m.On("ListUsers", mock.Anything, 10, 0).Return(users, int64(1), nil)
			},
			expectedError: codes.OK,
			expectedCount: 1,
		},
		{
			name:          "limit exceeds maximum",
			limit:         150,
			offset:        0,
			mockSetup:     func(m *MockUserService) {},
			expectedError: codes.InvalidArgument,
		},
		{
			name:          "negative offset",
			limit:         10,
			offset:        -5,
			mockSetup:     func(m *MockUserService) {},
			expectedError: codes.InvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockUserService)
			tt.mockSetup(mockService)
			logger := observability.NewLogger("test", "grpc-test")
			server := grpcTransport.NewServer(mockService, logger)

			req := &pb.ListUsersRequest{
				Limit:  tt.limit,
				Offset: tt.offset,
			}
			resp, err := server.ListUsers(context.Background(), req)

			if tt.expectedError == codes.OK {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Len(t, resp.Users, tt.expectedCount)
			} else {
				assert.Error(t, err)
				st, ok := status.FromError(err)
				assert.True(t, ok)
				assert.Equal(t, tt.expectedError, st.Code())
			}

			mockService.AssertExpectations(t)
		})
	}
}

// TestHandleServiceError tests all domain error mappings to gRPC status codes
func TestHandleServiceError(t *testing.T) {
	tests := []struct {
		name          string
		domainError   error
		expectedCode  codes.Code
		rpcMethod     string
	}{
		{
			name:         "ErrUserAlreadyExists maps to AlreadyExists",
			domainError:  domain.ErrUserAlreadyExists,
			expectedCode: codes.AlreadyExists,
			rpcMethod:    "GetUser",
		},
		{
			name:         "ErrInvalidCredentials maps to Unauthenticated",
			domainError:  domain.ErrInvalidCredentials,
			expectedCode: codes.Unauthenticated,
			rpcMethod:    "GetUser",
		},
		{
			name:         "ErrInvalidKYCStatus maps to InvalidArgument",
			domainError:  domain.ErrInvalidKYCStatus,
			expectedCode: codes.InvalidArgument,
			rpcMethod:    "UpdateKYCStatus",
		},
		{
			name:         "ErrInvalidEmail maps to InvalidArgument",
			domainError:  domain.ErrInvalidEmail,
			expectedCode: codes.InvalidArgument,
			rpcMethod:    "GetUserByEmail",
		},
		{
			name:         "ErrWeakPassword maps to InvalidArgument",
			domainError:  domain.ErrWeakPassword,
			expectedCode: codes.InvalidArgument,
			rpcMethod:    "GetUser",
		},
		{
			name:         "unknown error maps to Internal",
			domainError:  errors.New("database connection failed"),
			expectedCode: codes.Internal,
			rpcMethod:    "GetUser",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockUserService)
			logger := observability.NewLogger("test", "grpc-test")
			server := grpcTransport.NewServer(mockService, logger)

			// Trigger the error based on the RPC method
			switch tt.rpcMethod {
			case "GetUser":
				mockService.On("GetByID", mock.Anything, mock.Anything).Return(nil, tt.domainError)
				_, err := server.GetUser(context.Background(), &pb.GetUserRequest{UserId: uuid.New().String()})
				assert.Error(t, err)
				st, ok := status.FromError(err)
				assert.True(t, ok)
				assert.Equal(t, tt.expectedCode, st.Code())

			case "GetUserByEmail":
				mockService.On("GetByEmail", mock.Anything, "test@example.com").Return(nil, tt.domainError)
				_, err := server.GetUserByEmail(context.Background(), &pb.GetUserByEmailRequest{Email: "test@example.com"})
				assert.Error(t, err)
				st, ok := status.FromError(err)
				assert.True(t, ok)
				assert.Equal(t, tt.expectedCode, st.Code())

			case "UpdateKYCStatus":
				mockService.On("UpdateKYC", mock.Anything, mock.Anything, mock.Anything).Return(nil, tt.domainError)
				_, err := server.UpdateKYCStatus(context.Background(), &pb.UpdateKYCRequest{
					UserId:    uuid.New().String(),
					KycStatus: "verified",
				})
				assert.Error(t, err)
				st, ok := status.FromError(err)
				assert.True(t, ok)
				assert.Equal(t, tt.expectedCode, st.Code())
			}

			mockService.AssertExpectations(t)
		})
	}
}

// TestGetUser_InternalError tests internal error handling
func TestGetUser_InternalError(t *testing.T) {
	mockService := new(MockUserService)
	logger := observability.NewLogger("test", "grpc-test")
	server := grpcTransport.NewServer(mockService, logger)

	mockService.On("GetByID", mock.Anything, mock.Anything).Return(nil, errors.New("database failure"))

	req := &pb.GetUserRequest{UserId: uuid.New().String()}
	resp, err := server.GetUser(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())

	mockService.AssertExpectations(t)
}

// TestGetUserByEmail_InternalError tests internal error handling
func TestGetUserByEmail_InternalError(t *testing.T) {
	mockService := new(MockUserService)
	logger := observability.NewLogger("test", "grpc-test")
	server := grpcTransport.NewServer(mockService, logger)

	mockService.On("GetByEmail", mock.Anything, "test@example.com").Return(nil, errors.New("database failure"))

	req := &pb.GetUserByEmailRequest{Email: "test@example.com"}
	resp, err := server.GetUserByEmail(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())

	mockService.AssertExpectations(t)
}

// TestUpdateKYCStatus_InternalError tests internal error handling
func TestUpdateKYCStatus_InternalError(t *testing.T) {
	mockService := new(MockUserService)
	logger := observability.NewLogger("test", "grpc-test")
	server := grpcTransport.NewServer(mockService, logger)

	mockService.On("UpdateKYC", mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("database failure"))

	req := &pb.UpdateKYCRequest{
		UserId:    uuid.New().String(),
		KycStatus: "verified",
		UpdatedBy: "admin@example.com",
	}
	resp, err := server.UpdateKYCStatus(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())

	mockService.AssertExpectations(t)
}

// TestValidateUser_InternalError tests internal error handling for non-NotFound errors
func TestValidateUser_InternalError(t *testing.T) {
	mockService := new(MockUserService)
	logger := observability.NewLogger("test", "grpc-test")
	server := grpcTransport.NewServer(mockService, logger)

	// Return a non-NotFound error (should propagate as Internal error)
	mockService.On("GetByID", mock.Anything, mock.Anything).Return(nil, errors.New("database connection lost"))

	req := &pb.ValidateUserRequest{UserId: uuid.New().String()}
	resp, err := server.ValidateUser(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())

	mockService.AssertExpectations(t)
}

// TestListUsers_InternalError tests internal error handling
func TestListUsers_InternalError(t *testing.T) {
	mockService := new(MockUserService)
	logger := observability.NewLogger("test", "grpc-test")
	server := grpcTransport.NewServer(mockService, logger)

	mockService.On("ListUsers", mock.Anything, 10, 0).Return([]*domain.User{}, int64(0), errors.New("database failure"))

	req := &pb.ListUsersRequest{Limit: 10, Offset: 0}
	resp, err := server.ListUsers(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())

	mockService.AssertExpectations(t)
}

// TestToProtoUser_WithDeletedUser tests timestamp conversion for soft-deleted users
func TestToProtoUser_WithDeletedUser(t *testing.T) {
	mockService := new(MockUserService)
	logger := observability.NewLogger("test", "grpc-test")
	server := grpcTransport.NewServer(mockService, logger)

	// Create a soft-deleted user
	deletedUser := createTestUser()
	deletedTime := time.Now()
	deletedUser.DeletedAt = &deletedTime

	mockService.On("GetByID", mock.Anything, mock.Anything).Return(deletedUser, nil)

	req := &pb.GetUserRequest{UserId: deletedUser.ID.String()}
	resp, err := server.GetUser(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotNil(t, resp.User)
	assert.NotNil(t, resp.User.DeletedAt)
	assert.Equal(t, deletedUser.Email, resp.User.Email)

	mockService.AssertExpectations(t)
}

// TestToProtoUser_WithActiveUser tests timestamp conversion for active users (nil DeletedAt)
func TestToProtoUser_WithActiveUser(t *testing.T) {
	mockService := new(MockUserService)
	logger := observability.NewLogger("test", "grpc-test")
	server := grpcTransport.NewServer(mockService, logger)

	// Create an active user (DeletedAt is nil)
	activeUser := createTestUser()
	activeUser.DeletedAt = nil

	mockService.On("GetByID", mock.Anything, mock.Anything).Return(activeUser, nil)

	req := &pb.GetUserRequest{UserId: activeUser.ID.String()}
	resp, err := server.GetUser(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotNil(t, resp.User)
	assert.Nil(t, resp.User.DeletedAt) // Should be nil for active users
	assert.Equal(t, activeUser.Email, resp.User.Email)

	mockService.AssertExpectations(t)
}

// TestListUsers_WithTotal tests that total count is properly returned
func TestListUsers_WithTotal(t *testing.T) {
	mockService := new(MockUserService)
	logger := observability.NewLogger("test", "grpc-test")
	server := grpcTransport.NewServer(mockService, logger)

	users := []*domain.User{createTestUser(), createTestUser()}
	totalUsers := int64(100) // Total in DB

	mockService.On("ListUsers", mock.Anything, 2, 10).Return(users, totalUsers, nil)

	req := &pb.ListUsersRequest{Limit: 2, Offset: 10}
	resp, err := server.ListUsers(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Users, 2)
	assert.Equal(t, totalUsers, resp.Total)

	mockService.AssertExpectations(t)
}
