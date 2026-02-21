package user

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSentinelErrors verifies all sentinel errors in the user package are defined
func TestSentinelErrors(t *testing.T) {
	sentinelErrors := []error{
		ErrNotFound,
		ErrAlreadyExists,
		ErrDeleted,
		ErrInvalidCredentials,
		ErrInvalidKYCStatus,
		ErrInvalidEmail,
		ErrWeakPassword,
		ErrInvalidRole,
		ErrInvalidInput,
	}

	for _, err := range sentinelErrors {
		assert.NotNil(t, err, "error should not be nil")
		assert.NotEmpty(t, err.Error(), "error message should not be empty")
	}
}

// TestErrorMessages verifies error messages are descriptive
func TestErrorMessages(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		wantMessage string
	}{
		{
			name:        "ErrNotFound",
			err:         ErrNotFound,
			wantMessage: "user not found",
		},
		{
			name:        "ErrAlreadyExists",
			err:         ErrAlreadyExists,
			wantMessage: "user already exists",
		},
		{
			name:        "ErrDeleted",
			err:         ErrDeleted,
			wantMessage: "user has been deleted",
		},
		{
			name:        "ErrInvalidCredentials",
			err:         ErrInvalidCredentials,
			wantMessage: "invalid email or password",
		},
		{
			name:        "ErrInvalidKYCStatus",
			err:         ErrInvalidKYCStatus,
			wantMessage: "invalid KYC status",
		},
		{
			name:        "ErrInvalidEmail",
			err:         ErrInvalidEmail,
			wantMessage: "invalid email format",
		},
		{
			name:        "ErrWeakPassword",
			err:         ErrWeakPassword,
			wantMessage: "password does not meet security requirements",
		},
		{
			name:        "ErrInvalidRole",
			err:         ErrInvalidRole,
			wantMessage: "invalid role",
		},
		{
			name:        "ErrInvalidInput",
			err:         ErrInvalidInput,
			wantMessage: "invalid input",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantMessage, tt.err.Error())
		})
	}
}
