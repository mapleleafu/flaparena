package handlers

import (
    "log"
    "net/http"
    "strconv"

    "github.com/gorilla/mux"
    "github.com/gorilla/websocket"
    "github.com/mapleleafu/flaparena/flaparena-backend/pkg/responses"
    "github.com/mapleleafu/flaparena/flaparena-backend/pkg/utils"
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

    connection := &Connection{send: make(chan []byte, 256), ws: conn}
    hub.register <- connection
    defer func() { hub.unregister <- connection }()
    
    go connection.writePump()

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

func (c *Connection) writePump() {
    defer func() {
        c.ws.Close()
    }()
    for {
        message, ok := <-c.send
        if !ok {
            // The hub closed the channel.
            c.ws.WriteMessage(websocket.CloseMessage, []byte{})
            return
        }

        c.ws.WriteMessage(websocket.TextMessage, message)
    }
}
