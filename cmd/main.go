package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/EasyPeek/EasyPeek-backend/internal/api"
	"github.com/EasyPeek/EasyPeek-backend/internal/config"
	"github.com/EasyPeek/EasyPeek-backend/internal/database"
	"github.com/EasyPeek/EasyPeek-backend/internal/models"
	"github.com/EasyPeek/EasyPeek-backend/internal/scheduler"
	"github.com/EasyPeek/EasyPeek-backend/internal/services"
)

func main() {
	// load config
	cfg, err := config.LoadConfig("internal/config/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// initialize database
	if err := database.Initialize(cfg); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.CloseDatabase()

	// execute database migration
	if err := database.Migrate(
		&models.User{},
		&models.News{},
		&models.Event{},
		&models.RSSSource{},
		&models.Comment{},
		&models.Message{},
		&models.Follow{},
	); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// initialize seed data
	seedService := services.NewSeedService()

	if err := seedService.SeedAllData(); err != nil {
		log.Printf("Warning: Failed to seed initial data: %v", err)
	}

	// initialize RSS scheduler
	rssScheduler := scheduler.NewRSSScheduler()
	if err := rssScheduler.Start(); err != nil {
		log.Fatalf("Failed to start RSS scheduler: %v", err)
	}
	defer rssScheduler.Stop()

	// set up routes
	router := api.SetupRoutes()

	// create http server
	server := &http.Server{
		Addr:         ":8080",
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Println("Shutting down server...")
		rssScheduler.Stop()

		if err := server.Close(); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
	}()

	log.Println("Server is starting on :8080")
	log.Println("RSS scheduler is running")

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Failed to start server: %v", err)
	}

	log.Println("Server stopped")
}
