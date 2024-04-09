package handlers

import (
	"log"
    "time"

	"github.com/mapleleafu/flaparena/flaparena-backend/models"
)

func startGame() {
    readyPlayers := 0

    for _, player := range currentGameState.Players {
        if player.Ready && readyPlayers <= 20 {
            readyPlayers++
        }
    }

    if readyPlayers >= 2 && !currentGameState.Started && checkAllPlayersReady() {
        GameID := startNewGameSession()
        currentGameState.Mutex.Lock()
        currentGameState.GameID = GameID
        currentGameState.Started = true
        currentGameState.Mutex.Unlock()

        gameStartedAction := models.GameAction{
            UserID:    "server",
            Action:    "start",
            Timestamp: time.Now().UnixNano() / int64(time.Millisecond),
        }
        handleGameAction(gameStartedAction, currentGameState.GameID)

        log.Println("Game started")
        broadcastMessage("gameStart", map[string]interface{}{
            "gameID": GameID,
        })
    } else if readyPlayers < 2 {
        log.Println("Not enough players to start the game.")
    } else if currentGameState.Started {
        log.Println("Game already started.")
    } else {
        log.Println("Not all players are ready.")
    }
}

func endGame() {
    if checkAllPlayersDead() {
        gameID := currentGameState.GameID
        log.Println("Game ended")

        gameEndedAction := models.GameAction{
            UserID:    "server",
            Action:    "end",
            Timestamp: time.Now().UnixNano() / int64(time.Millisecond),
        }
        handleGameAction(gameEndedAction, currentGameState.GameID)

        broadcastMessage("gameEnd", map[string]interface{}{
            "gameID": gameID,
        })
        realGameID, session := saveGameSessionToMongoDB(gameID)
        saveGameDataToPostgres(realGameID, session)
        resetGameState()
    } else {
        log.Println("Not all players dead yet.")
    }
}

func resetGameState() {
    currentGameState.GameID = "" // Reset placeholder ID
    currentGameState.Players = make(map[string]*models.PlayerState) // Reset players
    currentGameState.Started = false // Reset game state
}

func checkAllPlayersReady() bool {
    currentGameState.Mutex.Lock()
    defer currentGameState.Mutex.Unlock()

    for _, player := range currentGameState.Players {
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
    
    log.Printf("Player %s scored", userID)
}
