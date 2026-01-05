package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
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

	fmt.Printf("🔗 Connecting to database...\n")

	// Connect to database
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		log.Fatalf("❌ Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatalf("❌ Failed to ping database: %v", err)
	}

	fmt.Println("✅ Database connection successful!")

	// Read schema file
	schemaContent, err := ioutil.ReadFile("internal/database/schema.sql")
	if err != nil {
		log.Fatalf("❌ Failed to read schema file: %v", err)
	}

	fmt.Println("📄 Executing database schema...")

	// Execute schema
	_, err = db.Exec(string(schemaContent))
	if err != nil {
		log.Fatalf("❌ Failed to execute schema: %v", err)
	}

	fmt.Println("✅ Database schema executed successfully!")

	// Verify tables were created
	var tableCount int
	err = db.QueryRow(`
		SELECT COUNT(*) 
		FROM information_schema.tables 
		WHERE table_schema = 'public' 
		AND table_name IN ('games', 'leaderboard', 'game_moves')
	`).Scan(&tableCount)
	
	if err != nil {
		log.Fatalf("❌ Failed to check tables: %v", err)
	}

	fmt.Printf("✅ Created %d tables successfully\n", tableCount)

	// Check for sample data
	var gameCount int
	err = db.QueryRow("SELECT COUNT(*) FROM games").Scan(&gameCount)
	if err != nil {
		log.Printf("⚠️  Could not count games: %v", err)
	} else {
		fmt.Printf("✅ Found %d sample games in database\n", gameCount)
	}

	// Check leaderboard
	var playerCount int
	err = db.QueryRow("SELECT COUNT(*) FROM leaderboard").Scan(&playerCount)
	if err != nil {
		log.Printf("⚠️  Could not count leaderboard entries: %v", err)
	} else {
		fmt.Printf("✅ Found %d players in leaderboard\n", playerCount)
	}

	// Show sample leaderboard data
	if playerCount > 0 {
		rows, err := db.Query(`
			SELECT username, wins, losses, win_rate 
			FROM leaderboard 
			ORDER BY win_rate DESC 
			LIMIT 5
		`)
		if err != nil {
			log.Printf("⚠️  Could not query leaderboard: %v", err)
		} else {
			defer rows.Close()
			fmt.Println("\n🏆 Sample Leaderboard:")
			fmt.Println("Username\t\tWins\tLosses\tWin Rate")
			fmt.Println("----------------------------------------")
			
			for rows.Next() {
				var username string
				var wins, losses int
				var winRate float64
				
				if err := rows.Scan(&username, &wins, &losses, &winRate); err != nil {
					log.Printf("Error scanning row: %v", err)
					continue
				}
				
				fmt.Printf("%-15s\t%d\t%d\t%.2f%%\n", username, wins, losses, winRate)
			}
		}
	}

	fmt.Println("\n🎉 Database setup complete!")
	fmt.Println("📝 Next steps:")
	fmt.Println("   1. Run: go run cmd/server/main.go")
	fmt.Println("   2. Test API: curl http://localhost:8080/api/leaderboard")
	fmt.Println("   3. Open frontend: http://localhost:3000")
}