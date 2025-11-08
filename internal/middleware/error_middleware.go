package middleware

import (
	"github.com/gin-gonic/gin"

	"github.com/alex-necsoiu/pandora-exchange/internal/errors"
	"github.com/alex-necsoiu/pandora-exchange/pkg/logger"
)

// ErrorMiddleware is a Gin middleware that handles errors and panics.
// It intercepts errors set via c.Error() and converts them to appropriate HTTP responses.
// It also recovers from panics and returns a 500 Internal Server Error.
func ErrorMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Setup panic recovery
		defer func() {
			if err := recover(); err != nil {
				// Log the panic
				log := logger.WithTrace(c.Request.Context(), logger.GetLogger())
				log.Error().
					Interface("panic", err).
					Str("path", c.Request.URL.Path).
					Str("method", c.Request.Method).
					Msg("Panic recovered")

				// Return 500 Internal Server Error
				statusCode, response := errors.ToHTTPError(c.Request.Context(), errors.ErrInternal)
				c.JSON(statusCode, response)
				c.Abort()
			}
		}()

		// Process request
		c.Next()

		// Check if there are any errors
		if len(c.Errors) > 0 {
			// Get the first error (we only handle the first one)
			err := c.Errors[0].Err

			// Log the error
			log := logger.WithTrace(c.Request.Context(), logger.GetLogger())
			log.Error().
				Err(err).
				Str("path", c.Request.URL.Path).
				Str("method", c.Request.Method).
				Msg("Request error")

			// Convert error to HTTP response
			statusCode, response := errors.ToHTTPError(c.Request.Context(), err)

			// Only set response if not already set
			if !c.Writer.Written() {
				c.JSON(statusCode, response)
			}

			c.Abort()
		}
	}
}
