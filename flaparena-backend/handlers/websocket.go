package handlers

import (
    "log"
    "net/http"
    "strconv"
    "encoding/json"
    "context"
    "sync"

    "github.com/gorilla/mux"
    "github.com/gorilla/websocket"
    "github.com/mapleleafu/flaparena/flaparena-backend/responses"
    "github.com/mapleleafu/flaparena/flaparena-backend/utils"
    "github.com/mapleleafu/flaparena/flaparena-backend/models"
    "github.com/mapleleafu/flaparena/flaparena-backend/repository"
)

var upgrader = websocket.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
    CheckOrigin:     func(r *http.Request) bool { return true }, // Note: Check the origin in production
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

    log.Printf("Token validated for user ID: %d, Username: %s", userID, claims.Username)

    conn, err := upgrader.Upgrade(w, r, nil)

    if err != nil {
        log.Println(err)
        return
    }
    defer conn.Close()

    connection := &Connection{send: make(chan []byte, 256), ws: conn, userID: userID}
    hub.register <- connection
    defer func() { hub.unregister <- connection }()
    
    go connection.writePump()
    go connection.readPump()

    for {
        _, message, err := conn.ReadMessage()
        if err != nil {
            log.Println("read:", err)
            break
        }
        log.Printf("recv: %s", message)

        // Broadcast the message to all connections
        hub.broadcast <- message
    }
}

func (c *Connection) readPump() {
    for {
        _, message, err := c.ws.ReadMessage()
        if err != nil {
            log.Printf("error: %v", err)
            break
        }

        var msg models.GameAction
        if err := json.Unmarshal(message, &msg); err != nil {
            log.Printf("error unmarshalling message: %v", err)
            continue
        }

        gameAction := models.GameAction{
            GameID:    "make it increase gameid automatically",
            PlayerID:  strconv.FormatUint(c.userID, 10), 
            Action:    msg.Action,
            Timestamp: msg.Timestamp,
        }

        // Insert the game action into MongoDB
        collection := repository.MongoDBClient.Database("flaparena").Collection("game_actions")
        _, err = collection.InsertOne(context.Background(), gameAction)
        if err != nil {
            log.Printf("error inserting game action into MongoDB: %v", err)
            continue
        }

        // Broadcast the message to all other players
        hub.broadcast <- message
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

func saveGameSessionToMongoDB(gameID string) {
    gameSessionsMutex.Lock()
    session, exists := gameSessions[gameID]
    gameSessionsMutex.Unlock()

    if !exists {
        log.Printf("Game session %s not found", gameID)
        return
    }

    // Example: Insert each action as a separate document
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
