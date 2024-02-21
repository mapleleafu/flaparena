package models

import (
	"sync"
)

type PlayerState struct {
    UserID   string
    Ready    bool
    Alive    bool
    Score    int
}

type GameState struct {
    Players map[string]*PlayerState
    Started bool
    Mutex   sync.Mutex
}
