package handlers

import (
    "github.com/gorilla/websocket"
)

// Connection represents a WebSocket connection and the user it belongs to.
type Connection struct {
    ws       *websocket.Conn
    send     chan []byte
    userID   uint64
    username string
}

// Hub maintains the set of active connections and broadcasts messages to the connections.
type Hub struct {
    // Registered connections.
    connections map[*Connection]bool

    // Inbound messages from the connections.
    broadcast chan []byte

    register chan *Connection

    unregister chan *Connection

    // mutex sync.Mutex // Ensure thread safety
}

var hub = &Hub{
    broadcast:   make(chan []byte),
    register:    make(chan *Connection),
    unregister:  make(chan *Connection),
    connections: make(map[*Connection]bool),
}

func (h *Hub) run() {
    for {
        select {
        case connection := <-h.register:
            h.connections[connection] = true
        case connection := <-h.unregister:
            if _, ok := h.connections[connection]; ok {
                delete(h.connections, connection)
                close(connection.send)
            }
        case message := <-h.broadcast:
            for connection := range h.connections {
                select {
                case connection.send <- message:
                default:
                    close(connection.send)
                    delete(h.connections, connection)
                }
            }
        }
    }
}

func init() {
    go hub.run()
}
