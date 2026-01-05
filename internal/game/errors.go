package game

import "errors"

var (
	ErrGameNotFound     = errors.New("game not found")
	ErrGameNotActive    = errors.New("game is not active")
	ErrPlayerNotInGame  = errors.New("player not in game")
	ErrNotPlayerTurn    = errors.New("not player's turn")
	ErrInvalidMove      = errors.New("invalid move")
)