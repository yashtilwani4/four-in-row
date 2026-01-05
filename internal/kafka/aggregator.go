package kafka

import (
	"fmt"
	"log"
	"sync"
	"time"

	"connect-four-backend/internal/database"
)

// MetricsAggregator handles real-time aggregation of game metrics
type MetricsAggregator struct {
	repo                *database.Repository
	gameMetrics         *GameMetrics
	playerMetrics       *PlayerMetrics
	hourlyMetrics       *HourlyMetrics
	dailyMetrics        *DailyMetrics
	mu                  sync.RWMutex
	lastFlush           time.Time
	flushInterval       time.Duration
}

// GameMetrics tracks game-related aggregated metrics
type GameMetrics struct {
	TotalGames          int64         `json:"total_games"`
	CompletedGames      int64         `json:"completed_games"`
	AverageGameDuration float64       `json:"average_game_duration"`
	TotalGameDuration   int64         `json:"total_game_duration"`
	WinnerFrequency     map[string]int64 `json:"winner_frequency"`
	WinTypeDistribution map[string]int64 `json:"win_type_distribution"`
	DrawCount           int64         `json:"draw_count"`
	BotGames            int64         `json:"bot_games"`
	HumanGames          int64         `json:"human_games"`
	mu                  sync.RWMutex
}

// PlayerMetrics tracks player-related aggregated metrics
type PlayerMetrics struct {
	ActivePlayers       map[string]*PlayerStats `json:"active_players"`
	TotalPlayers        int64                   `json:"total_players"`
	NewPlayersToday     int64                   `json:"new_players_today"`
	TotalMoves          int64                   `json:"total_moves"`
	TotalDisconnections int64                   `json:"total_disconnections"`
	TotalReconnections  int64                   `json:"total_reconnections"`
	mu                  sync.RWMutex
}

// PlayerStats tracks individual player statistics
type PlayerStats struct {
	Name                string        `json:"name"`
	GamesPlayed         int64         `json:"games_played"`
	GamesWon            int64         `json:"games_won"`
	GamesLost           int64         `json:"games_lost"`
	GamesDrawn          int64         `json:"games_drawn"`
	TotalMoves          int64         `json:"total_moves"`
	TotalGameTime       int64         `json:"total_game_time"`
	AverageGameTime     float64       `json:"average_game_time"`
	WinRate             float64       `json:"win_rate"`
	Disconnections      int64         `json:"disconnections"`
	Reconnections       int64         `json:"reconnections"`
	TotalOfflineTime    time.Duration `json:"total_offline_time"`
	FirstSeen           time.Time     `json:"first_seen"`
	LastSeen            time.Time     `json:"last_seen"`
	IsActive            bool          `json:"is_active"`
}

// HourlyMetrics tracks hourly game statistics
type HourlyMetrics struct {
	GamesPerHour        map[string]int64 `json:"games_per_hour"` // key: "2024-01-01-15"
	MovesPerHour        map[string]int64 `json:"moves_per_hour"`
	PlayersPerHour      map[string]int64 `json:"players_per_hour"`
	AverageDurationHour map[string]float64 `json:"average_duration_hour"`
	CurrentHour         string           `json:"current_hour"`
	mu                  sync.RWMutex
}

// DailyMetrics tracks daily game statistics
type DailyMetrics struct {
	GamesPerDay         map[string]int64 `json:"games_per_day"` // key: "2024-01-01"
	MovesPerDay         map[string]int64 `json:"moves_per_day"`
	PlayersPerDay       map[string]int64 `json:"players_per_day"`
	AverageDurationDay  map[string]float64 `json:"average_duration_day"`
	NewPlayersPerDay    map[string]int64 `json:"new_players_per_day"`
	CurrentDay          string           `json:"current_day"`
	mu                  sync.RWMutex
}

// NewMetricsAggregator creates a new metrics aggregator
func NewMetricsAggregator(repo *database.Repository) (*MetricsAggregator, error) {
	return &MetricsAggregator{
		repo: repo,
		gameMetrics: &GameMetrics{
			WinnerFrequency:     make(map[string]int64),
			WinTypeDistribution: make(map[string]int64),
		},
		playerMetrics: &PlayerMetrics{
			ActivePlayers: make(map[string]*PlayerStats),
		},
		hourlyMetrics: &HourlyMetrics{
			GamesPerHour:        make(map[string]int64),
			MovesPerHour:        make(map[string]int64),
			PlayersPerHour:      make(map[string]int64),
			AverageDurationHour: make(map[string]float64),
		},
		dailyMetrics: &DailyMetrics{
			GamesPerDay:        make(map[string]int64),
			MovesPerDay:        make(map[string]int64),
			PlayersPerDay:      make(map[string]int64),
			AverageDurationDay: make(map[string]float64),
			NewPlayersPerDay:   make(map[string]int64),
		},
		lastFlush:     time.Now(),
		flushInterval: 5 * time.Minute,
	}, nil
}

// RecordGameStart processes a game started event
func (ma *MetricsAggregator) RecordGameStart(event GameStartedEvent) error {
	ma.mu.Lock()
	defer ma.mu.Unlock()

	// Update game metrics
	ma.gameMetrics.mu.Lock()
	ma.gameMetrics.TotalGames++
	
	// Check if it's a bot game
	hasBots := false
	for _, player := range event.Players {
		if player.IsBot {
			hasBots = true
			break
		}
	}
	
	if hasBots {
		ma.gameMetrics.BotGames++
	} else {
		ma.gameMetrics.HumanGames++
	}
	ma.gameMetrics.mu.Unlock()

	// Update hourly metrics
	hourKey := event.Timestamp.Format("2006-01-02-15")
	ma.hourlyMetrics.mu.Lock()
	ma.hourlyMetrics.GamesPerHour[hourKey]++
	ma.hourlyMetrics.CurrentHour = hourKey
	ma.hourlyMetrics.mu.Unlock()

	// Update daily metrics
	dayKey := event.Timestamp.Format("2006-01-02")
	ma.dailyMetrics.mu.Lock()
	ma.dailyMetrics.GamesPerDay[dayKey]++
	ma.dailyMetrics.CurrentDay = dayKey
	ma.dailyMetrics.mu.Unlock()

	// Update player metrics
	ma.playerMetrics.mu.Lock()
	uniquePlayers := make(map[string]bool)
	for _, player := range event.Players {
		uniquePlayers[player.Name] = true
		
		if _, exists := ma.playerMetrics.ActivePlayers[player.Name]; !exists {
			ma.playerMetrics.ActivePlayers[player.Name] = &PlayerStats{
				Name:      player.Name,
				FirstSeen: event.Timestamp,
				LastSeen:  event.Timestamp,
				IsActive:  true,
			}
			ma.playerMetrics.TotalPlayers++
			
			// Check if new player today
			if event.Timestamp.Format("2006-01-02") == time.Now().Format("2006-01-02") {
				ma.playerMetrics.NewPlayersToday++
				ma.dailyMetrics.NewPlayersPerDay[dayKey]++
			}
		}
		
		ma.playerMetrics.ActivePlayers[player.Name].GamesPlayed++
		ma.playerMetrics.ActivePlayers[player.Name].LastSeen = event.Timestamp
		ma.playerMetrics.ActivePlayers[player.Name].IsActive = true
	}
	
	// Update unique players per hour/day
	ma.hourlyMetrics.PlayersPerHour[hourKey] = int64(len(uniquePlayers))
	ma.dailyMetrics.PlayersPerDay[dayKey] = int64(len(uniquePlayers))
	ma.playerMetrics.mu.Unlock()

	log.Printf("Aggregated game start: Total games: %d, Active players: %d", 
		ma.gameMetrics.TotalGames, len(ma.playerMetrics.ActivePlayers))

	return nil
}

// RecordMove processes a move played event
func (ma *MetricsAggregator) RecordMove(event MovePlayedEvent) error {
	ma.mu.Lock()
	defer ma.mu.Unlock()

	// Update hourly metrics
	hourKey := event.Timestamp.Format("2006-01-02-15")
	ma.hourlyMetrics.mu.Lock()
	ma.hourlyMetrics.MovesPerHour[hourKey]++
	ma.hourlyMetrics.mu.Unlock()

	// Update daily metrics
	dayKey := event.Timestamp.Format("2006-01-02")
	ma.dailyMetrics.mu.Lock()
	ma.dailyMetrics.MovesPerDay[dayKey]++
	ma.dailyMetrics.mu.Unlock()

	// Update player metrics
	ma.playerMetrics.mu.Lock()
	ma.playerMetrics.TotalMoves++
	
	if player, exists := ma.playerMetrics.ActivePlayers[event.Player.Name]; exists {
		player.TotalMoves++
		player.LastSeen = event.Timestamp
	}
	ma.playerMetrics.mu.Unlock()

	return nil
}

// RecordGameEnd processes a game ended event
func (ma *MetricsAggregator) RecordGameEnd(event GameEndedEvent) error {
	ma.mu.Lock()
	defer ma.mu.Unlock()

	// Update game metrics
	ma.gameMetrics.mu.Lock()
	ma.gameMetrics.CompletedGames++
	ma.gameMetrics.TotalGameDuration += event.Duration
	
	if ma.gameMetrics.CompletedGames > 0 {
		ma.gameMetrics.AverageGameDuration = float64(ma.gameMetrics.TotalGameDuration) / float64(ma.gameMetrics.CompletedGames)
	}

	if event.IsDraw {
		ma.gameMetrics.DrawCount++
	} else if event.Winner != nil {
		ma.gameMetrics.WinnerFrequency[event.Winner.Name]++
	}

	if event.WinType != "" {
		ma.gameMetrics.WinTypeDistribution[event.WinType]++
	}
	ma.gameMetrics.mu.Unlock()

	// Update hourly metrics
	hourKey := event.Timestamp.Format("2006-01-02-15")
	ma.hourlyMetrics.mu.Lock()
	if count := ma.hourlyMetrics.GamesPerHour[hourKey]; count > 0 {
		currentAvg := ma.hourlyMetrics.AverageDurationHour[hourKey]
		ma.hourlyMetrics.AverageDurationHour[hourKey] = (currentAvg*float64(count-1) + float64(event.Duration)) / float64(count)
	}
	ma.hourlyMetrics.mu.Unlock()

	// Update daily metrics
	dayKey := event.Timestamp.Format("2006-01-02")
	ma.dailyMetrics.mu.Lock()
	if count := ma.dailyMetrics.GamesPerDay[dayKey]; count > 0 {
		currentAvg := ma.dailyMetrics.AverageDurationDay[dayKey]
		ma.dailyMetrics.AverageDurationDay[dayKey] = (currentAvg*float64(count-1) + float64(event.Duration)) / float64(count)
	}
	ma.dailyMetrics.mu.Unlock()

	// Update player metrics
	ma.playerMetrics.mu.Lock()
	for _, player := range event.Players {
		if playerStats, exists := ma.playerMetrics.ActivePlayers[player.Name]; exists {
			playerStats.TotalGameTime += event.Duration
			playerStats.LastSeen = event.Timestamp
			
			if playerStats.GamesPlayed > 0 {
				playerStats.AverageGameTime = float64(playerStats.TotalGameTime) / float64(playerStats.GamesPlayed)
			}

			if event.IsDraw {
				playerStats.GamesDrawn++
			} else if event.Winner != nil && event.Winner.Name == player.Name {
				playerStats.GamesWon++
			} else {
				playerStats.GamesLost++
			}

			// Calculate win rate
			totalGames := playerStats.GamesWon + playerStats.GamesLost + playerStats.GamesDrawn
			if totalGames > 0 {
				playerStats.WinRate = float64(playerStats.GamesWon) / float64(totalGames) * 100
			}
		}
	}
	ma.playerMetrics.mu.Unlock()

	log.Printf("Aggregated game end: Completed games: %d, Average duration: %.1fs", 
		ma.gameMetrics.CompletedGames, ma.gameMetrics.AverageGameDuration)

	return nil
}

// RecordDisconnection processes a player disconnected event
func (ma *MetricsAggregator) RecordDisconnection(event PlayerDisconnectedEvent) error {
	ma.mu.Lock()
	defer ma.mu.Unlock()

	ma.playerMetrics.mu.Lock()
	ma.playerMetrics.TotalDisconnections++
	
	if player, exists := ma.playerMetrics.ActivePlayers[event.Player.Name]; exists {
		player.Disconnections++
		player.LastSeen = event.Timestamp
		player.IsActive = false
	}
	ma.playerMetrics.mu.Unlock()

	return nil
}

// RecordReconnection processes a player reconnected event
func (ma *MetricsAggregator) RecordReconnection(event PlayerReconnectedEvent) error {
	ma.mu.Lock()
	defer ma.mu.Unlock()

	ma.playerMetrics.mu.Lock()
	ma.playerMetrics.TotalReconnections++
	
	if player, exists := ma.playerMetrics.ActivePlayers[event.Player.Name]; exists {
		player.Reconnections++
		player.TotalOfflineTime += event.OfflineDuration
		player.LastSeen = event.Timestamp
		player.IsActive = true
	}
	ma.playerMetrics.mu.Unlock()

	return nil
}

// AggregateMetrics performs periodic aggregation and persistence
func (ma *MetricsAggregator) AggregateMetrics() error {
	ma.mu.Lock()
	defer ma.mu.Unlock()

	log.Println("Starting metrics aggregation...")

	// Clean up old data (keep last 7 days for hourly, 30 days for daily)
	ma.cleanupOldMetrics()

	// Persist current metrics to database (if needed)
	if err := ma.persistMetrics(); err != nil {
		return fmt.Errorf("failed to persist metrics: %w", err)
	}

	// Update last flush time
	ma.lastFlush = time.Now()

	log.Printf("Metrics aggregation completed. Games: %d, Players: %d, Avg Duration: %.1fs",
		ma.gameMetrics.TotalGames, ma.playerMetrics.TotalPlayers, ma.gameMetrics.AverageGameDuration)

	return nil
}

// GetGameMetrics returns current game metrics
func (ma *MetricsAggregator) GetGameMetrics() GameMetrics {
	ma.gameMetrics.mu.RLock()
	defer ma.gameMetrics.mu.RUnlock()
	
	// Create a copy to avoid race conditions
	metrics := *ma.gameMetrics
	metrics.WinnerFrequency = make(map[string]int64)
	metrics.WinTypeDistribution = make(map[string]int64)
	
	for k, v := range ma.gameMetrics.WinnerFrequency {
		metrics.WinnerFrequency[k] = v
	}
	
	for k, v := range ma.gameMetrics.WinTypeDistribution {
		metrics.WinTypeDistribution[k] = v
	}
	
	return metrics
}

// GetPlayerMetrics returns current player metrics
func (ma *MetricsAggregator) GetPlayerMetrics() PlayerMetrics {
	ma.playerMetrics.mu.RLock()
	defer ma.playerMetrics.mu.RUnlock()
	
	// Create a copy to avoid race conditions
	metrics := *ma.playerMetrics
	metrics.ActivePlayers = make(map[string]*PlayerStats)
	
	for k, v := range ma.playerMetrics.ActivePlayers {
		playerCopy := *v
		metrics.ActivePlayers[k] = &playerCopy
	}
	
	return metrics
}

// GetHourlyMetrics returns current hourly metrics
func (ma *MetricsAggregator) GetHourlyMetrics() HourlyMetrics {
	ma.hourlyMetrics.mu.RLock()
	defer ma.hourlyMetrics.mu.RUnlock()
	
	// Create a copy to avoid race conditions
	metrics := *ma.hourlyMetrics
	metrics.GamesPerHour = make(map[string]int64)
	metrics.MovesPerHour = make(map[string]int64)
	metrics.PlayersPerHour = make(map[string]int64)
	metrics.AverageDurationHour = make(map[string]float64)
	
	for k, v := range ma.hourlyMetrics.GamesPerHour {
		metrics.GamesPerHour[k] = v
	}
	for k, v := range ma.hourlyMetrics.MovesPerHour {
		metrics.MovesPerHour[k] = v
	}
	for k, v := range ma.hourlyMetrics.PlayersPerHour {
		metrics.PlayersPerHour[k] = v
	}
	for k, v := range ma.hourlyMetrics.AverageDurationHour {
		metrics.AverageDurationHour[k] = v
	}
	
	return metrics
}

// GetDailyMetrics returns current daily metrics
func (ma *MetricsAggregator) GetDailyMetrics() DailyMetrics {
	ma.dailyMetrics.mu.RLock()
	defer ma.dailyMetrics.mu.RUnlock()
	
	// Create a copy to avoid race conditions
	metrics := *ma.dailyMetrics
	metrics.GamesPerDay = make(map[string]int64)
	metrics.MovesPerDay = make(map[string]int64)
	metrics.PlayersPerDay = make(map[string]int64)
	metrics.AverageDurationDay = make(map[string]float64)
	metrics.NewPlayersPerDay = make(map[string]int64)
	
	for k, v := range ma.dailyMetrics.GamesPerDay {
		metrics.GamesPerDay[k] = v
	}
	for k, v := range ma.dailyMetrics.MovesPerDay {
		metrics.MovesPerDay[k] = v
	}
	for k, v := range ma.dailyMetrics.PlayersPerDay {
		metrics.PlayersPerDay[k] = v
	}
	for k, v := range ma.dailyMetrics.AverageDurationDay {
		metrics.AverageDurationDay[k] = v
	}
	for k, v := range ma.dailyMetrics.NewPlayersPerDay {
		metrics.NewPlayersPerDay[k] = v
	}
	
	return metrics
}

// GetTopWinners returns the most frequent winners
func (ma *MetricsAggregator) GetTopWinners(limit int) []struct {
	Name string
	Wins int64
} {
	ma.gameMetrics.mu.RLock()
	defer ma.gameMetrics.mu.RUnlock()

	type winner struct {
		Name string
		Wins int64
	}

	winners := make([]winner, 0, len(ma.gameMetrics.WinnerFrequency))
	for name, wins := range ma.gameMetrics.WinnerFrequency {
		winners = append(winners, winner{Name: name, Wins: wins})
	}

	// Simple bubble sort by wins (descending)
	for i := 0; i < len(winners)-1; i++ {
		for j := 0; j < len(winners)-i-1; j++ {
			if winners[j].Wins < winners[j+1].Wins {
				winners[j], winners[j+1] = winners[j+1], winners[j]
			}
		}
	}

	if len(winners) > limit {
		winners = winners[:limit]
	}

	result := make([]struct {
		Name string
		Wins int64
	}, len(winners))

	for i, w := range winners {
		result[i] = struct {
			Name string
			Wins int64
		}{Name: w.Name, Wins: w.Wins}
	}

	return result
}

// Flush persists all current metrics
func (ma *MetricsAggregator) Flush() error {
	return ma.AggregateMetrics()
}

// cleanupOldMetrics removes old metric data to prevent memory leaks
func (ma *MetricsAggregator) cleanupOldMetrics() {
	now := time.Now()
	
	// Clean hourly metrics (keep last 7 days)
	cutoffHour := now.Add(-7 * 24 * time.Hour).Format("2006-01-02-15")
	ma.hourlyMetrics.mu.Lock()
	for key := range ma.hourlyMetrics.GamesPerHour {
		if key < cutoffHour {
			delete(ma.hourlyMetrics.GamesPerHour, key)
			delete(ma.hourlyMetrics.MovesPerHour, key)
			delete(ma.hourlyMetrics.PlayersPerHour, key)
			delete(ma.hourlyMetrics.AverageDurationHour, key)
		}
	}
	ma.hourlyMetrics.mu.Unlock()

	// Clean daily metrics (keep last 30 days)
	cutoffDay := now.Add(-30 * 24 * time.Hour).Format("2006-01-02")
	ma.dailyMetrics.mu.Lock()
	for key := range ma.dailyMetrics.GamesPerDay {
		if key < cutoffDay {
			delete(ma.dailyMetrics.GamesPerDay, key)
			delete(ma.dailyMetrics.MovesPerDay, key)
			delete(ma.dailyMetrics.PlayersPerDay, key)
			delete(ma.dailyMetrics.AverageDurationDay, key)
			delete(ma.dailyMetrics.NewPlayersPerDay, key)
		}
	}
	ma.dailyMetrics.mu.Unlock()

	// Mark inactive players (not seen in last 24 hours)
	cutoffTime := now.Add(-24 * time.Hour)
	ma.playerMetrics.mu.Lock()
	for _, player := range ma.playerMetrics.ActivePlayers {
		if player.LastSeen.Before(cutoffTime) {
			player.IsActive = false
		}
	}
	ma.playerMetrics.mu.Unlock()
}

// persistMetrics saves current metrics to database (optional implementation)
func (ma *MetricsAggregator) persistMetrics() error {
	// This could save aggregated metrics to a separate analytics table
	// For now, we'll just log the current state
	
	gameMetrics := ma.GetGameMetrics()
	playerMetrics := ma.GetPlayerMetrics()
	
	log.Printf("Persisting metrics: %d games, %d players, %.1fs avg duration",
		gameMetrics.TotalGames, playerMetrics.TotalPlayers, gameMetrics.AverageGameDuration)
	
	// TODO: Implement actual database persistence if needed
	// This could involve creating analytics tables and storing aggregated data
	
	return nil
}