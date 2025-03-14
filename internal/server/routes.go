package server

import (
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"io/fs"

	"github.com/hail2skins/armory/cmd/web"
	authviews "github.com/hail2skins/armory/cmd/web/views/auth"
	"github.com/hail2skins/armory/cmd/web/views/data"
	"github.com/hail2skins/armory/internal/controller"
)

// HTMLRender is an interface for HTML renderers
type HTMLRender interface {
	Instance(string, any) Render
}

// Render is an interface for rendering templates
type Render interface {
	Render(http.ResponseWriter) error
	WriteContentType(http.ResponseWriter)
}

func (s *Server) RegisterRoutes() http.Handler {
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"}, // Add your frontend URL
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true, // Enable cookies/auth
	}))

	// Create controllers
	authController := controller.NewAuthController(s.db)
	homeController := controller.NewHomeController(s.db)

	// Set up middleware for auth data
	r.Use(func(c *gin.Context) {
		// Get the current user's authentication status and email
		userInfo, authenticated := authController.GetCurrentUser(c)

		// Create AuthData with authentication status and email
		authData := data.NewAuthData()
		authData.Authenticated = authenticated

		// Set email if authenticated
		if authenticated {
			authData.Email = userInfo.GetUserName()
		}

		// Add authData to context
		c.Set("authData", authData)
		c.Set("authController", authController)

		c.Next()
	})

	// Override the render methods to use our templates
	authController.RenderLogin = func(c *gin.Context, d interface{}) {
		loginData := d.(authviews.LoginData)
		// Set authentication state
		_, authenticated := authController.GetCurrentUser(c)
		// Preserve the Title field
		title := loginData.Title
		if title == "" {
			title = "Login"
		}
		// Create new AuthData with all fields
		authData := data.NewAuthData()
		authData.Authenticated = authenticated
		authData.Title = title
		loginData.AuthData = authData
		authviews.Login(loginData).Render(c.Request.Context(), c.Writer)
	}

	authController.RenderRegister = func(c *gin.Context, d interface{}) {
		registerData := d.(authviews.RegisterData)
		// Set authentication state
		_, authenticated := authController.GetCurrentUser(c)
		// Preserve the Title field
		title := registerData.Title
		if title == "" {
			title = "Register"
		}
		// Create new AuthData with all fields
		authData := data.NewAuthData()
		authData.Authenticated = authenticated
		authData.Title = title
		registerData.AuthData = authData
		authviews.Register(registerData).Render(c.Request.Context(), c.Writer)
	}

	authController.RenderLogout = func(c *gin.Context, d interface{}) {
		logoutData := d.(authviews.LogoutData)
		// Set authentication state - should be false after logout
		// Preserve the Title field
		title := logoutData.Title
		if title == "" {
			title = "Logout"
		}
		// Create new AuthData with all fields
		authData := data.NewAuthData()
		authData.Authenticated = false
		authData.Title = title
		logoutData.AuthData = authData
		authviews.Logout(logoutData).Render(c.Request.Context(), c.Writer)
	}

	// Health check
	r.GET("/health", s.healthHandler)

	// Static files
	staticFiles, _ := fs.Sub(web.Files, "assets")
	r.StaticFS("/assets", http.FS(staticFiles))

	// Home page
	r.GET("/", homeController.HomeHandler)

	// Auth routes
	r.GET("/login", authController.LoginHandler)
	r.POST("/login", authController.LoginHandler)
	r.GET("/register", authController.RegisterHandler)
	r.POST("/register", authController.RegisterHandler)
	r.GET("/logout", authController.LogoutHandler)

	return r
}

// TemplRender is a custom HTML renderer for templ templates
type TemplRender struct {
	Templates map[string]interface{}
}

// Instance returns a renderer for the given template name
func (t *TemplRender) Instance(name string, data any) Render {
	return &TemplInstance{
		Template: t.Templates[name],
		Data:     data,
	}
}

// TemplInstance represents a single template instance
type TemplInstance struct {
	Template interface{}
	Data     interface{}
}

// Render renders the template to the response writer
func (t *TemplInstance) Render(w http.ResponseWriter) error {
	// Different template types have different function signatures
	switch t.Template.(type) {
	case func(map[string]interface{}) interface{}:
		return nil // Handle this case appropriately
	default:
		return nil
	}
}

// WriteContentType writes the content type header
func (t *TemplInstance) WriteContentType(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
}

func (s *Server) healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, s.db.Health())
}
