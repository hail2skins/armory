package errors

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/logger"
)

type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	ID      string `json:"id,omitempty"` // For tracking
}

func HandleError(c *gin.Context, err error) {
	var response ErrorResponse

	switch e := err.(type) {
	case *ValidationError:
		response = ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: e.Error(),
		}
	case *AuthError:
		response = ErrorResponse{
			Code:    http.StatusUnauthorized,
			Message: e.Error(),
		}
	case *NotFoundError:
		response = ErrorResponse{
			Code:    http.StatusNotFound,
			Message: e.Error(),
		}
	case *PaymentError:
		response = ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: e.Error(),
		}
	default:
		response = ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "An internal error occurred",
			ID:      generateErrorID(), // For tracking in logs
		}
		// Log internal errors with the structured logger
		logger.Error("Internal server error", err, map[string]interface{}{
			"error_id": response.ID,
			"path":     c.Request.URL.Path,
		})
	}

	// Determine response format based on Accept header
	acceptHeader := c.GetHeader("Accept")
	if strings.Contains(acceptHeader, "application/json") {
		// Return JSON response
		c.JSON(response.Code, response)
	} else {
		// Check if we're in test mode
		if gin.Mode() == gin.TestMode {
			// In test mode, just set the status code and a simple text response
			c.String(response.Code, response.Message)
		} else {
			// Try to render HTML response, fall back to string if template is not available
			defer func() {
				if r := recover(); r != nil {
					// If rendering HTML fails, fall back to string response
					c.String(response.Code, response.Message)
				}
			}()

			// Return HTML response
			c.HTML(response.Code, "partials/error.templ", gin.H{
				"errorMsg": response.Message,
			})
		}
	}
}

// generateErrorID creates a random ID for tracking errors
func generateErrorID() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return "error-generating-id"
	}
	return hex.EncodeToString(bytes)
}

// NoRouteHandler returns a 404 handler for Gin
func NoRouteHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		HandleError(c, NewNotFoundError("Page not found"))
	}
}

// NoMethodHandler returns a 405 handler for Gin
func NoMethodHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusMethodNotAllowed, ErrorResponse{
			Code:    http.StatusMethodNotAllowed,
			Message: "Method not allowed",
		})
	}
}

// RecoveryHandler returns a recovery middleware for Gin
func RecoveryHandler() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		errorID := generateErrorID()

		// Log the panic
		logger.Error("Panic recovered", nil, map[string]interface{}{
			"error_id":  errorID,
			"path":      c.Request.URL.Path,
			"recovered": recovered,
		})

		// Return a 500 error
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "An internal server error occurred",
			ID:      errorID,
		})
	})
}
