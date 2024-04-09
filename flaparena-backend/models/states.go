package models

import (
	"sync"
)

type PlayerState struct {
    UserID string
    Username string
    Connected bool
    Ready bool
    Alive bool
    Score int
}

type GameState struct {
    Players map[string]*PlayerState
    Started bool
    Mutex   sync.Mutex
    GameID string
}
