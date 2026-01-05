package bot

import (
	"math/rand"
	"time"

	"connect-four-backend/internal/models"

	"github.com/google/uuid"
)

// NewBot creates a simple AI bot
func NewBot() *models.Player {
	return &models.Player{
		ID:        uuid.New(),
		Name:      "ConnectBot",
		IsBot:     true,
		Connected: true,
		LastSeen:  time.Now(),
	}
}

// GetBestMove picks the best move for the bot
// Strategy: Win > Block > Center > Random
func GetBestMove(game *models.Game, botColor models.PlayerColor) int {
	// Try to win first
	if move := findWinningMove(game, botColor); move != -1 {
		return move
	}

	// Block opponent from winning
	opponentColor := models.PlayerRed
	if botColor == models.PlayerRed {
		opponentColor = models.PlayerYellow
	}
	if move := findWinningMove(game, opponentColor); move != -1 {
		return move
	}

	// Prefer center columns (better strategy)
	centerColumns := []int{3, 2, 4, 1, 5, 0, 6}
	for _, col := range centerColumns {
		if game.IsValidMove(col) {
			return col
		}
	}

	// Fallback to any valid move
	validMoves := make([]int, 0)
	for col := 0; col < 7; col++ {
		if game.IsValidMove(col) {
			validMoves = append(validMoves, col)
		}
	}

	if len(validMoves) > 0 {
		return validMoves[rand.Intn(len(validMoves))]
	}

	return -1 // No valid moves (shouldn't happen)
}

// findWinningMove checks if we can win in one move
func findWinningMove(game *models.Game, color models.PlayerColor) int {
	for col := 0; col < 7; col++ {
		if !game.IsValidMove(col) {
			continue
		}

		// Try this move and see if it wins
		testGame := *game
		testGame.Board = game.Board // Copy board state

		move := testGame.MakeMove(col, color)
		if move == nil {
			continue
		}

		// Check if this move creates a win
		if winner := testGame.CheckWinner(); winner != nil && *winner == color {
			return col
		}
	}

	return -1 // No winning move found
}