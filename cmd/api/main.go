package main

import (
	"context"
	"fmt"
	"net/http"
	"os/signal"
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
	// Initialize the server (which also sets up logging)
	server := server.NewServer()

	// Ensure logger is reset when the application exits
	defer logger.ResetLogging()

	// Create a done channel to signal when the shutdown is complete
	done := make(chan bool, 1)

	// Run graceful shutdown in a separate goroutine
	go gracefulShutdown(server, done)

	logger.Info("Starting server", nil)
	err := server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		logger.Error("HTTP server error", err, nil)
		panic(fmt.Sprintf("http server error: %s", err))
	}

	// Wait for the graceful shutdown to complete
	<-done
	logger.Info("Graceful shutdown complete", nil)
}
