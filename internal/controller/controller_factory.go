package controller

import (
	"github.com/hail2skins/armory/internal/controller/testing"
	"github.com/hail2skins/armory/internal/database"
)

// ControllerFactoryImpl is the actual implementation of the controller factory
type ControllerFactoryImpl struct{}

// NewAuthController creates a new AuthController instance
func (f *ControllerFactoryImpl) NewAuthController(db database.Service) testing.AuthControllerInterface {
	return NewAuthController(db)
}

// NewOwnerController creates a new OwnerController instance
func (f *ControllerFactoryImpl) NewOwnerController(db database.Service) testing.OwnerControllerInterface {
	return NewOwnerController(db)
}

// RegisterFactory registers the controller factory with the testing package
func RegisterFactory() {
	testing.RegisterControllerFactory(&ControllerFactoryImpl{})
}
