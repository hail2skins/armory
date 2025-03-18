package testing

import (
	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/database"
)

// AuthControllerInterface defines methods that AuthController implements
type AuthControllerInterface interface {
	LoginHandler(c *gin.Context)
	RegisterHandler(c *gin.Context)
	LogoutHandler(c *gin.Context)
	VerifyEmailHandler(c *gin.Context)
	ForgotPasswordHandler(c *gin.Context)
	ResetPasswordHandler(c *gin.Context)
}

// OwnerControllerInterface defines methods that OwnerController implements
type OwnerControllerInterface interface {
	Index(c *gin.Context)
	Profile(c *gin.Context)
	EditProfile(c *gin.Context)
	UpdateProfile(c *gin.Context)
	DeleteAccountConfirm(c *gin.Context)
	DeleteAccountHandler(c *gin.Context)
}

// ControllerFactory defines a function that creates controllers
type ControllerFactory interface {
	NewAuthController(db database.Service) AuthControllerInterface
	NewOwnerController(db database.Service) OwnerControllerInterface
}

var factory ControllerFactory

// RegisterControllerFactory allows the application to provide controller factory
func RegisterControllerFactory(f ControllerFactory) {
	factory = f
}

// NewAuthController creates a new AuthController
func NewAuthController(db database.Service) AuthControllerInterface {
	// This will panic if factory is not registered
	if factory == nil {
		panic("Controller factory not registered. Call RegisterControllerFactory first.")
	}
	return factory.NewAuthController(db)
}

// NewOwnerController creates a new OwnerController
func NewOwnerController(db database.Service) OwnerControllerInterface {
	if factory == nil {
		panic("Controller factory not registered. Call RegisterControllerFactory first.")
	}
	return factory.NewOwnerController(db)
}
