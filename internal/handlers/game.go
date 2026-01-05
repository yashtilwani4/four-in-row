package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"connect-four-backend/internal/game"
	"connect-four-backend/internal/kafka"
	"connect-four-backend/internal/matchmaking"
	"connect-four-backend/internal/models"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type GameHandler struct {
	gameManager      *game.Manager
	matchmaker       *matchmaking.Matchmaker
	analyticsService *kafka.AnalyticsService
	upgrader         websocket.Upgrader
}

func NewGameHandler(gameManager *game.Manager, matchmaker *matchmaking.Matchmaker, analyticsService *kafka.AnalyticsService) *GameHandler {
	return &GameHandler{
		gameManager:      gameManager,
		matchmaker:       matchmaker,
		analyticsService: analyticsService,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // TODO: Add proper origin checking for production
			},
		},
	}
}

func (h *GameHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	log.Printf("New WebSocket connection established from %s", r.RemoteAddr)

	var playerID uuid.UUID

	// Main message loop
	for {
		var msg models.WSMessage
		if err := conn.ReadJSON(&msg); err != nil {
			// Check if it's a normal close (not an actual error)
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
				log.Printf("WebSocket unexpected close: %v", err)
			}
			// For normal closes (1001 going away, 1000 normal), just break silently
			break
		}

		switch msg.Type {
		case models.MsgJoinQueue:
			playerID, _ = h.handleJoinQueue(conn, msg.Payload)

		case models.MsgLeaveQueue:
			h.handleLeaveQueue(playerID)

		case models.MsgMakeMove:
			h.handleMakeMove(conn, playerID, msg.Payload)

		case models.MsgReconnect:
			playerID, _ = h.handleReconnect(conn, msg.Payload)

		case models.MsgHeartbeat:
			h.handleHeartbeat(conn, playerID)

		default:
			h.sendError(conn, "UNKNOWN_MESSAGE", "Unknown message type", "")
		}
	}

	// Clean up when player disconnects
	if playerID != uuid.Nil {
		h.gameManager.RemovePlayerConnection(playerID)
		h.matchmaker.LeaveQueue(playerID)
		log.Printf("Player %s disconnected cleanly", playerID)
	} else {
		log.Printf("WebSocket connection closed from %s", r.RemoteAddr)
	}
}

func (h *GameHandler) handleJoinQueue(conn *websocket.Conn, payload interface{}) (uuid.UUID, uuid.UUID) {
	var joinPayload models.JoinQueuePayload
	if err := h.parsePayload(payload, &joinPayload); err != nil {
		h.sendError(conn, "INVALID_PAYLOAD", "Invalid join queue payload", "")
		return uuid.Nil, uuid.Nil
	}

	player := h.matchmaker.JoinQueue(joinPayload.PlayerName, conn)

	// Send analytics event
	h.analyticsService.SendEvent("player_joined_queue", map[string]interface{}{
		"player_id":   player.ID.String(),
		"player_name": player.Name,
	})

	return player.ID, uuid.Nil
}

func (h *GameHandler) handleLeaveQueue(playerID uuid.UUID) {
	if playerID != uuid.Nil {
		h.matchmaker.LeaveQueue(playerID)

		// Send analytics event
		h.analyticsService.SendEvent("player_left_queue", map[string]interface{}{
			"player_id": playerID.String(),
		})
	}
}

func (h *GameHandler) handleMakeMove(conn *websocket.Conn, playerID uuid.UUID, payload interface{}) {
	var movePayload models.MakeMovePayload
	if err := h.parsePayload(payload, &movePayload); err != nil {
		h.sendError(conn, "INVALID_PAYLOAD", "Invalid move payload", "")
		return
	}

	move, err := h.gameManager.MakeMove(movePayload.GameID, playerID, movePayload.Column)
	if err != nil {
		// Get current game state for error response
		gameInstance, _ := h.gameManager.GetGame(movePayload.GameID)
		
		conn.WriteJSON(models.NewWSMessage(models.MsgMoveResult, models.MoveResultPayload{
			Success:    false,
			Error:      err.Error(),
			GameState:  gameInstance,
			IsGameOver: gameInstance != nil && gameInstance.State == models.GameStateFinished,
		}))
		return
	}

	// Get updated game state
	gameInstance, _ := h.gameManager.GetGame(movePayload.GameID)

	// Prepare move result payload
	moveResult := models.MoveResultPayload{
		Success:    true,
		Move:       move,
		GameState:  gameInstance,
		IsGameOver: gameInstance.State == models.GameStateFinished,
		NextTurn:   int(gameInstance.CurrentTurn),
	}

	// Add win result if game is finished
	if gameInstance.State == models.GameStateFinished {
		// Note: WinResult is not available in models.Game, would need to be added or calculated
		// moveResult.WinResult = gameInstance.WinResult
	}

	// Send move result to all players
	h.gameManager.BroadcastToGame(movePayload.GameID, models.NewWSMessage(models.MsgMoveResult, moveResult))

	// Send analytics event
	h.analyticsService.SendEvent("move_made", map[string]interface{}{
		"game_id":   movePayload.GameID.String(),
		"player_id": playerID.String(),
		"column":    movePayload.Column,
		"row":       move.Row,
	})

	// Check if game ended
	if gameInstance.State == models.GameStateFinished {
		gameEndPayload := models.GameEndPayload{
			GameID:    gameInstance.ID,
			Reason:    "game_completed",
			GameState: gameInstance,
			Duration:  int(gameInstance.FinishedAt.Sub(gameInstance.CreatedAt).Seconds()),
			IsDraw:    gameInstance.Winner == nil,
		}

		if gameInstance.Winner != nil {
			// Convert PlayerColor to Player
			winnerColor := *gameInstance.Winner
			if winnerColor == models.PlayerRed {
				gameEndPayload.Winner = gameInstance.Players[0]
			} else if winnerColor == models.PlayerYellow {
				gameEndPayload.Winner = gameInstance.Players[1]
			}
		}

		h.gameManager.BroadcastToGame(movePayload.GameID, models.NewWSMessage(models.MsgGameEnd, gameEndPayload))

		// Send analytics event
		reason := "draw"
		if gameInstance.Winner != nil {
			reason = "win"
		}

		h.analyticsService.SendEvent("game_ended", map[string]interface{}{
			"game_id":  movePayload.GameID.String(),
			"winner":   gameInstance.Winner,
			"reason":   reason,
			"duration": gameInstance.FinishedAt.Sub(gameInstance.CreatedAt).Seconds(),
		})
	}
}

func (h *GameHandler) handleReconnect(conn *websocket.Conn, payload interface{}) (uuid.UUID, uuid.UUID) {
	var reconnectPayload models.ReconnectPayload
	if err := h.parsePayload(payload, &reconnectPayload); err != nil {
		h.sendError(conn, "INVALID_PAYLOAD", "Invalid reconnect payload", "")
		return uuid.Nil, uuid.Nil
	}

	// Verify game and player exist
	gameInstance, exists := h.gameManager.GetGame(reconnectPayload.GameID)
	if !exists {
		h.sendError(conn, "GAME_NOT_FOUND", "Game not found", "")
		return uuid.Nil, uuid.Nil
	}

	// Verify player is in the game
	var playerInGame bool
	for _, player := range gameInstance.Players {
		if player.ID == reconnectPayload.PlayerID {
			playerInGame = true
			break
		}
	}

	if !playerInGame {
		h.sendError(conn, "PLAYER_NOT_IN_GAME", "Player not in game", "")
		return uuid.Nil, uuid.Nil
	}

	// Re-establish connection
	h.gameManager.AddPlayerConnection(reconnectPayload.PlayerID, reconnectPayload.GameID, conn)

	// Send reconnect success message
	conn.WriteJSON(models.NewWSMessage(models.MsgReconnectSuccess, models.ReconnectSuccessPayload{
		GameID:         reconnectPayload.GameID,
		PlayerID:       reconnectPayload.PlayerID,
		GameState:      gameInstance,
		QueuedMessages: 0, // TODO: Implement message queuing
		Message:        "Successfully reconnected to game",
	}))

	// Send analytics event
	h.analyticsService.SendEvent("player_reconnected", map[string]interface{}{
		"game_id":   reconnectPayload.GameID.String(),
		"player_id": reconnectPayload.PlayerID.String(),
	})

	return reconnectPayload.PlayerID, reconnectPayload.GameID
}

func (h *GameHandler) handleHeartbeat(conn *websocket.Conn, playerID uuid.UUID) {
	if playerID != uuid.Nil {
		if playerConn, exists := h.gameManager.GetPlayerConnection(playerID); exists {
			// Update last seen time
			playerConn.LastSeen = time.Now()
		}
	}

	// Send heartbeat acknowledgment
	conn.WriteJSON(models.NewWSMessage(models.MsgHeartbeatAck, map[string]interface{}{
		"server_time":   time.Now(),
		"connection_id": playerID.String(),
	}))
}

func (h *GameHandler) sendError(conn *websocket.Conn, code, message, details string) {
	conn.WriteJSON(models.NewWSMessage(models.MsgError, models.ErrorPayload{
		Code:    code,
		Message: message,
		Details: details,
	}))
}

func (h *GameHandler) parsePayload(payload interface{}, target interface{}) error {
	// Convert payload to JSON and back to parse into target struct
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	return json.Unmarshal(jsonData, target)
}