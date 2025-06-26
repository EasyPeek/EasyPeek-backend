package main

import (
	"log"
	"net/http"
	"time"

	"github.com/EasyPeek/EasyPeek-backend/internal/api"
	"github.com/EasyPeek/EasyPeek-backend/internal/config"
	"github.com/EasyPeek/EasyPeek-backend/internal/database"
	"github.com/EasyPeek/EasyPeek-backend/internal/models"
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
	if err := database.Migrate(&models.User{}); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

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

	log.Println("Server is starting on :8080")
	log.Println("API documentation available at: http://localhost:8080/swagger/index.html")
	log.Println("Health check available at: http://localhost:8080/health")

	// start server
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Failed to start server: %v", err)
	}
}
