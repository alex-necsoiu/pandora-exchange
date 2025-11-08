package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"pandora-exchange/internal/errors"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// TestErrorMiddleware_DomainError verifies domain errors are mapped correctly
func TestErrorMiddleware_DomainError(t *testing.T) {
	tests := []struct {
		name               string
		handlerError       error
		expectedStatus     int
		expectedErrorCode  string
		expectedMessage    string
	}{
		{
			name:               "UserNotFound error",
			handlerError:       errors.ErrUserNotFound,
			expectedStatus:     http.StatusNotFound,
			expectedErrorCode:  "USER_NOT_FOUND",
			expectedMessage:    "User not found",
		},
		{
			name:               "InvalidCredentials error",
			handlerError:       errors.ErrInvalidCredentials,
			expectedStatus:     http.StatusUnauthorized,
			expectedErrorCode:  "INVALID_CREDENTIALS",
			expectedMessage:    "Invalid credentials",
		},
		{
			name:               "UserAlreadyExists error",
			handlerError:       errors.ErrUserAlreadyExists,
			expectedStatus:     http.StatusConflict,
			expectedErrorCode:  "USER_ALREADY_EXISTS",
			expectedMessage:    "User already exists",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(ErrorMiddleware())
			
			router.GET("/test", func(c *gin.Context) {
				c.Error(tt.handlerError)
			})

			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			errorData, ok := response["error"].(map[string]interface{})
			require.True(t, ok, "Response should have error field")

			assert.Equal(t, tt.expectedErrorCode, errorData["code"])
			assert.Equal(t, tt.expectedMessage, errorData["message"])
			assert.NotEmpty(t, errorData["trace_id"])
		})
	}
}

// TestErrorMiddleware_AppError verifies AppError is handled correctly
func TestErrorMiddleware_AppError(t *testing.T) {
	router := gin.New()
	router.Use(ErrorMiddleware())
	
	router.GET("/test", func(c *gin.Context) {
		appErr := errors.NewAppError(c.Request.Context(), "CUSTOM_ERROR", "Custom error message", http.StatusTeapot)
		c.Error(appErr)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTeapot, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	errorData := response["error"].(map[string]interface{})
	assert.Equal(t, "CUSTOM_ERROR", errorData["code"])
	assert.Equal(t, "Custom error message", errorData["message"])
}

// TestErrorMiddleware_PanicRecovery verifies panic recovery
func TestErrorMiddleware_PanicRecovery(t *testing.T) {
	router := gin.New()
	router.Use(ErrorMiddleware())
	
	router.GET("/test", func(c *gin.Context) {
		panic("something went wrong!")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	errorData := response["error"].(map[string]interface{})
	assert.Equal(t, "INTERNAL_ERROR", errorData["code"])
	assert.Equal(t, "Internal server error", errorData["message"])
	assert.NotEmpty(t, errorData["trace_id"])
}

// TestErrorMiddleware_NoError verifies successful requests pass through
func TestErrorMiddleware_NoError(t *testing.T) {
	router := gin.New()
	router.Use(ErrorMiddleware())
	
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "success", response["message"])
}

// TestErrorMiddleware_MultipleErrors verifies only first error is returned
func TestErrorMiddleware_MultipleErrors(t *testing.T) {
	router := gin.New()
	router.Use(ErrorMiddleware())
	
	router.GET("/test", func(c *gin.Context) {
		c.Error(errors.ErrUserNotFound)
		c.Error(errors.ErrInvalidInput)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	errorData := response["error"].(map[string]interface{})
	assert.Equal(t, "USER_NOT_FOUND", errorData["code"])
}
