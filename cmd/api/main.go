package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/spf13/viper"

	"sonet/internal/adapters"
	"sonet/internal/api"
	"sonet/internal/config"
)

func main() {
	// Initialize configuration
	if err := config.LoadConfig(); err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize the app
	app := fiber.New(fiber.Config{
		AppName:      "Sonet API",
		ErrorHandler: api.ErrorHandler,
	})

	// Middleware
	app.Use(logger.New())
	app.Use(recover.New())
	app.Use(cors.New())
	app.Use(api.RateLimiterMiddleware())

	// Initialize the database adapter
	dbAdapter, err := adapters.NewDatabaseAdapter()
	if err != nil {
		log.Printf("Failed to initialize database adapter: %v", err)
		log.Printf("Check your DB_ADAPTER and DB_CONNECTION_STRING configuration")
		log.Fatalf("Exiting due to database initialization failure")
	}

	// Initialize API routes
	api.SetupRoutes(app, dbAdapter)

	// Start the server
	port := viper.GetString("PORT")
	if port == "" {
		port = "8080"
	}

	// Handle graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		fmt.Println("Gracefully shutting down...")
		_ = app.Shutdown()
	}()

	// Start the server
	log.Printf("Starting Sonet on :%s", port)
	if err := app.Listen(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
