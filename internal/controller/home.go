package controller

import (
	"fmt"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/cmd/web/views/data"
	"github.com/hail2skins/armory/cmd/web/views/home"
	"github.com/hail2skins/armory/internal/database"
	"github.com/hail2skins/armory/internal/logger"
	"github.com/hail2skins/armory/internal/models"
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

	// Print debug info about active promotion
	if promotion, exists := c.Get("active_promotion"); exists {
		if promo, ok := promotion.(*models.Promotion); ok {
			logger.Info("Found active promotion in HomeHandler", map[string]interface{}{
				"name":        promo.Name,
				"benefitDays": promo.BenefitDays,
				"type":        promo.Type,
				"active":      promo.Active,
			})
		} else {
			logger.Warn("Active promotion is not a *models.Promotion", map[string]interface{}{
				"type": fmt.Sprintf("%T", promotion),
			})
		}
	} else {
		logger.Warn("No active promotion in context in HomeHandler", nil)
	}

	// Try to get auth data from context first
	if authDataInterface, exists := c.Get("authData"); exists {
		if data, ok := authDataInterface.(data.AuthData); ok {
			// Use the auth data that already has roles
			homeData.AuthData = data.WithTitle("Home")

			// Log authData for debugging
			logger.Info("HomeHandler authData", map[string]interface{}{
				"email":              homeData.AuthData.Email,
				"authenticated":      homeData.AuthData.Authenticated,
				"roles":              homeData.AuthData.Roles,
				"isCasbinAdmin":      homeData.AuthData.IsCasbinAdmin,
				"hasAdminRole":       homeData.AuthData.HasRole("admin"),
				"hasActivePromotion": homeData.AuthData.ActivePromotion != nil,
			})

			// Check for flash messages directly from session
			session := sessions.Default(c)
			if flashes := session.Flashes(); len(flashes) > 0 {
				session.Save()
				if msg, ok := flashes[0].(string); ok {
					homeData.AuthData = homeData.AuthData.WithSuccess(msg)
				}
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
						homeData.AuthData = homeData.AuthData.WithRoles(roles)
					}
				}
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

		// Check for flash messages directly from session
		session := sessions.Default(c)
		if flashes := session.Flashes(); len(flashes) > 0 {
			session.Save()
			if msg, ok := flashes[0].(string); ok {
				authData = authData.WithSuccess(msg)
			}
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
	var aboutData home.AboutData

	// Try to get auth data from context first
	if authDataInterface, exists := c.Get("authData"); exists {
		if data, ok := authDataInterface.(data.AuthData); ok {
			// Use the auth data that already has roles
			aboutData.AuthData = data.WithTitle("About")

			// Re-fetch roles from Casbin to ensure they're fresh
			if data.Authenticated && data.Email != "" {
				// Get Casbin from context
				if casbinAuth, exists := c.Get("casbinAuth"); exists && casbinAuth != nil {
					if ca, ok := casbinAuth.(interface{ GetUserRoles(string) []string }); ok {
						roles := ca.GetUserRoles(data.Email)
						aboutData.AuthData = aboutData.AuthData.WithRoles(roles)

						// Log roles for debugging
						logger.Info("About page - Casbin roles for user", map[string]interface{}{
							"email":   data.Email,
							"roles":   roles,
							"isAdmin": aboutData.AuthData.IsCasbinAdmin,
						})
					}
				}
			}

			// Check for flash messages directly from session
			session := sessions.Default(c)
			if flashes := session.Flashes(); len(flashes) > 0 {
				session.Save()
				if msg, ok := flashes[0].(string); ok {
					aboutData.AuthData = aboutData.AuthData.WithSuccess(msg)
				}
			}
		}
	}

	// If we couldn't get auth data from context, create a new one (fallback)
	if aboutData.AuthData.Title == "" {
		// Get the current user's authentication status and email
		userInfo, authenticated := c.MustGet("auth").(AuthService).GetCurrentUser(c)

		// Create auth data
		authData := data.NewAuthData()
		authData.Title = "About"
		authData.Authenticated = authenticated

		// Check for flash messages directly from session
		session := sessions.Default(c)
		if flashes := session.Flashes(); len(flashes) > 0 {
			session.Save()
			if msg, ok := flashes[0].(string); ok {
				authData = authData.WithSuccess(msg)
			}
		}

		// Set email if authenticated
		if authenticated {
			authData = authData.WithEmail(userInfo.GetUserName())

			// Get Casbin from context
			if casbinAuth, exists := c.Get("casbinAuth"); exists && casbinAuth != nil {
				if ca, ok := casbinAuth.(interface{ GetUserRoles(string) []string }); ok {
					roles := ca.GetUserRoles(userInfo.GetUserName())
					authData = authData.WithRoles(roles)

					// Log roles for debugging
					logger.Info("About page - Casbin roles for user (fallback)", map[string]interface{}{
						"email":   userInfo.GetUserName(),
						"roles":   roles,
						"isAdmin": authData.IsCasbinAdmin,
					})
				}
			}
		}

		aboutData.AuthData = authData
	}

	// Render the about page with the data
	home.About(aboutData).Render(c.Request.Context(), c.Writer)
}

// ContactHandler handles the contact page route
func (h *HomeController) ContactHandler(c *gin.Context) {
	// Get the authData from context that already contains roles
	var contactData home.ContactData

	// Try to get auth data from context first
	if authDataInterface, exists := c.Get("authData"); exists {
		if data, ok := authDataInterface.(data.AuthData); ok {
			// Use the auth data that already has roles
			contactData.AuthData = data.WithTitle("Contact")

			// Re-fetch roles from Casbin to ensure they're fresh
			if data.Authenticated && data.Email != "" {
				// Get Casbin from context
				if casbinAuth, exists := c.Get("casbinAuth"); exists && casbinAuth != nil {
					if ca, ok := casbinAuth.(interface{ GetUserRoles(string) []string }); ok {
						roles := ca.GetUserRoles(data.Email)
						contactData.AuthData = contactData.AuthData.WithRoles(roles)

						// Log roles for debugging
						logger.Info("Contact page - Casbin roles for user", map[string]interface{}{
							"email":   data.Email,
							"roles":   roles,
							"isAdmin": contactData.AuthData.IsCasbinAdmin,
						})
					}
				}
			}

			// Check for flash messages directly from session
			session := sessions.Default(c)
			if flashes := session.Flashes(); len(flashes) > 0 {
				session.Save()
				if msg, ok := flashes[0].(string); ok {
					contactData.AuthData = contactData.AuthData.WithSuccess(msg)
				}
			}
		}
	}

	// If we couldn't get auth data from context, create a new one (fallback)
	if contactData.AuthData.Title == "" {
		// Get the current user's authentication status and email
		userInfo, authenticated := c.MustGet("auth").(AuthService).GetCurrentUser(c)

		// Create auth data
		authData := data.NewAuthData()
		authData.Title = "Contact"
		authData.Authenticated = authenticated

		// Check for flash messages directly from session
		session := sessions.Default(c)
		if flashes := session.Flashes(); len(flashes) > 0 {
			session.Save()
			if msg, ok := flashes[0].(string); ok {
				authData = authData.WithSuccess(msg)
			}
		}

		// Set email if authenticated
		if authenticated {
			authData = authData.WithEmail(userInfo.GetUserName())

			// Get Casbin from context
			if casbinAuth, exists := c.Get("casbinAuth"); exists && casbinAuth != nil {
				if ca, ok := casbinAuth.(interface{ GetUserRoles(string) []string }); ok {
					roles := ca.GetUserRoles(userInfo.GetUserName())
					authData = authData.WithRoles(roles)

					// Log roles for debugging
					logger.Info("Contact page - Casbin roles for user (fallback)", map[string]interface{}{
						"email":   userInfo.GetUserName(),
						"roles":   roles,
						"isAdmin": authData.IsCasbinAdmin,
					})
				}
			}
		}

		contactData.AuthData = authData
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
