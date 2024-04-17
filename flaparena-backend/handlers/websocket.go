package handlers

import (
	"log"
	"net/http"
	"strconv"
	"sync"
    "encoding/json"
    "time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/mapleleafu/flaparena/flaparena-backend/models"
	"github.com/mapleleafu/flaparena/flaparena-backend/responses"
	"github.com/mapleleafu/flaparena/flaparena-backend/utils"
)

var upgrader = websocket.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
    CheckOrigin:     func(r *http.Request) bool { return true },
}

var (
    gameSessions = make(map[string]*models.GameSession)
    gameSessionsMutex = &sync.Mutex{}
)

var currentGameState = &models.GameState{
    Players: make(map[string]*models.PlayerState),
    Started: false,
    Mutex:   sync.Mutex{},
    GameID: "",
}

func WsHandler(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    tokenStr := vars["token"]

    // Validate the token
    claims, err := ValidateToken(tokenStr)
    if err != nil {
        log.Println(err)
        utils.HandleError(w, responses.UnauthorizedError{Msg: "Error validating token."})
        return
    }

    // Convert ID back to uint
    userID, err := strconv.ParseUint(claims.ID, 10, 64)
    if err != nil {
        log.Println(err)
        return
    }

    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Println("Upgrade error:", err)
        return
    }
    defer conn.Close()

    connection := &Connection{send: make(chan []byte, 256), ws: conn, userID: userID, username: claims.Username}

    // Register the connection to the hub for broadcasting and message handling
    hub.register <- connection

    // Convert userID to string for map index
    userIDStr := strconv.FormatUint(userID, 10)

    // Update the player's state in currentGameState
    currentGameState.Mutex.Lock()
    currentGameState.Players[userIDStr] = &models.PlayerState{
        UserID:   userIDStr,
        Username: claims.Username,
        Connected: true,
        Ready:    false,
        Alive:    false,
        Score:    0,
    }
    currentGameState.Mutex.Unlock()

    log.Printf("User %s connected", userIDStr)

    // Broadcast updated game state to all connections
    broadcastGameState()

    // Setup clean up for when the connection is closed
    defer func() { 
        hub.unregister <- connection
        // Remove the player from the game state
        currentGameState.Mutex.Lock()
        delete(currentGameState.Players, userIDStr)
        currentGameState.Mutex.Unlock()
        broadcastGameState()
    }()

    go connection.writePump()
    connection.readPump()
}

func (c *Connection) readPump() {
    defer func() {
        // Before unregistering, check if the game has started and mark the player as dead
        if currentGameState.Started {
            userIDStr := strconv.FormatUint(c.userID, 10)
            if player, exists := currentGameState.Players[userIDStr]; exists && player.Alive {
                deadAction := models.GameAction{
                    UserID:    userIDStr,
                    Action:    "dead",
                    Timestamp: time.Now().UnixMilli(),
                }
                handleDeadAction(deadAction)
            }
        }
        
        // Unregister the connection and close the WebSocket
        hub.unregister <- c
        c.ws.Close()
        log.Printf("User %d disconnected", c.userID)
    }()

    for {
        _, message, err := c.ws.ReadMessage()
        if err != nil {
            log.Printf("Error reading message from userID %d: %v", c.userID, err)
            break
        }
        processMessage(c, message)
    }
}

func (c *Connection) writePump() {
    defer func() {
        c.ws.Close()
    }()

    for message := range c.send {
        if err := c.ws.WriteMessage(websocket.TextMessage, message); err != nil {
            log.Printf("error writing message: %v", err)
            break
        }
    }
}

func broadcastGameState() {
    currentGameState.Mutex.Lock()
    defer currentGameState.Mutex.Unlock()

    gameState := make([]map[string]interface{}, 0)
    for _, player := range currentGameState.Players {
        gameState = append(gameState, map[string]interface{}{
            "userID":   player.UserID,
            "username": player.Username,
            "connected": player.Connected,
            "ready":    player.Ready,
            "alive":    player.Alive,
            "score":    player.Score,
        })
    }

    message, _ := json.Marshal(map[string]interface{}{
        "type": "gameState",
        "data": gameState,
    })
    hub.broadcast <- message
}