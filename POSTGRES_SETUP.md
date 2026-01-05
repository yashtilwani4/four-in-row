# PostgreSQL Setup Notes

Had to set up PostgreSQL for the leaderboard. Here's what I did:

## Quick Setup

1. **Install PostgreSQL** (I used pgAdmin4)
2. **Create database** called `connect_four`
3. **Run the setup script**: `go run setup_database.go`

## Database Schema

Two main tables:

### games table
Stores completed games with winner, duration, moves, etc.

### leaderboard view  
Auto-calculates player stats from the games table. Updates automatically when new games are added.

## Connection

Update your `.env` file:
```
DATABASE_URL=postgres://username:password@localhost:5432/connect_four
```

## Testing

```bash
# Test connection
go run test_db_connection.go

# Add some sample data
go run add_sample_data.go
```

## Troubleshooting

- Make sure PostgreSQL is running
- Check your username/password in the connection string
- Database name should be `connect_four`
- Default port is 5432

The setup script creates everything automatically, including indexes and triggers for the leaderboard updates.
    win_type VARCHAR(50), -- 'horizontal', 'vertical', 'diagonal_positive', 'diagonal_negative', 'forfeit'
    final_board JSONB, -- Complete board state as JSON
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    started_at TIMESTAMP WITH TIME ZONE,
    finished_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Leaderboard table for player statistics
CREATE TABLE IF NOT EXISTS leaderboard (
    id SERIAL PRIMARY KEY,
    player_name VARCHAR(255) UNIQUE NOT NULL,
    total_games INTEGER DEFAULT 0,
    wins INTEGER DEFAULT 0,
    losses INTEGER DEFAULT 0,
    draws INTEGER DEFAULT 0,
    win_rate DECIMAL(5,2) DEFAULT 0.00,
    average_game_duration DECIMAL(8,2) DEFAULT 0.00,
    total_playtime_seconds BIGINT DEFAULT 0,
    horizontal_wins INTEGER DEFAULT 0,
    vertical_wins INTEGER DEFAULT 0,
    diagonal_wins INTEGER DEFAULT 0,
    forfeit_wins INTEGER DEFAULT 0,
    wins_vs_humans INTEGER DEFAULT 0,
    wins_vs_bots INTEGER DEFAULT 0,
    losses_vs_humans INTEGER DEFAULT 0,
    losses_vs_bots INTEGER DEFAULT 0,
    current_win_streak INTEGER DEFAULT 0,
    longest_win_streak INTEGER DEFAULT 0,
    first_game_at TIMESTAMP WITH TIME ZONE,
    last_game_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_games_created_at ON games(created_at);
CREATE INDEX IF NOT EXISTS idx_games_finished_at ON games(finished_at);
CREATE INDEX IF NOT EXISTS idx_games_player1_name ON games(player1_name);
CREATE INDEX IF NOT EXISTS idx_games_player2_name ON games(player2_name);
CREATE INDEX IF NOT EXISTS idx_games_winner_name ON games(winner_name);
CREATE INDEX IF NOT EXISTS idx_leaderboard_player_name ON leaderboard(player_name);
CREATE INDEX IF NOT EXISTS idx_leaderboard_win_rate ON leaderboard(win_rate DESC);
CREATE INDEX IF NOT EXISTS idx_leaderboard_wins ON leaderboard(wins DESC);

-- Create a view for easy leaderboard queries
CREATE OR REPLACE VIEW leaderboard_view AS
SELECT 
    ROW_NUMBER() OVER (ORDER BY win_rate DESC, wins DESC, total_games DESC) as rank,
    player_name,
    total_games,
    wins,
    losses,
    draws,
    win_rate,
    average_game_duration,
    wins_vs_humans,
    wins_vs_bots,
    current_win_streak,
    longest_win_streak,
    last_game_at
FROM leaderboard
WHERE total_games >= 1
ORDER BY win_rate DESC, wins DESC, total_games DESC;

-- Function to update leaderboard statistics
CREATE OR REPLACE FUNCTION update_leaderboard_stats()
RETURNS TRIGGER AS $$
DECLARE
    p1_stats RECORD;
    p2_stats RECORD;
    p1_win_streak INTEGER := 0;
    p2_win_streak INTEGER := 0;
BEGIN
    -- Update Player 1 statistics
    INSERT INTO leaderboard (player_name, first_game_at, last_game_at)
    VALUES (NEW.player1_name, NEW.finished_at, NEW.finished_at)
    ON CONFLICT (player_name) 
    DO UPDATE SET 
        last_game_at = NEW.finished_at,
        updated_at = NOW();

    -- Update Player 2 statistics
    INSERT INTO leaderboard (player_name, first_game_at, last_game_at)
    VALUES (NEW.player2_name, NEW.finished_at, NEW.finished_at)
    ON CONFLICT (player_name) 
    DO UPDATE SET 
        last_game_at = NEW.finished_at,
        updated_at = NOW();

    -- Calculate Player 1 stats
    SELECT 
        COUNT(*) as total_games,
        SUM(CASE WHEN winner_name = NEW.player1_name THEN 1 ELSE 0 END) as wins,
        SUM(CASE WHEN winner_name != NEW.player1_name AND NOT is_draw THEN 1 ELSE 0 END) as losses,
        SUM(CASE WHEN is_draw THEN 1 ELSE 0 END) as draws,
        SUM(duration_seconds) as total_playtime,
        AVG(duration_seconds) as avg_duration,
        SUM(CASE WHEN winner_name = NEW.player1_name AND win_type = 'horizontal' THEN 1 ELSE 0 END) as horizontal_wins,
        SUM(CASE WHEN winner_name = NEW.player1_name AND win_type = 'vertical' THEN 1 ELSE 0 END) as vertical_wins,
        SUM(CASE WHEN winner_name = NEW.player1_name AND win_type LIKE 'diagonal%' THEN 1 ELSE 0 END) as diagonal_wins,
        SUM(CASE WHEN winner_name = NEW.player1_name AND win_type = 'forfeit' THEN 1 ELSE 0 END) as forfeit_wins,
        SUM(CASE WHEN winner_name = NEW.player1_name AND NOT player2_is_bot THEN 1 ELSE 0 END) as wins_vs_humans,
        SUM(CASE WHEN winner_name = NEW.player1_name AND player2_is_bot THEN 1 ELSE 0 END) as wins_vs_bots,
        SUM(CASE WHEN winner_name != NEW.player1_name AND NOT is_draw AND NOT player2_is_bot THEN 1 ELSE 0 END) as losses_vs_humans,
        SUM(CASE WHEN winner_name != NEW.player1_name AND NOT is_draw AND player2_is_bot THEN 1 ELSE 0 END) as losses_vs_bots
    INTO p1_stats
    FROM games 
    WHERE player1_name = NEW.player1_name OR player2_name = NEW.player1_name;

    -- Calculate Player 2 stats
    SELECT 
        COUNT(*) as total_games,
        SUM(CASE WHEN winner_name = NEW.player2_name THEN 1 ELSE 0 END) as wins,
        SUM(CASE WHEN winner_name != NEW.player2_name AND NOT is_draw THEN 1 ELSE 0 END) as losses,
        SUM(CASE WHEN is_draw THEN 1 ELSE 0 END) as draws,
        SUM(duration_seconds) as total_playtime,
        AVG(duration_seconds) as avg_duration,
        SUM(CASE WHEN winner_name = NEW.player2_name AND win_type = 'horizontal' THEN 1 ELSE 0 END) as horizontal_wins,
        SUM(CASE WHEN winner_name = NEW.player2_name AND win_type = 'vertical' THEN 1 ELSE 0 END) as vertical_wins,
        SUM(CASE WHEN winner_name = NEW.player2_name AND win_type LIKE 'diagonal%' THEN 1 ELSE 0 END) as diagonal_wins,
        SUM(CASE WHEN winner_name = NEW.player2_name AND win_type = 'forfeit' THEN 1 ELSE 0 END) as forfeit_wins,
        SUM(CASE WHEN winner_name = NEW.player2_name AND NOT player1_is_bot THEN 1 ELSE 0 END) as wins_vs_humans,
        SUM(CASE WHEN winner_name = NEW.player2_name AND player1_is_bot THEN 1 ELSE 0 END) as wins_vs_bots,
        SUM(CASE WHEN winner_name != NEW.player2_name AND NOT is_draw AND NOT player1_is_bot THEN 1 ELSE 0 END) as losses_vs_humans,
        SUM(CASE WHEN winner_name != NEW.player2_name AND NOT is_draw AND player1_is_bot THEN 1 ELSE 0 END) as losses_vs_bots
    INTO p2_stats
    FROM games 
    WHERE player1_name = NEW.player2_name OR player2_name = NEW.player2_name;

    -- Update Player 1 leaderboard entry
    UPDATE leaderboard SET
        total_games = p1_stats.total_games,
        wins = p1_stats.wins,
        losses = p1_stats.losses,
        draws = p1_stats.draws,
        win_rate = CASE WHEN p1_stats.total_games > 0 THEN ROUND((p1_stats.wins::DECIMAL / p1_stats.total_games::DECIMAL) * 100, 2) ELSE 0 END,
        average_game_duration = COALESCE(p1_stats.avg_duration, 0),
        total_playtime_seconds = COALESCE(p1_stats.total_playtime, 0),
        horizontal_wins = p1_stats.horizontal_wins,
        vertical_wins = p1_stats.vertical_wins,
        diagonal_wins = p1_stats.diagonal_wins,
        forfeit_wins = p1_stats.forfeit_wins,
        wins_vs_humans = p1_stats.wins_vs_humans,
        wins_vs_bots = p1_stats.wins_vs_bots,
        losses_vs_humans = p1_stats.losses_vs_humans,
        losses_vs_bots = p1_stats.losses_vs_bots,
        updated_at = NOW()
    WHERE player_name = NEW.player1_name;

    -- Update Player 2 leaderboard entry
    UPDATE leaderboard SET
        total_games = p2_stats.total_games,
        wins = p2_stats.wins,
        losses = p2_stats.losses,
        draws = p2_stats.draws,
        win_rate = CASE WHEN p2_stats.total_games > 0 THEN ROUND((p2_stats.wins::DECIMAL / p2_stats.total_games::DECIMAL) * 100, 2) ELSE 0 END,
        average_game_duration = COALESCE(p2_stats.avg_duration, 0),
        total_playtime_seconds = COALESCE(p2_stats.total_playtime, 0),
        horizontal_wins = p2_stats.horizontal_wins,
        vertical_wins = p2_stats.vertical_wins,
        diagonal_wins = p2_stats.diagonal_wins,
        forfeit_wins = p2_stats.forfeit_wins,
        wins_vs_humans = p2_stats.wins_vs_humans,
        wins_vs_bots = p2_stats.wins_vs_bots,
        losses_vs_humans = p2_stats.losses_vs_humans,
        losses_vs_bots = p2_stats.losses_vs_bots,
        updated_at = NOW()
    WHERE player_name = NEW.player2_name;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger to automatically update leaderboard
DROP TRIGGER IF EXISTS trigger_update_leaderboard ON games;
CREATE TRIGGER trigger_update_leaderboard
    AFTER INSERT ON games
    FOR EACH ROW
    EXECUTE FUNCTION update_leaderboard_stats();

-- Insert some sample data for testing
INSERT INTO games (
    player1_id, player1_name, player1_is_bot,
    player2_id, player2_name, player2_is_bot,
    winner_id, winner_name, is_draw,
    total_moves, duration_seconds, win_type,
    final_board, started_at, finished_at
) VALUES 
(
    gen_random_uuid(), 'Alice', false,
    gen_random_uuid(), 'ConnectBot', true,
    (SELECT player1_id FROM (VALUES (gen_random_uuid())) AS t(player1_id)), 'Alice', false,
    15, 180, 'horizontal',
    '[[0,0,0,0,0,0,0],[0,0,0,0,0,0,0],[0,0,1,1,1,1,0],[0,0,2,1,2,2,0],[0,2,1,2,1,1,0],[2,1,2,1,2,1,0]]'::jsonb,
    NOW() - INTERVAL '3 minutes', NOW() - INTERVAL '1 minute'
),
(
    gen_random_uuid(), 'Bob', false,
    gen_random_uuid(), 'ConnectBot', true,
    (SELECT player2_id FROM (VALUES (gen_random_uuid())) AS t(player2_id)), 'ConnectBot', false,
    12, 145, 'vertical',
    '[[0,0,0,0,0,0,0],[0,0,0,0,0,0,0],[0,0,2,0,0,0,0],[0,0,2,0,0,0,0],[0,1,2,1,0,0,0],[1,1,2,1,0,0,0]]'::jsonb,
    NOW() - INTERVAL '5 minutes', NOW() - INTERVAL '2 minutes'
),
(
    gen_random_uuid(), 'Charlie', false,
    gen_random_uuid(), 'Diana', false,
    NULL, NULL, true,
    42, 420, NULL,
    '[[1,2,1,2,1,2,1],[2,1,2,1,2,1,2],[1,2,1,2,1,2,1],[2,1,2,1,2,1,2],[1,2,1,2,1,2,1],[2,1,2,1,2,1,2]]'::jsonb,
    NOW() - INTERVAL '10 minutes', NOW() - INTERVAL '3 minutes'
);
```

### Step 3: Verify Database Setup

Execute these queries to verify everything is working:

```sql
-- Check tables exist
SELECT table_name 
FROM information_schema.tables 
WHERE table_schema = 'public';

-- Check sample data
SELECT * FROM games;

-- Check leaderboard
SELECT * FROM leaderboard_view;

-- Test leaderboard function
SELECT player_name, wins, losses, win_rate 
FROM leaderboard 
ORDER BY win_rate DESC;
```

## 🔗 Connection Configuration

### Update .env file

Update your `.env` file with your PostgreSQL connection details:

```env
# Database Configuration
DATABASE_URL=postgres://username:password@localhost:5432/connect_four?sslmode=disable

# Replace with your actual values:
# username: your PostgreSQL username (usually 'postgres')
# password: your PostgreSQL password
# localhost: your PostgreSQL host (usually localhost)
# 5432: your PostgreSQL port (usually 5432)
```

### Example Connection Strings

```env
# Local PostgreSQL (default)
DATABASE_URL=postgres://postgres:password@localhost:5432/connect_four?sslmode=disable

# PostgreSQL with custom port
DATABASE_URL=postgres://postgres:password@localhost:5433/connect_four?sslmode=disable

# Remote PostgreSQL
DATABASE_URL=postgres://username:password@your-server.com:5432/connect_four?sslmode=require
```

## 🧪 Test Database Connection

### Option 1: Using Go (if Go is installed)

Create a test file `test_db.go`:

```go
package main

import (
    "database/sql"
    "fmt"
    "log"
    
    _ "github.com/lib/pq"
)

func main() {
    // Update with your connection string
    db, err := sql.Open("postgres", "postgres://postgres:password@localhost:5432/connect_four?sslmode=disable")
    if err != nil {
        log.Fatal("Failed to connect:", err)
    }
    defer db.Close()
    
    if err := db.Ping(); err != nil {
        log.Fatal("Failed to ping:", err)
    }
    
    fmt.Println("✅ Database connection successful!")
    
    // Test query
    var count int
    err = db.QueryRow("SELECT COUNT(*) FROM games").Scan(&count)
    if err != nil {
        log.Fatal("Failed to query:", err)
    }
    
    fmt.Printf("✅ Found %d games in database\n", count)
}
```

Run: `go run test_db.go`

### Option 2: Using pgAdmin4 Query Tool

Execute this test query:

```sql
-- Test connection and data
SELECT 
    'Database connection successful!' as status,
    COUNT(*) as total_games,
    (SELECT COUNT(*) FROM leaderboard) as total_players
FROM games;
```

## 🚀 Running the Application

Once the database is set up:

1. **Update .env file** with your database connection
2. **Start the Go backend**:
   ```bash
   go run cmd/server/main.go
   ```
3. **Test the API**:
   ```bash
   curl http://localhost:8080/api/leaderboard
   ```

## 🔍 Troubleshooting

### Common Issues

1. **Connection Refused**
   - Check if PostgreSQL service is running
   - Verify port number (usually 5432)
   - Check firewall settings

2. **Authentication Failed**
   - Verify username and password
   - Check pg_hba.conf for authentication method
   - Ensure user has database access

3. **Database Not Found**
   - Verify database name is correct
   - Ensure database was created successfully
   - Check connection string format

4. **Permission Denied**
   - Ensure user has CREATE, INSERT, UPDATE, DELETE permissions
   - Grant necessary privileges:
     ```sql
     GRANT ALL PRIVILEGES ON DATABASE connect_four TO your_username;
     GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO your_username;
     ```

### Useful pgAdmin4 Queries

```sql
-- Check current connections
SELECT * FROM pg_stat_activity WHERE datname = 'connect_four';

-- Check table sizes
SELECT 
    schemaname,
    tablename,
    attname,
    n_distinct,
    correlation
FROM pg_stats
WHERE schemaname = 'public';

-- Check recent games
SELECT 
    player1_name,
    player2_name,
    winner_name,
    duration_seconds,
    finished_at
FROM games
ORDER BY finished_at DESC
LIMIT 10;
```

## 📊 Database Schema Overview

### Tables Created:
- **games**: Stores completed game records
- **leaderboard**: Player statistics and rankings

### Views Created:
- **leaderboard_view**: Ranked leaderboard with calculated positions

### Functions Created:
- **update_leaderboard_stats()**: Automatically updates player statistics

### Triggers Created:
- **trigger_update_leaderboard**: Fires after game insertion to update stats

The database is now ready for the Connect Four application! 🎮