package middleware

import (
	"encoding/json"
	stderrors "errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/errors"
	"github.com/stretchr/testify/assert"
)

func TestErrorMiddleware(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name         string
		err          error
		acceptHeader string
		expectedCode int
		expectedBody string
		isJSON       bool
	}{
		{
			name:         "ValidationError HTML",
			err:          errors.NewValidationError("Invalid input"),
			acceptHeader: "text/html",
			expectedCode: http.StatusBadRequest,
			expectedBody: "Invalid input",
			isJSON:       false,
		},
		{
			name:         "ValidationError JSON",
			err:          errors.NewValidationError("Invalid input"),
			acceptHeader: "application/json",
			expectedCode: http.StatusBadRequest,
			isJSON:       true,
		},
		{
			name:         "AuthError HTML",
			err:          errors.NewAuthError("Unauthorized access"),
			acceptHeader: "text/html",
			expectedCode: http.StatusUnauthorized,
			expectedBody: "Unauthorized access",
			isJSON:       false,
		},
		{
			name:         "NotFoundError HTML",
			err:          errors.NewNotFoundError("Resource not found"),
			acceptHeader: "text/html",
			expectedCode: http.StatusNotFound,
			expectedBody: "Resource not found",
			isJSON:       false,
		},
		{
			name:         "Generic Error HTML",
			err:          errors.NewPaymentError("Payment failed", "PAYMENT_FAILED"),
			acceptHeader: "text/html",
			expectedCode: http.StatusBadRequest,
			expectedBody: "Payment failed",
			isJSON:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new Gin router
			r := gin.New()

			// Add the error middleware
			r.Use(ErrorHandler())

			// Add a test route that returns an error
			r.GET("/test", func(c *gin.Context) {
				// Simulate error
				errors.HandleError(c, tt.err)
			})

			// Create a test request
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Accept", tt.acceptHeader)

			// Serve the request
			r.ServeHTTP(w, req)

			// Assert status code
			assert.Equal(t, tt.expectedCode, w.Code)

			if tt.isJSON {
				// For JSON responses
				var response errors.ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedCode, response.Code)
			} else {
				// For HTML responses
				assert.Contains(t, w.Body.String(), tt.expectedBody)
			}
		})
	}
}

func TestErrorMiddlewareWithNormalRequest(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create a new Gin router
	r := gin.New()

	// Add the error middleware
	r.Use(ErrorHandler())

	// Add a test route that succeeds
	r.GET("/success", func(c *gin.Context) {
		c.String(http.StatusOK, "success")
	})

	// Create a test request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/success", nil)

	// Serve the request
	r.ServeHTTP(w, req)

	// Assert the middleware didn't interfere with normal operation
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "success", w.Body.String())
}

func TestErrorMiddlewareWithGinErrors(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name         string
		err          error
		acceptHeader string
		expectedCode int
	}{
		{
			name:         "ValidationError via c.Error()",
			err:          errors.NewValidationError("Invalid input"),
			acceptHeader: "application/json",
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "AuthError via c.Error()",
			err:          errors.NewAuthError("Unauthorized access"),
			acceptHeader: "application/json",
			expectedCode: http.StatusUnauthorized,
		},
		{
			name:         "NotFoundError via c.Error()",
			err:          errors.NewNotFoundError("Resource not found"),
			acceptHeader: "application/json",
			expectedCode: http.StatusNotFound,
		},
		{
			name:         "Standard error via c.Error()",
			err:          stderrors.New("Standard error"),
			acceptHeader: "application/json",
			expectedCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new Gin router
			r := gin.New()

			// Add the error middleware
			r.Use(ErrorHandler())

			// Add a test route that adds an error to the context
			r.GET("/error", func(c *gin.Context) {
				c.Error(tt.err)
				// Don't call HandleError here, let the middleware handle it
			})

			// Create a test request
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/error", nil)
			req.Header.Set("Accept", tt.acceptHeader)

			// Serve the request
			r.ServeHTTP(w, req)

			// Assert status code
			assert.Equal(t, tt.expectedCode, w.Code)

			// For JSON responses
			var response errors.ErrorResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedCode, response.Code)
		})
	}
}

func TestPanicRecovery(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create a new Gin router with recovery middleware
	r := gin.New()
	r.Use(errors.RecoveryHandler())

	// Add a route that will panic
	r.GET("/panic", func(c *gin.Context) {
		panic("test panic")
	})

	// Create a test request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/panic", nil)
	req.Header.Set("Accept", "application/json")

	// Serve the request
	r.ServeHTTP(w, req)

	// Assert status code
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	// Check the response body
	var response errors.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, response.Code)
	assert.Equal(t, "An internal server error occurred", response.Message)
	assert.NotEmpty(t, response.ID)
}
