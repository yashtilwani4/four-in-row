package matchmaking

import (
	"sync"
	"time"

	"connect-four-backend/internal/game"
	"connect-four-backend/internal/models"
	"github.com/google/uuid"
)

// QueueEntry represents a player waiting in the matchmaking queue
type QueueEntry struct {
	PlayerID    uuid.UUID `json:"player_id"`
	Username    string    `json:"username"`
	JoinedAt    time.Time `json:"joined_at"`
	BotTimer    *time.Timer `json:"-"`
	Preferences *MatchPreferences `json:"preferences,omitempty"`
	
	// Additional fields for compatibility with matchmaker
	Player      *models.Player `json:"-"`
	Conn        game.WSConnection `json:"-"`
}

// MatchPreferences holds player preferences for matchmaking
type MatchPreferences struct {
	AllowBots     bool `json:"allow_bots"`
	SkillLevel    int  `json:"skill_level"`    // 1-10 scale
	MaxWaitTime   int  `json:"max_wait_time"`  // seconds
}

// Queue manages the matchmaking queue with thread-safe operations
type Queue struct {
	entries map[uuid.UUID]*QueueEntry
	mutex   sync.RWMutex
	
	// Channels for queue operations
	addChan    chan *QueueEntry
	removeChan chan uuid.UUID
	
	// Queue statistics
	stats QueueStats
}

// QueueStats holds queue statistics
type QueueStats struct {
	TotalJoined     int64         `json:"total_joined"`
	TotalLeft       int64         `json:"total_left"`
	TotalMatched    int64         `json:"total_matched"`
	TotalBotMatches int64         `json:"total_bot_matches"`
	CurrentSize     int           `json:"current_size"`
	AverageWaitTime time.Duration `json:"average_wait_time"`
	mutex           sync.RWMutex
}

// NewQueue creates a new matchmaking queue
func NewQueue() *Queue {
	return &Queue{
		entries:    make(map[uuid.UUID]*QueueEntry),
		addChan:    make(chan *QueueEntry, 100),
		removeChan: make(chan uuid.UUID, 100),
	}
}

// Add adds a player to the queue
func (q *Queue) Add(playerID uuid.UUID, username string, preferences *MatchPreferences) *QueueEntry {
	if preferences == nil {
		preferences = &MatchPreferences{
			AllowBots:   true,
			SkillLevel:  5,
			MaxWaitTime: 10,
		}
	}

	entry := &QueueEntry{
		PlayerID:    playerID,
		Username:    username,
		JoinedAt:    time.Now(),
		Preferences: preferences,
	}

	q.addChan <- entry
	return entry
}

// Remove removes a player from the queue
func (q *Queue) Remove(playerID uuid.UUID) bool {
	q.removeChan <- playerID
	return true
}

// GetEntry gets a queue entry by player ID
func (q *Queue) GetEntry(playerID uuid.UUID) (*QueueEntry, bool) {
	q.mutex.RLock()
	defer q.mutex.RUnlock()
	
	entry, exists := q.entries[playerID]
	return entry, exists
}

// GetSize returns the current queue size
func (q *Queue) GetSize() int {
	q.mutex.RLock()
	defer q.mutex.RUnlock()
	return len(q.entries)
}

// GetOldestEntry returns the oldest entry in the queue
func (q *Queue) GetOldestEntry() *QueueEntry {
	q.mutex.RLock()
	defer q.mutex.RUnlock()

	var oldest *QueueEntry
	for _, entry := range q.entries {
		if oldest == nil || entry.JoinedAt.Before(oldest.JoinedAt) {
			oldest = entry
		}
	}
	return oldest
}

// GetCompatibleMatch finds a compatible match for the given entry
func (q *Queue) GetCompatibleMatch(entry *QueueEntry) *QueueEntry {
	q.mutex.RLock()
	defer q.mutex.RUnlock()

	for _, candidate := range q.entries {
		if candidate.PlayerID == entry.PlayerID {
			continue
		}

		if q.areCompatible(entry, candidate) {
			return candidate
		}
	}
	return nil
}

// GetAllEntries returns all queue entries (for debugging/monitoring)
func (q *Queue) GetAllEntries() []*QueueEntry {
	q.mutex.RLock()
	defer q.mutex.RUnlock()

	entries := make([]*QueueEntry, 0, len(q.entries))
	for _, entry := range q.entries {
		entries = append(entries, entry)
	}
	return entries
}

// GetStats returns queue statistics
func (q *Queue) GetStats() QueueStats {
	q.stats.mutex.RLock()
	defer q.stats.mutex.RUnlock()
	
	q.mutex.RLock()
	q.stats.CurrentSize = len(q.entries)
	q.mutex.RUnlock()
	
	return q.stats
}

// processAdd handles adding entries to the queue
func (q *Queue) processAdd(entry *QueueEntry) {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	q.entries[entry.PlayerID] = entry
	
	// Update statistics
	q.stats.mutex.Lock()
	q.stats.TotalJoined++
	q.stats.mutex.Unlock()
}

// processRemove handles removing entries from the queue
func (q *Queue) processRemove(playerID uuid.UUID) {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	if entry, exists := q.entries[playerID]; exists {
		// Cancel bot timer if it exists
		if entry.BotTimer != nil {
			entry.BotTimer.Stop()
		}
		
		delete(q.entries, playerID)
		
		// Update statistics
		q.stats.mutex.Lock()
		q.stats.TotalLeft++
		q.stats.mutex.Unlock()
	}
}

// areCompatible checks if two players are compatible for matching
func (q *Queue) areCompatible(player1, player2 *QueueEntry) bool {
	// Basic compatibility check - can be extended with more sophisticated logic
	skillDiff := abs(player1.Preferences.SkillLevel - player2.Preferences.SkillLevel)
	
	// Allow skill difference of up to 2 levels
	if skillDiff > 2 {
		return false
	}

	// Both players must allow the match
	return true
}

// updateAverageWaitTime updates the average wait time statistic
func (q *Queue) updateAverageWaitTime(waitTime time.Duration) {
	q.stats.mutex.Lock()
	defer q.stats.mutex.Unlock()

	// Simple moving average calculation
	if q.stats.AverageWaitTime == 0 {
		q.stats.AverageWaitTime = waitTime
	} else {
		q.stats.AverageWaitTime = (q.stats.AverageWaitTime + waitTime) / 2
	}
}

// incrementMatched increments the matched counter
func (q *Queue) incrementMatched() {
	q.stats.mutex.Lock()
	q.stats.TotalMatched++
	q.stats.mutex.Unlock()
}

// incrementBotMatches increments the bot match counter
func (q *Queue) incrementBotMatches() {
	q.stats.mutex.Lock()
	q.stats.TotalBotMatches++
	q.stats.mutex.Unlock()
}

// Helper function
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}