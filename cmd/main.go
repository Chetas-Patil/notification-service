package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"notification-service/internal/app"
	"notification-service/config"
)

func main() {
	log.Println("Starting Notification Service")

	cfg := config.LoadConfig()
	application := app.NewApp(cfg)
	if err := application.Init(); err != nil {
		log.Fatalf("Failed to initialize app: %v", err)
	}

	serverErr := make(chan error, 1)
	go func() { serverErr <- application.Serve() }()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigChan:
		log.Printf("Received %s, shutting down", sig)
	case err := <-serverErr:
		if err != nil {
			log.Printf("HTTP server stopped: %v", err)
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	application.Shutdown(shutdownCtx)
	log.Println("Service exited")
}
