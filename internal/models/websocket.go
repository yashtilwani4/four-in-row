package models

import (
	"time"
	"github.com/google/uuid"
)

type MessageType string

const (
	// Client messages
	MsgJoinQueue    MessageType = "join_queue"
	MsgLeaveQueue   MessageType = "leave_queue"
	MsgMakeMove     MessageType = "make_move"
	MsgReconnect    MessageType = "reconnect"
	MsgHeartbeat    MessageType = "heartbeat"
	MsgGetGameState MessageType = "get_game_state"

	// Server messages
	MsgGameFound          MessageType = "game_found"
	MsgGameState          MessageType = "game_state"
	MsgMoveResult         MessageType = "move_result"
	MsgGameEnd            MessageType = "game_end"
	MsgError              MessageType = "error"
	MsgPlayerJoined       MessageType = "player_joined"
	MsgPlayerLeft         MessageType = "player_left"
	MsgTurnChanged        MessageType = "turn_changed"
	MsgHeartbeatAck       MessageType = "heartbeat_ack"
	MsgBotMove            MessageType = "bot_move"
	MsgReconnectSuccess   MessageType = "reconnect_success"
	MsgPlayerDisconnected MessageType = "player_disconnected"
	MsgPlayerReconnected  MessageType = "player_reconnected"
)

type WSMessage struct {
	Type      MessageType `json:"type"`
	Payload   interface{} `json:"payload,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
	MessageID string      `json:"message_id"`
}

// Payload structs for different message types
type JoinQueuePayload struct {
	PlayerName string `json:"player_name"`
}

type MakeMovePayload struct {
	GameID uuid.UUID `json:"game_id"`
	Column int       `json:"column"`
}

type ReconnectPayload struct {
	GameID   uuid.UUID `json:"game_id"`
	PlayerID uuid.UUID `json:"player_id"`
	Username string    `json:"username"`
	LastSeen time.Time `json:"last_seen,omitempty"`
}

type GetGameStatePayload struct {
	GameID uuid.UUID `json:"game_id"`
}

type GameFoundPayload struct {
	Game     *Game     `json:"game"`
	PlayerID uuid.UUID `json:"player_id"`
}

type MoveResultPayload struct {
	Success      bool          `json:"success"`
	Move         *Move         `json:"move,omitempty"`
	GameState    *Game         `json:"game_state"`
	Error        string        `json:"error,omitempty"`
	IsGameOver   bool          `json:"is_game_over"`
	WinResult    *WinResult    `json:"win_result,omitempty"`
	NextTurn     int           `json:"next_turn,omitempty"`
}

type GameEndPayload struct {
	GameID    uuid.UUID     `json:"game_id"`
	Winner    *Player       `json:"winner,omitempty"`
	Reason    string        `json:"reason"`
	GameState *Game         `json:"game_state"`
	Duration  int           `json:"duration"`
	IsDraw    bool          `json:"is_draw"`
}

type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

type BotMovePayload struct {
	GameID     uuid.UUID `json:"game_id"`
	Move       *Move     `json:"move"`
	Reasoning  string    `json:"reasoning"`
	Confidence int       `json:"confidence"`
	GameState  *Game     `json:"game_state"`
}

type ReconnectSuccessPayload struct {
	GameID         uuid.UUID `json:"game_id"`
	PlayerID       uuid.UUID `json:"player_id"`
	GameState      *Game     `json:"game_state"`
	QueuedMessages int       `json:"queued_messages"`
	Message        string    `json:"message"`
}

type PlayerDisconnectedPayload struct {
	Player               *Player   `json:"player"`
	DisconnectTime       time.Time `json:"disconnect_time"`
	Reason               string    `json:"reason"`
	GameState            string    `json:"game_state"`
	MoveNumber           int       `json:"move_number"`
	GracePeriodSeconds   int       `json:"grace_period_seconds"`
}

type PlayerReconnectedPayload struct {
	Player             *Player   `json:"player"`
	ReconnectTime      time.Time `json:"reconnect_time"`
	DisconnectTime     time.Time `json:"disconnect_time"`
	OfflineDurationMs  int64     `json:"offline_duration_ms"`
	MissedMoves        int       `json:"missed_moves"`
	GameState          string    `json:"game_state"`
}

// Helper to create WebSocket messages
func NewWSMessage(msgType MessageType, payload interface{}) WSMessage {
	return WSMessage{
		Type:      msgType,
		Payload:   payload,
		Timestamp: time.Now(),
		MessageID: uuid.New().String(),
	}
}