package main

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/errors"
	"github.com/hail2skins/armory/internal/logger"
	"github.com/hail2skins/armory/internal/middleware"
)

// User represents a user in the system
type User struct {
	ID    uint   `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

// GetID returns the user ID (used by the error middleware)
func (u User) GetID() uint {
	return u.ID
}

func main() {
	// Initialize logger
	logger.SetupFileLogging("app.log")
	defer logger.ResetLogging()

	// Create a new Gin router
	router := gin.Default()

	// Set up error handling
	middleware.SetupErrorHandling(router)

	// Set up routes
	setupRoutes(router)

	// Start the server
	logger.Info("Starting server on :8080", nil)
	router.Run(":8080")
}

func setupRoutes(router *gin.Engine) {
	// Public routes
	router.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "Welcome to the API")
	})

	// User routes
	userRoutes := router.Group("/users")
	{
		userRoutes.GET("/:id", getUserHandler)
		userRoutes.POST("/", createUserHandler)
		userRoutes.GET("/error", errorExampleHandler)
		userRoutes.GET("/panic", panicExampleHandler)
	}
}

// getUserHandler handles GET /users/:id
func getUserHandler(c *gin.Context) {
	// Get the user ID from the URL
	id := c.Param("id")

	// In a real app, this would be a database call
	if id == "123" {
		// Return a sample user
		user := User{
			ID:    123,
			Email: "user@example.com",
			Name:  "John Doe",
		}
		c.JSON(http.StatusOK, user)
	} else if id == "not-found" {
		// Simulate a not found error
		err := errors.NewNotFoundError("User not found")
		c.Error(err)
	} else if id == "unauthorized" {
		// Simulate an auth error
		err := errors.NewAuthError("Unauthorized access")
		c.Error(err)
	} else {
		// Simulate a database error
		err := sql.ErrNoRows
		logger.Error("Database error", err, map[string]interface{}{
			"user_id": id,
		})
		c.Error(err)
	}
}

// createUserHandler handles POST /users
func createUserHandler(c *gin.Context) {
	var user User
	if err := c.ShouldBindJSON(&user); err != nil {
		// Validation error
		c.Error(errors.NewValidationError("Invalid user data"))
		return
	}

	// In a real app, this would save the user to a database
	// For this example, just return the user with an ID
	user.ID = 999
	c.JSON(http.StatusCreated, user)
}

// errorExampleHandler demonstrates different error types
func errorExampleHandler(c *gin.Context) {
	// Get the error type from the query string
	errorType := c.Query("type")

	switch errorType {
	case "validation":
		c.Error(errors.NewValidationError("Validation failed"))
	case "auth":
		c.Error(errors.NewAuthError("Authentication required"))
	case "notfound":
		c.Error(errors.NewNotFoundError("Resource not found"))
	case "payment":
		c.Error(errors.NewPaymentError("Payment failed", "CARD_DECLINED"))
	default:
		// Standard error
		c.Error(sql.ErrConnDone)
	}
}

// panicExampleHandler demonstrates panic recovery
func panicExampleHandler(c *gin.Context) {
	// This will be caught by the recovery middleware
	panic("This is a test panic")
}
