package matchmaking

import (
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
)

// DefaultGameCreator implements GameCreator interface
type DefaultGameCreator struct{}

// CreateGame creates a new game between two players
func (gc *DefaultGameCreator) CreateGame(player1, player2 *Player) (*Match, error) {
	if player1 == nil || player2 == nil {
		return nil, ErrInvalidRequest
	}
	
	gameID := uuid.New()
	
	match := &Match{
		GameID:    gameID,
		Player1:   player1,
		Player2:   player2,
		CreatedAt: time.Now(),
		IsBot:     player2.IsBot,
	}
	
	return match, nil
}

// DefaultBotProvider implements BotProvider interface
type DefaultBotProvider struct {
	botCounter int
}

// CreateBot creates a new bot player
func (bp *DefaultBotProvider) CreateBot() *Player {
	bp.botCounter++
	
	botNames := []string{
		"ConnectBot", "AI_Master", "BotPlayer", "SmartBot", 
		"ChallengerBot", "ProBot", "GameBot", "WinBot",
	}
	
	botName := fmt.Sprintf("%s_%d", botNames[bp.botCounter%len(botNames)], bp.botCounter)
	
	return &Player{
		ID:       uuid.New(),
		Username: botName,
		IsBot:    true,
	}
}

// DefaultEventPublisher implements EventPublisher interface
type DefaultEventPublisher struct {
	matchFoundHandlers    []func(*Match)
	playerJoinedHandlers  []func(uuid.UUID, string)
	playerLeftHandlers    []func(uuid.UUID, string)
}

// NewDefaultEventPublisher creates a new default event publisher
func NewDefaultEventPublisher() *DefaultEventPublisher {
	return &DefaultEventPublisher{
		matchFoundHandlers:   make([]func(*Match), 0),
		playerJoinedHandlers: make([]func(uuid.UUID, string), 0),
		playerLeftHandlers:   make([]func(uuid.UUID, string), 0),
	}
}

// PublishMatchFound publishes a match found event
func (ep *DefaultEventPublisher) PublishMatchFound(match *Match) error {
	log.Printf("Match found: %s vs %s (Game: %s, Bot: %v)", 
		match.Player1.Username, match.Player2.Username, match.GameID, match.IsBot)
	
	for _, handler := range ep.matchFoundHandlers {
		go handler(match)
	}
	
	return nil
}

// PublishPlayerJoined publishes a player joined event
func (ep *DefaultEventPublisher) PublishPlayerJoined(playerID uuid.UUID, username string) error {
	log.Printf("Player joined queue: %s (%s)", username, playerID)
	
	for _, handler := range ep.playerJoinedHandlers {
		go handler(playerID, username)
	}
	
	return nil
}

// PublishPlayerLeft publishes a player left event
func (ep *DefaultEventPublisher) PublishPlayerLeft(playerID uuid.UUID, username string) error {
	log.Printf("Player left queue: %s (%s)", username, playerID)
	
	for _, handler := range ep.playerLeftHandlers {
		go handler(playerID, username)
	}
	
	return nil
}

// OnMatchFound registers a handler for match found events
func (ep *DefaultEventPublisher) OnMatchFound(handler func(*Match)) {
	ep.matchFoundHandlers = append(ep.matchFoundHandlers, handler)
}

// OnPlayerJoined registers a handler for player joined events
func (ep *DefaultEventPublisher) OnPlayerJoined(handler func(uuid.UUID, string)) {
	ep.playerJoinedHandlers = append(ep.playerJoinedHandlers, handler)
}

// OnPlayerLeft registers a handler for player left events
func (ep *DefaultEventPublisher) OnPlayerLeft(handler func(uuid.UUID, string)) {
	ep.playerLeftHandlers = append(ep.playerLeftHandlers, handler)
}