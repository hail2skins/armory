package server

import (
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"io/fs"

	"github.com/hail2skins/armory/cmd/web"
	webviews "github.com/hail2skins/armory/cmd/web/views"
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

	// Create auth controller
	authController := controller.NewAuthController(s.db)

	// Override the render methods to use our templates
	authController.RenderLogin = func(c *gin.Context, data gin.H) {
		webviews.Login(data).Render(c.Request.Context(), c.Writer)
	}

	authController.RenderRegister = func(c *gin.Context, data gin.H) {
		webviews.Register(data).Render(c.Request.Context(), c.Writer)
	}

	// Health check
	r.GET("/health", s.healthHandler)

	// Static files
	staticFiles, _ := fs.Sub(web.Files, "assets")
	r.StaticFS("/assets", http.FS(staticFiles))

	// Home page
	r.GET("/", func(c *gin.Context) {
		// Check if user is authenticated
		userInfo, authenticated := authController.GetCurrentUser(c)

		data := gin.H{
			"authenticated": authenticated,
		}

		if authenticated {
			data["email"] = userInfo.GetUserName()
		}

		webviews.Home(data).Render(c.Request.Context(), c.Writer)
	})

	// Auth routes
	r.GET("/login", authController.LoginHandler)
	r.POST("/login", authController.LoginHandler)
	r.GET("/register", authController.RegisterHandler)
	r.POST("/register", authController.RegisterHandler)
	r.GET("/logout", authController.LogoutHandler)

	// Add auth-links endpoint for HTMX to load the appropriate nav links
	r.GET("/auth-links", func(c *gin.Context) {
		_, authenticated := authController.GetCurrentUser(c)
		c.Header("Content-Type", "text/html")
		webviews.NavAuth(authenticated).Render(c.Request.Context(), c.Writer)
	})

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
