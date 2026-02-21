package user_test

import (
	"testing"
	"time"

	"github.com/alex-necsoiu/pandora-exchange/internal/domain/auth"
	"github.com/alex-necsoiu/pandora-exchange/internal/domain/user"
	"github.com/google/uuid"
)

// TestRole_IsValid tests the Role.IsValid method.
func TestRole_IsValid(t *testing.T) {
	tests := []struct {
		name string
		role user.Role
		want bool
	}{
		{
			name: "user role is valid",
			role: user.RoleUser,
			want: true,
		},
		{
			name: "admin role is valid",
			role: user.RoleAdmin,
			want: true,
		},
		{
			name: "invalid role",
			role: user.Role("invalid"),
			want: false,
		},
		{
			name: "empty role",
			role: user.Role(""),
			want: false,
		},
		{
			name: "uppercase ADMIN is invalid",
			role: user.Role("ADMIN"),
			want: false,
		},
		{
			name: "uppercase USER is invalid",
			role: user.Role("USER"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.role.IsValid(); got != tt.want {
				t.Errorf("Role.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestRole_String tests the Role.String method.
func TestRole_String(t *testing.T) {
	tests := []struct {
		name string
		role user.Role
		want string
	}{
		{
			name: "user role to string",
			role: user.RoleUser,
			want: "user",
		},
		{
			name: "admin role to string",
			role: user.RoleAdmin,
			want: "admin",
		},
		{
			name: "custom role to string",
			role: user.Role("custom"),
			want: "custom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.role.String(); got != tt.want {
				t.Errorf("Role.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestUser_IsAdmin tests the User.IsAdmin method.
func TestUser_IsAdmin(t *testing.T) {
	tests := []struct {
		name string
		user *user.User
		want bool
	}{
		{
			name: "admin user is admin",
			user: &user.User{
				ID:   uuid.New(),
				Role: user.RoleAdmin,
			},
			want: true,
		},
		{
			name: "regular user is not admin",
			user: &user.User{
				ID:   uuid.New(),
				Role: user.RoleUser,
			},
			want: false,
		},
		{
			name: "user with invalid role is not admin",
			user: &user.User{
				ID:   uuid.New(),
				Role: user.Role("invalid"),
			},
			want: false,
		},
		{
			name: "user with empty role is not admin",
			user: &user.User{
				ID:   uuid.New(),
				Role: user.Role(""),
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.user.IsAdmin(); got != tt.want {
				t.Errorf("User.IsAdmin() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestKYCStatus_IsValid(t *testing.T) {
	tests := []struct {
		name   string
		status user.KYCStatus
		want   bool
	}{
		{
			name:   "pending is valid",
			status: user.KYCStatusPending,
			want:   true,
		},
		{
			name:   "verified is valid",
			status: user.KYCStatusVerified,
			want:   true,
		},
		{
			name:   "rejected is valid",
			status: user.KYCStatusRejected,
			want:   true,
		},
		{
			name:   "invalid status",
			status: user.KYCStatus("invalid"),
			want:   false,
		},
		{
			name:   "empty status",
			status: user.KYCStatus(""),
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.IsValid(); got != tt.want {
				t.Errorf("KYCStatus.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestKYCStatus_String(t *testing.T) {
	tests := []struct {
		name   string
		status user.KYCStatus
		want   string
	}{
		{
			name:   "pending to string",
			status: user.KYCStatusPending,
			want:   "pending",
		},
		{
			name:   "verified to string",
			status: user.KYCStatusVerified,
			want:   "verified",
		},
		{
			name:   "rejected to string",
			status: user.KYCStatusRejected,
			want:   "rejected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.String(); got != tt.want {
				t.Errorf("KYCStatus.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUser_IsDeleted(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name string
		user *user.User
		want bool
	}{
		{
			name: "active user is not deleted",
			user: &user.User{
				ID:        uuid.New(),
				DeletedAt: nil,
			},
			want: false,
		},
		{
			name: "deleted user is deleted",
			user: &user.User{
				ID:        uuid.New(),
				DeletedAt: &now,
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.user.IsDeleted(); got != tt.want {
				t.Errorf("User.IsDeleted() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUser_IsKYCVerified(t *testing.T) {
	tests := []struct {
		name string
		user *user.User
		want bool
	}{
		{
			name: "verified user is KYC verified",
			user: &user.User{
				ID:        uuid.New(),
				KYCStatus: user.KYCStatusVerified,
			},
			want: true,
		},
		{
			name: "pending user is not KYC verified",
			user: &user.User{
				ID:        uuid.New(),
				KYCStatus: user.KYCStatusPending,
			},
			want: false,
		},
		{
			name: "rejected user is not KYC verified",
			user: &user.User{
				ID:        uuid.New(),
				KYCStatus: user.KYCStatusRejected,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.user.IsKYCVerified(); got != tt.want {
				t.Errorf("User.IsKYCVerified() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRefreshToken_IsActive(t *testing.T) {
	now := time.Now()
	future := now.Add(1 * time.Hour)
	past := now.Add(-1 * time.Hour)

	tests := []struct {
		name  string
		token *auth.RefreshToken
		want  bool
	}{
		{
			name: "active token (not revoked, not expired)",
			token: &auth.RefreshToken{
				Token:     "active_token",
				ExpiresAt: future,
				RevokedAt: nil,
			},
			want: true,
		},
		{
			name: "revoked token is not active",
			token: &auth.RefreshToken{
				Token:     "revoked_token",
				ExpiresAt: future,
				RevokedAt: &now,
			},
			want: false,
		},
		{
			name: "expired token is not active",
			token: &auth.RefreshToken{
				Token:     "expired_token",
				ExpiresAt: past,
				RevokedAt: nil,
			},
			want: false,
		},
		{
			name: "revoked and expired token is not active",
			token: &auth.RefreshToken{
				Token:     "revoked_expired_token",
				ExpiresAt: past,
				RevokedAt: &now,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.token.IsActive(); got != tt.want {
				t.Errorf("RefreshToken.IsActive() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRefreshToken_IsExpired(t *testing.T) {
	now := time.Now()
	future := now.Add(1 * time.Hour)
	past := now.Add(-1 * time.Hour)

	tests := []struct {
		name  string
		token *auth.RefreshToken
		want  bool
	}{
		{
			name: "future expiration is not expired",
			token: &auth.RefreshToken{
				ExpiresAt: future,
			},
			want: false,
		},
		{
			name: "past expiration is expired",
			token: &auth.RefreshToken{
				ExpiresAt: past,
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.token.IsExpired(); got != tt.want {
				t.Errorf("RefreshToken.IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRefreshToken_IsRevoked(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name  string
		token *auth.RefreshToken
		want  bool
	}{
		{
			name: "token with RevokedAt set is revoked",
			token: &auth.RefreshToken{
				RevokedAt: &now,
			},
			want: true,
		},
		{
			name: "token with nil RevokedAt is not revoked",
			token: &auth.RefreshToken{
				RevokedAt: nil,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.token.IsRevoked(); got != tt.want {
				t.Errorf("RefreshToken.IsRevoked() = %v, want %v", got, tt.want)
			}
		})
	}
}
