package unit

import (
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/sitemap"
	"github.com/stretchr/testify/assert"
)

// Sitemap XML structure for testing
type SitemapXML struct {
	XMLName xml.Name `xml:"urlset"`
	Xmlns   string   `xml:"xmlns,attr"`
	URLs    []URL    `xml:"url"`
}

type URL struct {
	Loc        string  `xml:"loc"`
	LastMod    string  `xml:"lastmod,omitempty"`
	ChangeFreq string  `xml:"changefreq,omitempty"`
	Priority   float64 `xml:"priority,omitempty"`
}

func TestSitemapGeneration(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create a gin router with some test routes
	router := gin.New()
	router.GET("/", func(c *gin.Context) {})
	router.GET("/about", func(c *gin.Context) {})
	router.GET("/contact", func(c *gin.Context) {})
	router.GET("/login", func(c *gin.Context) {})
	router.GET("/register", func(c *gin.Context) {})
	router.POST("/login", func(c *gin.Context) {}) // This shouldn't be in sitemap

	// Create the sitemap generator with the router
	host := "https://example.com"
	generator := sitemap.NewGenerator(router, host)

	// Generate the sitemap
	sitemap := generator.Generate()

	// Create a request to test the handler
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/sitemap.xml", nil)

	// Setup the handler
	sitemapHandler := sitemap.GetSitemapHandler()

	// Serve the request using the handler
	sitemapHandler(w, req)

	// Check response code
	assert.Equal(t, http.StatusOK, w.Code)

	// Parse the XML response
	var sitemapXML SitemapXML
	err := xml.Unmarshal(w.Body.Bytes(), &sitemapXML)
	assert.NoError(t, err)

	// Check the sitemap structure
	assert.Equal(t, "http://www.sitemaps.org/schemas/sitemap/0.9", sitemapXML.Xmlns)

	// We should only have GET routes in the sitemap (not POST, PUT, DELETE)
	// And we should exclude admin routes
	expectedURLs := []string{
		host + "/",
		host + "/about",
		host + "/contact",
		host + "/login",
		host + "/register",
	}

	// Check that all expected URLs are in the sitemap
	urlMap := make(map[string]bool)
	for _, url := range sitemapXML.URLs {
		urlMap[url.Loc] = true
	}

	for _, expectedURL := range expectedURLs {
		assert.True(t, urlMap[expectedURL], "Expected URL %s not found in sitemap", expectedURL)
	}

	// Check that we don't have POST routes in the sitemap
	assert.Len(t, sitemapXML.URLs, len(expectedURLs))
}

func TestSitemapExclusions(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create a gin router with some test routes including admin routes
	router := gin.New()
	router.GET("/", func(c *gin.Context) {})
	router.GET("/about", func(c *gin.Context) {})
	router.GET("/admin", func(c *gin.Context) {})
	router.GET("/admin/users", func(c *gin.Context) {})

	// Create the sitemap generator with the router and exclusions
	host := "https://example.com"
	generator := sitemap.NewGenerator(router, host)

	// Add exclusion patterns
	generator.ExcludePattern("^/admin")

	// Generate the sitemap
	sitemap := generator.Generate()

	// Create a request to test the handler
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/sitemap.xml", nil)

	// Setup the handler
	sitemapHandler := sitemap.GetSitemapHandler()

	// Serve the request using the handler
	sitemapHandler(w, req)

	// Parse the XML response
	var sitemapXML SitemapXML
	err := xml.Unmarshal(w.Body.Bytes(), &sitemapXML)
	assert.NoError(t, err)

	// Check that admin routes are not included
	for _, url := range sitemapXML.URLs {
		assert.NotContains(t, url.Loc, "/admin", "Admin URL %s found in sitemap", url.Loc)
	}

	// We should only have / and /about in the sitemap
	assert.Len(t, sitemapXML.URLs, 2)
}

func TestAddCustomURL(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create a gin router with some test routes
	router := gin.New()

	// Create the sitemap generator with the router
	host := "https://example.com"
	generator := sitemap.NewGenerator(router, host)

	// Add a custom URL
	generator.AddURL("/custom-page", sitemap.URLOptions{
		ChangeFreq: "weekly",
		Priority:   0.8,
		LastMod:    "2023-01-01",
	})

	// Generate the sitemap
	sitemap := generator.Generate()

	// Create a request to test the handler
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/sitemap.xml", nil)

	// Setup the handler
	sitemapHandler := sitemap.GetSitemapHandler()

	// Serve the request using the handler
	sitemapHandler(w, req)

	// Parse the XML response
	var sitemapXML SitemapXML
	err := xml.Unmarshal(w.Body.Bytes(), &sitemapXML)
	assert.NoError(t, err)

	// Find the custom URL
	var customURL *URL
	for i, url := range sitemapXML.URLs {
		if url.Loc == host+"/custom-page" {
			customURL = &sitemapXML.URLs[i]
			break
		}
	}

	// Verify the custom URL exists with the right properties
	assert.NotNil(t, customURL, "Custom URL not found in sitemap")
	if customURL != nil {
		assert.Equal(t, "weekly", customURL.ChangeFreq)
		assert.Equal(t, 0.8, customURL.Priority)
		assert.Equal(t, "2023-01-01", customURL.LastMod)
	}
}
