package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"sync"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/mapleleafu/flaparena/flaparena-backend/models"
	"github.com/mapleleafu/flaparena/flaparena-backend/repository"
	"github.com/mapleleafu/flaparena/flaparena-backend/responses"
	"github.com/mapleleafu/flaparena/flaparena-backend/utils"
)

var upgrader = websocket.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
    CheckOrigin:     func(r *http.Request) bool { return true },
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

    connection := &Connection{
        ws:     conn,
        send:   make(chan []byte, 256),
        userID: userID,
    }

    // Register the connection to the hub for broadcasting and message handling
    hub.register <- connection

    // Cleanup on disconnect
    defer func() { hub.unregister <- connection }()

    // Setup message pumps
    go connection.writePump()
    go connection.readPump()
}

func (c *Connection) readPump() {
    defer func() {
        c.ws.Close()
    }()
    
    for {
        _, message, err := c.ws.ReadMessage()
        if err != nil {
            if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
                log.Printf("error: %v", err)
            }
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
            return
        }
    }
}

func processMessage(c *Connection, message []byte) {
    var msg models.GameAction
    if err := json.Unmarshal(message, &msg); err != nil {
        log.Printf("error unmarshalling message: %v", err)
        return
    }
    
    switch msg.Action {
    case "ready":
        log.Printf("ready message")
    case "up":
        log.Printf("up message")
    }
    
    // Broadcast the message to other players
    hub.broadcast <- message
}

var (
    gameSessions = make(map[string]*models.GameSession)
    gameSessionsMutex = &sync.Mutex{}
)

func handleGameAction(wsMessage []byte, gameID string, userID string) {
    // Unmarshal the incoming WebSocket message into a GameAction
    var action models.GameAction
    json.Unmarshal(wsMessage, &action)

    // Safely access the gameSessions map
    gameSessionsMutex.Lock()
    defer gameSessionsMutex.Unlock()

    // Initialize the game session in the map if it doesn't exist
    if _, exists := gameSessions[gameID]; !exists {
        gameSessions[gameID] = &models.GameSession{}
    }

    // Add the action to the session
    gameSessions[gameID].Actions = append(gameSessions[gameID].Actions, action)
    log.Printf("Game session %s: %v", gameID, gameSessions[gameID])
}

func saveGameSessionToMongoDB(gameID string) {
    gameSessionsMutex.Lock()
    session, exists := gameSessions[gameID]
    gameSessionsMutex.Unlock()

    if !exists {
        log.Printf("Game session %s not found", gameID)
        return
    }

    collection := repository.MongoDBClient.Database("flaparena").Collection("game_actions")
    for _, action := range session.Actions {
        _, err := collection.InsertOne(context.Background(), action)
        if err != nil {
            log.Printf("Failed to insert game action into MongoDB: %v", err)
        }
    }

    // Cleanup: Remove the session from the map after saving
    gameSessionsMutex.Lock()
    delete(gameSessions, gameID)
    gameSessionsMutex.Unlock()

    log.Printf("Game session %s actions saved to MongoDB", gameID)
}

var currentGameState = &models.GameState{
    Players: make(map[string]*models.PlayerState),
    Started: false,
}

func playerReady(userID string) {
    currentGameState.Mutex.Lock()
    defer currentGameState.Mutex.Unlock()

    if player, exists := currentGameState.Players[userID]; exists {
        player.Ready = true
    } else {
        currentGameState.Players[userID] = &models.PlayerState{
            UserID:   userID,
            Ready:    true,
            Alive:    true,
            Score:    0,
        }
    }
    log.Printf("Player %s is ready", userID)
}

func checkAllPlayersReady() bool {
    for _, player := range currentGameState.Players {
        if !player.Ready {
            return false
        }
    }
    return true
}

func startGame() {
    currentGameState.Mutex.Lock()
    defer currentGameState.Mutex.Unlock()

    if checkAllPlayersReady() && !currentGameState.Started {
        currentGameState.Started = true

        log.Println("Game started")
    }
}
