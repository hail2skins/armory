package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// setupTestRouter creates a test router with real authentication handling
func setupTestRouter(t *testing.T) (*gin.Engine, *MockDBWithContext) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create a mock database
	mockDB := new(MockDBWithContext)

	// Setup mock responses
	testUser := &database.User{
		Email: "test@example.com",
	}
	database.SetUserID(testUser, 1)

	testUser2 := &database.User{
		Email: "test2@example.com",
	}
	database.SetUserID(testUser2, 2)

	// GetUserByEmail returns nil for a new user, then returns the user after creation
	mockDB.On("GetUserByEmail", mock.Anything, "test@example.com").Return(nil, nil).Once()
	mockDB.On("GetUserByEmail", mock.Anything, "test@example.com").Return(testUser, nil).Times(3)
	mockDB.On("GetUserByEmail", mock.Anything, "test2@example.com").Return(nil, nil).Once()
	mockDB.On("GetUserByEmail", mock.Anything, "test2@example.com").Return(testUser2, nil).Times(3)

	// CreateUser returns a new user
	mockDB.On("CreateUser", mock.Anything, "test@example.com", "password123").Return(testUser, nil)
	mockDB.On("CreateUser", mock.Anything, "test2@example.com", "password123").Return(testUser2, nil)

	// AuthenticateUser returns the user when credentials are correct
	mockDB.On("AuthenticateUser", mock.Anything, "test@example.com", "password123").Return(testUser, nil)
	mockDB.On("AuthenticateUser", mock.Anything, "test2@example.com", "password123").Return(testUser2, nil)

	// Health check
	mockDB.On("Health").Return(map[string]string{"status": "up"})

	// Create a server with the mock DB
	server := &Server{db: mockDB}

	// Get the router
	router := server.RegisterRoutes().(*gin.Engine)

	return router, mockDB
}

// MockDBWithContext is a mock implementation of the database.Service interface with context
type MockDBWithContext struct {
	mock.Mock
}

func (m *MockDBWithContext) Health() map[string]string {
	args := m.Called()
	return args.Get(0).(map[string]string)
}

func (m *MockDBWithContext) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockDBWithContext) CreateUser(ctx context.Context, email, password string) (*database.User, error) {
	args := m.Called(ctx, email, password)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.User), args.Error(1)
}

func (m *MockDBWithContext) GetUserByEmail(ctx context.Context, email string) (*database.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.User), args.Error(1)
}

func (m *MockDBWithContext) AuthenticateUser(ctx context.Context, email, password string) (*database.User, error) {
	args := m.Called(ctx, email, password)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.User), args.Error(1)
}

func TestAuthenticationFlow(t *testing.T) {
	// Setup
	router, _ := setupTestRouter(t)

	// Test the full authentication flow
	t.Run("Authentication flow changes navigation bar in HTML", func(t *testing.T) {
		// Step 1: Check unauthenticated state - Home page should show Login and Register
		t.Run("Unauthenticated user sees login and register links in HTML", func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			require.Equal(t, http.StatusOK, resp.Code)
			body := resp.Body.String()

			// Verify the actual HTML contains login and register links
			assert.Contains(t, body, `href="/login"`)
			assert.Contains(t, body, `href="/register"`)
			assert.NotContains(t, body, `You are logged in as`)
		})

		// Step 2: Register a new user
		t.Run("User can register and HTML changes to show logout link", func(t *testing.T) {
			// Submit registration form
			form := url.Values{}
			form.Add("email", "test@example.com")
			form.Add("password", "password123")
			form.Add("confirm_password", "password123")

			req := httptest.NewRequest("POST", "/register", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			// Should redirect to home page
			require.Equal(t, http.StatusSeeOther, resp.Code)
			require.Equal(t, "/", resp.Header().Get("Location"))

			// Extract auth cookie
			cookies := resp.Result().Cookies()
			var authCookie *http.Cookie
			for _, cookie := range cookies {
				if cookie.Name == "auth-session" {
					authCookie = cookie
					break
				}
			}
			require.NotNil(t, authCookie, "Auth cookie should be set")

			// Now check the home page with the auth cookie
			req = httptest.NewRequest("GET", "/", nil)
			req.AddCookie(authCookie)
			resp = httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			// Check that the home page shows the logout link
			homeHTML := resp.Body.String()
			assert.Contains(t, homeHTML, `/logout`)
			assert.NotContains(t, homeHTML, `/login`)
			assert.NotContains(t, homeHTML, `/register`)
		})

		// Step 3: Logout
		t.Run("User can logout and HTML changes back to show login/register links", func(t *testing.T) {
			// Logout
			req := httptest.NewRequest("GET", "/logout", nil)
			req.AddCookie(&http.Cookie{
				Name:  "auth-session",
				Value: "1", // User ID from the test user
			})
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			// Should redirect to home page
			require.Equal(t, http.StatusSeeOther, resp.Code)
			require.Equal(t, "/", resp.Header().Get("Location"))

			// Check that auth cookie is cleared
			cookies := resp.Result().Cookies()
			var authCookie *http.Cookie
			for _, cookie := range cookies {
				if cookie.Name == "auth-session" {
					authCookie = cookie
					break
				}
			}
			require.NotNil(t, authCookie, "Auth cookie should be present")
			assert.Equal(t, "", authCookie.Value, "Auth cookie should be cleared")
			assert.Less(t, authCookie.MaxAge, 0, "Auth cookie should be expired")

			// Now check the home page to verify unauthenticated state
			req = httptest.NewRequest("GET", "/", nil)
			resp = httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			// Verify the HTML shows unauthenticated state
			homeHTML := resp.Body.String()
			assert.Contains(t, homeHTML, `/login`)
			assert.Contains(t, homeHTML, `/register`)
			assert.NotContains(t, homeHTML, `/logout`)
		})
	})
}

// TestRealHTMLOutput tests the actual HTML output of the templates with different authentication states
func TestRealHTMLOutput(t *testing.T) {
	// Setup
	router, _ := setupTestRouter(t)

	t.Run("Home page renders correct HTML based on authentication", func(t *testing.T) {
		// Test unauthenticated state
		req := httptest.NewRequest("GET", "/", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		// Verify HTML contains login and register links
		unauthHTML := resp.Body.String()
		assert.Contains(t, unauthHTML, `href="/login"`)
		assert.Contains(t, unauthHTML, `href="/register"`)
		assert.NotContains(t, unauthHTML, `href="/logout"`)

		// Test authenticated state - first register to set up the authentication
		form := url.Values{}
		form.Add("email", "test2@example.com")
		form.Add("password", "password123")
		form.Add("confirm_password", "password123")

		regReq := httptest.NewRequest("POST", "/register", strings.NewReader(form.Encode()))
		regReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		regResp := httptest.NewRecorder()
		router.ServeHTTP(regResp, regReq)

		// Extract auth cookie
		cookies := regResp.Result().Cookies()
		var authCookie *http.Cookie
		for _, cookie := range cookies {
			if cookie.Name == "auth-session" {
				authCookie = cookie
				break
			}
		}
		require.NotNil(t, authCookie, "Auth cookie should be set")

		// Now test the home page with the auth cookie
		authReq := httptest.NewRequest("GET", "/", nil)
		authReq.AddCookie(authCookie)
		authResp := httptest.NewRecorder()
		router.ServeHTTP(authResp, authReq)

		// Verify HTML contains logout link and user email
		authHTML := authResp.Body.String()
		assert.Contains(t, authHTML, `/logout`)
		assert.NotContains(t, authHTML, `/login`)
		assert.NotContains(t, authHTML, `/register`)
	})
}
