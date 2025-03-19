package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/cmd/web/views/data"
	"github.com/hail2skins/armory/cmd/web/views/home"
	"github.com/hail2skins/armory/internal/database"
	"github.com/hail2skins/armory/internal/logger"
	"github.com/hail2skins/armory/internal/services/email"
)

// HomeController handles home page related routes
type HomeController struct {
	db           database.Service
	emailService email.EmailService
}

// NewHomeController creates a new home controller
func NewHomeController(db database.Service) *HomeController {
	emailService, _ := email.NewMailjetService()
	return &HomeController{
		db:           db,
		emailService: emailService,
	}
}

// HomeHandler handles the home page route
func (h *HomeController) HomeHandler(c *gin.Context) {
	// Get the authData from context that already contains roles
	var homeData home.HomeData

	// Try to get auth data from context first
	if authDataInterface, exists := c.Get("authData"); exists {
		if authData, ok := authDataInterface.(data.AuthData); ok {
			// Use the auth data that already has roles
			authData = authData.WithTitle("Home")

			// Check for flash messages
			if flashCookie, err := c.Cookie("flash"); err == nil && flashCookie != "" {
				// Add flash message to success messages
				authData = authData.WithSuccess(flashCookie)
				// Clear the flash cookie
				c.SetCookie("flash", "", -1, "/", "", false, false)
			}

			// Make sure RBAC information is correctly passed
			if c.MustGet("auth").(AuthService).IsAuthenticated(c) {
				// Get user info
				userInfo, _ := c.MustGet("auth").(AuthService).GetCurrentUser(c)

				// Get Casbin directly
				if s, exists := c.Get("casbinAuth"); exists && s != nil {
					casbinAuth, ok := s.(interface{ GetUserRoles(string) []string })
					if ok {
						roles := casbinAuth.GetUserRoles(userInfo.GetUserName())
						// Apply roles directly to authData
						authData = authData.WithRoles(roles)
					}
				}
			}

			homeData = home.HomeData{
				AuthData: authData,
			}
		}
	}

	// If we couldn't get auth data from context, create a new one (fallback)
	if homeData.AuthData.Title == "" {
		// Get the current user's authentication status and email
		userInfo, authenticated := c.MustGet("auth").(AuthService).GetCurrentUser(c)

		// Create HomeData with proper AuthData
		authData := data.NewAuthData()
		authData.Authenticated = authenticated
		authData.Title = "Home"

		// Check for flash messages
		if flashCookie, err := c.Cookie("flash"); err == nil && flashCookie != "" {
			// Add flash message to success messages
			authData.Success = flashCookie
			// Clear the flash cookie
			c.SetCookie("flash", "", -1, "/", "", false, false)
		}

		// Set email if authenticated
		if authenticated {
			authData = authData.WithEmail(userInfo.GetUserName())

			// Get Casbin from context or server
			if casbinAuth, exists := c.Get("casbinAuth"); exists && casbinAuth != nil {
				if ca, ok := casbinAuth.(interface{ GetUserRoles(string) []string }); ok {
					roles := ca.GetUserRoles(userInfo.GetUserName())
					authData = authData.WithRoles(roles)
				}
			}
		}

		homeData = home.HomeData{
			AuthData: authData,
		}
	}

	// Render the home page with the data
	// Direct debug output for authData
	logger.Info("HomeHandler authData", map[string]interface{}{
		"email":         homeData.AuthData.Email,
		"authenticated": homeData.AuthData.Authenticated,
		"roles":         homeData.AuthData.Roles,
		"isCasbinAdmin": homeData.AuthData.IsCasbinAdmin,
		"hasAdminRole":  homeData.AuthData.HasRole("admin"),
	})

	home.Home(homeData).Render(c.Request.Context(), c.Writer)
}

// AboutHandler handles the about page route
func (h *HomeController) AboutHandler(c *gin.Context) {
	// Get the authData from context that already contains roles
	var authData data.AuthData

	// Try to get auth data from context first
	if authDataInterface, exists := c.Get("authData"); exists {
		if data, ok := authDataInterface.(data.AuthData); ok {
			// Use the auth data that already has roles
			authData = data.WithTitle("About")

			// Check for flash messages
			if flashCookie, err := c.Cookie("flash"); err == nil && flashCookie != "" {
				// Add flash message to success messages
				authData = authData.WithSuccess(flashCookie)
				// Clear the flash cookie
				c.SetCookie("flash", "", -1, "/", "", false, false)
			}
		}
	}

	// If we couldn't get auth data from context, create a new one (fallback)
	if authData.Title == "" {
		// Get the current user's authentication status and email
		userInfo, authenticated := c.MustGet("auth").(AuthService).GetCurrentUser(c)

		// Create auth data
		authData = data.NewAuthData()
		authData.Title = "About"
		authData.Authenticated = authenticated

		// Set email if authenticated
		if authenticated {
			authData = authData.WithEmail(userInfo.GetUserName())

			// Get Casbin from context
			if casbinAuth, exists := c.Get("casbinAuth"); exists && casbinAuth != nil {
				if ca, ok := casbinAuth.(interface{ GetUserRoles(string) []string }); ok {
					roles := ca.GetUserRoles(userInfo.GetUserName())
					authData = authData.WithRoles(roles)
				}
			}
		}
	}

	// Render the about page with the data
	aboutData := home.AboutData{
		AuthData: authData,
	}
	home.About(aboutData).Render(c.Request.Context(), c.Writer)
}

// ContactHandler handles the contact page route
func (h *HomeController) ContactHandler(c *gin.Context) {
	// Get the authData from context that already contains roles
	var authData data.AuthData

	// Try to get auth data from context first
	if authDataInterface, exists := c.Get("authData"); exists {
		if data, ok := authDataInterface.(data.AuthData); ok {
			// Use the auth data that already has roles
			authData = data.WithTitle("Contact")

			// Check for flash messages
			if flashCookie, err := c.Cookie("flash"); err == nil && flashCookie != "" {
				// Add flash message to success messages
				authData = authData.WithSuccess(flashCookie)
				// Clear the flash cookie
				c.SetCookie("flash", "", -1, "/", "", false, false)
			}
		}
	}

	// If we couldn't get auth data from context, create a new one (fallback)
	if authData.Title == "" {
		// Get the current user's authentication status and email
		userInfo, authenticated := c.MustGet("auth").(AuthService).GetCurrentUser(c)

		// Create auth data
		authData = data.NewAuthData()
		authData.Title = "Contact"
		authData.Authenticated = authenticated

		// Set email if authenticated
		if authenticated {
			authData = authData.WithEmail(userInfo.GetUserName())

			// Get Casbin from context
			if casbinAuth, exists := c.Get("casbinAuth"); exists && casbinAuth != nil {
				if ca, ok := casbinAuth.(interface{ GetUserRoles(string) []string }); ok {
					roles := ca.GetUserRoles(userInfo.GetUserName())
					authData = authData.WithRoles(roles)
				}
			}
		}
	}

	// Create contact data with auth data
	contactData := home.ContactData{
		AuthData: authData,
	}

	// Check if this is a form submission
	if c.Request.Method == "POST" {
		// Get form data
		name := c.PostForm("name")
		email := c.PostForm("email")
		subject := c.PostForm("subject")
		message := c.PostForm("message")

		// Validate form data
		if name == "" || email == "" || subject == "" || message == "" {
			contactData.Message = "Please fill out all fields."
			contactData.MessageType = "error"
			home.Contact(contactData).Render(c.Request.Context(), c.Writer)
			return
		}

		// Send email
		if h.emailService != nil {
			err := h.emailService.SendContactEmail(name, email, subject, message)
			if err != nil {
				contactData.Message = "There was an error sending your message. Please try again later."
				contactData.MessageType = "error"
			} else {
				contactData.Message = "Thank you for your message. We'll get back to you soon."
				contactData.MessageType = "success"
			}
		} else {
			// Email service not available, but still show success to the user
			contactData.Message = "Thank you for your message. We'll get back to you soon."
			contactData.MessageType = "success"
		}
	}

	// Render the contact page with the data
	home.Contact(contactData).Render(c.Request.Context(), c.Writer)
}

// SetEmailService allows replacing the email service (mainly for testing)
func (h *HomeController) SetEmailService(emailService email.EmailService) {
	h.emailService = emailService
}
