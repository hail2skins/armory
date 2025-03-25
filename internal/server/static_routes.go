package server

import (
	"io/fs"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/cmd/web"
)

// RegisterStaticRoutes registers all static file routes
func (s *Server) RegisterStaticRoutes(r *gin.Engine) {
	// Static files
	staticFiles, _ := fs.Sub(web.Files, "assets")
	r.StaticFS("/assets", http.FS(staticFiles))

	// You can add more static file routes here if needed
	// r.StaticFS("/docs", http.FS(docsFiles))

	// Serve favicon.ico from the embedded assets
	r.StaticFileFS("/favicon.ico", "favicon.ico", http.FS(staticFiles))
}
