package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"sync"
    "fmt"
    "time"

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

var (
    gameSessions = make(map[string]*models.GameSession)
    gameSessionsMutex = &sync.Mutex{}
)

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

func processMessage(c *Connection, rawMessage []byte) {
    log.Printf("Raw message: %s", string(rawMessage))

    var gameActionMsg models.GameActionMessage
    err := json.Unmarshal(rawMessage, &gameActionMsg)
    if err != nil {
        log.Printf("Error unmarshalling game action message: %v", err)
        return
    }

    userIDStr := strconv.FormatUint(c.userID, 10)
    gameAction := models.GameAction{
        UserID:    userIDStr,
        Action:    gameActionMsg.Action,
        Timestamp: gameActionMsg.Timestamp,
    }

    switch gameAction.Action {
    case "ready":
        handleReadyAction(gameAction, c)
    case "flap":
        handleFlapAction(gameAction, c)
    case "score":
        handleScoreAction(gameAction, c)
    case "dead":
        handleDeadAction(gameAction, c)
    default:
        log.Printf("Unhandled game action: %s", gameAction.Action)
    }
}

func handleReadyAction(action models.GameAction, c *Connection) {
    log.Printf("Player %s is ready", action.UserID)

    currentGameState.Mutex.Lock()
    defer currentGameState.Mutex.Unlock()

    if !currentGameState.Started {
        if _, exists := currentGameState.Players[action.UserID]; !exists {
            currentGameState.Players[action.UserID] = &models.PlayerState{
                UserID:   action.UserID,
                Ready:    true,
                Alive:    true,
                Score:    0,
            }
            hub.broadcast <- []byte(fmt.Sprintf("'SERVER' - Player %s readied up.", action.UserID))
            log.Printf("after broadcast in handleReadyAction")
            startGame()
            log.Printf("after startGame() in handleReadyAction")
        } else {
            hub.broadcast <- []byte(fmt.Sprintf("'SERVER' - You already readied up UserID: %s", action.UserID))
        }
    } else {
        hub.broadcast <- []byte(fmt.Sprintf("'SERVER' - Game already started UserID: %s", action.UserID))
    }
}

func handleFlapAction(action models.GameAction, c *Connection) {
    log.Printf("Player %s flapped", action.UserID)

    currentGameState.Mutex.Lock()
    defer currentGameState.Mutex.Unlock()

    if currentGameState.Started {
        if _, exists := currentGameState.Players[action.UserID]; exists {
            broadcastAction := fmt.Sprintf("'SERVER' - Player %s flapped.", action.UserID)
            hub.broadcast <- []byte(broadcastAction)
            handleGameAction(action, currentGameState.GameID)
        } else {
            message := fmt.Sprintf("'SERVER' - Player not found. Ready up UserID: %s", action.UserID)
            hub.broadcast <- []byte(message)
        }
    } else {
        message := fmt.Sprintf("'SERVER' - Game not started UserID: %s", action.UserID)
        hub.broadcast <- []byte(message)
    }
}

func handleScoreAction(action models.GameAction, c *Connection) {
    log.Printf("Player %s scored", action.UserID)

    currentGameState.Mutex.Lock()
    defer currentGameState.Mutex.Unlock()

    if currentGameState.Started {
        if _, exists := currentGameState.Players[action.UserID]; exists {
            playerScored(action.UserID)
            handleGameAction(action, currentGameState.GameID)
        } else {
            hub.broadcast <- []byte(fmt.Sprintf("'SERVER' - Player not found. Ready up UserID: %s", action.UserID))
        }
    } else {
        hub.broadcast <- []byte(fmt.Sprintf("'SERVER' - Game not started UserID: %s", action.UserID))
    }
}

func handleDeadAction(action models.GameAction, c *Connection) {
    currentGameState.Mutex.Lock()
    defer currentGameState.Mutex.Unlock()

    if currentGameState.Started {
        if player, exists := currentGameState.Players[action.UserID]; exists {
            // Player is dead
            player.Alive = false
            log.Printf("Player %s is dead", action.UserID)

            // Broadcast the 'dead' action to all clients
            broadcastAction := fmt.Sprintf("'SERVER' - Player %s is dead.", action.UserID)
            hub.broadcast <- []byte(broadcastAction)

            // Log the action in the game session
            handleGameAction(action, currentGameState.GameID)

            // Check if all players are dead and end the game if so
            if checkAllPlayersDead() {
                endGame()
            }
        } else {
            // Player not found
            missingPlayerMessage := fmt.Sprintf("'SERVER' - Player not found. Ready up UserID: %s", action.UserID)
            hub.broadcast <- []byte(missingPlayerMessage)
        }
    } else {
        // Game not started
        gameNotStartedMessage := fmt.Sprintf("'SERVER' - Game not started UserID: %s", action.UserID)
        hub.broadcast <- []byte(gameNotStartedMessage)
    }
}

func handleGameAction(action models.GameAction, gameID string) {
    // Safely access the gameSessions map.
    gameSessionsMutex.Lock()
    defer gameSessionsMutex.Unlock()

    // Initialize the game session in the map if it doesn't exist.
    if _, exists := gameSessions[gameID]; !exists {
        gameSessions[gameID] = &models.GameSession{}
    }

    // Add the action to the session.
    gameSessions[gameID].Actions = append(gameSessions[gameID].Actions, action)
}


// func handleGameAction(wsMessage []byte, gameID string, userID string) {
//     // Unmarshal the incoming WebSocket message into a GameAction
//     var action models.GameAction
//     log.Printf("Raw message: %s", string(wsMessage))
//     err := json.Unmarshal(wsMessage, &action)
//     if err != nil {
//         log.Printf("Error unmarshalling wsMessage: %v", err)
//         return
//     }

//     // Safely access the gameSessions map
//     gameSessionsMutex.Lock()
//     defer gameSessionsMutex.Unlock()

//     // Initialize the game session in the map if it doesn't exist
//     if _, exists := gameSessions[gameID]; !exists {
//         gameSessions[gameID] = &models.GameSession{}
//     }

//     // Add the action to the session
//     gameSessions[gameID].Actions = append(gameSessions[gameID].Actions, action)
// }

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


func startGame() {
    // currentGameState.Mutex.Lock()
    // defer currentGameState.Mutex.Unlock()

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
        currentGameState.GameID = GameID
        currentGameState.Started = true
        log.Println("Game started")
        gameStartedAction := models.GameAction{
            UserID:    "server",
            Action:    "startGame",
            Timestamp: time.Now().UnixNano() / int64(time.Millisecond),
        }
        handleGameAction(gameStartedAction, currentGameState.GameID)
        hub.broadcast <- []byte("'SERVER' - Game started!")
    } else {
        log.Println("Not all players ready yet or game already started or not enough players ready yet.")
    }
}

func endGame() {
    log.Printf("Ending game with placeholderID %s", currentGameState.GameID)
    if checkAllPlayersDead() {
        currentGameState.Mutex.Lock()
        gameID := currentGameState.GameID
        currentGameState.Mutex.Unlock()
        currentGameState.Started = false
        log.Println("Game ended")

        gameEndedAction := models.GameAction{
            UserID:    "server",
            Action:    "Game ended",
            Timestamp: time.Now().UnixNano() / int64(time.Millisecond),
        }
        handleGameAction(gameEndedAction, currentGameState.GameID)

        hub.broadcast <- []byte("'SERVER' - Game ended!")
        saveGameSessionToMongoDB(gameID)
        currentGameState.GameID = "" // Reset placeholder ID
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
