package handlers

import (
	"log"
	"strconv"
	"encoding/json"
	"github.com/mapleleafu/flaparena/flaparena-backend/models"
)

func processMessage(c *Connection, rawMessage []byte) {
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
            handleReadyAction(gameAction)
        case "flap":
            handleFlapAction(gameAction)
        case "score":
            handleScoreAction(gameAction)
        case "dead":
            handleDeadAction(gameAction)
        case "info":
            broadcastLobbyInfo()
        default:
            log.Printf("Unhandled game action: %s", gameAction.Action)
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

func broadcastMessage(messageType string, data interface{}) {
    message, err := json.Marshal(struct {
        Type string      `json:"type"`
        Data interface{} `json:"data"`
    }{
        Type: messageType,
        Data: data,
    })
    if err != nil {
        log.Printf("Error marshalling broadcast message: %v", err)
        return
    }
    hub.broadcast <- message
}

func handleReadyAction(action models.GameAction) {
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
            broadcastMessage("playerReady", map[string]string{"userID": action.UserID})
            startGame()
            log.Printf("Player %s is ready", action.UserID)
        } else {
            broadcastMessage("playerAlreadyReady", map[string]string{"userID": action.UserID})
        }
    } else {
        broadcastMessage("gameAlreadyStarted", map[string]string{"userID": action.UserID})
    }
}

func handleFlapAction(action models.GameAction) {
    if currentGameState.Started {
        if _, exists := currentGameState.Players[action.UserID]; exists {
            broadcastMessage("playerAction", map[string]interface{}{"action": "flap", "userID": action.UserID})
            handleGameAction(action, currentGameState.GameID)
            log.Printf("Player %s flapped", action.UserID)
        } else {
            broadcastMessage("playerNotFound", map[string]string{"userID": action.UserID})
        }
    } else {
        broadcastMessage("gameNotStarted", map[string]string{"userID": action.UserID})
    }
}

func handleScoreAction(action models.GameAction) {
    if currentGameState.Started {
        if _, exists := currentGameState.Players[action.UserID]; exists {
            playerScored(action.UserID)
            broadcastMessage("playerScored", map[string]interface{}{"userID": action.UserID})
            handleGameAction(action, currentGameState.GameID)
            log.Printf("Player %s scored", action.UserID)
        } else {
            broadcastMessage("playerNotFound", map[string]string{"userID": action.UserID})
        }
    } else {
        broadcastMessage("gameNotStarted", map[string]string{"userID": action.UserID})
    }
}

func handleDeadAction(action models.GameAction) {
    if currentGameState.Started {
        if player, exists := currentGameState.Players[action.UserID]; exists {
            player.Alive = false
            broadcastMessage("playerDeath", map[string]string{"userID": action.UserID})
            handleGameAction(action, currentGameState.GameID)
            log.Printf("Player %s is dead", action.UserID)

            if checkAllPlayersDead() {
                endGame()
            }
        } else {
            broadcastMessage("playerNotFound", map[string]string{"userID": action.UserID})
        }
    } else {
        broadcastMessage("gameNotStarted", map[string]string{"userID": action.UserID})
    }
}
