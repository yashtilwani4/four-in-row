package matchmaking

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Manager provides a high-level interface for matchmaking operations
type Manager struct {
	service        *MatchmakingService
	eventPublisher *DefaultEventPublisher
	
	// Active matches tracking
	activeMatches map[uuid.UUID]*Match
	matchesMutex  sync.RWMutex
	
	// Match callbacks
	onMatchFound func(*Match)
	onMatchEnd   func(uuid.UUID)
	
	// Configuration
	config ManagerConfig
}

// ManagerConfig holds configuration for the matchmaking manager
type ManagerConfig struct {
	BotMatchTimeout    time.Duration `json:"bot_match_timeout"`
	MatchCheckInterval time.Duration `json:"match_check_interval"`
	MaxQueueSize       int           `json:"max_queue_size"`
	EnableBotMatches   bool          `json:"enable_bot_matches"`
	EnableMetrics      bool          `json:"enable_metrics"`
}

// NewManager creates a new matchmaking manager
func NewManager(ctx context.Context, config ManagerConfig) *Manager {
	// Set default configuration
	if config.BotMatchTimeout == 0 {
		config.BotMatchTimeout = 10 * time.Second
	}
	if config.MatchCheckInterval == 0 {
		config.MatchCheckInterval = 1 * time.Second
	}
	if config.MaxQueueSize == 0 {
		config.MaxQueueSize = 1000
	}
	
	// Create event publisher
	eventPublisher := NewDefaultEventPublisher()
	
	// Create service configuration
	serviceConfig := MatchmakingConfig{
		BotMatchTimeout:    config.BotMatchTimeout,
		MatchCheckInterval: config.MatchCheckInterval,
		MaxQueueSize:       config.MaxQueueSize,
		EnableBotMatches:   config.EnableBotMatches,
	}
	
	// Create service
	service := NewMatchmakingService(
		ctx,
		serviceConfig,
		&DefaultGameCreator{},
		&DefaultBotProvider{},
		eventPublisher,
	)
	
	manager := &Manager{
		service:        service,
		eventPublisher: eventPublisher,
		activeMatches:  make(map[uuid.UUID]*Match),
		config:         config,
	}
	
	// Register event handlers
	eventPublisher.OnMatchFound(manager.handleMatchFound)
	
	return manager
}

// Start starts the matchmaking manager
func (m *Manager) Start() error {
	return m.service.Start()
}

// Stop stops the matchmaking manager
func (m *Manager) Stop() error {
	return m.service.Stop()
}

// JoinQueue adds a player to the matchmaking queue
func (m *Manager) JoinQueue(username string) (*JoinResponse, error) {
	if username == "" {
		return &JoinResponse{
			Success: false,
			Message: "Username is required",
		}, ErrInvalidUsername
	}
	
	playerID := uuid.New()
	
	// Default preferences
	preferences := &MatchPreferences{
		AllowBots:   m.config.EnableBotMatches,
		SkillLevel:  5,
		MaxWaitTime: int(m.config.BotMatchTimeout.Seconds()),
	}
	
	return m.service.JoinQueue(playerID, username, preferences)
}

// JoinQueueWithPreferences adds a player to the matchmaking queue with custom preferences
func (m *Manager) JoinQueueWithPreferences(username string, preferences *MatchPreferences) (*JoinResponse, error) {
	if username == "" {
		return &JoinResponse{
			Success: false,
			Message: "Username is required",
		}, ErrInvalidUsername
	}
	
	playerID := uuid.New()
	
	// Apply default preferences if not provided
	if preferences == nil {
		preferences = &MatchPreferences{
			AllowBots:   m.config.EnableBotMatches,
			SkillLevel:  5,
			MaxWaitTime: int(m.config.BotMatchTimeout.Seconds()),
		}
	}
	
	return m.service.JoinQueue(playerID, username, preferences)
}

// LeaveQueue removes a player from the matchmaking queue
func (m *Manager) LeaveQueue(playerID uuid.UUID) (*LeaveResponse, error) {
	return m.service.LeaveQueue(playerID)
}

// GetQueueStatus returns the current queue status
func (m *Manager) GetQueueStatus() QueueStatus {
	stats := m.service.GetQueueStats()
	
	return QueueStatus{
		Size:            stats.CurrentSize,
		TotalJoined:     stats.TotalJoined,
		TotalMatched:    stats.TotalMatched,
		TotalBotMatches: stats.TotalBotMatches,
		AverageWaitTime: stats.AverageWaitTime,
		ActiveMatches:   m.getActiveMatchCount(),
	}
}

// GetActiveMatch returns an active match by game ID
func (m *Manager) GetActiveMatch(gameID uuid.UUID) (*Match, bool) {
	m.matchesMutex.RLock()
	defer m.matchesMutex.RUnlock()
	
	match, exists := m.activeMatches[gameID]
	return match, exists
}

// EndMatch marks a match as ended
func (m *Manager) EndMatch(gameID uuid.UUID) {
	m.matchesMutex.Lock()
	defer m.matchesMutex.Unlock()
	
	if _, exists := m.activeMatches[gameID]; exists {
		delete(m.activeMatches, gameID)
		
		if m.onMatchEnd != nil {
			go m.onMatchEnd(gameID)
		}
	}
}

// OnMatchFound registers a callback for when matches are found
func (m *Manager) OnMatchFound(callback func(*Match)) {
	m.onMatchFound = callback
}

// OnMatchEnd registers a callback for when matches end
func (m *Manager) OnMatchEnd(callback func(uuid.UUID)) {
	m.onMatchEnd = callback
}

// GetMetrics returns matchmaking metrics
func (m *Manager) GetMetrics() MatchmakingMetrics {
	stats := m.service.GetQueueStats()
	
	return MatchmakingMetrics{
		QueueSize:       stats.CurrentSize,
		TotalJoined:     stats.TotalJoined,
		TotalLeft:       stats.TotalLeft,
		TotalMatched:    stats.TotalMatched,
		TotalBotMatches: stats.TotalBotMatches,
		AverageWaitTime: stats.AverageWaitTime,
		ActiveMatches:   m.getActiveMatchCount(),
		Timestamp:       time.Now(),
	}
}

// handleMatchFound handles match found events
func (m *Manager) handleMatchFound(match *Match) {
	// Add to active matches
	m.matchesMutex.Lock()
	m.activeMatches[match.GameID] = match
	m.matchesMutex.Unlock()
	
	// Call user callback if registered
	if m.onMatchFound != nil {
		m.onMatchFound(match)
	}
}

// getActiveMatchCount returns the number of active matches
func (m *Manager) getActiveMatchCount() int {
	m.matchesMutex.RLock()
	defer m.matchesMutex.RUnlock()
	return len(m.activeMatches)
}

// QueueStatus represents the current status of the matchmaking queue
type QueueStatus struct {
	Size            int           `json:"size"`
	TotalJoined     int64         `json:"total_joined"`
	TotalMatched    int64         `json:"total_matched"`
	TotalBotMatches int64         `json:"total_bot_matches"`
	AverageWaitTime time.Duration `json:"average_wait_time"`
	ActiveMatches   int           `json:"active_matches"`
}

// MatchmakingMetrics represents comprehensive matchmaking metrics
type MatchmakingMetrics struct {
	QueueSize       int           `json:"queue_size"`
	TotalJoined     int64         `json:"total_joined"`
	TotalLeft       int64         `json:"total_left"`
	TotalMatched    int64         `json:"total_matched"`
	TotalBotMatches int64         `json:"total_bot_matches"`
	AverageWaitTime time.Duration `json:"average_wait_time"`
	ActiveMatches   int           `json:"active_matches"`
	Timestamp       time.Time     `json:"timestamp"`
}