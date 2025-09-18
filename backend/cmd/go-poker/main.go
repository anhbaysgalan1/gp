package main

import (
	"log/slog"
	"os"

	"github.com/evanofslack/go-poker/internal/server"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		slog.Warn("No .env file found, using environment variables")
	}

	// Create and start poker server
	pokerServer, err := server.NewPokerServer()
	if err != nil {
		slog.Error("Failed to create poker server", "error", err)
		os.Exit(1)
	}

	// Start server (blocks until shutdown)
	if err := pokerServer.Start(); err != nil {
		slog.Error("Failed to start poker server", "error", err)
		os.Exit(1)
	}
}
