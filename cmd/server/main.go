package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"connect-four-backend/internal/config"
	"connect-four-backend/internal/database"
	"connect-four-backend/internal/game"
	"connect-four-backend/internal/handlers"
	"connect-four-backend/internal/kafka"
	"connect-four-backend/internal/matchmaking"
	"connect-four-backend/internal/server"

	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	cfg := config.Load()

	// Initialize database
	db, err := database.NewPostgresDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Initialize Kafka producer
	kafkaConfig := kafka.DefaultProducerConfig(cfg.KafkaBrokers)
	kafkaProducer, err := kafka.NewProducer(kafkaConfig)
	if err != nil {
		log.Fatal("Failed to create Kafka producer:", err)
	}
	defer kafkaProducer.Close()

	// Initialize services
	gameManager := game.NewManager()
	matchmaker := matchmaking.NewMatchmaker(gameManager)
	analyticsService := kafka.NewAnalyticsService(kafkaProducer, true)

	// Initialize handlers
	gameHandler := handlers.NewGameHandler(gameManager, matchmaker, analyticsService)
	leaderboardHandler := handlers.NewLeaderboardHandler(db)

	// Initialize server
	srv := server.NewServer(cfg, gameHandler, leaderboardHandler)

	// Start matchmaker
	go matchmaker.Start()

	// Start server
	go func() {
		log.Printf("Server starting on port %s", cfg.Port)
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Server failed to start:", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exited")
}