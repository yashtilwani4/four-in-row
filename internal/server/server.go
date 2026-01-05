package server

import (
	"context"
	"net/http"
	"time"

	"connect-four-backend/internal/config"
	"connect-four-backend/internal/handlers"

	"github.com/gorilla/mux"
)

type Server struct {
	httpServer *http.Server
	config     *config.Config
}

func NewServer(cfg *config.Config, gameHandler *handlers.GameHandler, leaderboardHandler *handlers.LeaderboardHandler) *Server {
	router := mux.NewRouter()

	// WebSocket endpoint for game connections
	router.HandleFunc("/ws", gameHandler.HandleWebSocket)

	// REST API endpoints
	api := router.PathPrefix("/api").Subrouter()
	api.HandleFunc("/leaderboard", leaderboardHandler.GetLeaderboard).Methods("GET")
	api.HandleFunc("/player/stats", leaderboardHandler.GetPlayerStats).Methods("GET")

	// Health check endpoint
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}).Methods("GET")

	// Serve static files (React frontend)
	router.PathPrefix("/").Handler(http.FileServer(http.Dir("./web/build/")))

	// CORS middleware
	router.Use(corsMiddleware)

	httpServer := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return &Server{
		httpServer: httpServer,
		config:     cfg,
	}
}

func (s *Server) Start() error {
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

func corsMiddleware(next http.Handler) http.Handler {
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