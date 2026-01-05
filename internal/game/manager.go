package game

import (
	"log"
	"sync"
	"time"

	"connect-four-backend/internal/models"

	"github.com/google/uuid"
)

type Manager struct {
	games   map[uuid.UUID]*models.Game
	players map[uuid.UUID]*PlayerConnection
	mutex   sync.RWMutex
}

type PlayerConnection struct {
	PlayerID uuid.UUID
	GameID   uuid.UUID
	Conn     WSConnection
	LastSeen time.Time
}

type WSConnection interface {
	WriteJSON(v interface{}) error
	Close() error
}

func NewManager() *Manager {
	manager := &Manager{
		games:   make(map[uuid.UUID]*models.Game),
		players: make(map[uuid.UUID]*PlayerConnection),
	}

	// Start cleanup routine for disconnected players
	go manager.cleanupRoutine()

	return manager
}

func (m *Manager) CreateGame(player1, player2 *models.Player) *models.Game {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	game := &models.Game{
		ID:          uuid.New(),
		State:       models.GameStatePlaying,
		Players:     [2]*models.Player{player1, player2},
		CurrentTurn: models.PlayerRed, // Red always starts
		CurrentTurnNumber: 1, // Red = 1
		CreatedAt:   time.Now(),
	}

	// Assign colors and numbers
	game.Players[0].Color = models.PlayerRed
	game.Players[0].Number = 1 // Red = 1
	game.Players[1].Color = models.PlayerYellow
	game.Players[1].Number = 2 // Yellow = 2

	log.Printf("DEBUG: Game created. Player1: %s (Color: %d, Number: %d), Player2: %s (Color: %d, Number: %d)", 
		game.Players[0].Name, game.Players[0].Color, game.Players[0].Number,
		game.Players[1].Name, game.Players[1].Color, game.Players[1].Number)

	m.games[game.ID] = game
	return game
}

func (m *Manager) GetGame(gameID uuid.UUID) (*models.Game, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	game, exists := m.games[gameID]
	return game, exists
}

func (m *Manager) MakeMove(gameID uuid.UUID, playerID uuid.UUID, column int) (*models.Move, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	game, exists := m.games[gameID]
	if !exists {
		return nil, ErrGameNotFound
	}

	if game.State != models.GameStatePlaying {
		return nil, ErrGameNotActive
	}

	// Find player and check if it's their turn
	var player *models.Player
	for _, p := range game.Players {
		if p.ID == playerID {
			player = p
			break
		}
	}

	if player == nil {
		return nil, ErrPlayerNotInGame
	}

	// Debug logging
	log.Printf("DEBUG: Player %s (Color: %d, Number: %d) trying to move. Current turn: %d (Number: %d)", 
		player.Name, player.Color, player.Number, game.CurrentTurn, game.CurrentTurnNumber)

	if player.Color != game.CurrentTurn {
		return nil, ErrNotPlayerTurn
	}

	// Try to make the move
	move := game.MakeMove(column, player.Color)
	if move == nil {
		return nil, ErrInvalidMove
	}

	move.PlayerID = playerID

	// Check if someone won
	if winner := game.CheckWinner(); winner != nil {
		game.Winner = winner
		game.State = models.GameStateFinished
		now := time.Now()
		game.FinishedAt = &now
	} else if game.IsBoardFull() {
		// It's a draw
		game.State = models.GameStateFinished
		now := time.Now()
		game.FinishedAt = &now
	} else {
		// Switch turns
		if game.CurrentTurn == models.PlayerRed {
			game.CurrentTurn = models.PlayerYellow
			game.CurrentTurnNumber = 2
		} else {
			game.CurrentTurn = models.PlayerRed
			game.CurrentTurnNumber = 1
		}
	}

	return move, nil
}

func (m *Manager) AddPlayerConnection(playerID, gameID uuid.UUID, conn WSConnection) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.players[playerID] = &PlayerConnection{
		PlayerID: playerID,
		GameID:   gameID,
		Conn:     conn,
		LastSeen: time.Now(),
	}

	// Update player connection status in game
	if game, exists := m.games[gameID]; exists {
		for _, player := range game.Players {
			if player.ID == playerID {
				player.Connected = true
				player.LastSeen = time.Now()
				break
			}
		}
	}
}

func (m *Manager) RemovePlayerConnection(playerID uuid.UUID) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if conn, exists := m.players[playerID]; exists {
		// Update player connection status in game
		if game, exists := m.games[conn.GameID]; exists {
			for _, player := range game.Players {
				if player.ID == playerID {
					player.Connected = false
					player.LastSeen = time.Now()
					break
				}
			}
		}

		delete(m.players, playerID)
	}
}

func (m *Manager) GetPlayerConnection(playerID uuid.UUID) (*PlayerConnection, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	conn, exists := m.players[playerID]
	return conn, exists
}

func (m *Manager) BroadcastToGame(gameID uuid.UUID, message interface{}) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	game, exists := m.games[gameID]
	if !exists {
		return
	}

	for _, player := range game.Players {
		if conn, exists := m.players[player.ID]; exists {
			conn.Conn.WriteJSON(message)
		}
	}
}

func (m *Manager) cleanupRoutine() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		m.cleanupDisconnectedPlayers()
	}
}

func (m *Manager) cleanupDisconnectedPlayers() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	now := time.Now()
	gracePeriod := 30 * time.Second

	for gameID, game := range m.games {
		if game.State != models.GameStatePlaying {
			continue
		}

		// Check if any player has been disconnected too long
		for _, player := range game.Players {
			if !player.Connected && now.Sub(player.LastSeen) > gracePeriod {
				// End game due to disconnection
				game.State = models.GameStateFinished
				now := time.Now()
				game.FinishedAt = &now

				// Determine winner (the connected player wins)
				for _, p := range game.Players {
					if p.Connected {
						game.Winner = &p.Color
						break
					}
				}

				// Broadcast game end
				m.BroadcastToGame(gameID, models.WSMessage{
					Type: models.MsgGameEnd,
					Payload: models.GameEndPayload{
						GameID:    gameID,
						GameState: game,
						Winner:    nil, // Will be set based on game.Winner
						Reason:    "Player disconnected",
						Duration:  0, // Calculate if needed
						IsDraw:    false,
					},
				})
				break
			}
		}
	}
}