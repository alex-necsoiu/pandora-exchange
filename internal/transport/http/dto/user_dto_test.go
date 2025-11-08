package dto

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestRegisterRequest_Validate verifies validation rules
func TestRegisterRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     RegisterRequest
		wantErr bool
	}{
		{
			name: "valid request",
			req: RegisterRequest{
				Email:     "test@example.com",
				Password:  "SecurePass123!",
				FirstName: "John",
				LastName:  "Doe",
			},
			wantErr: false,
		},
		{
			name: "missing email",
			req: RegisterRequest{
				Password:  "SecurePass123!",
				FirstName: "John",
				LastName:  "Doe",
			},
			wantErr: true,
		},
		{
			name: "invalid email format",
			req: RegisterRequest{
				Email:     "not-an-email",
				Password:  "SecurePass123!",
				FirstName: "John",
				LastName:  "Doe",
			},
			wantErr: true,
		},
		{
			name: "password too short",
			req: RegisterRequest{
				Email:     "test@example.com",
				Password:  "short",
				FirstName: "John",
				LastName:  "Doe",
			},
			wantErr: true,
		},
		{
			name: "missing first name",
			req: RegisterRequest{
				Email:    "test@example.com",
				Password: "SecurePass123!",
				LastName: "Doe",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestLoginRequest_Validate verifies login validation
func TestLoginRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     LoginRequest
		wantErr bool
	}{
		{
			name: "valid login",
			req: LoginRequest{
				Email:    "test@example.com",
				Password: "password123",
			},
			wantErr: false,
		},
		{
			name: "missing email",
			req: LoginRequest{
				Password: "password123",
			},
			wantErr: true,
		},
		{
			name: "missing password",
			req: LoginRequest{
				Email: "test@example.com",
			},
			wantErr: true,
		},
		{
			name: "invalid email format",
			req: LoginRequest{
				Email:    "not-an-email",
				Password: "password123",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestUserResponse_Structure verifies response structure
func TestUserResponse_Structure(t *testing.T) {
	resp := UserResponse{
		ID:          "550e8400-e29b-41d4-a716-446655440000",
		Email:       "test@example.com",
		FirstName:   "John",
		LastName:    "Doe",
		Role:        "user",
		KYCStatus:   "pending",
		IsActive:    true,
		CreatedAt:   "2025-11-08T10:00:00Z",
		UpdatedAt:   "2025-11-08T10:00:00Z",
	}

	assert.NotEmpty(t, resp.ID)
	assert.NotEmpty(t, resp.Email)
	assert.NotEmpty(t, resp.FirstName)
	assert.Equal(t, "user", resp.Role)
	assert.True(t, resp.IsActive)
}
