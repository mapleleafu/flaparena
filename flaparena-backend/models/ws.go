package models
import "sync"

type ConnectionInfo struct {
    UserID    string
    Username  string
    Connected bool
    Ready     bool
}

type LobbyInfo struct {
    UserID    string `json:"userID"`
    Username  string `json:"username"`
    Connected bool   `json:"connected"`
    Ready     bool   `json:"ready"`
}

// Lobby represents a collection of active connections and their states.
type Lobby struct {
    Connections map[string]*ConnectionInfo
    Mutex       sync.Mutex // Ensure thread-safe access to the Connections map.
}

