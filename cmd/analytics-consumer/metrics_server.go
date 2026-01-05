package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"connect-four-backend/internal/kafka"

	"github.com/gorilla/mux"
)

// MetricsServer provides HTTP API for analytics metrics
type MetricsServer struct {
	consumer *kafka.Consumer
	server   *http.Server
	router   *mux.Router
}

// MetricsResponse represents the structure of metrics API responses
type MetricsResponse struct {
	Status    string      `json:"status"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
	Error     string      `json:"error,omitempty"`
}

// NewMetricsServer creates a new metrics API server
func NewMetricsServer(consumer *kafka.Consumer, addr string) *MetricsServer {
	router := mux.NewRouter()
	
	server := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	ms := &MetricsServer{
		consumer: consumer,
		server:   server,
		router:   router,
	}

	ms.setupRoutes()
	return ms
}

// Start starts the metrics server
func (ms *MetricsServer) Start() error {
	log.Printf("Starting metrics API server on %s", ms.server.Addr)
	return ms.server.ListenAndServe()
}

// Stop stops the metrics server
func (ms *MetricsServer) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return ms.server.Shutdown(ctx)
}

// setupRoutes configures all API routes
func (ms *MetricsServer) setupRoutes() {
	// Add CORS middleware
	ms.router.Use(ms.corsMiddleware)
	ms.router.Use(ms.loggingMiddleware)

	// Health check
	ms.router.HandleFunc("/health", ms.handleHealth).Methods("GET")

	// Consumer statistics
	ms.router.HandleFunc("/api/consumer/stats", ms.handleConsumerStats).Methods("GET")

	// Game metrics
	ms.router.HandleFunc("/api/metrics/games", ms.handleGameMetrics).Methods("GET")
	ms.router.HandleFunc("/api/metrics/games/winners", ms.handleTopWinners).Methods("GET")
	ms.router.HandleFunc("/api/metrics/games/duration", ms.handleGameDuration).Methods("GET")

	// Player metrics
	ms.router.HandleFunc("/api/metrics/players", ms.handlePlayerMetrics).Methods("GET")
	ms.router.HandleFunc("/api/metrics/players/top", ms.handleTopPlayers).Methods("GET")
	ms.router.HandleFunc("/api/metrics/players/{name}", ms.handlePlayerStats).Methods("GET")

	// Time-based metrics
	ms.router.HandleFunc("/api/metrics/hourly", ms.handleHourlyMetrics).Methods("GET")
	ms.router.HandleFunc("/api/metrics/daily", ms.handleDailyMetrics).Methods("GET")

	// Real-time metrics
	ms.router.HandleFunc("/api/metrics/realtime", ms.handleRealtimeMetrics).Methods("GET")

	// Dashboard data
	ms.router.HandleFunc("/api/dashboard", ms.handleDashboard).Methods("GET")
}

// Middleware

func (ms *MetricsServer) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (ms *MetricsServer) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %v", r.Method, r.URL.Path, time.Since(start))
	})
}

// Handler methods

func (ms *MetricsServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	stats := ms.consumer.GetStats()
	
	health := map[string]interface{}{
		"status":             "healthy",
		"uptime":             stats.Uptime.String(),
		"messages_processed": stats.MessagesProcessed,
		"messages_errored":   stats.MessagesErrored,
		"last_message":       stats.LastMessageTime,
	}

	ms.writeResponse(w, http.StatusOK, health)
}

func (ms *MetricsServer) handleConsumerStats(w http.ResponseWriter, r *http.Request) {
	stats := ms.consumer.GetStats()
	ms.writeResponse(w, http.StatusOK, stats)
}

func (ms *MetricsServer) handleGameMetrics(w http.ResponseWriter, r *http.Request) {
	// This would need access to the processor's aggregator
	// For now, return mock data structure
	gameMetrics := map[string]interface{}{
		"total_games":          1000,
		"completed_games":      950,
		"average_duration":     180.5,
		"draw_count":          95,
		"bot_games":           300,
		"human_games":         700,
		"win_type_distribution": map[string]int{
			"horizontal": 400,
			"vertical":   300,
			"diagonal":   250,
		},
	}

	ms.writeResponse(w, http.StatusOK, gameMetrics)
}

func (ms *MetricsServer) handleTopWinners(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 10
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	// Mock data - in real implementation, get from aggregator
	topWinners := []map[string]interface{}{
		{"name": "Alice", "wins": 45, "games": 60, "win_rate": 75.0},
		{"name": "Bob", "wins": 38, "games": 55, "win_rate": 69.1},
		{"name": "Charlie", "wins": 32, "games": 50, "win_rate": 64.0},
		{"name": "Diana", "wins": 28, "games": 45, "win_rate": 62.2},
		{"name": "Eve", "wins": 25, "games": 42, "win_rate": 59.5},
	}

	if len(topWinners) > limit {
		topWinners = topWinners[:limit]
	}

	ms.writeResponse(w, http.StatusOK, topWinners)
}

func (ms *MetricsServer) handleGameDuration(w http.ResponseWriter, r *http.Request) {
	durationStats := map[string]interface{}{
		"average_duration":    180.5,
		"median_duration":     165.0,
		"min_duration":        45.0,
		"max_duration":        600.0,
		"duration_buckets": map[string]int{
			"0-60s":    50,
			"60-120s":  200,
			"120-180s": 300,
			"180-300s": 250,
			"300s+":    150,
		},
	}

	ms.writeResponse(w, http.StatusOK, durationStats)
}

func (ms *MetricsServer) handlePlayerMetrics(w http.ResponseWriter, r *http.Request) {
	playerMetrics := map[string]interface{}{
		"total_players":        500,
		"active_players":       150,
		"new_players_today":    25,
		"total_moves":          50000,
		"total_disconnections": 200,
		"total_reconnections":  180,
		"average_session_time": 1200.0,
	}

	ms.writeResponse(w, http.StatusOK, playerMetrics)
}

func (ms *MetricsServer) handleTopPlayers(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 10
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	// Mock data - in real implementation, get from player tracker
	topPlayers := []map[string]interface{}{
		{
			"name":           "Alice",
			"games_played":   60,
			"games_won":      45,
			"win_rate":       75.0,
			"total_moves":    1200,
			"avg_game_time":  180.0,
			"is_online":      true,
		},
		{
			"name":           "Bob",
			"games_played":   55,
			"games_won":      38,
			"win_rate":       69.1,
			"total_moves":    1100,
			"avg_game_time":  175.0,
			"is_online":      false,
		},
	}

	if len(topPlayers) > limit {
		topPlayers = topPlayers[:limit]
	}

	ms.writeResponse(w, http.StatusOK, topPlayers)
}

func (ms *MetricsServer) handlePlayerStats(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	playerName := vars["name"]

	// Mock data - in real implementation, get from player tracker
	playerStats := map[string]interface{}{
		"name":              playerName,
		"games_played":      60,
		"games_won":         45,
		"games_lost":        12,
		"games_drawn":       3,
		"win_rate":          75.0,
		"total_moves":       1200,
		"avg_game_time":     180.0,
		"total_game_time":   10800,
		"disconnections":    5,
		"reconnections":     4,
		"total_offline_time": "2m30s",
		"first_seen":        "2024-01-01T10:00:00Z",
		"last_seen":         "2024-01-04T15:30:00Z",
		"is_online":         true,
	}

	ms.writeResponse(w, http.StatusOK, playerStats)
}

func (ms *MetricsServer) handleHourlyMetrics(w http.ResponseWriter, r *http.Request) {
	hoursStr := r.URL.Query().Get("hours")
	hours := 24
	if hoursStr != "" {
		if h, err := strconv.Atoi(hoursStr); err == nil && h > 0 {
			hours = h
		}
	}

	// Mock hourly data
	hourlyMetrics := make([]map[string]interface{}, hours)
	now := time.Now()
	
	for i := 0; i < hours; i++ {
		hourTime := now.Add(-time.Duration(i) * time.Hour)
		hourlyMetrics[i] = map[string]interface{}{
			"hour":             hourTime.Format("2006-01-02-15"),
			"games_started":    10 + i%5,
			"games_completed":  8 + i%4,
			"total_moves":      200 + i*10,
			"unique_players":   15 + i%3,
			"avg_duration":     180.0 + float64(i%30),
		}
	}

	ms.writeResponse(w, http.StatusOK, hourlyMetrics)
}

func (ms *MetricsServer) handleDailyMetrics(w http.ResponseWriter, r *http.Request) {
	daysStr := r.URL.Query().Get("days")
	days := 7
	if daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 {
			days = d
		}
	}

	// Mock daily data
	dailyMetrics := make([]map[string]interface{}, days)
	now := time.Now()
	
	for i := 0; i < days; i++ {
		dayTime := now.Add(-time.Duration(i) * 24 * time.Hour)
		dailyMetrics[i] = map[string]interface{}{
			"day":              dayTime.Format("2006-01-02"),
			"games_started":    200 + i*20,
			"games_completed":  180 + i*18,
			"total_moves":      4000 + i*400,
			"unique_players":   100 + i*10,
			"new_players":      20 + i*2,
			"avg_duration":     185.0 + float64(i*5),
		}
	}

	ms.writeResponse(w, http.StatusOK, dailyMetrics)
}

func (ms *MetricsServer) handleRealtimeMetrics(w http.ResponseWriter, r *http.Request) {
	realtimeMetrics := map[string]interface{}{
		"active_games":       25,
		"online_players":     75,
		"games_this_hour":    12,
		"moves_this_hour":    240,
		"avg_response_time":  150.0,
		"server_uptime":      "2d 14h 30m",
		"last_updated":       time.Now(),
	}

	ms.writeResponse(w, http.StatusOK, realtimeMetrics)
}

func (ms *MetricsServer) handleDashboard(w http.ResponseWriter, r *http.Request) {
	dashboard := map[string]interface{}{
		"overview": map[string]interface{}{
			"total_games":      1000,
			"active_games":     25,
			"total_players":    500,
			"online_players":   75,
			"games_today":      120,
			"avg_duration":     180.5,
		},
		"recent_activity": []map[string]interface{}{
			{
				"type":      "game_completed",
				"players":   []string{"Alice", "Bob"},
				"winner":    "Alice",
				"duration":  165,
				"timestamp": time.Now().Add(-5 * time.Minute),
			},
			{
				"type":      "player_joined",
				"player":    "Charlie",
				"timestamp": time.Now().Add(-10 * time.Minute),
			},
		},
		"top_players": []map[string]interface{}{
			{"name": "Alice", "wins": 45, "win_rate": 75.0},
			{"name": "Bob", "wins": 38, "win_rate": 69.1},
			{"name": "Charlie", "wins": 32, "win_rate": 64.0},
		},
		"hourly_games": []int{8, 12, 15, 18, 22, 25, 20, 16, 14, 10, 8, 12},
	}

	ms.writeResponse(w, http.StatusOK, dashboard)
}

// Helper methods

func (ms *MetricsServer) writeResponse(w http.ResponseWriter, status int, data interface{}) {
	response := MetricsResponse{
		Status:    "success",
		Timestamp: time.Now(),
		Data:      data,
	}

	if status >= 400 {
		response.Status = "error"
		if errMsg, ok := data.(string); ok {
			response.Error = errMsg
			response.Data = nil
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}

func (ms *MetricsServer) writeError(w http.ResponseWriter, status int, message string) {
	ms.writeResponse(w, status, message)
}