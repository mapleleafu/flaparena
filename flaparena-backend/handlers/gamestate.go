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
        } else {
            log.Printf("Max is 20 players, %d players are ready", readyPlayers)
        }
    }

    if readyPlayers >= 2 && !currentGameState.Started && checkAllPlayersReady() {
        log.Printf("Starting game with %d players", readyPlayers)
        GameID := startNewGameSession()
        currentGameState.GameID = GameID
        currentGameState.Started = true
        log.Println("Game started")
        gameStartedAction := models.GameAction{
            UserID:    "server",
            Action:    "start",
            Timestamp: time.Now().UnixNano() / int64(time.Millisecond),
        }
        handleGameAction(gameStartedAction, currentGameState.GameID)
        hub.broadcast <- []byte("'SERVER' - Game started!")
    } else if readyPlayers < 2 {
        log.Println("Not enough players to start game")
    } else if currentGameState.Started {
        log.Println("Game already started")
    } else {
        log.Println("Not all players are ready")
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

        hub.broadcast <- []byte("'SERVER' - Game ended!")
        realGameID, session := saveGameSessionToMongoDB(gameID)
        saveGameDataToPostgres(realGameID, session)
        currentGameState.GameID = "" // Reset placeholder ID
        currentGameState.Players = make(map[string]*models.PlayerState) // Reset players
        currentGameState.Started = false // Reset game state
    } else {
        log.Println("Not all players dead yet.")
    }
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
    
    log.Printf("Player %s scored", userID)
}
