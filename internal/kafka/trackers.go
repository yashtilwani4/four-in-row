package kafka

import (
	"sync"
	"time"
)

// GameTracker tracks active games and their states
type GameTracker struct {
	activeGames map[string]*ActiveGame
	mu          sync.RWMutex
}

// ActiveGame represents a game currently being tracked
type ActiveGame struct {
	GameID      string    `json:"game_id"`
	Players     []string  `json:"players"`
	StartTime   time.Time `json:"start_time"`
	EndTime     *time.Time `json:"end_time,omitempty"`
	Winner      string    `json:"winner,omitempty"`
	Duration    int64     `json:"duration"`
	MoveCount   int       `json:"move_count"`
	LastMove    time.Time `json:"last_move"`
	IsCompleted bool      `json:"is_completed"`
}

// NewGameTracker creates a new game tracker
func NewGameTracker() *GameTracker {
	return &GameTracker{
		activeGames: make(map[string]*ActiveGame),
	}
}

// StartGame records a new game start
func (gt *GameTracker) StartGame(gameID string, players []PlayerInfo, startTime time.Time) {
	gt.mu.Lock()
	defer gt.mu.Unlock()

	playerNames := make([]string, len(players))
	for i, player := range players {
		playerNames[i] = player.Name
	}

	gt.activeGames[gameID] = &ActiveGame{
		GameID:      gameID,
		Players:     playerNames,
		StartTime:   startTime,
		LastMove:    startTime,
		IsCompleted: false,
	}
}

// RecordMove records a move in an active game
func (gt *GameTracker) RecordMove(gameID, playerName string, moveTime time.Time) {
	gt.mu.Lock()
	defer gt.mu.Unlock()

	if game, exists := gt.activeGames[gameID]; exists {
		game.MoveCount++
		game.LastMove = moveTime
	}
}

// EndGame records a game completion
func (gt *GameTracker) EndGame(gameID, winner string, duration int64, endTime time.Time) {
	gt.mu.Lock()
	defer gt.mu.Unlock()

	if game, exists := gt.activeGames[gameID]; exists {
		game.Winner = winner
		game.Duration = duration
		game.EndTime = &endTime
		game.IsCompleted = true
	}
}

// GetActiveGameCount returns the number of active games
func (gt *GameTracker) GetActiveGameCount() int {
	gt.mu.RLock()
	defer gt.mu.RUnlock()

	count := 0
	for _, game := range gt.activeGames {
		if !game.IsCompleted {
			count++
		}
	}
	return count
}

// GetActiveGames returns all active games
func (gt *GameTracker) GetActiveGames() []*ActiveGame {
	gt.mu.RLock()
	defer gt.mu.RUnlock()

	var activeGames []*ActiveGame
	for _, game := range gt.activeGames {
		if !game.IsCompleted {
			gameCopy := *game
			activeGames = append(activeGames, &gameCopy)
		}
	}
	return activeGames
}

// CleanupCompletedGames removes completed games older than the specified duration
func (gt *GameTracker) CleanupCompletedGames(maxAge time.Duration) {
	gt.mu.Lock()
	defer gt.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	for gameID, game := range gt.activeGames {
		if game.IsCompleted && game.EndTime != nil && game.EndTime.Before(cutoff) {
			delete(gt.activeGames, gameID)
		}
	}
}

// PlayerTracker tracks player activities and statistics
type PlayerTracker struct {
	players map[string]*TrackedPlayer
	mu      sync.RWMutex
}

// TrackedPlayer represents a player being tracked
type TrackedPlayer struct {
	Name                string        `json:"name"`
	FirstSeen           time.Time     `json:"first_seen"`
	LastSeen            time.Time     `json:"last_seen"`
	GamesPlayed         int           `json:"games_played"`
	GamesWon            int           `json:"games_won"`
	GamesLost           int           `json:"games_lost"`
	GamesDrawn          int           `json:"games_drawn"`
	TotalMoves          int           `json:"total_moves"`
	TotalGameTime       int64         `json:"total_game_time"`
	Disconnections      int           `json:"disconnections"`
	Reconnections       int           `json:"reconnections"`
	TotalOfflineTime    time.Duration `json:"total_offline_time"`
	IsOnline            bool          `json:"is_online"`
	CurrentGameID       string        `json:"current_game_id,omitempty"`
	SessionStartTime    time.Time     `json:"session_start_time"`
	TotalSessionTime    time.Duration `json:"total_session_time"`
}

// NewPlayerTracker creates a new player tracker
func NewPlayerTracker() *PlayerTracker {
	return &PlayerTracker{
		players: make(map[string]*TrackedPlayer),
	}
}

// TrackPlayer starts tracking a player
func (pt *PlayerTracker) TrackPlayer(playerName string, timestamp time.Time) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	if _, exists := pt.players[playerName]; !exists {
		pt.players[playerName] = &TrackedPlayer{
			Name:             playerName,
			FirstSeen:        timestamp,
			LastSeen:         timestamp,
			IsOnline:         true,
			SessionStartTime: timestamp,
		}
	} else {
		player := pt.players[playerName]
		player.LastSeen = timestamp
		if !player.IsOnline {
			player.IsOnline = true
			player.SessionStartTime = timestamp
		}
	}
}

// RecordMove records a move by a player
func (pt *PlayerTracker) RecordMove(playerName string, timestamp time.Time) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	if player, exists := pt.players[playerName]; exists {
		player.TotalMoves++
		player.LastSeen = timestamp
	}
}

// RecordGameEnd records a game end for a player
func (pt *PlayerTracker) RecordGameEnd(playerName string, isWinner, isDraw bool, duration int64, timestamp time.Time) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	if player, exists := pt.players[playerName]; exists {
		player.GamesPlayed++
		player.TotalGameTime += duration
		player.LastSeen = timestamp
		player.CurrentGameID = ""

		if isDraw {
			player.GamesDrawn++
		} else if isWinner {
			player.GamesWon++
		} else {
			player.GamesLost++
		}
	}
}

// RecordDisconnection records a player disconnection
func (pt *PlayerTracker) RecordDisconnection(playerName string, timestamp time.Time) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	if player, exists := pt.players[playerName]; exists {
		player.Disconnections++
		player.IsOnline = false
		player.LastSeen = timestamp
		
		// Add session time
		if !player.SessionStartTime.IsZero() {
			player.TotalSessionTime += timestamp.Sub(player.SessionStartTime)
		}
	}
}

// RecordReconnection records a player reconnection
func (pt *PlayerTracker) RecordReconnection(playerName string, offlineDuration time.Duration, timestamp time.Time) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	if player, exists := pt.players[playerName]; exists {
		player.Reconnections++
		player.TotalOfflineTime += offlineDuration
		player.IsOnline = true
		player.LastSeen = timestamp
		player.SessionStartTime = timestamp
	}
}

// GetPlayerCount returns the total number of tracked players
func (pt *PlayerTracker) GetPlayerCount() int {
	pt.mu.RLock()
	defer pt.mu.RUnlock()
	return len(pt.players)
}

// GetOnlinePlayerCount returns the number of online players
func (pt *PlayerTracker) GetOnlinePlayerCount() int {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	count := 0
	for _, player := range pt.players {
		if player.IsOnline {
			count++
		}
	}
	return count
}

// GetTopPlayers returns the top players by games won
func (pt *PlayerTracker) GetTopPlayers(limit int) []*TrackedPlayer {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	players := make([]*TrackedPlayer, 0, len(pt.players))
	for _, player := range pt.players {
		playerCopy := *player
		players = append(players, &playerCopy)
	}

	// Simple bubble sort by games won (descending)
	for i := 0; i < len(players)-1; i++ {
		for j := 0; j < len(players)-i-1; j++ {
			if players[j].GamesWon < players[j+1].GamesWon {
				players[j], players[j+1] = players[j+1], players[j]
			}
		}
	}

	if len(players) > limit {
		players = players[:limit]
	}

	return players
}

// GetPlayerStats returns statistics for a specific player
func (pt *PlayerTracker) GetPlayerStats(playerName string) *TrackedPlayer {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	if player, exists := pt.players[playerName]; exists {
		playerCopy := *player
		return &playerCopy
	}
	return nil
}

// UpdatePlayerActivity updates player activity status based on last seen time
func (pt *PlayerTracker) UpdatePlayerActivity(inactiveThreshold time.Duration) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	cutoff := time.Now().Add(-inactiveThreshold)
	for _, player := range pt.players {
		if player.IsOnline && player.LastSeen.Before(cutoff) {
			player.IsOnline = false
			// Add final session time
			if !player.SessionStartTime.IsZero() {
				player.TotalSessionTime += player.LastSeen.Sub(player.SessionStartTime)
			}
		}
	}
}

// HourlyTracker tracks hourly game statistics
type HourlyTracker struct {
	hourlyStats map[string]*HourlyStats
	mu          sync.RWMutex
}

// HourlyStats represents statistics for a specific hour
type HourlyStats struct {
	Hour            string    `json:"hour"`
	GamesStarted    int       `json:"games_started"`
	GamesCompleted  int       `json:"games_completed"`
	TotalMoves      int       `json:"total_moves"`
	UniquePlayers   int       `json:"unique_players"`
	TotalDuration   int64     `json:"total_duration"`
	AverageDuration float64   `json:"average_duration"`
	LastUpdated     time.Time `json:"last_updated"`
}

// NewHourlyTracker creates a new hourly tracker
func NewHourlyTracker() *HourlyTracker {
	return &HourlyTracker{
		hourlyStats: make(map[string]*HourlyStats),
	}
}

// RecordGameStart records a game start for hourly tracking
func (ht *HourlyTracker) RecordGameStart(timestamp time.Time) {
	ht.mu.Lock()
	defer ht.mu.Unlock()

	hourKey := timestamp.Format("2006-01-02-15")
	if _, exists := ht.hourlyStats[hourKey]; !exists {
		ht.hourlyStats[hourKey] = &HourlyStats{
			Hour: hourKey,
		}
	}

	ht.hourlyStats[hourKey].GamesStarted++
	ht.hourlyStats[hourKey].LastUpdated = timestamp
}

// RecordGameEnd records a game end for hourly tracking
func (ht *HourlyTracker) RecordGameEnd(timestamp time.Time, duration int64) {
	ht.mu.Lock()
	defer ht.mu.Unlock()

	hourKey := timestamp.Format("2006-01-02-15")
	if _, exists := ht.hourlyStats[hourKey]; !exists {
		ht.hourlyStats[hourKey] = &HourlyStats{
			Hour: hourKey,
		}
	}

	stats := ht.hourlyStats[hourKey]
	stats.GamesCompleted++
	stats.TotalDuration += duration
	
	if stats.GamesCompleted > 0 {
		stats.AverageDuration = float64(stats.TotalDuration) / float64(stats.GamesCompleted)
	}
	
	stats.LastUpdated = timestamp
}

// GetGamesToday returns the number of games started today
func (ht *HourlyTracker) GetGamesToday() int {
	ht.mu.RLock()
	defer ht.mu.RUnlock()

	today := time.Now().Format("2006-01-02")
	total := 0

	for hourKey, stats := range ht.hourlyStats {
		if len(hourKey) >= 10 && hourKey[:10] == today {
			total += stats.GamesStarted
		}
	}

	return total
}

// GetGamesThisHour returns the number of games started this hour
func (ht *HourlyTracker) GetGamesThisHour() int {
	ht.mu.RLock()
	defer ht.mu.RUnlock()

	currentHour := time.Now().Format("2006-01-02-15")
	if stats, exists := ht.hourlyStats[currentHour]; exists {
		return stats.GamesStarted
	}
	return 0
}

// GetHourlyStats returns statistics for a specific hour
func (ht *HourlyTracker) GetHourlyStats(hour string) *HourlyStats {
	ht.mu.RLock()
	defer ht.mu.RUnlock()

	if stats, exists := ht.hourlyStats[hour]; exists {
		statsCopy := *stats
		return &statsCopy
	}
	return nil
}

// GetRecentHours returns statistics for the last N hours
func (ht *HourlyTracker) GetRecentHours(hours int) []*HourlyStats {
	ht.mu.RLock()
	defer ht.mu.RUnlock()

	var recentStats []*HourlyStats
	now := time.Now()

	for i := 0; i < hours; i++ {
		hourTime := now.Add(-time.Duration(i) * time.Hour)
		hourKey := hourTime.Format("2006-01-02-15")
		
		if stats, exists := ht.hourlyStats[hourKey]; exists {
			statsCopy := *stats
			recentStats = append(recentStats, &statsCopy)
		} else {
			// Create empty stats for missing hours
			recentStats = append(recentStats, &HourlyStats{
				Hour: hourKey,
			})
		}
	}

	return recentStats
}

// CleanupOldStats removes statistics older than the specified duration
func (ht *HourlyTracker) CleanupOldStats(maxAge time.Duration) {
	ht.mu.Lock()
	defer ht.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	cutoffKey := cutoff.Format("2006-01-02-15")

	for hourKey := range ht.hourlyStats {
		if hourKey < cutoffKey {
			delete(ht.hourlyStats, hourKey)
		}
	}
}

// GetDailyTotals returns daily totals from hourly data
func (ht *HourlyTracker) GetDailyTotals(days int) map[string]*DailyTotals {
	ht.mu.RLock()
	defer ht.mu.RUnlock()

	dailyTotals := make(map[string]*DailyTotals)
	now := time.Now()

	for i := 0; i < days; i++ {
		dayTime := now.Add(-time.Duration(i) * 24 * time.Hour)
		dayKey := dayTime.Format("2006-01-02")
		
		dailyTotals[dayKey] = &DailyTotals{
			Day: dayKey,
		}

		// Aggregate hourly stats for this day
		for hourKey, stats := range ht.hourlyStats {
			if len(hourKey) >= 10 && hourKey[:10] == dayKey {
				dailyTotals[dayKey].GamesStarted += stats.GamesStarted
				dailyTotals[dayKey].GamesCompleted += stats.GamesCompleted
				dailyTotals[dayKey].TotalMoves += stats.TotalMoves
				dailyTotals[dayKey].TotalDuration += stats.TotalDuration
			}
		}

		// Calculate average duration
		if dailyTotals[dayKey].GamesCompleted > 0 {
			dailyTotals[dayKey].AverageDuration = float64(dailyTotals[dayKey].TotalDuration) / float64(dailyTotals[dayKey].GamesCompleted)
		}
	}

	return dailyTotals
}

// DailyTotals represents aggregated daily statistics
type DailyTotals struct {
	Day             string  `json:"day"`
	GamesStarted    int     `json:"games_started"`
	GamesCompleted  int     `json:"games_completed"`
	TotalMoves      int     `json:"total_moves"`
	TotalDuration   int64   `json:"total_duration"`
	AverageDuration float64 `json:"average_duration"`
}