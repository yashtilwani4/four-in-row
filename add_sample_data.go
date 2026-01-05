package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// Get database URL from environment
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	// Connect to database
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		log.Fatalf("❌ Failed to connect to database: %v", err)
	}
	defer db.Close()

	fmt.Println("🎮 Adding sample game data...")

	// Insert sample games
	sampleGames := []string{
		`INSERT INTO games (
			player1_id, player1_name, player1_is_bot,
			player2_id, player2_name, player2_is_bot,
			winner_id, winner_name, is_draw,
			total_moves, duration_seconds, win_type,
			final_board, started_at, finished_at
		) VALUES (
			'550e8400-e29b-41d4-a716-446655440001', 'Alice', false,
			'550e8400-e29b-41d4-a716-446655440002', 'ConnectBot', true,
			'550e8400-e29b-41d4-a716-446655440001', 'Alice', false,
			15, 180, 'horizontal',
			'[[0,0,0,0,0,0,0],[0,0,0,0,0,0,0],[0,0,1,1,1,1,0],[0,0,2,1,2,2,0],[0,2,1,2,1,1,0],[2,1,2,1,2,1,0]]'::jsonb,
			NOW() - INTERVAL '3 minutes', NOW() - INTERVAL '1 minute'
		)`,
		`INSERT INTO games (
			player1_id, player1_name, player1_is_bot,
			player2_id, player2_name, player2_is_bot,
			winner_id, winner_name, is_draw,
			total_moves, duration_seconds, win_type,
			final_board, started_at, finished_at
		) VALUES (
			'550e8400-e29b-41d4-a716-446655440003', 'Bob', false,
			'550e8400-e29b-41d4-a716-446655440002', 'ConnectBot', true,
			'550e8400-e29b-41d4-a716-446655440002', 'ConnectBot', false,
			12, 145, 'vertical',
			'[[0,0,0,0,0,0,0],[0,0,0,0,0,0,0],[0,0,2,0,0,0,0],[0,0,2,0,0,0,0],[0,1,2,1,0,0,0],[1,1,2,1,0,0,0]]'::jsonb,
			NOW() - INTERVAL '5 minutes', NOW() - INTERVAL '2 minutes'
		)`,
		`INSERT INTO games (
			player1_id, player1_name, player1_is_bot,
			player2_id, player2_name, player2_is_bot,
			winner_id, winner_name, is_draw,
			total_moves, duration_seconds, win_type,
			final_board, started_at, finished_at
		) VALUES (
			'550e8400-e29b-41d4-a716-446655440004', 'Charlie', false,
			'550e8400-e29b-41d4-a716-446655440005', 'Diana', false,
			NULL, NULL, true,
			42, 420, NULL,
			'[[1,2,1,2,1,2,1],[2,1,2,1,2,1,2],[1,2,1,2,1,2,1],[2,1,2,1,2,1,2],[1,2,1,2,1,2,1],[2,1,2,1,2,1,2]]'::jsonb,
			NOW() - INTERVAL '10 minutes', NOW() - INTERVAL '3 minutes'
		)`,
		`INSERT INTO games (
			player1_id, player1_name, player1_is_bot,
			player2_id, player2_name, player2_is_bot,
			winner_id, winner_name, is_draw,
			total_moves, duration_seconds, win_type,
			final_board, started_at, finished_at
		) VALUES (
			'550e8400-e29b-41d4-a716-446655440001', 'Alice', false,
			'550e8400-e29b-41d4-a716-446655440003', 'Bob', false,
			'550e8400-e29b-41d4-a716-446655440001', 'Alice', false,
			18, 220, 'diagonal_positive',
			'[[0,0,0,0,0,0,0],[0,0,0,0,0,0,0],[0,0,0,1,0,0,0],[0,0,1,2,0,0,0],[0,1,2,1,0,0,0],[1,2,1,2,0,0,0]]'::jsonb,
			NOW() - INTERVAL '15 minutes', NOW() - INTERVAL '12 minutes'
		)`,
		`INSERT INTO games (
			player1_id, player1_name, player1_is_bot,
			player2_id, player2_name, player2_is_bot,
			winner_id, winner_name, is_draw,
			total_moves, duration_seconds, win_type,
			final_board, started_at, finished_at
		) VALUES (
			'550e8400-e29b-41d4-a716-446655440003', 'Bob', false,
			'550e8400-e29b-41d4-a716-446655440005', 'Diana', false,
			'550e8400-e29b-41d4-a716-446655440005', 'Diana', false,
			21, 280, 'vertical',
			'[[0,0,0,0,0,0,0],[0,0,0,0,0,0,0],[0,0,0,2,0,0,0],[0,0,0,2,0,0,0],[0,0,1,2,1,0,0],[1,1,2,2,1,0,0]]'::jsonb,
			NOW() - INTERVAL '20 minutes', NOW() - INTERVAL '16 minutes'
		)`,
	}

	for i, gameSQL := range sampleGames {
		_, err := db.Exec(gameSQL)
		if err != nil {
			log.Printf("⚠️  Failed to insert sample game %d: %v", i+1, err)
		} else {
			fmt.Printf("✅ Inserted sample game %d\n", i+1)
		}
	}

	// Check results
	var gameCount int
	err = db.QueryRow("SELECT COUNT(*) FROM games").Scan(&gameCount)
	if err != nil {
		log.Printf("⚠️  Could not count games: %v", err)
	} else {
		fmt.Printf("✅ Total games in database: %d\n", gameCount)
	}

	var playerCount int
	err = db.QueryRow("SELECT COUNT(*) FROM leaderboard").Scan(&playerCount)
	if err != nil {
		log.Printf("⚠️  Could not count leaderboard entries: %v", err)
	} else {
		fmt.Printf("✅ Total players in leaderboard: %d\n", playerCount)
	}

	// Show leaderboard
	if playerCount > 0 {
		rows, err := db.Query(`
			SELECT username, wins, losses, draws, win_rate 
			FROM leaderboard 
			ORDER BY win_rate DESC, wins DESC
		`)
		if err != nil {
			log.Printf("⚠️  Could not query leaderboard: %v", err)
		} else {
			defer rows.Close()
			fmt.Println("\n🏆 Current Leaderboard:")
			fmt.Println("Username\t\tWins\tLosses\tDraws\tWin Rate")
			fmt.Println("------------------------------------------------")
			
			for rows.Next() {
				var username string
				var wins, losses, draws int
				var winRate float64
				
				if err := rows.Scan(&username, &wins, &losses, &draws, &winRate); err != nil {
					log.Printf("Error scanning row: %v", err)
					continue
				}
				
				fmt.Printf("%-15s\t%d\t%d\t%d\t%.2f%%\n", username, wins, losses, draws, winRate)
			}
		}
	}

	fmt.Println("\n🎉 Sample data added successfully!")
}