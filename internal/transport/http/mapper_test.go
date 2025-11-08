package http

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/alex-necsoiu/pandora-exchange/internal/domain"
	"github.com/alex-necsoiu/pandora-exchange/internal/transport/http/dto"
)

// TestToUserResponse verifies domain.User to UserResponse mapping
func TestToUserResponse(t *testing.T) {
	tests := []struct {
		name     string
		user     domain.User
		expected dto.UserResponse
	}{
		{
			name: "complete user",
			user: domain.User{
				ID:        uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
				Email:     "test@example.com",
				FirstName: "John",
				LastName:  "Doe",
				Role:      domain.RoleUser,
				KYCStatus: domain.KYCStatusVerified,
				DeletedAt: nil,  // Active user
				CreatedAt: time.Date(2025, 11, 8, 10, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2025, 11, 8, 12, 0, 0, 0, time.UTC),
			},
			expected: dto.UserResponse{
				ID:        "550e8400-e29b-41d4-a716-446655440000",
				Email:     "test@example.com",
				FirstName: "John",
				LastName:  "Doe",
				Role:      "user",
				KYCStatus: "verified",
				IsActive:  true,
				CreatedAt: "2025-11-08T10:00:00Z",
				UpdatedAt: "2025-11-08T12:00:00Z",
			},
		},
		{
			name: "inactive user",
			user: domain.User{
				ID:        uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"),
				Email:     "inactive@example.com",
				FirstName: "Jane",
				LastName:  "Smith",
				Role:      domain.RoleUser,
				KYCStatus: domain.KYCStatusPending,
				DeletedAt: timePtr(time.Date(2025, 11, 1, 12, 0, 0, 0, time.UTC)),  // Deleted user
				CreatedAt: time.Date(2025, 11, 1, 10, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2025, 11, 1, 10, 0, 0, 0, time.UTC),
			},
			expected: dto.UserResponse{
				ID:        "550e8400-e29b-41d4-a716-446655440001",
				Email:     "inactive@example.com",
				FirstName: "Jane",
				LastName:  "Smith",
				Role:      "user",
				KYCStatus: "pending",
				IsActive:  false,
				CreatedAt: "2025-11-01T10:00:00Z",
				UpdatedAt: "2025-11-01T10:00:00Z",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToUserResponse(tt.user)

			assert.Equal(t, tt.expected.ID, result.ID)
			assert.Equal(t, tt.expected.Email, result.Email)
			assert.Equal(t, tt.expected.FirstName, result.FirstName)
			assert.Equal(t, tt.expected.LastName, result.LastName)
			assert.Equal(t, tt.expected.Role, result.Role)
			assert.Equal(t, tt.expected.KYCStatus, result.KYCStatus)
			assert.Equal(t, tt.expected.IsActive, result.IsActive)
			assert.Equal(t, tt.expected.CreatedAt, result.CreatedAt)
			assert.Equal(t, tt.expected.UpdatedAt, result.UpdatedAt)
		})
	}
}

// TestFromRegisterRequest verifies RegisterRequest to service params mapping
func TestFromRegisterRequest(t *testing.T) {
	tests := []struct {
		name              string
		req               dto.RegisterRequest
		expectedEmail     string
		expectedPassword  string
		expectedFirstName string
		expectedLastName  string
	}{
		{
			name: "valid registration request",
			req: dto.RegisterRequest{
				Email:     "newuser@example.com",
				Password:  "SecurePass123!",
				FirstName: "Alice",
				LastName:  "Johnson",
			},
			expectedEmail:     "newuser@example.com",
			expectedPassword:  "SecurePass123!",
			expectedFirstName: "Alice",
			expectedLastName:  "Johnson",
		},
		{
			name: "with whitespace",
			req: dto.RegisterRequest{
				Email:     "  user@example.com  ",
				Password:  "password",
				FirstName: "  Bob  ",
				LastName:  "  Smith  ",
			},
			expectedEmail:     "user@example.com",
			expectedPassword:  "password",
			expectedFirstName: "Bob",
			expectedLastName:  "Smith",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			email, password, firstName, lastName := FromRegisterRequest(tt.req)

			assert.Equal(t, tt.expectedEmail, email)
			assert.Equal(t, tt.expectedPassword, password)
			assert.Equal(t, tt.expectedFirstName, firstName)
			assert.Equal(t, tt.expectedLastName, lastName)
		})
	}
}

// TestToTokenResponse verifies token response mapping
func TestToTokenResponse(t *testing.T) {
	accessToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
	refreshToken := "refresh-token-value"
	expiresIn := 900 // 15 minutes

	result := ToTokenResponse(accessToken, refreshToken, expiresIn)

	assert.Equal(t, accessToken, result.AccessToken)
	assert.Equal(t, refreshToken, result.RefreshToken)
	assert.Equal(t, "Bearer", result.TokenType)
	assert.Equal(t, expiresIn, result.ExpiresIn)
}

// timePtr returns a pointer to the given time value
func timePtr(t time.Time) *time.Time {
	return &t
}
