package domain_test

import (
	"testing"
	"time"

	"github.com/alex-necsoiu/pandora-exchange/internal/domain"
	"github.com/google/uuid"
)

func TestKYCStatus_IsValid(t *testing.T) {
	tests := []struct {
		name   string
		status domain.KYCStatus
		want   bool
	}{
		{
			name:   "pending is valid",
			status: domain.KYCStatusPending,
			want:   true,
		},
		{
			name:   "verified is valid",
			status: domain.KYCStatusVerified,
			want:   true,
		},
		{
			name:   "rejected is valid",
			status: domain.KYCStatusRejected,
			want:   true,
		},
		{
			name:   "invalid status",
			status: domain.KYCStatus("invalid"),
			want:   false,
		},
		{
			name:   "empty status",
			status: domain.KYCStatus(""),
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
		status domain.KYCStatus
		want   string
	}{
		{
			name:   "pending to string",
			status: domain.KYCStatusPending,
			want:   "pending",
		},
		{
			name:   "verified to string",
			status: domain.KYCStatusVerified,
			want:   "verified",
		},
		{
			name:   "rejected to string",
			status: domain.KYCStatusRejected,
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
		user *domain.User
		want bool
	}{
		{
			name: "active user is not deleted",
			user: &domain.User{
				ID:        uuid.New(),
				DeletedAt: nil,
			},
			want: false,
		},
		{
			name: "deleted user is deleted",
			user: &domain.User{
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
		user *domain.User
		want bool
	}{
		{
			name: "verified user is KYC verified",
			user: &domain.User{
				ID:        uuid.New(),
				KYCStatus: domain.KYCStatusVerified,
			},
			want: true,
		},
		{
			name: "pending user is not KYC verified",
			user: &domain.User{
				ID:        uuid.New(),
				KYCStatus: domain.KYCStatusPending,
			},
			want: false,
		},
		{
			name: "rejected user is not KYC verified",
			user: &domain.User{
				ID:        uuid.New(),
				KYCStatus: domain.KYCStatusRejected,
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
		token *domain.RefreshToken
		want  bool
	}{
		{
			name: "active token (not revoked, not expired)",
			token: &domain.RefreshToken{
				Token:     "active_token",
				ExpiresAt: future,
				RevokedAt: nil,
			},
			want: true,
		},
		{
			name: "revoked token is not active",
			token: &domain.RefreshToken{
				Token:     "revoked_token",
				ExpiresAt: future,
				RevokedAt: &now,
			},
			want: false,
		},
		{
			name: "expired token is not active",
			token: &domain.RefreshToken{
				Token:     "expired_token",
				ExpiresAt: past,
				RevokedAt: nil,
			},
			want: false,
		},
		{
			name: "revoked and expired token is not active",
			token: &domain.RefreshToken{
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
		token *domain.RefreshToken
		want  bool
	}{
		{
			name: "future expiration is not expired",
			token: &domain.RefreshToken{
				ExpiresAt: future,
			},
			want: false,
		},
		{
			name: "past expiration is expired",
			token: &domain.RefreshToken{
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
		token *domain.RefreshToken
		want  bool
	}{
		{
			name: "token with RevokedAt set is revoked",
			token: &domain.RefreshToken{
				RevokedAt: &now,
			},
			want: true,
		},
		{
			name: "token with nil RevokedAt is not revoked",
			token: &domain.RefreshToken{
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
