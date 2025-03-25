package controller

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/render"
	"github.com/hail2skins/armory/cmd/web/views/data"
	errortempl "github.com/hail2skins/armory/cmd/web/views/error"
	"github.com/hail2skins/armory/internal/logger"
)

// ErrorController handles error-related functionality
type ErrorController struct{}

// NewErrorController creates a new error controller
func NewErrorController() *ErrorController {
	return &ErrorController{}
}

// CreateTemplRenderer creates a renderer for templ components
// This enables Gin to render templ-based HTML responses for error pages
func (c *ErrorController) CreateTemplRenderer() *ErrorTemplRenderer {
	return &ErrorTemplRenderer{}
}

// NoRouteHandler returns a 404 handler for undefined routes
func (c *ErrorController) NoRouteHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		acceptHeader := ctx.GetHeader("Accept")
		logger.Debug("404 Not Found", map[string]interface{}{
			"path":          ctx.Request.URL.Path,
			"method":        ctx.Request.Method,
			"accept_header": acceptHeader,
			"test_mode":     gin.Mode() == gin.TestMode,
			"wants_json":    strings.Contains(acceptHeader, "application/json"),
		})

		// Only return JSON if explicitly requested (not based on test mode)
		if strings.Contains(acceptHeader, "application/json") {
			logger.Debug("Returning JSON response for 404", nil)
			ctx.JSON(http.StatusNotFound, gin.H{
				"code":    http.StatusNotFound,
				"message": "Page not found",
			})
			return
		}

		logger.Debug("Rendering HTML template for 404", nil)
		// Otherwise render the HTML template using the RenderNotFound method
		c.RenderNotFound(ctx, "The page you're looking for doesn't exist.")
	}
}

// NoMethodHandler returns a 405 handler for method not allowed
func (c *ErrorController) NoMethodHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		acceptHeader := ctx.GetHeader("Accept")
		logger.Debug("405 Method Not Allowed", map[string]interface{}{
			"path":          ctx.Request.URL.Path,
			"method":        ctx.Request.Method,
			"accept_header": acceptHeader,
			"wants_json":    strings.Contains(acceptHeader, "application/json"),
		})

		// Only return JSON if explicitly requested (not based on test mode)
		if strings.Contains(acceptHeader, "application/json") {
			logger.Debug("Returning JSON response for 405", nil)
			ctx.JSON(http.StatusMethodNotAllowed, gin.H{
				"code":    http.StatusMethodNotAllowed,
				"message": "Method not allowed",
			})
			return
		}

		logger.Debug("Rendering HTML template for 405", nil)
		// Create proper AuthData
		authData := data.AuthData{
			Title:       "405 - Method Not Allowed",
			SiteName:    "Virtual Armory",
			CurrentPath: ctx.Request.URL.Path,
		}

		// Get auth information if available
		auth, exists := ctx.Get("auth")
		if exists {
			if authController, ok := auth.(*AuthController); ok {
				user, authenticated := authController.GetCurrentUser(ctx)
				if authenticated && user != nil {
					authData.Authenticated = true
					authData.Email = user.GetUserName()
				}
			}
		}

		// Get CSRF token if available
		token, exists := ctx.Get("csrf_token")
		if exists && token != nil {
			if tokenStr, ok := token.(string); ok && tokenStr != "" {
				authData.CSRFToken = tokenStr
			}
		}

		// Check for active promotion
		activePromo, exists := ctx.Get("active_promotion")
		if exists && activePromo != nil {
			authData.ActivePromotion = activePromo
		}

		// Render the error template with the populated authData
		ctx.HTML(http.StatusMethodNotAllowed, "error.Error", gin.H{
			"errorMsg":  "This method is not allowed for this resource.",
			"errorCode": http.StatusMethodNotAllowed,
			"authData":  authData,
		})
	}
}

// RecoveryHandler returns a recovery middleware for panic recovery
func (c *ErrorController) RecoveryHandler() gin.HandlerFunc {
	return gin.CustomRecovery(func(ctx *gin.Context, recovered interface{}) {
		// Generate an error ID for tracking
		errorID := generateErrorID()

		acceptHeader := ctx.GetHeader("Accept")
		// Log the panic
		logger.Error("Panic recovered", nil, map[string]interface{}{
			"error_id":      errorID,
			"path":          ctx.Request.URL.Path,
			"recovered":     recovered,
			"accept_header": acceptHeader,
			"wants_json":    strings.Contains(acceptHeader, "application/json"),
		})

		// Only return JSON if explicitly requested (not based on test mode)
		if strings.Contains(acceptHeader, "application/json") {
			logger.Debug("Returning JSON response for 500", nil)
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"code":    http.StatusInternalServerError,
				"message": "An internal server error occurred",
				"id":      errorID,
			})
			return
		}

		logger.Debug("Rendering HTML template for 500", nil)
		// Otherwise render the HTML template
		c.RenderInternalServerError(ctx, "An internal server error occurred", errorID)
	})
}

// generateErrorID creates a random ID for tracking errors
func generateErrorID() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return "error-generating-id"
	}
	return hex.EncodeToString(bytes)
}

// RenderNotFound renders a 404 Not Found error page
func (c *ErrorController) RenderNotFound(ctx *gin.Context, message string) {
	logger.Debug("RenderNotFound called", map[string]interface{}{
		"message":      message,
		"content_type": ctx.ContentType(),
		"path":         ctx.Request.URL.Path,
	})

	// Create proper AuthData
	authData := data.AuthData{
		Title:       "404 - Page Not Found",
		SiteName:    "Virtual Armory",
		CurrentPath: ctx.Request.URL.Path,
	}

	// Get auth information if available
	auth, exists := ctx.Get("auth")
	if exists {
		if authController, ok := auth.(*AuthController); ok {
			user, authenticated := authController.GetCurrentUser(ctx)
			if authenticated && user != nil {
				authData.Authenticated = true
				authData.Email = user.GetUserName()
			}
		}
	}

	// Just to be sure, always set the content type to HTML
	ctx.Header("Content-Type", "text/html; charset=utf-8")

	// Force gin to render HTML, not JSON
	logger.Debug("Rendering 404 HTML response", map[string]interface{}{
		"template": "error.NotFound",
	})

	ctx.HTML(http.StatusNotFound, "error.NotFound", gin.H{
		"errorMsg": message,
		"authData": authData,
	})
}

// RenderError renders a generic error page
func (c *ErrorController) RenderError(ctx *gin.Context, message string, code int) {
	// Create proper AuthData
	authData := data.AuthData{
		Title:       "Error " + strconv.Itoa(code),
		SiteName:    "Virtual Armory",
		CurrentPath: ctx.Request.URL.Path,
	}

	// Get auth information if available
	auth, exists := ctx.Get("auth")
	if exists {
		if authController, ok := auth.(*AuthController); ok {
			user, authenticated := authController.GetCurrentUser(ctx)
			if authenticated && user != nil {
				authData.Authenticated = true
				authData.Email = user.GetUserName()
			}
		}
	}

	ctx.HTML(code, "error.Error", gin.H{
		"errorMsg":  message,
		"errorCode": code,
		"authData":  authData,
	})
}

// RenderForbidden renders a 403 Forbidden error page
func (c *ErrorController) RenderForbidden(ctx *gin.Context, message string) {
	// Create proper AuthData
	authData := data.AuthData{
		Title:       "403 - Forbidden",
		SiteName:    "Virtual Armory",
		CurrentPath: ctx.Request.URL.Path,
	}

	// Get auth information if available
	auth, exists := ctx.Get("auth")
	if exists {
		if authController, ok := auth.(*AuthController); ok {
			user, authenticated := authController.GetCurrentUser(ctx)
			if authenticated && user != nil {
				authData.Authenticated = true
				authData.Email = user.GetUserName()
			}
		}
	}

	ctx.HTML(http.StatusForbidden, "error.Forbidden", gin.H{
		"errorMsg": message,
		"authData": authData,
	})
}

// RenderUnauthorized renders a 401 Unauthorized error page
func (c *ErrorController) RenderUnauthorized(ctx *gin.Context, message string) {
	// Create proper AuthData
	authData := data.AuthData{
		Title:       "401 - Unauthorized",
		SiteName:    "Virtual Armory",
		CurrentPath: ctx.Request.URL.Path,
	}

	ctx.HTML(http.StatusUnauthorized, "error.Unauthorized", gin.H{
		"errorMsg": message,
		"authData": authData,
	})
}

// RenderInternalServerError renders a 500 Internal Server Error page
func (c *ErrorController) RenderInternalServerError(ctx *gin.Context, message string, errorID string) {
	// Create proper AuthData
	authData := data.AuthData{
		Title:       "500 - Internal Server Error",
		SiteName:    "Virtual Armory",
		CurrentPath: ctx.Request.URL.Path,
	}

	// Get auth information if available
	auth, exists := ctx.Get("auth")
	if exists {
		if authController, ok := auth.(*AuthController); ok {
			user, authenticated := authController.GetCurrentUser(ctx)
			if authenticated && user != nil {
				authData.Authenticated = true
				authData.Email = user.GetUserName()
			}
		}
	}

	ctx.HTML(http.StatusInternalServerError, "error.InternalServerError", gin.H{
		"errorMsg": message,
		"errorID":  errorID,
		"authData": authData,
	})
}

// ErrorTemplInstance represents a templ component instance for error pages
type ErrorTemplInstance struct {
	Component templ.Component
}

// Render implements the render.Render interface
func (t *ErrorTemplInstance) Render(w http.ResponseWriter) error {
	logger.Debug("ErrorTemplInstance.Render called", map[string]interface{}{
		"component_nil":  t.Component == nil,
		"component_type": fmt.Sprintf("%T", t.Component),
	})

	if t.Component == nil {
		return errors.New("no such template")
	}

	// Create a valid context instead of passing nil
	ctx := context.Background()
	err := t.Component.Render(ctx, w)

	if err != nil {
		logger.Error("Error rendering templ component", err, map[string]interface{}{
			"component_type": fmt.Sprintf("%T", t.Component),
		})
	} else {
		logger.Debug("Successfully rendered templ component", map[string]interface{}{
			"component_type": fmt.Sprintf("%T", t.Component),
		})
	}

	return err
}

// WriteContentType writes the content type header
func (t *ErrorTemplInstance) WriteContentType(w http.ResponseWriter) {
	logger.Debug("ErrorTemplInstance.WriteContentType called", nil)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
}

// ErrorTemplRenderer is a custom renderer for templ error components
type ErrorTemplRenderer struct{}

// Instance implements the render.HTMLRender interface
func (t *ErrorTemplRenderer) Instance(name string, data interface{}) render.Render {
	var component templ.Component

	// Debug log for template rendering
	logger.Debug("ErrorTemplRenderer.Instance called", map[string]interface{}{
		"template_name": name,
		"data_type":     fmt.Sprintf("%T", data),
	})

	// Parse the data from gin.H
	var errorMsg, errorID string
	var errorCode int

	if h, ok := data.(gin.H); ok {
		logger.Debug("Parsed gin.H data", map[string]interface{}{
			"keys": fmt.Sprintf("%v", h),
		})

		if msg, exists := h["errorMsg"]; exists {
			if str, ok := msg.(string); ok {
				errorMsg = str
			}
		}

		if code, exists := h["errorCode"]; exists {
			if num, ok := code.(int); ok {
				errorCode = num
			}
		}

		if id, exists := h["errorID"]; exists {
			if str, ok := id.(string); ok {
				errorID = str
			}
		}
	}

	// Route to the appropriate templ component
	switch name {
	case "error.NotFound":
		component = errortempl.NotFound(errorMsg)
		logger.Debug("Rendering NotFound template", map[string]interface{}{
			"errorMsg": errorMsg,
		})
	case "error.Error":
		component = errortempl.Error(errorMsg, errorCode)
		logger.Debug("Rendering Error template", map[string]interface{}{
			"errorMsg":  errorMsg,
			"errorCode": errorCode,
		})
	case "error.Forbidden":
		component = errortempl.Forbidden(errorMsg)
		logger.Debug("Rendering Forbidden template", map[string]interface{}{
			"errorMsg": errorMsg,
		})
	case "error.Unauthorized":
		component = errortempl.Unauthorized(errorMsg)
		logger.Debug("Rendering Unauthorized template", map[string]interface{}{
			"errorMsg": errorMsg,
		})
	case "error.InternalServerError":
		component = errortempl.InternalServerError(errorMsg, errorID)
		logger.Debug("Rendering InternalServerError template", map[string]interface{}{
			"errorMsg": errorMsg,
			"errorID":  errorID,
		})
	default:
		// Fallback to generic error
		logger.Warn("Unknown template requested, falling back to generic error", map[string]interface{}{
			"requestedTemplate": name,
		})
		component = errortempl.Error("Unknown error occurred", http.StatusInternalServerError)
	}

	return &ErrorTemplInstance{
		Component: component,
	}
}
