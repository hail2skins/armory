package mocks

import (
	"github.com/gin-gonic/gin"
	"github.com/shaj13/go-guardian/v2/auth"
	"github.com/stretchr/testify/mock"
)

// MockAuthInfo implements the auth.Info interface for testing
type MockAuthInfo struct {
	username   string
	id         string
	groups     []string
	extensions auth.Extensions
}

func (m *MockAuthInfo) GetUserName() string {
	return m.username
}

func (m *MockAuthInfo) SetUserName(username string) {
	m.username = username
}

func (m *MockAuthInfo) GetID() string {
	if m.id == "" {
		return "1"
	}
	return m.id
}

func (m *MockAuthInfo) SetID(id string) {
	m.id = id
}

func (m *MockAuthInfo) GetGroups() []string {
	return m.groups
}

func (m *MockAuthInfo) SetGroups(groups []string) {
	m.groups = groups
}

func (m *MockAuthInfo) GetExtensions() auth.Extensions {
	if m.extensions == nil {
		return auth.Extensions{}
	}
	return m.extensions
}

func (m *MockAuthInfo) SetExtensions(exts auth.Extensions) {
	m.extensions = exts
}

// MockAuthController is a mock of AuthController
type MockAuthController struct {
	mock.Mock
}

// GetCurrentUser mocks the GetCurrentUser method
func (m *MockAuthController) GetCurrentUser(c *gin.Context) (auth.Info, bool) {
	args := m.Called(c)
	if args.Get(0) == nil {
		return nil, args.Bool(1)
	}
	return args.Get(0).(auth.Info), args.Bool(1)
}
