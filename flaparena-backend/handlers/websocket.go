package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"sync"

    "go.mongodb.org/mongo-driver/bson/primitive"
    "github.com/google/uuid"
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

    connection := &Connection{send: make(chan []byte, 256), ws: conn, userID: userID}

    // Register the connection to the hub for broadcasting and message handling
    hub.register <- connection

    // Cleanup on disconnect
    defer func() { hub.unregister <- connection }()

    go connection.writePump()
    connection.readPump()
}

func (c *Connection) readPump() {
    defer func() {
        hub.unregister <- c
        c.ws.Close()
        }()

    for {
        _, message, err := c.ws.ReadMessage()
        if err != nil {
            log.Printf("error reading message: %v", err)
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
            log.Printf("error writing message: %v", err)
            break
        }
    }
}

func processMessage(c *Connection, message []byte) {
    var msg models.GameAction
    if err := json.Unmarshal(message, &msg); err != nil {
        log.Printf("error unmarshalling message: %v", err)
        return
    }
    
    userIDStr := strconv.FormatUint(c.userID, 10)
    message = append([]byte("UserID: " + userIDStr + ": "), message...)
    
    switch msg.Action {
    case "ready":
        if !currentGameState.Started {
            if _, exists := currentGameState.Players[userIDStr]; exists {
                hub.broadcast <- append([]byte("You already readied up UserID: "), []byte(userIDStr)...)
            } else {
                playerReady(userIDStr)
                hub.broadcast <- message
                startGame()
            }
        } else {
            hub.broadcast <- append([]byte("Game already started UserID: "), []byte(userIDStr)...)
        }
    case "flap":
        if currentGameState.Started {
            if _, exists := currentGameState.Players[userIDStr]; exists {
                hub.broadcast <- message
                handleGameAction(message, msg.GameID, userIDStr)
            } else {
                hub.broadcast <- append([]byte("Player not found UserID: "), []byte(userIDStr)...)
            }
        } else {
            hub.broadcast <- append([]byte("Game not started UserID: "), []byte(userIDStr)...)
        }
    case "score":
        if currentGameState.Started {
            if _, exists := currentGameState.Players[userIDStr]; exists {
                playerScored(userIDStr)
                handleGameAction(message, msg.GameID, userIDStr)
            } else {
                hub.broadcast <- append([]byte("Player not found UserID: "), []byte(userIDStr)...)
            }
        } else {
            hub.broadcast <- append([]byte("Game not started UserID: "), []byte(userIDStr)...)
        }
    case "dead":
        if currentGameState.Started {
            if _, exists := currentGameState.Players[userIDStr]; exists {
                hub.broadcast <- message
                playerDead(userIDStr)
                handleGameAction(message, msg.GameID, userIDStr)
                if checkAllPlayersDead() {
                    endGame()
                }
            } else {
                hub.broadcast <- append([]byte("Player not found UserID: "), []byte(userIDStr)...)
            }
        } else {
            hub.broadcast <- append([]byte("Game not started UserID: "), []byte(userIDStr)...)
        }
    }
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
}

func saveGameSessionToMongoDB(placeholderID string) {
    gameSessionsMutex.Lock()
    session, exists := gameSessions[placeholderID]
    if !exists {
        log.Printf("Game session with placeholder ID %s not found", placeholderID)
        gameSessionsMutex.Unlock()
        return
    }
    delete(gameSessions, placeholderID) // Remove the session from the map
    gameSessionsMutex.Unlock()

    collection := repository.MongoDBClient.Database("flaparena").Collection("game_sessions")
    result, err := collection.InsertOne(context.Background(), session)
    if err != nil {
        log.Printf("Failed to insert game session into MongoDB: %v", err)
        return
    }

    // Correctly handle the InsertedID as primitive.ObjectID and convert it to string
    realGameID := result.InsertedID.(primitive.ObjectID).Hex()
    log.Printf("Game session saved to MongoDB with ID %s", realGameID)
}

var currentGameState = &models.GameState{
    Players: make(map[string]*models.PlayerState),
    Started: false,
    Mutex:   sync.Mutex{},
    GameID: "",
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
        log.Print(player)
        if !player.Ready {
            return false
        }
    }
    return true
}

func checkAllPlayersDead() bool {
    for _, player := range currentGameState.Players {
        if player.Alive {
            return false
        }
    }
    return true
}

func playerScored(userID string) {
    currentGameState.Mutex.Lock()
    defer currentGameState.Mutex.Unlock()

    if player, exists := currentGameState.Players[userID]; exists {
        player.Score++
    }
}

func playerDead(userID string) {
    currentGameState.Mutex.Lock()
    defer currentGameState.Mutex.Unlock()

    if player, exists := currentGameState.Players[userID]; exists {
        player.Alive = false
    }
}

func startGame() {
    currentGameState.Mutex.Lock()
    defer currentGameState.Mutex.Unlock()

    readyPlayers := 0
    for _, player := range currentGameState.Players {
        if player.Ready && readyPlayers <= 20 {
            readyPlayers++
        } else {
            log.Printf("Max is 20 players, %d players are ready", readyPlayers)
        }
    }
    log.Printf("readyPlayers: %d", readyPlayers)
    if readyPlayers >= 2 && !currentGameState.Started && checkAllPlayersReady() {
        log.Printf("Starting game with %d players", readyPlayers)
        GameID := startNewGameSession()
        log.Printf("after startNewGameSession: %s", GameID)
        currentGameState.GameID = GameID
        currentGameState.Started = true
        log.Println("Game started")
        hub.broadcast <- []byte("Game started!")
    }
}

func endGame() {
    log.Printf("Ending game with placeholderID %s", currentGameState.GameID)
    if checkAllPlayersDead() {
        currentGameState.Mutex.Lock()
        gameID := currentGameState.GameID
        currentGameState.Mutex.Unlock()
        currentGameState.Started = false
        saveGameSessionToMongoDB(gameID)
        currentGameState.GameID = "" // Reset placeholder ID
        log.Println("Game ended")
        hub.broadcast <- []byte("Game ended!")
    } else {
        log.Println("Not all players dead yet.")
    }
}

func startNewGameSession() string {
    log.Println("Generating GameID...")
    placeholderID  := generatePlaceholderID()
    log.Println("Generated GameID:", placeholderID )

    // Initialize a new game session with this ID
    gameSessionsMutex.Lock()
    defer gameSessionsMutex.Unlock()
    gameSessions[placeholderID] = &models.GameSession{}
    return placeholderID
}

func generatePlaceholderID() string {
    return uuid.New().String()
}
