package controller

import (
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/logger"
	"github.com/hail2skins/armory/internal/sitemap"
)

// SitemapController handles sitemap.xml generation and serving
type SitemapController struct {
	generator *sitemap.Generator
}

// NewSitemapController creates a new sitemap controller with default settings
func NewSitemapController(router *gin.Engine) *SitemapController {
	// Get host name from environment or use default
	host := os.Getenv("SITE_HOST")
	if host == "" {
		host = "https://virtualarmory.co" // Default host
	}

	// Create the sitemap generator
	generator := sitemap.NewGenerator(router, host)

	// Add default exclusions
	generator.WithDefaultExclusions()

	// Add high-priority custom URLs with more specific options
	generator.AddURL("/", sitemap.URLOptions{
		ChangeFreq: "daily",
		Priority:   1.0,
		LastMod:    time.Now().Format("2006-01-02"),
	})

	generator.AddURL("/about", sitemap.URLOptions{
		ChangeFreq: "monthly",
		Priority:   0.8,
		LastMod:    time.Now().Format("2006-01-02"),
	})

	generator.AddURL("/pricing", sitemap.URLOptions{
		ChangeFreq: "weekly",
		Priority:   0.9,
		LastMod:    time.Now().Format("2006-01-02"),
	})

	// Generate the initial sitemap
	generator.Generate()

	return &SitemapController{
		generator: generator,
	}
}

// SitemapHandler serves the sitemap.xml file
func (sc *SitemapController) SitemapHandler(c *gin.Context) {
	handler := sc.generator.GetSitemapHandler()
	handler(c.Writer, c.Request)
}

// RobotsHandler serves the robots.txt file which points to the sitemap
func (sc *SitemapController) RobotsHandler(c *gin.Context) {
	host := sc.generator.GetHost()
	if host == "" {
		host = "https://virtualarmory.co" // Default host
	}

	// Create robots.txt content
	robotsTxt := `User-agent: *
Allow: /

# Disallow admin paths
Disallow: /admin/
Disallow: /wp-admin/

# Sitemap location
Sitemap: ` + host + `/sitemap.xml
`

	c.Header("Content-Type", "text/plain")
	c.String(http.StatusOK, robotsTxt)
}

// RegisterRoutes registers the sitemap-related routes
func (sc *SitemapController) RegisterRoutes(router *gin.Engine) {
	router.GET("/sitemap.xml", sc.SitemapHandler)
	router.GET("/robots.txt", sc.RobotsHandler)

	// Log that routes were registered
	logger.Info("Registered sitemap routes", map[string]interface{}{
		"routes": []string{"/sitemap.xml", "/robots.txt"},
	})
}
