// Package grpc provides gRPC server implementation for internal service-to-service communication.
// This is NOT for external clients - use HTTP/REST for external APIs.
package grpc

import (
	"context"
	"errors"

	"github.com/alex-necsoiu/pandora-exchange/internal/domain"
	"github.com/alex-necsoiu/pandora-exchange/internal/observability"
	pb "github.com/alex-necsoiu/pandora-exchange/internal/transport/grpc/proto"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Server implements the gRPC UserService server
type Server struct {
	pb.UnimplementedUserServiceServer
	userService domain.UserService
	logger      *observability.Logger
}

// NewServer creates a new gRPC server instance
func NewServer(userService domain.UserService, logger *observability.Logger) *Server {
	return &Server{
		userService: userService,
		logger:      logger,
	}
}

// GetUser retrieves a user by ID (internal services only)
func (s *Server) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	s.logger.WithFields(map[string]interface{}{
		"user_id": req.UserId,
		"method":  "GetUser",
	}).Debug("gRPC request received")

	// Validate UUID
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		s.logger.WithField("error", err.Error()).Warn("Invalid user ID format")
		return nil, status.Error(codes.InvalidArgument, "invalid user ID format")
	}

	// Get user from service
	user, err := s.userService.GetByID(ctx, userID)
	if err != nil {
		return nil, s.handleServiceError(err, "failed to get user")
	}

	s.logger.WithField("user_id", userID).Debug("User retrieved successfully")

	return &pb.GetUserResponse{
		User: toProtoUser(user),
	}, nil
}

// GetUserByEmail retrieves a user by email (internal services only)
func (s *Server) GetUserByEmail(ctx context.Context, req *pb.GetUserByEmailRequest) (*pb.GetUserResponse, error) {
	s.logger.WithFields(map[string]interface{}{
		"email":  req.Email,
		"method": "GetUserByEmail",
	}).Debug("gRPC request received")

	// Validate email
	if req.Email == "" {
		return nil, status.Error(codes.InvalidArgument, "email is required")
	}

	// Get user from service
	user, err := s.userService.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, s.handleServiceError(err, "failed to get user by email")
	}

	s.logger.WithField("email", req.Email).Debug("User retrieved successfully")

	return &pb.GetUserResponse{
		User: toProtoUser(user),
	}, nil
}

// UpdateKYCStatus updates a user's KYC verification status
func (s *Server) UpdateKYCStatus(ctx context.Context, req *pb.UpdateKYCRequest) (*pb.UpdateKYCResponse, error) {
	s.logger.WithFields(map[string]interface{}{
		"user_id":    req.UserId,
		"kyc_status": req.KycStatus,
		"updated_by": req.UpdatedBy,
		"method":     "UpdateKYCStatus",
	}).Info("gRPC KYC update request received")

	// Validate UUID
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		s.logger.WithField("error", err.Error()).Warn("Invalid user ID format")
		return nil, status.Error(codes.InvalidArgument, "invalid user ID format")
	}

	// Parse KYC status
	kycStatus := domain.KYCStatus(req.KycStatus)

	// Update KYC status
	user, err := s.userService.UpdateKYC(ctx, userID, kycStatus)
	if err != nil {
		return nil, s.handleServiceError(err, "failed to update KYC status")
	}

	s.logger.WithFields(map[string]interface{}{
		"user_id":    userID,
		"kyc_status": req.KycStatus,
	}).Info("KYC status updated successfully")

	return &pb.UpdateKYCResponse{
		User: toProtoUser(user),
	}, nil
}

// ValidateUser checks if a user exists and is active
func (s *Server) ValidateUser(ctx context.Context, req *pb.ValidateUserRequest) (*pb.ValidateUserResponse, error) {
	s.logger.WithFields(map[string]interface{}{
		"user_id": req.UserId,
		"method":  "ValidateUser",
	}).Debug("gRPC validation request received")

	// Validate UUID
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		s.logger.WithField("error", err.Error()).Warn("Invalid user ID format")
		return nil, status.Error(codes.InvalidArgument, "invalid user ID format")
	}

	// Get user from service
	user, err := s.userService.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			// User not found - return valid response with is_valid=false
			return &pb.ValidateUserResponse{
				IsValid:   false,
				IsActive:  false,
				KycStatus: "",
			}, nil
		}
		return nil, s.handleServiceError(err, "failed to validate user")
	}

	// Check if user is active (not deleted)
	isActive := !user.IsDeleted()

	s.logger.WithFields(map[string]interface{}{
		"user_id":   userID,
		"is_valid":  true,
		"is_active": isActive,
	}).Debug("User validated")

	return &pb.ValidateUserResponse{
		IsValid:   true,
		IsActive:  isActive,
		KycStatus: user.KYCStatus.String(),
	}, nil
}

// ListUsers returns a paginated list of users (admin operations)
func (s *Server) ListUsers(ctx context.Context, req *pb.ListUsersRequest) (*pb.ListUsersResponse, error) {
	s.logger.WithFields(map[string]interface{}{
		"limit":  req.Limit,
		"offset": req.Offset,
		"method": "ListUsers",
	}).Debug("gRPC list users request received")

	// Validate pagination
	if req.Limit <= 0 {
		req.Limit = 10 // Default
	}
	if req.Limit > 100 {
		return nil, status.Error(codes.InvalidArgument, "limit cannot exceed 100")
	}
	if req.Offset < 0 {
		return nil, status.Error(codes.InvalidArgument, "offset must be non-negative")
	}

	// Get users from service
	users, total, err := s.userService.ListUsers(ctx, int(req.Limit), int(req.Offset))
	if err != nil {
		return nil, s.handleServiceError(err, "failed to list users")
	}

	// Convert to proto users
	protoUsers := make([]*pb.User, len(users))
	for i, user := range users {
		protoUsers[i] = toProtoUser(user)
	}

	s.logger.WithFields(map[string]interface{}{
		"count": len(users),
		"total": total,
	}).Debug("Users listed successfully")

	return &pb.ListUsersResponse{
		Users: protoUsers,
		Total: total,
	}, nil
}

// handleServiceError converts domain errors to appropriate gRPC status codes
func (s *Server) handleServiceError(err error, context string) error {
	s.logger.WithFields(map[string]interface{}{
		"error":   err.Error(),
		"context": context,
	}).Error("Service error in gRPC handler")

	switch {
	case errors.Is(err, domain.ErrUserNotFound):
		return status.Error(codes.NotFound, "user not found")
	case errors.Is(err, domain.ErrUserAlreadyExists):
		return status.Error(codes.AlreadyExists, "user already exists")
	case errors.Is(err, domain.ErrInvalidCredentials):
		return status.Error(codes.Unauthenticated, "invalid credentials")
	case errors.Is(err, domain.ErrInvalidKYCStatus):
		return status.Error(codes.InvalidArgument, "invalid KYC status")
	case errors.Is(err, domain.ErrInvalidEmail):
		return status.Error(codes.InvalidArgument, "invalid email format")
	case errors.Is(err, domain.ErrWeakPassword):
		return status.Error(codes.InvalidArgument, "password does not meet requirements")
	default:
		return status.Error(codes.Internal, "internal server error")
	}
}

// toProtoUser converts a domain User to a protobuf User
func toProtoUser(user *domain.User) *pb.User {
	protoUser := &pb.User{
		Id:        user.ID.String(),
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		KycStatus: user.KYCStatus.String(),
		Role:      user.Role.String(),
		CreatedAt: timestamppb.New(user.CreatedAt),
		UpdatedAt: timestamppb.New(user.UpdatedAt),
	}

	// Only set DeletedAt if user is soft-deleted
	if user.DeletedAt != nil {
		protoUser.DeletedAt = timestamppb.New(*user.DeletedAt)
	}

	return protoUser
}
