package errors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSentinelErrors verifies all domain errors are defined
func TestSentinelErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"ErrUserNotFound", ErrUserNotFound},
		{"ErrUserAlreadyExists", ErrUserAlreadyExists},
		{"ErrInvalidCredentials", ErrInvalidCredentials},
		{"ErrInvalidInput", ErrInvalidInput},
		{"ErrInvalidToken", ErrInvalidToken},
		{"ErrTokenExpired", ErrTokenExpired},
		{"ErrUnauthorized", ErrUnauthorized},
		{"ErrForbidden", ErrForbidden},
		{"ErrInternal", ErrInternal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotNil(t, tt.err)
			assert.NotEmpty(t, tt.err.Error())
		})
	}
}

// TestErrorIs verifies errors.Is works with sentinel errors
func TestErrorIs(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		target   error
		expected bool
	}{
		{
			name:     "exact match",
			err:      ErrUserNotFound,
			target:   ErrUserNotFound,
			expected: true,
		},
		{
			name:     "wrapped error matches",
			err:      errors.Join(ErrUserNotFound, errors.New("additional context")),
			target:   ErrUserNotFound,
			expected: true,
		},
		{
			name:     "different errors don't match",
			err:      ErrUserNotFound,
			target:   ErrUserAlreadyExists,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := errors.Is(tt.err, tt.target)
			assert.Equal(t, tt.expected, result)
		})
	}
}
