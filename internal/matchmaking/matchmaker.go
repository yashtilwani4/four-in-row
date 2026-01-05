package matchmaking

import (
	"sync"
	"time"

	"connect-four-backend/internal/game"
	"connect-four-backend/internal/models"

	"github.com/google/uuid"
)

type Matchmaker struct {
	queue       []*QueueEntry
	gameManager *game.Manager
	mutex       sync.Mutex
}

func NewMatchmaker(gameManager *game.Manager) *Matchmaker {
	return &Matchmaker{
		queue:       make([]*QueueEntry, 0),
		gameManager: gameManager,
	}
}

func (m *Matchmaker) Start() {
	// Matchmaker runs continuously
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		m.processQueue()
	}
}

func (m *Matchmaker) JoinQueue(playerName string, conn game.WSConnection) *models.Player {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	player := &models.Player{
		ID:        uuid.New(),
		Name:      playerName,
		Connected: true,
		LastSeen:  time.Now(),
	}

	entry := &QueueEntry{
		Player:   player,
		Conn:     conn,
		JoinedAt: time.Now(),
	}

	// Set up bot timer (10 seconds)
	entry.BotTimer = time.AfterFunc(10*time.Second, func() {
		m.matchWithBot(entry)
	})

	m.queue = append(m.queue, entry)
	return player
}

func (m *Matchmaker) LeaveQueue(playerID uuid.UUID) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for i, entry := range m.queue {
		if entry.Player.ID == playerID {
			// Cancel bot timer
			if entry.BotTimer != nil {
				entry.BotTimer.Stop()
			}

			// Remove from queue
			m.queue = append(m.queue[:i], m.queue[i+1:]...)
			break
		}
	}
}

func (m *Matchmaker) processQueue() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Match players if we have at least 2
	for len(m.queue) >= 2 {
		player1Entry := m.queue[0]
		player2Entry := m.queue[1]

		// Cancel bot timers
		if player1Entry.BotTimer != nil {
			player1Entry.BotTimer.Stop()
		}
		if player2Entry.BotTimer != nil {
			player2Entry.BotTimer.Stop()
		}

		// Create game
		game := m.gameManager.CreateGame(player1Entry.Player, player2Entry.Player)

		// Add player connections
		m.gameManager.AddPlayerConnection(player1Entry.Player.ID, game.ID, player1Entry.Conn)
		m.gameManager.AddPlayerConnection(player2Entry.Player.ID, game.ID, player2Entry.Conn)

		// Notify players
		m.notifyGameFound(player1Entry, game)
		m.notifyGameFound(player2Entry, game)

		// Remove from queue
		m.queue = m.queue[2:]
	}
}

func (m *Matchmaker) matchWithBot(entry *QueueEntry) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Check if player is still in queue
	found := false
	for i, queueEntry := range m.queue {
		if queueEntry.Player.ID == entry.Player.ID {
			// Remove from queue
			m.queue = append(m.queue[:i], m.queue[i+1:]...)
			found = true
			break
		}
	}

	if !found {
		return // Player already matched or left
	}

	// Create bot player
	bot := game.NewBot()

	// Create game with bot
	gameInstance := m.gameManager.CreateGame(entry.Player, bot)

	// Add player connection (bot doesn't need connection)
	m.gameManager.AddPlayerConnection(entry.Player.ID, gameInstance.ID, entry.Conn)

	// Notify player
	m.notifyGameFound(entry, gameInstance)

	// Start bot AI routine
	go m.runBotAI(gameInstance.ID, bot.ID)
}

func (m *Matchmaker) notifyGameFound(entry *QueueEntry, game *models.Game) {
	message := models.WSMessage{
		Type: models.MsgGameFound,
		Payload: models.GameFoundPayload{
			Game:     game,
			PlayerID: entry.Player.ID,
		},
	}

	entry.Conn.WriteJSON(message)
}

func (m *Matchmaker) runBotAI(gameID, botID uuid.UUID) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		gameInstance, exists := m.gameManager.GetGame(gameID)
		if !exists || gameInstance.State != models.GameStatePlaying {
			return
		}

		// Check if it's bot's turn
		var botColor models.PlayerColor
		var isBot bool
		for _, player := range gameInstance.Players {
			if player.ID == botID {
				botColor = player.Color
				isBot = true
				break
			}
		}

		if !isBot || gameInstance.CurrentTurn != botColor {
			continue
		}

		// Add small delay for realism
		time.Sleep(500 * time.Millisecond)

		// Get best move
		column := game.GetBestMove(gameInstance, botColor)
		if column == -1 {
			continue
		}

		// Make the move
		move, err := m.gameManager.MakeMove(gameID, botID, column)
		if err != nil {
			continue
		}

		// Broadcast move result
		m.gameManager.BroadcastToGame(gameID, models.WSMessage{
			Type: models.MsgMoveResult,
			Payload: models.MoveResultPayload{
				Success:    true,
				Move:       move,
				GameState:  gameInstance,
				IsGameOver: gameInstance.State == models.GameStateFinished,
			},
		})

		// Check if game ended
		if gameInstance.State == models.GameStateFinished {
			m.gameManager.BroadcastToGame(gameID, models.WSMessage{
				Type: models.MsgGameEnd,
				Payload: models.GameEndPayload{
					GameID:    gameID,
					GameState: gameInstance,
					Winner:    nil, // Will need to convert from PlayerColor to Player
					Reason:    "Game completed",
					Duration:  0, // Calculate if needed
					IsDraw:    false, // Set based on game state
				},
			})
			return
		}
	}
}