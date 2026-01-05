# Database Package

This package provides PostgreSQL database operations for the Connect Four game backend.

## Components

### Repository (Recommended)
- **File**: `repository.go`
- **Purpose**: Modern, comprehensive database operations with full feature support
- **Features**:
  - Complete game record storage with detailed metadata
  - Comprehensive leaderboard with advanced statistics
  - Player game history tracking
  - Automatic leaderboard updates via database triggers
  - Move-by-move game history (optional)
  - Database health checks and statistics

### PostgresDB (Legacy)
- **File**: `postgres.go`
- **Purpose**: Backward compatibility layer
- **Status**: Deprecated - use Repository instead

### Database Schema
- **File**: `schema.sql`
- **Purpose**: Complete PostgreSQL schema with tables, indexes, views, and functions
- **Features**:
  - Comprehensive game tracking
  - Automatic leaderboard maintenance
  - Advanced player statistics
  - Performance-optimized indexes

## Usage

### Basic Setup

```go
import "connect-four-backend/internal/database"

// Create repository
repo, err := database.NewRepository(databaseURL)
if err != nil {
    log.Fatal(err)
}
defer repo.Close()
```

### Save Completed Game

```go
// After a game finishes
err := repo.SaveCompletedGame(game)
if err != nil {
    log.Printf("Failed to save game: %v", err)
}
```

### Get Leaderboard

```go
// Get top 10 players
leaderboard, err := repo.GetLeaderboard(10)
if err != nil {
    log.Printf("Failed to get leaderboard: %v", err)
    return
}

for _, entry := range leaderboard {
    fmt.Printf("%d. %s - %d wins (%.1f%% win rate)\n", 
        entry.Rank, entry.Username, entry.Wins, entry.WinRate)
}
```

### Get Player Statistics

```go
stats, err := repo.GetPlayerStats("username")
if err != nil {
    log.Printf("Player not found: %v", err)
    return
}

fmt.Printf("Player: %s (Rank #%d)\n", stats.Username, stats.Rank)
fmt.Printf("Games: %d | Win Rate: %.1f%%\n", stats.TotalGames, stats.WinRate)
```

### Get Player Game History

```go
history, err := repo.GetPlayerGameHistory("username", 20)
if err != nil {
    log.Printf("Failed to get history: %v", err)
    return
}

for _, game := range history {
    fmt.Printf("%s vs %s - %s\n", "username", game.OpponentName, game.Outcome)
}
```

## Database Schema Overview

### Core Tables

#### `games`
Stores completed game records with comprehensive metadata:
- Player information (names, IDs, bot flags)
- Game outcome (winner, draw, forfeit)
- Game statistics (moves, duration, win type)
- Final board state (JSON)
- Timestamps (created, started, finished)

#### `leaderboard`
Automatically maintained player statistics:
- Basic stats (games, wins, losses, draws, win rate)
- Performance metrics (average duration, playtime)
- Win type breakdown (horizontal, vertical, diagonal, forfeit)
- Opponent type stats (vs humans, vs bots)
- Streak tracking (current and longest win streaks)

#### `game_moves` (Optional)
Detailed move-by-move history:
- Individual move records
- Board states before/after moves
- Bot reasoning and confidence (for AI moves)
- Move timing information

### Views

#### `player_game_history`
Convenient view for player-centric game history queries.

#### `leaderboard_ranking`
Ranked leaderboard with player tiers (beginner, casual, active).

### Functions & Triggers

#### `update_leaderboard_stats()`
Automatically updates leaderboard when games are inserted.

#### `recalculate_leaderboard_stats()`
Maintenance function to rebuild leaderboard from scratch.

## Configuration

### Environment Variables

```bash
DATABASE_URL="postgres://user:password@localhost/connect_four?sslmode=disable"
```

### Connection Pool Settings

The repository automatically configures connection pooling:
- Max Open Connections: 25
- Max Idle Connections: 5
- Connection Max Lifetime: 5 minutes

## Error Handling

All repository methods return descriptive errors:

```go
if err != nil {
    if err == sql.ErrNoRows {
        // Handle not found case
    } else {
        // Handle other database errors
        log.Printf("Database error: %v", err)
    }
}
```

## Performance Considerations

### Indexes
The schema includes optimized indexes for:
- Player lookups (by ID and name)
- Game queries (by date, duration, outcome)
- Leaderboard sorting (by win rate, wins, games)

### Automatic Updates
Leaderboard statistics are updated automatically via database triggers, ensuring consistency without application-level complexity.

### Query Optimization
- Uses CTEs and window functions for efficient ranking
- Composite indexes for multi-column sorts
- JSON storage for flexible board state representation

## Migration Strategy

### Initial Setup
1. Run the complete `schema.sql` to create all tables, indexes, and functions
2. The schema is designed to be idempotent (safe to run multiple times)

### Data Migration
If migrating from the legacy PostgresDB:
```go
// Use the recalculation function to rebuild leaderboard
err := repo.RecalculateLeaderboard()
```

## Monitoring & Maintenance

### Health Checks
```go
if err := repo.HealthCheck(); err != nil {
    log.Printf("Database unhealthy: %v", err)
}
```

### Statistics
```go
stats, err := repo.GetDatabaseStats()
// Returns: total_games, total_players, games_today
```

### Maintenance
- The `recalculate_leaderboard_stats()` function can be called periodically to ensure data consistency
- Regular VACUUM and ANALYZE operations recommended for PostgreSQL performance

## Example Integration

See `examples/database_example.go` for a complete working example demonstrating all repository features.

## Best Practices

1. **Use Repository over PostgresDB**: The new Repository provides better error handling, more features, and cleaner API
2. **Handle Errors Gracefully**: Always check for `sql.ErrNoRows` when querying for specific records
3. **Use Transactions**: For complex operations, consider using database transactions
4. **Monitor Performance**: Use the health check and statistics methods for monitoring
5. **Regular Maintenance**: Run `recalculate_leaderboard_stats()` periodically to ensure data consistency