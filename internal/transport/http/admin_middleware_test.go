package http_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alex-necsoiu/pandora-exchange/internal/domain"
	httpTransport "github.com/alex-necsoiu/pandora-exchange/internal/transport/http"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// TestAdminMiddleware tests the AdminMiddleware function
func TestAdminMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name           string
		setupContext   func(c *gin.Context)
		expectedStatus int
		expectedError  string
	}{
		{
			name: "admin user passes middleware",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", uuid.New())
				c.Set("user_role", string(domain.RoleAdmin))
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "regular user is rejected",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", uuid.New())
				c.Set("user_role", string(domain.RoleUser))
			},
			expectedStatus: http.StatusForbidden,
			expectedError:  "forbidden",
		},
		{
			name: "missing role in context",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", uuid.New())
				// No role set
			},
			expectedStatus: http.StatusForbidden,
			expectedError:  "forbidden",
		},
		{
			name: "invalid role type in context",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", uuid.New())
				c.Set("user_role", 12345) // Invalid type (int instead of string)
			},
			expectedStatus: http.StatusForbidden,
			expectedError:  "forbidden",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			logger := getTestLogger()
			middleware := httpTransport.AdminMiddleware(logger)

			router := gin.New()
			router.Use(func(c *gin.Context) {
				tc.setupContext(c)
				c.Next()
			})
			router.Use(middleware)
			router.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)

			if tc.expectedError != "" {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedError, response["error"])
			}
		})
	}
}

// TestGetUserIDFromContext tests the GetUserIDFromContext helper function
func TestGetUserIDFromContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name          string
		setupContext  func(c *gin.Context)
		expectError   bool
		expectedID    uuid.UUID
	}{
		{
			name: "valid user ID in context",
			setupContext: func(c *gin.Context) {
				userID := uuid.New()
				c.Set("user_id", userID)
				// Store for validation
				c.Set("expected_id", userID)
			},
			expectError: false,
		},
		{
			name: "missing user ID in context",
			setupContext: func(c *gin.Context) {
				// No user_id set
			},
			expectError: true,
		},
		{
			name: "invalid user ID type in context",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", "not-a-uuid") // String instead of UUID
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			router := gin.New()
			router.GET("/test", func(c *gin.Context) {
				tc.setupContext(c)

				userID, err := httpTransport.GetUserIDFromContext(c)

				if tc.expectError {
					assert.Error(t, err)
					assert.Equal(t, uuid.Nil, userID)
					assert.Equal(t, domain.ErrUnauthorized, err)
				} else {
					assert.NoError(t, err)
					expectedID, _ := c.Get("expected_id")
					assert.Equal(t, expectedID.(uuid.UUID), userID)
				}

				c.JSON(http.StatusOK, gin.H{"success": true})
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

// TestGetUserRoleFromContext tests the GetUserRoleFromContext helper function
func TestGetUserRoleFromContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name         string
		setupContext func(c *gin.Context)
		expectError  bool
		expectedRole domain.Role
	}{
		{
			name: "valid admin role in context",
			setupContext: func(c *gin.Context) {
				c.Set("user_role", string(domain.RoleAdmin))
			},
			expectError:  false,
			expectedRole: domain.RoleAdmin,
		},
		{
			name: "valid user role in context",
			setupContext: func(c *gin.Context) {
				c.Set("user_role", string(domain.RoleUser))
			},
			expectError:  false,
			expectedRole: domain.RoleUser,
		},
		{
			name: "missing role in context",
			setupContext: func(c *gin.Context) {
				// No role set
			},
			expectError: true,
		},
		{
			name: "invalid role type in context",
			setupContext: func(c *gin.Context) {
				c.Set("user_role", 12345) // Int instead of string
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			router := gin.New()
			router.GET("/test", func(c *gin.Context) {
				tc.setupContext(c)

				role, err := httpTransport.GetUserRoleFromContext(c)

				if tc.expectError {
					assert.Error(t, err)
					assert.Equal(t, domain.Role(""), role)
					assert.Equal(t, domain.ErrUnauthorized, err)
				} else {
					assert.NoError(t, err)
					assert.Equal(t, tc.expectedRole, role)
				}

				c.JSON(http.StatusOK, gin.H{"success": true})
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}
