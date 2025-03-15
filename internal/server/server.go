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
)

type Server struct {
	port int

	db database.Service
}

func NewServer() *http.Server {
	// Initialize logger
	logPath := os.Getenv("LOG_FILE")
	if logPath == "" {
		logPath = "logs/app.log"
	}

	// Ensure logs directory exists
	os.MkdirAll("logs", 0755)

	// Setup file logging
	err := logger.SetupFileLogging(logPath)
	if err != nil {
		logger.Error("Failed to set up file logging", err, nil)
		// Continue with console logging
	}

	logger.Info("Initializing server", nil)

	port, _ := strconv.Atoi(os.Getenv("PORT"))
	NewServer := &Server{
		port: port,

		db: database.New(),
	}

	// Declare Server config
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", NewServer.port),
		Handler:      NewServer.RegisterRoutes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	logger.Info("Server initialized", map[string]interface{}{
		"port": port,
	})

	return server
}
