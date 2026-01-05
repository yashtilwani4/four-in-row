package database

import (
	"database/sql"
	"fmt"
	"time"

	"connect-four-backend/internal/models"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// PostgresDB provides legacy database operations (deprecated - use Repository instead)
// This is kept for backward compatibility
type PostgresDB struct {
	db *sql.DB
}

// Legacy types for backward compatibility
type LeaderboardEntry struct {
	Rank                    int        `json:"rank"`
	Username                string     `json:"username"`
	PlayerName              string     `json:"player_name"` // Alias for Username for backward compatibility
	TotalGames              int        `json:"total_games"`
	Wins                    int        `json:"wins"`
	Losses                  int        `json:"losses"`
	Draws                   int        `json:"draws"`
	WinRate                 float64    `json:"win_rate"`
	AverageGameDuration     float64    `json:"average_game_duration"`
	TotalPlaytimeSeconds    int64      `json:"total_playtime_seconds"`
	HorizontalWins          int        `json:"horizontal_wins"`
	VerticalWins            int        `json:"vertical_wins"`
	DiagonalWins            int        `json:"diagonal_wins"`
	ForfeitWins             int        `json:"forfeit_wins"`
	WinsVsHumans            int        `json:"wins_vs_humans"`
	WinsVsBots              int        `json:"wins_vs_bots"`
	LossesVsHumans          int        `json:"losses_vs_humans"`
	LossesVsBots            int        `json:"losses_vs_bots"`
	CurrentWinStreak        int        `json:"current_win_streak"`
	LongestWinStreak        int        `json:"longest_win_streak"`
	FirstGameAt             *time.Time `json:"first_game_at,omitempty"`
	LastGameAt              *time.Time `json:"last_game_at,omitempty"`
}

type PlayerStats struct {
	PlayerName          string  `json:"player_name"`
	TotalGames          int     `json:"total_games"`
	Wins                int     `json:"wins"`
	Losses              int     `json:"losses"`
	Draws               int     `json:"draws"`
	WinRate             float64 `json:"win_rate"`
	AverageGameDuration float64 `json:"average_game_duration"`
}

// NewPostgresDB creates a new PostgresDB instance (deprecated - use NewRepository instead)
func NewPostgresDB(databaseURL string) (*PostgresDB, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	pgDB := &PostgresDB{db: db}
	if err := pgDB.createTables(); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return pgDB, nil
}

func (p *PostgresDB) Close() error {
	return p.db.Close()
}

// createTables creates basic tables (simplified version)
func (p *PostgresDB) createTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS games (
			id UUID PRIMARY KEY,
			player1_id UUID NOT NULL,
			player1_name VARCHAR(255) NOT NULL,
			player2_id UUID NOT NULL,
			player2_name VARCHAR(255) NOT NULL,
			winner_id UUID,
			is_draw BOOLEAN DEFAULT FALSE,
			duration_seconds INTEGER NOT NULL,
			total_moves INTEGER NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			finished_at TIMESTAMP WITH TIME ZONE NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_games_player1 ON games(player1_id)`,
		`CREATE INDEX IF NOT EXISTS idx_games_player2 ON games(player2_id)`,
		`CREATE INDEX IF NOT EXISTS idx_games_winner ON games(winner_id)`,
		`CREATE INDEX IF NOT EXISTS idx_games_created_at ON games(created_at)`,
	}

	for _, query := range queries {
		if _, err := p.db.Exec(query); err != nil {
			return fmt.Errorf("failed to execute query: %w", err)
		}
	}

	return nil
}

// SaveGameResult saves a game result (simplified version)
func (p *PostgresDB) SaveGameResult(result *models.GameResult) error {
	query := `
		INSERT INTO games (id, player1_id, player1_name, player2_id, player2_name, winner_id, is_draw, duration_seconds, total_moves, created_at, finished_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	// Note: This is a simplified version for backward compatibility
	// For full functionality, use the Repository instead
	_, err := p.db.Exec(query,
		result.GameID,
		uuid.New(), // placeholder - you'd get this from the game
		"Player1",  // placeholder - you'd get this from the game
		uuid.New(), // placeholder - you'd get this from the game
		"Player2",  // placeholder - you'd get this from the game
		result.WinnerID,
		result.IsDraw,
		result.Duration,
		result.TotalMoves,
		result.CreatedAt,
		time.Now(),
	)

	if err != nil {
		return fmt.Errorf("failed to save game result: %w", err)
	}

	return nil
}

// GetLeaderboard retrieves leaderboard (simplified version)
func (p *PostgresDB) GetLeaderboard(limit int) ([]LeaderboardEntry, error) {
	query := `
		WITH player_stats AS (
			SELECT 
				player_name,
				COUNT(*) as total_games,
				SUM(CASE WHEN winner_name = player_name THEN 1 ELSE 0 END) as wins,
				SUM(CASE WHEN winner_name != player_name AND NOT is_draw THEN 1 ELSE 0 END) as losses,
				SUM(CASE WHEN is_draw THEN 1 ELSE 0 END) as draws
			FROM (
				SELECT player1_name as player_name, 
					   CASE WHEN winner_id = player1_id THEN player1_name
							WHEN winner_id = player2_id THEN player2_name
							ELSE NULL END as winner_name,
					   is_draw
				FROM games
				UNION ALL
				SELECT player2_name as player_name,
					   CASE WHEN winner_id = player1_id THEN player1_name
							WHEN winner_id = player2_id THEN player2_name
							ELSE NULL END as winner_name,
					   is_draw
				FROM games
			) all_games
			GROUP BY player_name
		)
		SELECT 
			player_name,
			wins,
			losses,
			draws,
			CASE WHEN total_games > 0 THEN ROUND((wins::numeric / total_games::numeric) * 100, 2) ELSE 0 END as win_rate
		FROM player_stats
		WHERE total_games >= 1  -- Only show players with at least 1 game
		ORDER BY win_rate DESC, wins DESC
		LIMIT $1
	`

	rows, err := p.db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query leaderboard: %w", err)
	}
	defer rows.Close()

	var leaderboard []LeaderboardEntry
	for rows.Next() {
		var entry LeaderboardEntry
		if err := rows.Scan(&entry.PlayerName, &entry.Wins, &entry.Losses, &entry.Draws, &entry.WinRate); err != nil {
			return nil, fmt.Errorf("failed to scan leaderboard entry: %w", err)
		}
		leaderboard = append(leaderboard, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating leaderboard rows: %w", err)
	}

	return leaderboard, nil
}

// GetPlayerStats retrieves player statistics (simplified version)
func (p *PostgresDB) GetPlayerStats(playerName string) (*PlayerStats, error) {
	query := `
		WITH player_games AS (
			SELECT 
				CASE WHEN winner_id = player1_id THEN player1_name
					 WHEN winner_id = player2_id THEN player2_name
					 ELSE NULL END as winner_name,
				is_draw,
				duration_seconds
			FROM games
			WHERE player1_name = $1 OR player2_name = $1
		)
		SELECT 
			COUNT(*) as total_games,
			SUM(CASE WHEN winner_name = $1 THEN 1 ELSE 0 END) as wins,
			SUM(CASE WHEN winner_name != $1 AND NOT is_draw THEN 1 ELSE 0 END) as losses,
			SUM(CASE WHEN is_draw THEN 1 ELSE 0 END) as draws,
			CASE WHEN COUNT(*) > 0 THEN ROUND((SUM(CASE WHEN winner_name = $1 THEN 1 ELSE 0 END)::float / COUNT(*)::float) * 100, 2) ELSE 0 END as win_rate,
			CASE WHEN COUNT(*) > 0 THEN ROUND(AVG(duration_seconds), 2) ELSE 0 END as avg_duration
		FROM player_games
	`

	var stats PlayerStats
	stats.PlayerName = playerName

	err := p.db.QueryRow(query, playerName).Scan(
		&stats.TotalGames,
		&stats.Wins,
		&stats.Losses,
		&stats.Draws,
		&stats.WinRate,
		&stats.AverageGameDuration,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("player not found: %s", playerName)
		}
		return nil, fmt.Errorf("failed to get player stats: %w", err)
	}

	return &stats, nil
}