package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"connect-four-backend/internal/models"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

// EventType represents the type of game event
type EventType string

const (
	EventGameStarted        EventType = "game_started"
	EventMovePlayed         EventType = "move_played"
	EventGameEnded          EventType = "game_ended"
	EventPlayerDisconnected EventType = "player_disconnected"
	EventPlayerReconnected  EventType = "player_reconnected"
	EventPlayerJoinedQueue  EventType = "player_joined_queue"
	EventPlayerLeftQueue    EventType = "player_left_queue"
	EventBotActivated       EventType = "bot_activated"
)

// Producer handles Kafka message production with async capabilities
type Producer struct {
	writer      *kafka.Writer
	errorChan   chan error
	stopChan    chan struct{}
	wg          sync.WaitGroup
	isRunning   bool
	mu          sync.RWMutex
	stats       ProducerStats
}

// ProducerStats tracks producer performance metrics
type ProducerStats struct {
	MessagesSent     int64     `json:"messages_sent"`
	MessagesErrored  int64     `json:"messages_errored"`
	LastMessageTime  time.Time `json:"last_message_time"`
	LastErrorTime    time.Time `json:"last_error_time"`
	LastError        string    `json:"last_error"`
}

// AnalyticsService provides high-level game event emission
type AnalyticsService struct {
	producer *Producer
	enabled  bool
}

// BaseEvent represents the common structure for all game events
type BaseEvent struct {
	EventType EventType `json:"event_type"`
	EventID   string    `json:"event_id"`
	Timestamp time.Time `json:"timestamp"`
	GameID    string    `json:"game_id"`
	Metadata  Metadata  `json:"metadata"`
}

// Metadata contains additional context for events
type Metadata struct {
	ServerID    string            `json:"server_id,omitempty"`
	Version     string            `json:"version,omitempty"`
	Environment string            `json:"environment,omitempty"`
	UserAgent   string            `json:"user_agent,omitempty"`
	IPAddress   string            `json:"ip_address,omitempty"`
	SessionID   string            `json:"session_id,omitempty"`
	Custom      map[string]string `json:"custom,omitempty"`
}

// PlayerInfo represents player information in events
type PlayerInfo struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Number    int    `json:"number"`
	IsBot     bool   `json:"is_bot"`
	IsActive  bool   `json:"is_active"`
	Connected bool   `json:"connected"`
}

// GameStartedEvent represents a game start event
type GameStartedEvent struct {
	BaseEvent
	Players     []PlayerInfo `json:"players"`
	GameMode    string       `json:"game_mode"`
	BoardSize   string       `json:"board_size"`
	StartPlayer int          `json:"start_player"`
}

// MovePlayedEvent represents a move event
type MovePlayedEvent struct {
	BaseEvent
	Player       PlayerInfo `json:"player"`
	Column       int        `json:"column"`
	Row          int        `json:"row"`
	MoveNumber   int        `json:"move_number"`
	TimeTaken    int64      `json:"time_taken_ms"`
	BoardState   [][]int    `json:"board_state"`
	ValidMoves   []int      `json:"valid_moves"`
	BotReasoning string     `json:"bot_reasoning,omitempty"`
}

// GameEndedEvent represents a game completion event
type GameEndedEvent struct {
	BaseEvent
	Players      []PlayerInfo `json:"players"`
	Winner       *PlayerInfo  `json:"winner,omitempty"`
	IsDraw       bool         `json:"is_draw"`
	WinType      string       `json:"win_type,omitempty"`
	TotalMoves   int          `json:"total_moves"`
	Duration     int64        `json:"duration_seconds"`
	EndReason    string       `json:"end_reason"`
	FinalBoard   [][]int      `json:"final_board"`
}

// PlayerDisconnectedEvent represents a player disconnection
type PlayerDisconnectedEvent struct {
	BaseEvent
	Player         PlayerInfo `json:"player"`
	DisconnectTime time.Time  `json:"disconnect_time"`
	Reason         string     `json:"reason"`
	GameState      string     `json:"game_state"`
	MoveNumber     int        `json:"move_number"`
	GracePeriod    int        `json:"grace_period_seconds"`
}

// PlayerReconnectedEvent represents a player reconnection
type PlayerReconnectedEvent struct {
	BaseEvent
	Player           PlayerInfo    `json:"player"`
	ReconnectTime    time.Time     `json:"reconnect_time"`
	DisconnectTime   time.Time     `json:"disconnect_time"`
	OfflineDuration  time.Duration `json:"offline_duration_ms"`
	MissedMoves      int           `json:"missed_moves"`
	GameState        string        `json:"game_state"`
}

// ProducerConfig holds configuration for the Kafka producer
type ProducerConfig struct {
	Brokers         []string      `json:"brokers"`
	Topic           string        `json:"topic"`
	RequiredAcks    int           `json:"required_acks"`
	BatchSize       int           `json:"batch_size"`
	BatchTimeout    time.Duration `json:"batch_timeout"`
	MaxMessageBytes int           `json:"max_message_bytes"`
	Compression     string        `json:"compression"`
	Retries         int           `json:"retries"`
	RetryBackoff    time.Duration `json:"retry_backoff"`
}

// DefaultProducerConfig returns a production-ready configuration
func DefaultProducerConfig(brokers []string) ProducerConfig {
	return ProducerConfig{
		Brokers:         brokers,
		Topic:           "connect-four-events",
		RequiredAcks:    1, // Wait for leader acknowledgment
		BatchSize:       100,
		BatchTimeout:    10 * time.Millisecond,
		MaxMessageBytes: 1000000, // 1MB
		Compression:     "snappy",
		Retries:         3,
		RetryBackoff:    100 * time.Millisecond,
	}
}

// NewProducer creates a new async Kafka producer
func NewProducer(config ProducerConfig) (*Producer, error) {
	// Configure compression
	var compression kafka.Compression
	switch config.Compression {
	case "gzip":
		compression = kafka.Gzip
	case "snappy":
		compression = kafka.Snappy
	case "lz4":
		compression = kafka.Lz4
	case "zstd":
		compression = kafka.Zstd
	default:
		compression = kafka.Snappy // Default to snappy
	}

	writer := &kafka.Writer{
		Addr:         kafka.TCP(config.Brokers...),
		Topic:        config.Topic,
		Balancer:     &kafka.Hash{}, // Use hash balancer for consistent partitioning
		RequiredAcks: kafka.RequiredAcks(config.RequiredAcks),
		Async:        true, // Enable async mode for better performance
		BatchSize:    config.BatchSize,
		BatchTimeout: config.BatchTimeout,
		Compression:  compression,
		MaxAttempts:  config.Retries,
		BatchBytes:   int64(config.MaxMessageBytes),
		ErrorLogger:  kafka.LoggerFunc(log.Printf),
	}

	producer := &Producer{
		writer:    writer,
		errorChan: make(chan error, 100), // Buffered channel for errors
		stopChan:  make(chan struct{}),
		stats:     ProducerStats{},
	}

	// Start error handling goroutine
	producer.wg.Add(1)
	go producer.handleErrors()

	producer.mu.Lock()
	producer.isRunning = true
	producer.mu.Unlock()

	return producer, nil
}

// Close gracefully shuts down the producer
func (p *Producer) Close() error {
	p.mu.Lock()
	if !p.isRunning {
		p.mu.Unlock()
		return nil
	}
	p.isRunning = false
	p.mu.Unlock()

	// Signal stop and wait for goroutines
	close(p.stopChan)
	p.wg.Wait()

	// Close error channel
	close(p.errorChan)

	// Close writer
	return p.writer.Close()
}

// SendMessage sends a message to Kafka asynchronously
func (p *Producer) SendMessage(key string, value []byte) error {
	p.mu.RLock()
	if !p.isRunning {
		p.mu.RUnlock()
		return fmt.Errorf("producer is not running")
	}
	p.mu.RUnlock()

	message := kafka.Message{
		Key:   []byte(key),
		Value: value,
		Time:  time.Now(),
	}

	// Send message asynchronously
	err := p.writer.WriteMessages(context.Background(), message)
	
	p.mu.Lock()
	if err != nil {
		p.stats.MessagesErrored++
		p.stats.LastErrorTime = time.Now()
		p.stats.LastError = err.Error()
	} else {
		p.stats.MessagesSent++
		p.stats.LastMessageTime = time.Now()
	}
	p.mu.Unlock()

	return err
}

// GetStats returns current producer statistics
func (p *Producer) GetStats() ProducerStats {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.stats
}

// handleErrors processes async errors from the Kafka writer
func (p *Producer) handleErrors() {
	defer p.wg.Done()

	for {
		select {
		case err := <-p.errorChan:
			if err != nil {
				log.Printf("Kafka producer error: %v", err)
				p.mu.Lock()
				p.stats.MessagesErrored++
				p.stats.LastErrorTime = time.Now()
				p.stats.LastError = err.Error()
				p.mu.Unlock()
			}
		case <-p.stopChan:
			return
		}
	}
}

// NewAnalyticsService creates a new analytics service
func NewAnalyticsService(producer *Producer, enabled bool) *AnalyticsService {
	return &AnalyticsService{
		producer: producer,
		enabled:  enabled,
	}
}

// IsEnabled returns whether analytics is enabled
func (a *AnalyticsService) IsEnabled() bool {
	return a.enabled
}

// SetEnabled enables or disables analytics
func (a *AnalyticsService) SetEnabled(enabled bool) {
	a.enabled = enabled
}

// EmitGameStarted emits a game started event
func (a *AnalyticsService) EmitGameStarted(game *models.Game, metadata Metadata) error {
	if !a.enabled {
		return nil
	}

	event := GameStartedEvent{
		BaseEvent: BaseEvent{
			EventType: EventGameStarted,
			EventID:   uuid.New().String(),
			Timestamp: time.Now(),
			GameID:    game.ID.String(),
			Metadata:  metadata,
		},
		Players:     convertPlayersToInfo(game.Players[:]),
		GameMode:    "1v1",
		BoardSize:   "7x6",
		StartPlayer: int(game.CurrentTurn),
	}

	return a.sendEvent(string(EventGameStarted), game.ID.String(), event)
}

// EmitMovePlayed emits a move played event
func (a *AnalyticsService) EmitMovePlayed(game *models.Game, move *models.Move, timeTaken time.Duration, botReasoning string, metadata Metadata) error {
	if !a.enabled {
		return nil
	}

	// Find the player who made the move
	var player *models.Player
	for _, p := range game.Players {
		if p.ID == move.PlayerID {
			player = p
			break
		}
	}
	if player == nil {
		return fmt.Errorf("player not found for move")
	}

	// Convert board grid for JSON
	boardState := make([][]int, 6)
	for i := range boardState {
		boardState[i] = make([]int, 7)
		for j := range boardState[i] {
			boardState[i][j] = game.Board[i][j]
		}
	}

	event := MovePlayedEvent{
		BaseEvent: BaseEvent{
			EventType: EventMovePlayed,
			EventID:   uuid.New().String(),
			Timestamp: time.Now(),
			GameID:    game.ID.String(),
			Metadata:  metadata,
		},
		Player:       convertPlayerToInfo(player),
		Column:       move.Column,
		Row:          move.Row,
		MoveNumber:   a.countMovesOnBoard(game.Board), // Use current move count
		TimeTaken:    timeTaken.Milliseconds(),
		BoardState:   boardState,
		ValidMoves:   a.getValidMoves(game), // Helper function to get valid moves
		BotReasoning: botReasoning,
	}

	return a.sendEvent(string(EventMovePlayed), game.ID.String(), event)
}

// EmitGameEnded emits a game ended event
func (a *AnalyticsService) EmitGameEnded(game *models.Game, endReason string, metadata Metadata) error {
	if !a.enabled {
		return nil
	}

	var winner *PlayerInfo
	if game.Winner != nil {
		// Find the winning player
		var winnerPlayer *models.Player
		if *game.Winner == models.PlayerRed {
			winnerPlayer = game.Players[0]
		} else {
			winnerPlayer = game.Players[1]
		}
		
		if winnerPlayer != nil {
			winnerInfo := convertPlayerToInfo(winnerPlayer)
			winner = &winnerInfo
		}
	}

	var winType string = "unknown"
	// For now, just use a default win type since we don't track this in models

	// Convert final board grid for JSON
	finalBoard := make([][]int, 6)
	for i := range finalBoard {
		finalBoard[i] = make([]int, 7)
		for j := range finalBoard[i] {
			finalBoard[i][j] = game.Board[i][j]
		}
	}

	event := GameEndedEvent{
		BaseEvent: BaseEvent{
			EventType: EventGameEnded,
			EventID:   uuid.New().String(),
			Timestamp: time.Now(),
			GameID:    game.ID.String(),
			Metadata:  metadata,
		},
		Players:    convertPlayersToInfo(game.Players[:]),
		Winner:     winner,
		IsDraw:     game.Winner == nil && game.State == models.GameStateFinished,
		WinType:    winType,
		TotalMoves: a.countMovesOnBoard(game.Board),
		Duration:   int64(game.FinishedAt.Sub(game.CreatedAt).Seconds()),
		EndReason:  endReason,
		FinalBoard: finalBoard,
	}

	return a.sendEvent(string(EventGameEnded), game.ID.String(), event)
}

// EmitPlayerDisconnected emits a player disconnected event
func (a *AnalyticsService) EmitPlayerDisconnected(game *models.Game, player *models.Player, reason string, gracePeriod int, metadata Metadata) error {
	if !a.enabled {
		return nil
	}

	event := PlayerDisconnectedEvent{
		BaseEvent: BaseEvent{
			EventType: EventPlayerDisconnected,
			EventID:   uuid.New().String(),
			Timestamp: time.Now(),
			GameID:    game.ID.String(),
			Metadata:  metadata,
		},
		Player:         convertPlayerToInfo(player),
		DisconnectTime: time.Now(),
		Reason:         reason,
		GameState:      string(game.State),
		MoveNumber:     a.countMovesOnBoard(game.Board),
		GracePeriod:    gracePeriod,
	}

	return a.sendEvent(string(EventPlayerDisconnected), game.ID.String(), event)
}

// EmitPlayerReconnected emits a player reconnected event
func (a *AnalyticsService) EmitPlayerReconnected(game *models.Game, player *models.Player, disconnectTime time.Time, missedMoves int, metadata Metadata) error {
	if !a.enabled {
		return nil
	}

	reconnectTime := time.Now()
	offlineDuration := reconnectTime.Sub(disconnectTime)

	event := PlayerReconnectedEvent{
		BaseEvent: BaseEvent{
			EventType: EventPlayerReconnected,
			EventID:   uuid.New().String(),
			Timestamp: reconnectTime,
			GameID:    game.ID.String(),
			Metadata:  metadata,
		},
		Player:          convertPlayerToInfo(player),
		ReconnectTime:   reconnectTime,
		DisconnectTime:  disconnectTime,
		OfflineDuration: offlineDuration,
		MissedMoves:     missedMoves,
		GameState:       string(game.State),
	}

	return a.sendEvent(string(EventPlayerReconnected), game.ID.String(), event)
}

// sendEvent is a helper method to send events to Kafka
func (a *AnalyticsService) sendEvent(eventType, gameID string, event interface{}) error {
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Use gameID as key for consistent partitioning
	key := fmt.Sprintf("%s:%s", eventType, gameID)
	
	return a.producer.SendMessage(key, eventJSON)
}

// Helper functions to convert engine types to event types

func convertPlayerToInfo(player *models.Player) PlayerInfo {
	return PlayerInfo{
		ID:        player.ID.String(),
		Name:      player.Name,
		Number:    int(player.Color), // Use color as number (0 for red, 1 for yellow)
		IsBot:     player.IsBot,
		IsActive:  player.Connected,
		Connected: player.Connected,
	}
}

func convertPlayersToInfo(players []*models.Player) []PlayerInfo {
	result := make([]PlayerInfo, len(players))
	for i, player := range players {
		if player != nil {
			result[i] = convertPlayerToInfo(player)
		}
	}
	return result
}

// Legacy method for backward compatibility
func (a *AnalyticsService) SendEvent(eventType string, data map[string]interface{}) {
	if !a.enabled {
		return
	}

	event := map[string]interface{}{
		"event_type": eventType,
		"event_id":   uuid.New().String(),
		"timestamp":  time.Now(),
		"data":       data,
	}

	eventJSON, err := json.Marshal(event)
	if err != nil {
		log.Printf("Failed to marshal legacy analytics event: %v", err)
		return
	}

	if err := a.producer.SendMessage(eventType, eventJSON); err != nil {
		log.Printf("Failed to send legacy analytics event: %v", err)
	}
}

// Helper function to count moves on the board
func (a *AnalyticsService) countMovesOnBoard(board [6][7]int) int {
	count := 0
	for i := 0; i < 6; i++ {
		for j := 0; j < 7; j++ {
			if board[i][j] != 0 {
				count++
			}
		}
	}
	return count
}
// Helper function to get valid moves
func (a *AnalyticsService) getValidMoves(game *models.Game) []int {
	var validMoves []int
	for col := 0; col < 7; col++ {
		if game.IsValidMove(col) {
			validMoves = append(validMoves, col)
		}
	}
	return validMoves
}