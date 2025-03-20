package server

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	_ "github.com/joho/godotenv/autoload"

	"github.com/hail2skins/armory/internal/database"
	"github.com/hail2skins/armory/internal/logger"
	"github.com/hail2skins/armory/internal/middleware"
	"github.com/hail2skins/armory/internal/services"
	"github.com/hail2skins/armory/internal/services/stripe"
)

type Server struct {
	port int

	db              database.Service
	casbinAuth      *middleware.CasbinAuth
	ipFilterService stripe.IPFilterService
	ipFilterStop    chan struct{} // Channel to stop the IP filter background refresh
}

// New creates a new server
func New() *Server {
	// Parse the port from environment variables
	portStr := os.Getenv("PORT")
	if portStr == "" {
		portStr = "8080" // Default port
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		logger.Warn("Invalid PORT env var, using default 8080", map[string]interface{}{
			"error": err.Error(),
		})
		port = 8080
	}

	// Create the database service
	dbService := database.New()

	// Create the IP filter service
	ipFilterService := stripe.NewIPFilterService(nil) // Use default HTTP client

	// Create the stop channel for background refresh
	ipFilterStop := make(chan struct{})

	// Create the server
	server := &Server{
		port:            port,
		db:              dbService,
		ipFilterService: ipFilterService,
		ipFilterStop:    ipFilterStop,
	}

	return server
}

// Start initializes and starts the server
func (s *Server) Start() error {
	logger.Info("Server Start method called", nil)

	if s.db == nil {
		logger.Info("No database service provided, creating a new one...", nil)
		s.db = database.New()
	}

	// Start the IP filter background refresh
	if s.ipFilterService != nil {
		logger.Info("Starting Stripe IP filter background refresh", nil)
		s.ipFilterService.StartBackgroundRefresh(s.ipFilterStop)
	}

	// Set up routes
	logger.Info("Setting up routes", nil)
	handler := s.RegisterRoutes()

	// Start the server
	addr := fmt.Sprintf(":%d", s.port)
	logger.Info("Starting server on "+addr, nil)

	// Use the handler from RegisterRoutes
	return http.ListenAndServe(addr, handler)
}

// Shutdown performs graceful shutdown operations
func (s *Server) Shutdown() {
	logger.Info("Shutting down server...", nil)

	// Stop the IP filter background refresh
	if s.ipFilterStop != nil {
		logger.Info("Stopping IP filter background refresh", nil)
		close(s.ipFilterStop)

		// Give it a moment to clean up
		time.Sleep(100 * time.Millisecond)
	}

	logger.Info("Server shutdown complete", nil)
}

// createPromotionService creates a new promotion service
func (s *Server) createPromotionService() *services.PromotionService {
	return services.NewPromotionService(s.db)
}
