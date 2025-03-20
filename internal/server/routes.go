package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
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

// RegisterRoutes registers all routes for the application
func (s *Server) RegisterRoutes() http.Handler {
	// Create a new Gin router
	r := gin.Default()

	// Create controllers
	authController := controller.NewAuthController(s.db)
	homeController := controller.NewHomeController(s.db)
	paymentController := controller.NewPaymentController(s.db)
	debugController := controller.NewDebugController()

	// Register middleware
	s.RegisterMiddleware(r, authController)

	// Add Stripe IP filter middleware if available
	if s.ipFilterService != nil {
		// Apply the middleware to all routes
		r.Use(s.ipFilterService.Middleware())
	}

	// Register static routes
	s.RegisterStaticRoutes(r)

	// Register API routes - pass the IP filter service
	s.RegisterAPIRoutes(r, s.ipFilterService)

	// Register home routes
	s.RegisterHomeRoutes(r, homeController)

	// Register auth routes
	s.RegisterAuthRoutes(r, authController)

	// Register payment routes
	s.RegisterPaymentRoutes(r, paymentController)

	// Register debug routes (add directly to main router)
	debugController.RegisterRoutes(r.Group(""))

	// Register admin routes
	s.RegisterAdminRoutes(r, authController)

	// Register owner routes
	RegisterOwnerRoutes(r, s.db, authController)

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
