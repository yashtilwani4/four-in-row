package matchmaking

import "errors"

var (
	// Service errors
	ErrServiceAlreadyRunning = errors.New("matchmaking service is already running")
	ErrServiceNotRunning     = errors.New("matchmaking service is not running")
	ErrServiceShuttingDown   = errors.New("matchmaking service is shutting down")
	
	// Queue errors
	ErrQueueFull         = errors.New("matchmaking queue is full")
	ErrPlayerNotInQueue  = errors.New("player is not in queue")
	ErrPlayerAlreadyInQueue = errors.New("player is already in queue")
	
	// Request errors
	ErrRequestTimeout    = errors.New("request timeout")
	ErrInvalidRequest    = errors.New("invalid request")
	ErrInvalidPlayerID   = errors.New("invalid player ID")
	ErrInvalidUsername   = errors.New("invalid username")
	
	// Match errors
	ErrMatchCreationFailed = errors.New("failed to create match")
	ErrBotCreationFailed   = errors.New("failed to create bot")
	ErrGameCreationFailed  = errors.New("failed to create game")
)