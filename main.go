package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/schizoidcock/zep-web-interface/internal/config"
	"github.com/schizoidcock/zep-web-interface/internal/server"
)

func main() {
	// Configure logging to use stdout instead of stderr
	log.SetOutput(os.Stdout)
	
	// Load configuration
	cfg := config.Load()

	// Create HTTP server
	srv, err := server.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Start server
	go func() {
		log.Printf("🌐 Zep Web Interface starting on %s:%d", cfg.Host, cfg.Port)
		log.Printf("🔗 Zep API URL: %s", cfg.ZepAPIURL)
		log.Printf("🔧 HOST env var: '%s'", os.Getenv("HOST"))
		log.Printf("🔧 Actual bind address: %s:%d", cfg.Host, cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("✅ Server exited")
}