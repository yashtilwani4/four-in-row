package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"connect-four-backend/internal/database"
	"connect-four-backend/internal/kafka"
)

func main() {
	// Command line flags
	var (
		brokers    = flag.String("brokers", getEnv("KAFKA_BROKERS", "localhost:9092"), "Kafka broker addresses")
		topic      = flag.String("topic", getEnv("KAFKA_TOPIC", "connect-four-events"), "Kafka topic to consume")
		groupID    = flag.String("group", getEnv("KAFKA_GROUP_ID", "analytics-processor"), "Kafka consumer group ID")
		dbURL      = flag.String("db", getEnv("DATABASE_URL", "postgres://user:password@localhost/connect_four?sslmode=disable"), "Database URL")
		logLevel   = flag.String("log-level", getEnv("LOG_LEVEL", "info"), "Log level (debug, info, warn, error)")
	)
	flag.Parse()

	log.Printf("Starting Connect Four Analytics Consumer")
	log.Printf("Brokers: %s", *brokers)
	log.Printf("Topic: %s", *topic)
	log.Printf("Group ID: %s", *groupID)
	log.Printf("Log Level: %s", *logLevel)

	// Setup database connection
	repo, err := database.NewRepository(*dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer repo.Close()

	// Test database connection
	if err := repo.HealthCheck(); err != nil {
		log.Fatalf("Database health check failed: %v", err)
	}
	log.Printf("✓ Database connection established")

	// Setup Kafka consumer
	brokerList := strings.Split(*brokers, ",")
	config := kafka.DefaultConsumerConfig(brokerList)
	config.Topic = *topic
	config.GroupID = *groupID

	consumer, err := kafka.NewConsumer(config, repo)
	if err != nil {
		log.Fatalf("Failed to create Kafka consumer: %v", err)
	}

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start consumer
	if err := consumer.Start(ctx); err != nil {
		log.Fatalf("Failed to start consumer: %v", err)
	}
	log.Printf("✓ Analytics consumer started successfully")

	// Start metrics API server (optional)
	metricsServer := NewMetricsServer(consumer, ":8082")
	go func() {
		if err := metricsServer.Start(); err != nil {
			log.Printf("Metrics server error: %v", err)
		}
	}()
	log.Printf("✓ Metrics API server started on :8082")

	// Wait for shutdown signal
	<-sigChan
	log.Printf("Shutdown signal received, stopping consumer...")

	// Stop metrics server
	if err := metricsServer.Stop(); err != nil {
		log.Printf("Error stopping metrics server: %v", err)
	}

	// Stop consumer with timeout
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer stopCancel()

	done := make(chan error, 1)
	go func() {
		done <- consumer.Stop()
	}()

	select {
	case err := <-done:
		if err != nil {
			log.Printf("Error stopping consumer: %v", err)
		} else {
			log.Printf("✓ Consumer stopped successfully")
		}
	case <-stopCtx.Done():
		log.Printf("⚠ Consumer stop timeout, forcing shutdown")
	}

	log.Printf("Analytics consumer shutdown complete")
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}