package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/hail2skins/armory/internal/logger"
	"github.com/hail2skins/armory/internal/server"
)

func gracefulShutdown(apiServer *http.Server, done chan bool) {
	// Create context that listens for the interrupt signal from the OS.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Listen for the interrupt signal.
	<-ctx.Done()

	logger.Info("Shutting down gracefully, press Ctrl+C again to force", nil)

	// The context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := apiServer.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", err, nil)
	}

	logger.Info("Server exiting", nil)

	// Notify the main goroutine that the shutdown is complete
	done <- true
}

func main() {
	// Configure logger based on environment
	env := os.Getenv("APP_ENV")
	switch strings.ToLower(env) {
	case "production":
		logger.Configure(logger.Production)
	case "staging":
		logger.Configure(logger.Staging)
	default:
		logger.Configure(logger.Development)
	}

	// Initialize the server (which also sets up logging)
	s := server.New()

	// Ensure logger is reset when the application exits
	defer logger.ResetLogging()

	// Start the server in a separate goroutine
	logger.Info("Starting server", map[string]interface{}{
		"environment": env,
	})

	err := s.Start()
	if err != nil {
		logger.Error("Server error", err, nil)
		panic(fmt.Sprintf("server error: %s", err))
	}

	// Create a channel to listen for OS signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	// Wait for interrupt signal
	<-stop

	logger.Info("Shutting down gracefully", nil)

	// Perform any cleanup needed
	// (e.g., database connections, etc.)

	logger.Info("Graceful shutdown complete", nil)
}
