package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"connect-four-backend/internal/database"

	"github.com/segmentio/kafka-go"
)

// Consumer handles Kafka message consumption and analytics processing
type Consumer struct {
	reader      *kafka.Reader
	processor   *EventProcessor
	stopChan    chan struct{}
	wg          sync.WaitGroup
	isRunning   bool
	mu          sync.RWMutex
	stats       ConsumerStats
}

// ConsumerStats tracks consumer performance metrics
type ConsumerStats struct {
	MessagesProcessed int64     `json:"messages_processed"`
	MessagesErrored   int64     `json:"messages_errored"`
	LastMessageTime   time.Time `json:"last_message_time"`
	LastErrorTime     time.Time `json:"last_error_time"`
	LastError         string    `json:"last_error"`
	StartTime         time.Time `json:"start_time"`
	Uptime            time.Duration `json:"uptime"`
}

// ConsumerConfig holds configuration for the Kafka consumer
type ConsumerConfig struct {
	Brokers       []string      `json:"brokers"`
	Topic         string        `json:"topic"`
	GroupID       string        `json:"group_id"`
	MinBytes      int           `json:"min_bytes"`
	MaxBytes      int           `json:"max_bytes"`
	MaxWait       time.Duration `json:"max_wait"`
	StartOffset   int64         `json:"start_offset"`
	CommitInterval time.Duration `json:"commit_interval"`
}

// DefaultConsumerConfig returns a production-ready consumer configuration
func DefaultConsumerConfig(brokers []string) ConsumerConfig {
	return ConsumerConfig{
		Brokers:        brokers,
		Topic:          "connect-four-events",
		GroupID:        "analytics-processor",
		MinBytes:       10e3,  // 10KB
		MaxBytes:       10e6,  // 10MB
		MaxWait:        1 * time.Second,
		StartOffset:    kafka.LastOffset,
		CommitInterval: 1 * time.Second,
	}
}

// NewConsumer creates a new Kafka consumer with analytics processing
func NewConsumer(config ConsumerConfig, repo *database.Repository) (*Consumer, error) {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        config.Brokers,
		Topic:          config.Topic,
		GroupID:        config.GroupID,
		MinBytes:       config.MinBytes,
		MaxBytes:       config.MaxBytes,
		MaxWait:        config.MaxWait,
		StartOffset:    config.StartOffset,
		CommitInterval: config.CommitInterval,
		ErrorLogger:    kafka.LoggerFunc(log.Printf),
	})

	processor, err := NewEventProcessor(repo)
	if err != nil {
		return nil, fmt.Errorf("failed to create event processor: %w", err)
	}

	consumer := &Consumer{
		reader:    reader,
		processor: processor,
		stopChan:  make(chan struct{}),
		stats: ConsumerStats{
			StartTime: time.Now(),
		},
	}

	return consumer, nil
}

// Start begins consuming messages from Kafka
func (c *Consumer) Start(ctx context.Context) error {
	c.mu.Lock()
	if c.isRunning {
		c.mu.Unlock()
		return fmt.Errorf("consumer is already running")
	}
	c.isRunning = true
	c.mu.Unlock()

	log.Printf("Starting Kafka consumer for topic: %s", c.reader.Config().Topic)

	// Start message processing goroutine
	c.wg.Add(1)
	go c.processMessages(ctx)

	// Start metrics aggregation goroutine
	c.wg.Add(1)
	go c.processor.StartAggregation(ctx, &c.wg)

	// Start periodic statistics reporting
	c.wg.Add(1)
	go c.reportStatistics(ctx)

	return nil
}

// Stop gracefully shuts down the consumer
func (c *Consumer) Stop() error {
	c.mu.Lock()
	if !c.isRunning {
		c.mu.Unlock()
		return nil
	}
	c.isRunning = false
	c.mu.Unlock()

	log.Println("Stopping Kafka consumer...")

	// Signal stop
	close(c.stopChan)

	// Wait for all goroutines to finish
	c.wg.Wait()

	// Close reader
	if err := c.reader.Close(); err != nil {
		return fmt.Errorf("failed to close reader: %w", err)
	}

	// Stop processor
	if err := c.processor.Stop(); err != nil {
		return fmt.Errorf("failed to stop processor: %w", err)
	}

	log.Println("Kafka consumer stopped successfully")
	return nil
}

// GetStats returns current consumer statistics
func (c *Consumer) GetStats() ConsumerStats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	stats := c.stats
	stats.Uptime = time.Since(stats.StartTime)
	return stats
}

// processMessages is the main message processing loop
func (c *Consumer) processMessages(ctx context.Context) {
	defer c.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopChan:
			return
		default:
			// Read message with timeout
			message, err := c.reader.ReadMessage(ctx)
			if err != nil {
				if err == context.Canceled {
					return
				}
				c.updateStats(false, err)
				log.Printf("Error reading message: %v", err)
				continue
			}

			// Process message
			if err := c.processor.ProcessMessage(message); err != nil {
				c.updateStats(false, err)
				log.Printf("Error processing message: %v", err)
			} else {
				c.updateStats(true, nil)
			}
		}
	}
}

// reportStatistics periodically reports consumer statistics
func (c *Consumer) reportStatistics(ctx context.Context) {
	defer c.wg.Done()

	ticker := time.NewTicker(60 * time.Second) // Report every minute
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopChan:
			return
		case <-ticker.C:
			c.logStatistics()
		}
	}
}

// updateStats updates consumer statistics
func (c *Consumer) updateStats(success bool, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if success {
		c.stats.MessagesProcessed++
		c.stats.LastMessageTime = time.Now()
	} else {
		c.stats.MessagesErrored++
		c.stats.LastErrorTime = time.Now()
		if err != nil {
			c.stats.LastError = err.Error()
		}
	}
}

// logStatistics logs current consumer statistics
func (c *Consumer) logStatistics() {
	stats := c.GetStats()
	
	log.Printf("=== Consumer Statistics ===")
	log.Printf("Uptime: %v", stats.Uptime.Round(time.Second))
	log.Printf("Messages Processed: %d", stats.MessagesProcessed)
	log.Printf("Messages Errored: %d", stats.MessagesErrored)
	
	if stats.MessagesProcessed > 0 {
		rate := float64(stats.MessagesProcessed) / stats.Uptime.Seconds()
		log.Printf("Processing Rate: %.2f messages/sec", rate)
	}
	
	if stats.LastError != "" {
		log.Printf("Last Error: %s (at %v)", stats.LastError, stats.LastErrorTime)
	}

	// Get processor statistics
	processorStats := c.processor.GetStats()
	log.Printf("Active Games: %d", processorStats.ActiveGames)
	log.Printf("Total Players: %d", processorStats.TotalPlayers)
	log.Printf("Games Completed Today: %d", processorStats.GamesToday)
	log.Printf("===========================")
}

// EventProcessor handles the processing and aggregation of game events
type EventProcessor struct {
	repo            *database.Repository
	aggregator      *MetricsAggregator
	gameTracker     *GameTracker
	playerTracker   *PlayerTracker
	hourlyTracker   *HourlyTracker
	mu              sync.RWMutex
	stopChan        chan struct{}
	isRunning       bool
}

// ProcessorStats tracks event processor statistics
type ProcessorStats struct {
	ActiveGames   int `json:"active_games"`
	TotalPlayers  int `json:"total_players"`
	GamesToday    int `json:"games_today"`
	GamesThisHour int `json:"games_this_hour"`
}

// NewEventProcessor creates a new event processor
func NewEventProcessor(repo *database.Repository) (*EventProcessor, error) {
	aggregator, err := NewMetricsAggregator(repo)
	if err != nil {
		return nil, fmt.Errorf("failed to create metrics aggregator: %w", err)
	}

	return &EventProcessor{
		repo:          repo,
		aggregator:    aggregator,
		gameTracker:   NewGameTracker(),
		playerTracker: NewPlayerTracker(),
		hourlyTracker: NewHourlyTracker(),
		stopChan:      make(chan struct{}),
	}, nil
}

// StartAggregation starts the metrics aggregation process
func (ep *EventProcessor) StartAggregation(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	ep.mu.Lock()
	ep.isRunning = true
	ep.mu.Unlock()

	// Start aggregation ticker (every 5 minutes)
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ep.stopChan:
			return
		case <-ticker.C:
			if err := ep.aggregator.AggregateMetrics(); err != nil {
				log.Printf("Error aggregating metrics: %v", err)
			}
		}
	}
}

// Stop stops the event processor
func (ep *EventProcessor) Stop() error {
	ep.mu.Lock()
	if !ep.isRunning {
		ep.mu.Unlock()
		return nil
	}
	ep.isRunning = false
	ep.mu.Unlock()

	close(ep.stopChan)
	return ep.aggregator.Flush()
}

// ProcessMessage processes a single Kafka message
func (ep *EventProcessor) ProcessMessage(message kafka.Message) error {
	// Log the raw event
	log.Printf("Processing event: %s", string(message.Key))

	// Parse base event to determine type
	var baseEvent BaseEvent
	if err := json.Unmarshal(message.Value, &baseEvent); err != nil {
		return fmt.Errorf("failed to parse base event: %w", err)
	}

	// Process based on event type
	switch baseEvent.EventType {
	case EventGameStarted:
		return ep.processGameStarted(message.Value)
	case EventMovePlayed:
		return ep.processMovePlayed(message.Value)
	case EventGameEnded:
		return ep.processGameEnded(message.Value)
	case EventPlayerDisconnected:
		return ep.processPlayerDisconnected(message.Value)
	case EventPlayerReconnected:
		return ep.processPlayerReconnected(message.Value)
	default:
		log.Printf("Unknown event type: %s", baseEvent.EventType)
		return nil
	}
}

// GetStats returns current processor statistics
func (ep *EventProcessor) GetStats() ProcessorStats {
	ep.mu.RLock()
	defer ep.mu.RUnlock()

	return ProcessorStats{
		ActiveGames:   ep.gameTracker.GetActiveGameCount(),
		TotalPlayers:  ep.playerTracker.GetPlayerCount(),
		GamesToday:    ep.hourlyTracker.GetGamesToday(),
		GamesThisHour: ep.hourlyTracker.GetGamesThisHour(),
	}
}

// Event processing methods

func (ep *EventProcessor) processGameStarted(data []byte) error {
	var event GameStartedEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return err
	}

	log.Printf("Game Started: %s with players %v", event.GameID, getPlayerNames(event.Players))

	// Track game
	ep.gameTracker.StartGame(event.GameID, event.Players, event.Timestamp)

	// Track players
	for _, player := range event.Players {
		ep.playerTracker.TrackPlayer(player.Name, event.Timestamp)
	}

	// Track hourly metrics
	ep.hourlyTracker.RecordGameStart(event.Timestamp)

	// Update aggregated metrics
	return ep.aggregator.RecordGameStart(event)
}

func (ep *EventProcessor) processMovePlayed(data []byte) error {
	var event MovePlayedEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return err
	}

	log.Printf("Move Played: Game %s, Player %s, Column %d", 
		event.GameID, event.Player.Name, event.Column)

	// Track move
	ep.gameTracker.RecordMove(event.GameID, event.Player.Name, event.Timestamp)
	ep.playerTracker.RecordMove(event.Player.Name, event.Timestamp)

	// Update aggregated metrics
	return ep.aggregator.RecordMove(event)
}

func (ep *EventProcessor) processGameEnded(data []byte) error {
	var event GameEndedEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return err
	}

	winnerName := ""
	if event.Winner != nil {
		winnerName = event.Winner.Name
	}

	log.Printf("Game Ended: %s, Winner: %s, Duration: %ds", 
		event.GameID, winnerName, event.Duration)

	// Track game completion
	ep.gameTracker.EndGame(event.GameID, winnerName, event.Duration, event.Timestamp)

	// Track players
	for _, player := range event.Players {
		isWinner := event.Winner != nil && event.Winner.Name == player.Name
		ep.playerTracker.RecordGameEnd(player.Name, isWinner, event.IsDraw, event.Duration, event.Timestamp)
	}

	// Track hourly metrics
	ep.hourlyTracker.RecordGameEnd(event.Timestamp, event.Duration)

	// Update aggregated metrics
	return ep.aggregator.RecordGameEnd(event)
}

func (ep *EventProcessor) processPlayerDisconnected(data []byte) error {
	var event PlayerDisconnectedEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return err
	}

	log.Printf("Player Disconnected: %s from game %s", event.Player.Name, event.GameID)

	// Track disconnection
	ep.playerTracker.RecordDisconnection(event.Player.Name, event.Timestamp)

	// Update aggregated metrics
	return ep.aggregator.RecordDisconnection(event)
}

func (ep *EventProcessor) processPlayerReconnected(data []byte) error {
	var event PlayerReconnectedEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return err
	}

	log.Printf("Player Reconnected: %s to game %s after %v", 
		event.Player.Name, event.GameID, event.OfflineDuration)

	// Track reconnection
	ep.playerTracker.RecordReconnection(event.Player.Name, event.OfflineDuration, event.Timestamp)

	// Update aggregated metrics
	return ep.aggregator.RecordReconnection(event)
}

// Helper functions

func getPlayerNames(players []PlayerInfo) []string {
	names := make([]string, len(players))
	for i, player := range players {
		names[i] = player.Name
	}
	return names
}