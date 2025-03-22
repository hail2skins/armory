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
	case *ForbiddenError:
		response = ErrorResponse{
			Code:    http.StatusForbidden,
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
	case *RateLimitError:
		response = ErrorResponse{
			Code:    http.StatusTooManyRequests,
			Message: e.Error(),
		}
	case *InternalServerError:
		response = ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: e.Error(),
			ID:      generateErrorID(), // For tracking in logs
		}
		// Log internal errors with the structured logger
		logger.Error("Internal server error", err, map[string]interface{}{
			"error_id": response.ID,
			"path":     c.Request.URL.Path,
		})
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
		// Try to render HTML response based on error type
		defer func() {
			if r := recover(); r != nil {
				// If rendering HTML fails, fall back to string response
				logger.Error("Failed to render error template", nil, map[string]interface{}{
					"error_code": response.Code,
					"panic":      r,
				})
				c.String(response.Code, response.Message)
			}
		}()

		// Use the new templ-based components
		switch response.Code {
		case http.StatusNotFound:
			// We use the component name directly since we're now using the templ package
			c.HTML(response.Code, "error.NotFound", gin.H{
				"errorMsg": response.Message,
			})
		case http.StatusForbidden:
			c.HTML(response.Code, "error.Forbidden", gin.H{
				"errorMsg": response.Message,
			})
		case http.StatusUnauthorized:
			c.HTML(response.Code, "error.Unauthorized", gin.H{
				"errorMsg": response.Message,
			})
		case http.StatusInternalServerError:
			c.HTML(response.Code, "error.InternalServerError", gin.H{
				"errorMsg": response.Message,
				"errorID":  response.ID,
			})
		default:
			// For other error types, use the generic error template
			c.HTML(response.Code, "error.Error", gin.H{
				"errorMsg":  response.Message,
				"errorCode": response.Code,
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
		logger.Debug("404 Not Found", map[string]interface{}{
			"path":   c.Request.URL.Path,
			"method": c.Request.Method,
		})

		// In test mode or if JSON requested, return JSON
		if gin.Mode() == gin.TestMode || strings.Contains(c.GetHeader("Accept"), "application/json") {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Code:    http.StatusNotFound,
				Message: "Page not found",
			})
			return
		}

		// In regular mode try templates, fallback to plain text
		defer func() {
			if r := recover(); r != nil {
				logger.Error("Failed to render 404 template", nil, map[string]interface{}{
					"panic": r,
				})
				c.String(http.StatusNotFound, "Page not found")
			}
		}()

		c.HTML(http.StatusNotFound, "error.NotFound", gin.H{
			"errorMsg": "The page you're looking for doesn't exist.",
		})
	}
}

// NoMethodHandler returns a 405 handler for Gin
func NoMethodHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		logger.Debug("405 Method Not Allowed", map[string]interface{}{
			"path":   c.Request.URL.Path,
			"method": c.Request.Method,
		})

		// In test mode or if JSON requested, return JSON
		if gin.Mode() == gin.TestMode || strings.Contains(c.GetHeader("Accept"), "application/json") {
			c.JSON(http.StatusMethodNotAllowed, ErrorResponse{
				Code:    http.StatusMethodNotAllowed,
				Message: "Method not allowed",
			})
			return
		}

		// In regular mode try templates, fallback to plain text
		defer func() {
			if r := recover(); r != nil {
				logger.Error("Failed to render 405 template", nil, map[string]interface{}{
					"panic": r,
				})
				c.String(http.StatusMethodNotAllowed, "Method not allowed")
			}
		}()

		c.HTML(http.StatusMethodNotAllowed, "error.Error", gin.H{
			"errorMsg":  "This method is not allowed for this resource.",
			"errorCode": http.StatusMethodNotAllowed,
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

		// In test mode or if JSON requested, return JSON
		if gin.Mode() == gin.TestMode || strings.Contains(c.GetHeader("Accept"), "application/json") {
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Code:    http.StatusInternalServerError,
				Message: "An internal server error occurred",
				ID:      errorID,
			})
			return
		}

		// In regular mode try templates, fallback to plain text
		defer func() {
			if r := recover(); r != nil {
				logger.Error("Failed to render 500 template", nil, map[string]interface{}{
					"panic": r,
				})
				c.String(http.StatusInternalServerError, "Internal server error")
			}
		}()

		c.HTML(http.StatusInternalServerError, "error.InternalServerError", gin.H{
			"errorMsg": "An internal server error occurred",
			"errorID":  errorID,
		})
	})
}
