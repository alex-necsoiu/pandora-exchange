package http

import (
	"strings"

	"github.com/alex-necsoiu/pandora-exchange/internal/domain"
	"github.com/alex-necsoiu/pandora-exchange/internal/transport/http/dto"
)

// ToUserResponse converts a domain.User to a UserResponse DTO.
// It excludes sensitive information like hashed passwords and only returns
// data suitable for API responses.
func ToUserResponse(user domain.User) dto.UserResponse {
	return dto.UserResponse{
		ID:        user.ID.String(),
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Role:      user.Role.String(),      // Convert custom type to string
		KYCStatus: user.KYCStatus.String(), // Convert custom type to string
		IsActive:  !user.IsDeleted(),       // Use IsDeleted() method
		CreatedAt: user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"), // ISO 8601
		UpdatedAt: user.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"), // ISO 8601
	}
}

// FromRegisterRequest extracts and sanitizes registration parameters from a RegisterRequest DTO.
// It returns the individual fields needed by the service layer.
func FromRegisterRequest(req dto.RegisterRequest) (email, password, firstName, lastName string) {
	return strings.TrimSpace(req.Email),
		req.Password, // Don't trim password - whitespace might be intentional
		strings.TrimSpace(req.FirstName),
		strings.TrimSpace(req.LastName)
}

// ToTokenResponse creates a TokenResponse DTO from authentication tokens.
// It includes the access token, refresh token, and expiry information.
func ToTokenResponse(accessToken, refreshToken string, expiresIn int) dto.TokenResponse {
	return dto.TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    expiresIn,
	}
}
