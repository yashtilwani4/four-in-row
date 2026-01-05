package database

import (
	"database/sql"
	"fmt"

	"connect-four-backend/internal/models"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// Repository provides database operations
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new repository
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// SaveCompletedGame saves a completed game to the database
func (r *Repository) SaveCompletedGame(game *models.Game) error {
	if game == nil || game.State != models.GameStateFinished {
		return fmt.Errorf("invalid game state")
	}

	// Count moves on the board
	totalMoves := 0
	for i := 0; i < 6; i++ {
		for j := 0; j < 7; j++ {
			if game.Board[i][j] != 0 {
				totalMoves++
			}
		}
	}

	// Determine winner
	var winnerID *uuid.UUID
	var winnerName *string
	isDraw := game.Winner == nil

	if !isDraw {
		if *game.Winner == models.PlayerRed {
			winnerID = &game.Players[0].ID
			winnerName = &game.Players[0].Name
		} else {
			winnerID = &game.Players[1].ID
			winnerName = &game.Players[1].Name
		}
	}

	duration := int(game.FinishedAt.Sub(game.CreatedAt).Seconds())

	query := `
		INSERT INTO games (
			id, player1_id, player1_name, player1_is_bot,
			player2_id, player2_name, player2_is_bot,
			winner_id, winner_name, is_draw,
			total_moves, duration_seconds,
			created_at, finished_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`

	_, err := r.db.Exec(query,
		game.ID,
		game.Players[0].ID, game.Players[0].Name, game.Players[0].IsBot,
		game.Players[1].ID, game.Players[1].Name, game.Players[1].IsBot,
		winnerID, winnerName, isDraw,
		totalMoves, duration,
		game.CreatedAt, game.FinishedAt,
	)

	return err
}

// GetLeaderboard returns the current leaderboard
func (r *Repository) GetLeaderboard(limit int) ([]LeaderboardEntry, error) {
	query := `SELECT * FROM leaderboard ORDER BY wins DESC, win_rate DESC LIMIT $1`
	
	rows, err := r.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []LeaderboardEntry
	for rows.Next() {
		var entry LeaderboardEntry
		err := rows.Scan(
			&entry.Rank, &entry.Username, &entry.PlayerName,
			&entry.TotalGames, &entry.Wins, &entry.Losses, &entry.Draws,
			&entry.WinRate, &entry.AverageGameDuration, &entry.TotalPlaytimeSeconds,
			&entry.HorizontalWins, &entry.VerticalWins, &entry.DiagonalWins,
			&entry.LastGameAt,
		)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// Close closes the database connection
func (r *Repository) Close() error {
	return r.db.Close()
}