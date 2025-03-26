package sitemap

import (
	"encoding/xml"
	"net/http"
	"regexp"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hail2skins/armory/internal/logger"
)

// URLSet is the root element of the sitemap XML
type URLSet struct {
	XMLName xml.Name `xml:"urlset"`
	Xmlns   string   `xml:"xmlns,attr"`
	URLs    []URL    `xml:"url"`
}

// URL represents a single URL entry in the sitemap
type URL struct {
	Loc        string  `xml:"loc"`
	LastMod    string  `xml:"lastmod,omitempty"`
	ChangeFreq string  `xml:"changefreq,omitempty"`
	Priority   float64 `xml:"priority,omitempty"`
}

// URLOptions contains options for a URL entry
type URLOptions struct {
	ChangeFreq string
	Priority   float64
	LastMod    string
}

// Generator creates and manages the sitemap
type Generator struct {
	router          *gin.Engine
	host            string
	excludePatterns []*regexp.Regexp
	customURLs      map[string]URLOptions
	generatedXML    []byte
	lastGenerated   time.Time
}

// NewGenerator creates a new sitemap generator
func NewGenerator(router *gin.Engine, host string) *Generator {
	return &Generator{
		router:          router,
		host:            host,
		excludePatterns: make([]*regexp.Regexp, 0),
		customURLs:      make(map[string]URLOptions),
	}
}

// ExcludePattern adds a regex pattern to exclude paths from the sitemap
func (g *Generator) ExcludePattern(pattern string) {
	regex, err := regexp.Compile(pattern)
	if err != nil {
		logger.Error("Failed to compile sitemap exclusion pattern", err, map[string]interface{}{
			"pattern": pattern,
		})
		return
	}
	g.excludePatterns = append(g.excludePatterns, regex)
}

// AddURL adds a custom URL to the sitemap
func (g *Generator) AddURL(path string, options URLOptions) {
	g.customURLs[path] = options
}

// shouldExcludePath checks if a path should be excluded from the sitemap
func (g *Generator) shouldExcludePath(path string) bool {
	for _, pattern := range g.excludePatterns {
		if pattern.MatchString(path) {
			return true
		}
	}
	return false
}

// Generate creates the sitemap XML
func (g *Generator) Generate() *Generator {
	// Create a new URLSet
	urlSet := URLSet{
		Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs:  make([]URL, 0),
	}

	// Map to track unique URLs
	uniqueURLs := make(map[string]bool)

	// Extract routes from the router
	routes := g.router.Routes()

	// Process routes
	for _, route := range routes {
		// Only include GET routes
		if route.Method != http.MethodGet {
			continue
		}

		// Skip excluded paths
		if g.shouldExcludePath(route.Path) {
			continue
		}

		// Skip routes with path parameters (for now)
		if regexp.MustCompile(`:[^/]+`).MatchString(route.Path) {
			continue
		}

		// Create the full URL
		fullURL := g.host + route.Path

		// Add to unique URLs map if not already added
		if !uniqueURLs[fullURL] {
			urlSet.URLs = append(urlSet.URLs, URL{
				Loc:        fullURL,
				ChangeFreq: "weekly", // Default to weekly
				Priority:   0.5,      // Default priority
			})
			uniqueURLs[fullURL] = true
		}
	}

	// Add custom URLs
	for path, options := range g.customURLs {
		// Skip excluded paths
		if g.shouldExcludePath(path) {
			continue
		}

		fullURL := g.host + path

		// Check if URL already exists
		if uniqueURLs[fullURL] {
			continue
		}

		urlSet.URLs = append(urlSet.URLs, URL{
			Loc:        fullURL,
			LastMod:    options.LastMod,
			ChangeFreq: options.ChangeFreq,
			Priority:   options.Priority,
		})
		uniqueURLs[fullURL] = true
	}

	// Sort URLs for consistency
	sort.Slice(urlSet.URLs, func(i, j int) bool {
		return urlSet.URLs[i].Loc < urlSet.URLs[j].Loc
	})

	// Marshal the URL set to XML
	output, err := xml.MarshalIndent(urlSet, "", "  ")
	if err != nil {
		logger.Error("Failed to marshal sitemap XML", err, nil)
		return g
	}

	// Add XML header
	xmlOutput := []byte(xml.Header + string(output))

	// Store the generated XML and timestamp
	g.generatedXML = xmlOutput
	g.lastGenerated = time.Now()

	return g
}

// GetSitemapHandler returns an HTTP handler that serves the sitemap XML
func (g *Generator) GetSitemapHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check if we need to regenerate
		if g.generatedXML == nil || time.Since(g.lastGenerated) > 1*time.Hour {
			g.Generate()
		}

		// Set content type
		w.Header().Set("Content-Type", "application/xml")
		w.Write(g.generatedXML)
	}
}

// WithDefaultExclusions adds common exclusion patterns
func (g *Generator) WithDefaultExclusions() *Generator {
	// Exclude admin routes
	g.ExcludePattern("^/admin")

	// Exclude auth-related routes
	g.ExcludePattern("^/reset-password")
	g.ExcludePattern("^/verification-")

	// Exclude API routes
	g.ExcludePattern("^/api/")

	// Exclude non-public routes
	g.ExcludePattern("^/owner")

	return g
}

// GetHost returns the host URL used by the generator
func (g *Generator) GetHost() string {
	return g.host
}
