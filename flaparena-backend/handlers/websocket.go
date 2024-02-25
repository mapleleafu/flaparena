package handlers

import (
	"log"
	"net/http"
	"strconv"
	"sync"

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
