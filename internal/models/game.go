package models

import (
	"time"

	"github.com/google/uuid"
)

type GameState int

const (
	GameStateWaiting GameState = iota
	GameStatePlaying
	GameStateFinished
)

type PlayerColor int

const (
	PlayerRed PlayerColor = iota
	PlayerYellow
)

type Player struct {
	ID       uuid.UUID   `json:"id"`
	Name     string      `json:"name"`
	Color    PlayerColor `json:"color"`
	Number   int         `json:"number"` // 1 for Red, 2 for Yellow (for frontend compatibility)
	IsBot    bool        `json:"is_bot"`
	Connected bool       `json:"connected"`
	LastSeen time.Time   `json:"last_seen"`
}

type Game struct {
	ID          uuid.UUID   `json:"id"`
	State       GameState   `json:"state"`
	Board       [6][7]int   `json:"board"` // 6 rows, 7 columns
	Players     [2]*Player  `json:"players"`
	CurrentTurn PlayerColor `json:"current_turn"`
	CurrentTurnNumber int   `json:"current_turn_number"` // 1 for Red, 2 for Yellow (for frontend)
	Winner      *PlayerColor `json:"winner,omitempty"`
	CreatedAt   time.Time   `json:"created_at"`
	FinishedAt  *time.Time  `json:"finished_at,omitempty"`
	LastMove    *Move       `json:"last_move,omitempty"`
}

type Move struct {
	PlayerID uuid.UUID   `json:"player_id"`
	Column   int         `json:"column"`
	Row      int         `json:"row"`
	Color    PlayerColor `json:"color"`
	Timestamp time.Time  `json:"timestamp"`
}

type GameResult struct {
	GameID     uuid.UUID  `json:"game_id"`
	WinnerID   *uuid.UUID `json:"winner_id,omitempty"`
	LoserID    *uuid.UUID `json:"loser_id,omitempty"`
	IsDraw     bool       `json:"is_draw"`
	Duration   int        `json:"duration_seconds"`
	TotalMoves int        `json:"total_moves"`
	CreatedAt  time.Time  `json:"created_at"`
}

type WinResult struct {
	Winner     *Player `json:"winner,omitempty"`
	WinType    string  `json:"win_type"` // "horizontal", "vertical", "diagonal_positive", "diagonal_negative", "forfeit"
	WinLine    []int   `json:"win_line,omitempty"` // Coordinates of winning line [row1, col1, row2, col2, row3, col3, row4, col4]
	IsDraw     bool    `json:"is_draw"`
	GameState  *Game   `json:"game_state"`
}

// Board methods
func (g *Game) IsValidMove(column int) bool {
	if column < 0 || column >= 7 {
		return false
	}
	return g.Board[0][column] == 0 // Top row must be empty
}

func (g *Game) MakeMove(column int, color PlayerColor) *Move {
	if !g.IsValidMove(column) {
		return nil
	}

	// Find the lowest empty row in the column
	row := -1
	for r := 5; r >= 0; r-- {
		if g.Board[r][column] == 0 {
			row = r
			break
		}
	}

	if row == -1 {
		return nil
	}

	// Place the piece
	g.Board[row][column] = int(color) + 1 // Store as 1 or 2

	move := &Move{
		Column:    column,
		Row:       row,
		Color:     color,
		Timestamp: time.Now(),
	}

	g.LastMove = move
	return move
}

func (g *Game) CheckWinner() *PlayerColor {
	// Check horizontal, vertical, and diagonal wins
	for row := 0; row < 6; row++ {
		for col := 0; col < 7; col++ {
			if g.Board[row][col] == 0 {
				continue
			}

			player := g.Board[row][col]

			// Check horizontal (right)
			if col <= 3 && g.checkLine(row, col, 0, 1, player) {
				color := PlayerColor(player - 1)
				return &color
			}

			// Check vertical (down)
			if row <= 2 && g.checkLine(row, col, 1, 0, player) {
				color := PlayerColor(player - 1)
				return &color
			}

			// Check diagonal (down-right)
			if row <= 2 && col <= 3 && g.checkLine(row, col, 1, 1, player) {
				color := PlayerColor(player - 1)
				return &color
			}

			// Check diagonal (down-left)
			if row <= 2 && col >= 3 && g.checkLine(row, col, 1, -1, player) {
				color := PlayerColor(player - 1)
				return &color
			}
		}
	}

	return nil
}

func (g *Game) checkLine(startRow, startCol, deltaRow, deltaCol, player int) bool {
	for i := 0; i < 4; i++ {
		row := startRow + i*deltaRow
		col := startCol + i*deltaCol
		if row < 0 || row >= 6 || col < 0 || col >= 7 || g.Board[row][col] != player {
			return false
		}
	}
	return true
}

func (g *Game) IsBoardFull() bool {
	for col := 0; col < 7; col++ {
		if g.Board[0][col] == 0 {
			return false
		}
	}
	return true
}