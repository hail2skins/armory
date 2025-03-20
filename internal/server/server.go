package server

import (
	"fmt"
	"net/http"
	"os"
	"strconv"

	_ "github.com/joho/godotenv/autoload"

	"github.com/hail2skins/armory/internal/database"
	"github.com/hail2skins/armory/internal/logger"
	"github.com/hail2skins/armory/internal/middleware"
	"github.com/hail2skins/armory/internal/services"
)

type Server struct {
	port int

	db         database.Service
	casbinAuth *middleware.CasbinAuth
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

	// Create the server
	NewServer := &Server{
		port: port,
		db:   database.New(),
	}

	return NewServer
}

// Start initializes and starts the server
func (s *Server) Start() error {
	logger.Info("Server Start method called", nil)

	if s.db == nil {
		logger.Info("No database service provided, creating a new one...", nil)
		s.db = database.New()
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

// createPromotionService creates a new promotion service
func (s *Server) createPromotionService() *services.PromotionService {
	return services.NewPromotionService(s.db)
}
