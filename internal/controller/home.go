package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/cmd/web/views/data"
	"github.com/hail2skins/armory/cmd/web/views/home"
	"github.com/hail2skins/armory/internal/database"
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
	// Get the current user's authentication status and email
	userInfo, authenticated := c.MustGet("auth").(AuthService).GetCurrentUser(c)

	// Create HomeData with proper AuthData
	authData := data.NewAuthData()
	authData.Authenticated = authenticated
	authData.Title = "Home"

	homeData := home.HomeData{
		AuthData: authData,
	}

	// Set email if authenticated
	if authenticated {
		homeData.Email = userInfo.GetUserName()
	}

	// Render the home page with the data
	home.Home(homeData).Render(c.Request.Context(), c.Writer)
}

// AboutHandler handles the about page route
func (h *HomeController) AboutHandler(c *gin.Context) {
	// Get the current user's authentication status and email
	userInfo, authenticated := c.MustGet("auth").(AuthService).GetCurrentUser(c)

	// Create AuthData
	authData := data.NewAuthData()
	authData.Authenticated = authenticated
	authData.Title = "About"

	// Create AboutData with the AuthData
	aboutData := home.AboutData{
		AuthData: authData,
	}

	// Set email if authenticated
	if authenticated {
		aboutData.Email = userInfo.GetUserName()
	}

	// Render the about page with the data
	home.About(aboutData).Render(c.Request.Context(), c.Writer)
}

// ContactHandler handles the contact page route
func (h *HomeController) ContactHandler(c *gin.Context) {
	// Get the current user's authentication status and email
	userInfo, authenticated := c.MustGet("auth").(AuthService).GetCurrentUser(c)

	// Create AuthData
	authData := data.NewAuthData()
	authData.Authenticated = authenticated
	authData.Title = "Contact"

	// Create ContactData with the AuthData
	contactData := home.ContactData{
		AuthData: authData,
	}

	// Set email if authenticated
	if authenticated {
		contactData.Email = userInfo.GetUserName()
	}

	// Check if this is a POST request
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
