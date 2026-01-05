package matchmaking

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

// MatchmakingService handles the core matchmaking logic
type MatchmakingService struct {
	queue           *Queue
	gameCreator     GameCreator
	botProvider     BotProvider
	eventPublisher  EventPublisher
	
	// Configuration
	config MatchmakingConfig
	
	// Channels for service operations
	joinRequests  chan *JoinRequest
	leaveRequests chan *LeaveRequest
	
	// Context for graceful shutdown
	ctx    context.Context
	cancel context.CancelFunc
	
	// Wait group for goroutines
	wg sync.WaitGroup
	
	// Service state
	running bool
	mutex   sync.RWMutex
}

// MatchmakingConfig holds configuration for the matchmaking service
type MatchmakingConfig struct {
	BotMatchTimeout    time.Duration `json:"bot_match_timeout"`
	MatchCheckInterval time.Duration `json:"match_check_interval"`
	MaxQueueSize       int           `json:"max_queue_size"`
	EnableBotMatches   bool          `json:"enable_bot_matches"`
}

// JoinRequest represents a request to join the matchmaking queue
type JoinRequest struct {
	PlayerID    uuid.UUID         `json:"player_id"`
	Username    string            `json:"username"`
	Preferences *MatchPreferences `json:"preferences,omitempty"`
	ResponseCh  chan *JoinResponse `json:"-"`
}

// JoinResponse represents the response to a join request
type JoinResponse struct {
	Success   bool      `json:"success"`
	Message   string    `json:"message"`
	QueueSize int       `json:"queue_size"`
	Position  int       `json:"position"`
	PlayerID  uuid.UUID `json:"player_id"`
}

// LeaveRequest represents a request to leave the matchmaking queue
type LeaveRequest struct {
	PlayerID   uuid.UUID          `json:"player_id"`
	ResponseCh chan *LeaveResponse `json:"-"`
}

// LeaveResponse represents the response to a leave request
type LeaveResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// Match represents a successful match between players
type Match struct {
	GameID    uuid.UUID `json:"game_id"`
	Player1   *Player   `json:"player1"`
	Player2   *Player   `json:"player2"`
	CreatedAt time.Time `json:"created_at"`
	IsBot     bool      `json:"is_bot"`
}

// Player represents a player in a match
type Player struct {
	ID       uuid.UUID `json:"id"`
	Username string    `json:"username"`
	IsBot    bool      `json:"is_bot"`
}

// Interfaces for dependency injection

// GameCreator interface for creating games
type GameCreator interface {
	CreateGame(player1, player2 *Player) (*Match, error)
}

// BotProvider interface for creating bot opponents
type BotProvider interface {
	CreateBot() *Player
}

// EventPublisher interface for publishing matchmaking events
type EventPublisher interface {
	PublishMatchFound(match *Match) error
	PublishPlayerJoined(playerID uuid.UUID, username string) error
	PublishPlayerLeft(playerID uuid.UUID, username string) error
}

// NewMatchmakingService creates a new matchmaking service
func NewMatchmakingService(ctx context.Context, config MatchmakingConfig, gameCreator GameCreator, botProvider BotProvider, eventPublisher EventPublisher) *MatchmakingService {
	serviceCtx, cancel := context.WithCancel(ctx)
	
	// Set default configuration values
	if config.BotMatchTimeout == 0 {
		config.BotMatchTimeout = 10 * time.Second
	}
	if config.MatchCheckInterval == 0 {
		config.MatchCheckInterval = 1 * time.Second
	}
	if config.MaxQueueSize == 0 {
		config.MaxQueueSize = 1000
	}
	
	return &MatchmakingService{
		queue:           NewQueue(),
		gameCreator:     gameCreator,
		botProvider:     botProvider,
		eventPublisher:  eventPublisher,
		config:          config,
		joinRequests:    make(chan *JoinRequest, 100),
		leaveRequests:   make(chan *LeaveRequest, 100),
		ctx:             serviceCtx,
		cancel:          cancel,
	}
}

// Start starts the matchmaking service
func (s *MatchmakingService) Start() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	if s.running {
		return ErrServiceAlreadyRunning
	}
	
	s.running = true
	
	// Start worker goroutines
	s.wg.Add(3)
	go s.requestProcessor()
	go s.matchProcessor()
	go s.queueProcessor()
	
	log.Println("Matchmaking service started")
	return nil
}

// Stop stops the matchmaking service
func (s *MatchmakingService) Stop() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	if !s.running {
		return ErrServiceNotRunning
	}
	
	s.running = false
	s.cancel()
	
	// Close channels
	close(s.joinRequests)
	close(s.leaveRequests)
	
	// Wait for goroutines to finish
	s.wg.Wait()
	
	log.Println("Matchmaking service stopped")
	return nil
}

// JoinQueue adds a player to the matchmaking queue
func (s *MatchmakingService) JoinQueue(playerID uuid.UUID, username string, preferences *MatchPreferences) (*JoinResponse, error) {
	if !s.isRunning() {
		return &JoinResponse{
			Success: false,
			Message: "Matchmaking service is not running",
		}, ErrServiceNotRunning
	}
	
	// Check queue size limit
	if s.queue.GetSize() >= s.config.MaxQueueSize {
		return &JoinResponse{
			Success: false,
			Message: "Queue is full",
		}, ErrQueueFull
	}
	
	request := &JoinRequest{
		PlayerID:    playerID,
		Username:    username,
		Preferences: preferences,
		ResponseCh:  make(chan *JoinResponse, 1),
	}
	
	select {
	case s.joinRequests <- request:
		// Wait for response
		select {
		case response := <-request.ResponseCh:
			return response, nil
		case <-s.ctx.Done():
			return &JoinResponse{
				Success: false,
				Message: "Service shutting down",
			}, ErrServiceShuttingDown
		case <-time.After(5 * time.Second):
			return &JoinResponse{
				Success: false,
				Message: "Request timeout",
			}, ErrRequestTimeout
		}
	case <-s.ctx.Done():
		return &JoinResponse{
			Success: false,
			Message: "Service shutting down",
		}, ErrServiceShuttingDown
	}
}

// LeaveQueue removes a player from the matchmaking queue
func (s *MatchmakingService) LeaveQueue(playerID uuid.UUID) (*LeaveResponse, error) {
	if !s.isRunning() {
		return &LeaveResponse{
			Success: false,
			Message: "Matchmaking service is not running",
		}, ErrServiceNotRunning
	}
	
	request := &LeaveRequest{
		PlayerID:   playerID,
		ResponseCh: make(chan *LeaveResponse, 1),
	}
	
	select {
	case s.leaveRequests <- request:
		// Wait for response
		select {
		case response := <-request.ResponseCh:
			return response, nil
		case <-s.ctx.Done():
			return &LeaveResponse{
				Success: false,
				Message: "Service shutting down",
			}, ErrServiceShuttingDown
		case <-time.After(5 * time.Second):
			return &LeaveResponse{
				Success: false,
				Message: "Request timeout",
			}, ErrRequestTimeout
		}
	case <-s.ctx.Done():
		return &LeaveResponse{
			Success: false,
			Message: "Service shutting down",
		}, ErrServiceShuttingDown
	}
}

// GetQueueStats returns queue statistics
func (s *MatchmakingService) GetQueueStats() QueueStats {
	return s.queue.GetStats()
}

// requestProcessor processes join and leave requests
func (s *MatchmakingService) requestProcessor() {
	defer s.wg.Done()
	
	for {
		select {
		case <-s.ctx.Done():
			return
			
		case request := <-s.joinRequests:
			if request == nil {
				return // Channel closed
			}
			s.handleJoinRequest(request)
			
		case request := <-s.leaveRequests:
			if request == nil {
				return // Channel closed
			}
			s.handleLeaveRequest(request)
		}
	}
}

// matchProcessor looks for matches between players
func (s *MatchmakingService) matchProcessor() {
	defer s.wg.Done()
	
	ticker := time.NewTicker(s.config.MatchCheckInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.processMatches()
		}
	}
}

// queueProcessor handles queue operations
func (s *MatchmakingService) queueProcessor() {
	defer s.wg.Done()
	
	for {
		select {
		case <-s.ctx.Done():
			return
		case entry := <-s.queue.addChan:
			s.queue.processAdd(entry)
		case playerID := <-s.queue.removeChan:
			s.queue.processRemove(playerID)
		}
	}
}

// handleJoinRequest processes a join request
func (s *MatchmakingService) handleJoinRequest(request *JoinRequest) {
	// Check if player is already in queue
	if _, exists := s.queue.GetEntry(request.PlayerID); exists {
		request.ResponseCh <- &JoinResponse{
			Success: false,
			Message: "Player already in queue",
		}
		return
	}
	
	// Add player to queue
	entry := s.queue.Add(request.PlayerID, request.Username, request.Preferences)
	
	// Set up bot timer if enabled
	if s.config.EnableBotMatches && entry.Preferences.AllowBots {
		timeout := s.config.BotMatchTimeout
		if entry.Preferences.MaxWaitTime > 0 {
			timeout = time.Duration(entry.Preferences.MaxWaitTime) * time.Second
		}
		
		entry.BotTimer = time.AfterFunc(timeout, func() {
			s.createBotMatch(entry)
		})
	}
	
	// Publish event
	if s.eventPublisher != nil {
		s.eventPublisher.PublishPlayerJoined(request.PlayerID, request.Username)
	}
	
	// Send response
	request.ResponseCh <- &JoinResponse{
		Success:   true,
		Message:   "Successfully joined queue",
		QueueSize: s.queue.GetSize(),
		Position:  s.calculatePosition(entry),
		PlayerID:  request.PlayerID,
	}
	
	log.Printf("Player %s (%s) joined matchmaking queue", request.Username, request.PlayerID)
}

// handleLeaveRequest processes a leave request
func (s *MatchmakingService) handleLeaveRequest(request *LeaveRequest) {
	// Check if player is in queue
	entry, exists := s.queue.GetEntry(request.PlayerID)
	if !exists {
		request.ResponseCh <- &LeaveResponse{
			Success: false,
			Message: "Player not in queue",
		}
		return
	}
	
	// Remove player from queue
	s.queue.Remove(request.PlayerID)
	
	// Publish event
	if s.eventPublisher != nil {
		s.eventPublisher.PublishPlayerLeft(request.PlayerID, entry.Username)
	}
	
	// Send response
	request.ResponseCh <- &LeaveResponse{
		Success: true,
		Message: "Successfully left queue",
	}
	
	log.Printf("Player %s (%s) left matchmaking queue", entry.Username, request.PlayerID)
}

// processMatches looks for and creates matches between players
func (s *MatchmakingService) processMatches() {
	entries := s.queue.GetAllEntries()
	
	// Simple matching algorithm - can be improved with more sophisticated logic
	for i := 0; i < len(entries); i++ {
		entry1 := entries[i]
		
		// Skip if entry no longer exists (may have been matched or left)
		if _, exists := s.queue.GetEntry(entry1.PlayerID); !exists {
			continue
		}
		
		// Look for a compatible match
		match := s.queue.GetCompatibleMatch(entry1)
		if match != nil {
			s.createPlayerMatch(entry1, match)
		}
	}
}

// createPlayerMatch creates a match between two players
func (s *MatchmakingService) createPlayerMatch(entry1, entry2 *QueueEntry) {
	// Remove both players from queue
	s.queue.Remove(entry1.PlayerID)
	s.queue.Remove(entry2.PlayerID)
	
	// Create players
	player1 := &Player{
		ID:       entry1.PlayerID,
		Username: entry1.Username,
		IsBot:    false,
	}
	
	player2 := &Player{
		ID:       entry2.PlayerID,
		Username: entry2.Username,
		IsBot:    false,
	}
	
	// Create match
	match, err := s.gameCreator.CreateGame(player1, player2)
	if err != nil {
		log.Printf("Failed to create game for players %s and %s: %v", entry1.Username, entry2.Username, err)
		// Re-add players to queue on failure
		s.queue.Add(entry1.PlayerID, entry1.Username, entry1.Preferences)
		s.queue.Add(entry2.PlayerID, entry2.Username, entry2.Preferences)
		return
	}
	
	// Update statistics
	s.queue.incrementMatched()
	s.queue.updateAverageWaitTime(time.Since(entry1.JoinedAt))
	s.queue.updateAverageWaitTime(time.Since(entry2.JoinedAt))
	
	// Publish match found event
	if s.eventPublisher != nil {
		s.eventPublisher.PublishMatchFound(match)
	}
	
	log.Printf("Match created: %s vs %s (Game ID: %s)", player1.Username, player2.Username, match.GameID)
}

// createBotMatch creates a match between a player and a bot
func (s *MatchmakingService) createBotMatch(entry *QueueEntry) {
	// Check if player is still in queue
	if _, exists := s.queue.GetEntry(entry.PlayerID); !exists {
		return
	}
	
	// Remove player from queue
	s.queue.Remove(entry.PlayerID)
	
	// Create player and bot
	player := &Player{
		ID:       entry.PlayerID,
		Username: entry.Username,
		IsBot:    false,
	}
	
	bot := s.botProvider.CreateBot()
	
	// Create match
	match, err := s.gameCreator.CreateGame(player, bot)
	if err != nil {
		log.Printf("Failed to create bot game for player %s: %v", entry.Username, err)
		// Re-add player to queue on failure
		s.queue.Add(entry.PlayerID, entry.Username, entry.Preferences)
		return
	}
	
	match.IsBot = true
	
	// Update statistics
	s.queue.incrementBotMatches()
	s.queue.updateAverageWaitTime(time.Since(entry.JoinedAt))
	
	// Publish match found event
	if s.eventPublisher != nil {
		s.eventPublisher.PublishMatchFound(match)
	}
	
	log.Printf("Bot match created: %s vs Bot (Game ID: %s)", player.Username, match.GameID)
}

// calculatePosition calculates the position of a player in the queue
func (s *MatchmakingService) calculatePosition(entry *QueueEntry) int {
	entries := s.queue.GetAllEntries()
	position := 1
	
	for _, e := range entries {
		if e.JoinedAt.Before(entry.JoinedAt) {
			position++
		}
	}
	
	return position
}

// isRunning checks if the service is running
func (s *MatchmakingService) isRunning() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.running
}