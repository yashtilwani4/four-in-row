package game

import (
	"math/rand"
	"time"

	"connect-four-backend/internal/models"

	"github.com/google/uuid"
)

type Bot struct {
	ID   uuid.UUID
	Name string
}

func NewBot() *models.Player {
	return &models.Player{
		ID:        uuid.New(),
		Name:      "ConnectBot",
		IsBot:     true,
		Connected: true,
		LastSeen:  time.Now(),
	}
}

// GetBestMove implements a simple AI strategy
func GetBestMove(game *models.Game, botColor models.PlayerColor) int {
	// Strategy priority:
	// 1. Win if possible
	// 2. Block opponent from winning
	// 3. Play center columns (better positioning)
	// 4. Random valid move

	// Check for winning move
	if move := findWinningMove(game, botColor); move != -1 {
		return move
	}

	// Check for blocking move
	opponentColor := models.PlayerRed
	if botColor == models.PlayerRed {
		opponentColor = models.PlayerYellow
	}
	if move := findWinningMove(game, opponentColor); move != -1 {
		return move
	}

	// Prefer center columns
	centerColumns := []int{3, 2, 4, 1, 5, 0, 6}
	for _, col := range centerColumns {
		if game.IsValidMove(col) {
			return col
		}
	}

	// Fallback to random valid move
	validMoves := make([]int, 0)
	for col := 0; col < 7; col++ {
		if game.IsValidMove(col) {
			validMoves = append(validMoves, col)
		}
	}

	if len(validMoves) > 0 {
		return validMoves[rand.Intn(len(validMoves))]
	}

	return -1 // No valid moves
}

func findWinningMove(game *models.Game, color models.PlayerColor) int {
	// Try each column to see if it results in a win
	for col := 0; col < 7; col++ {
		if !game.IsValidMove(col) {
			continue
		}

		// Create a copy of the game to test the move
		testGame := *game
		testGame.Board = game.Board // Copy the board

		// Make the test move
		move := testGame.MakeMove(col, color)
		if move == nil {
			continue
		}

		// Check if this move wins
		if winner := testGame.CheckWinner(); winner != nil && *winner == color {
			return col
		}
	}

	return -1 // No winning move found
}