package handlers

import (
	"encoding/json"
	"net/http"

	"connect-four-backend/internal/database"
)

type LeaderboardHandler struct {
	db *database.PostgresDB
}

func NewLeaderboardHandler(db *database.PostgresDB) *LeaderboardHandler {
	return &LeaderboardHandler{
		db: db,
	}
}

func (h *LeaderboardHandler) GetLeaderboard(w http.ResponseWriter, r *http.Request) {
	leaderboard, err := h.db.GetLeaderboard(50) // Top 50 players
	if err != nil {
		http.Error(w, "Failed to fetch leaderboard", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(leaderboard)
}

func (h *LeaderboardHandler) GetPlayerStats(w http.ResponseWriter, r *http.Request) {
	playerName := r.URL.Query().Get("name")
	if playerName == "" {
		http.Error(w, "Player name is required", http.StatusBadRequest)
		return
	}

	stats, err := h.db.GetPlayerStats(playerName)
	if err != nil {
		http.Error(w, "Failed to fetch player stats", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}