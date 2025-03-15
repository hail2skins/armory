package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/errors"
	"github.com/stretchr/testify/assert"
)

func TestSetupErrorHandling(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create a new Gin router
	router := gin.New()

	// Set up error handling
	SetupErrorHandling(router)

	// Add a test route that succeeds
	router.GET("/success", func(c *gin.Context) {
		c.String(http.StatusOK, "success")
	})

	// Add a test route that returns an error
	router.GET("/error", func(c *gin.Context) {
		c.Error(errors.NewValidationError("Invalid input"))
	})

	// Add a test route that panics
	router.GET("/panic", func(c *gin.Context) {
		panic("test panic")
	})

	// Test cases
	tests := []struct {
		name         string
		path         string
		expectedCode int
		expectedBody string
	}{
		{
			name:         "Success route",
			path:         "/success",
			expectedCode: http.StatusOK,
			expectedBody: "success",
		},
		{
			name:         "Error route",
			path:         "/error",
			expectedCode: http.StatusBadRequest,
			expectedBody: "Invalid input",
		},
		{
			name:         "Panic route",
			path:         "/panic",
			expectedCode: http.StatusInternalServerError,
			expectedBody: "An internal server error occurred",
		},
		{
			name:         "Not found route",
			path:         "/not-found",
			expectedCode: http.StatusNotFound,
			expectedBody: "Page not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test request
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, tt.path, nil)
			req.Header.Set("Accept", "application/json")

			// Serve the request
			router.ServeHTTP(w, req)

			// Assert status code
			assert.Equal(t, tt.expectedCode, w.Code)

			// For success route, check the body directly
			if tt.path == "/success" {
				assert.Equal(t, tt.expectedBody, w.Body.String())
			} else {
				// For error routes, check the JSON response
				var response errors.ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedCode, response.Code)

				// For internal server errors, check that we have an ID
				if tt.expectedCode == http.StatusInternalServerError {
					assert.NotEmpty(t, response.ID)
				}

				// Check the message
				assert.Equal(t, tt.expectedBody, response.Message)
			}
		})
	}
}
