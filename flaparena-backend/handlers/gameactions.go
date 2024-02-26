package handlers

import (
	"fmt"
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
    default:
        log.Printf("Unhandled game action: %s", gameAction.Action)
    }
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
            hub.broadcast <- []byte(fmt.Sprintf("'SERVER' - Player %s readied up.", action.UserID))
            startGame()
            log.Printf("Player %s is ready", action.UserID)
        } else {
            hub.broadcast <- []byte(fmt.Sprintf("'SERVER' - You already readied up UserID: %s", action.UserID))
        }
    } else {
        hub.broadcast <- []byte(fmt.Sprintf("'SERVER' - Game already started UserID: %s", action.UserID))
    }
}

func handleFlapAction(action models.GameAction) {
    currentGameState.Mutex.Lock()
    defer currentGameState.Mutex.Unlock()

    if currentGameState.Started {
        if _, exists := currentGameState.Players[action.UserID]; exists {
            broadcastAction := fmt.Sprintf("'SERVER' - Player %s flapped.", action.UserID)
            hub.broadcast <- []byte(broadcastAction)
            handleGameAction(action, currentGameState.GameID)
            log.Printf("Player %s flapped", action.UserID)
        } else {
            message := fmt.Sprintf("'SERVER' - Player not found. Ready up UserID: %s", action.UserID)
            hub.broadcast <- []byte(message)
        }
    } else {
        message := fmt.Sprintf("'SERVER' - Game not started UserID: %s", action.UserID)
        hub.broadcast <- []byte(message)
    }
}

func handleScoreAction(action models.GameAction) {
    if currentGameState.Started {
        if _, exists := currentGameState.Players[action.UserID]; exists {
            playerScored(action.UserID)
            handleGameAction(action, currentGameState.GameID)
            log.Printf("Player %s scored", action.UserID)
        } else {
            hub.broadcast <- []byte(fmt.Sprintf("'SERVER' - Player not found. Ready up UserID: %s", action.UserID))
        }
    } else {
        hub.broadcast <- []byte(fmt.Sprintf("'SERVER' - Game not started UserID: %s", action.UserID))
    }
}

func handleDeadAction(action models.GameAction) {
    currentGameState.Mutex.Lock()
    defer currentGameState.Mutex.Unlock()

    if currentGameState.Started {
        if player, exists := currentGameState.Players[action.UserID]; exists {
            player.Alive = false
            log.Printf("Player %s is dead", action.UserID)

            broadcastAction := fmt.Sprintf("'SERVER' - Player %s is dead.", action.UserID)
            hub.broadcast <- []byte(broadcastAction)

            handleGameAction(action, currentGameState.GameID)

            // Check if all players are dead and end the game if so
            if checkAllPlayersDead() {
                endGame()
            } else {
                log.Printf("Not all players dead yet.")
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
