package handlers

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"github.com/mapleleafu/flaparena/flaparena-backend/models"
	"github.com/mapleleafu/flaparena/flaparena-backend/repository"
)

func saveGameSessionToMongoDB(placeholderID string) (string, *models.GameSession) {
    gameSessionsMutex.Lock()
    session, exists := gameSessions[placeholderID]
    if !exists {
        log.Printf("Game session with placeholder ID %s not found", placeholderID)
        gameSessionsMutex.Unlock()
        return "", session
    }
    delete(gameSessions, placeholderID) // Remove the session from the map
    gameSessionsMutex.Unlock()
    
    collection := repository.MongoDBClient.Database("flaparena").Collection("game_sessions")
    result, err := collection.InsertOne(context.Background(), session)
    if err != nil {
        log.Printf("Failed to insert game session into MongoDB: %v", err)
        return "", session
    }

    // Correctly handle the InsertedID as primitive.ObjectID and convert it to string
    realGameID := result.InsertedID.(primitive.ObjectID).Hex()
    log.Printf("Game session saved to MongoDB with ID %s", realGameID)
    return realGameID, session
}

func saveGameDataToPostgres(gameID string, session *models.GameSession) {
    var serverStart, serverEnd int64
    
    var userIds []string
    
    for userID := range currentGameState.Players {
        userIds = append(userIds, userID)
    }

    for _, event := range session.Actions {
        switch event.Action {
        case "start":
            serverStart = event.Timestamp
        case "end":
            serverEnd = event.Timestamp
        }
    }

    serverStartTime := time.UnixMilli(serverStart).UTC().Format(time.RFC3339)
    serverEndTime := time.UnixMilli(serverEnd).UTC().Format(time.RFC3339)

    db := repository.PostgreSQLDB
    userIdsForDB := pq.Array(userIds)

    // Save the game session to the PostgreSQL database
    _, err := db.Exec("INSERT INTO games (id, created_at, finished_at, user_ids) VALUES ($1, $2, $3, $4)", 
    gameID, serverStartTime, serverEndTime, userIdsForDB)
    if err != nil {
        log.Printf("Failed to insert game session into PostgreSQL: %v", err)
        return
    }
    
    log.Printf("Game session saved to PostgreSQL with ID %s", gameID)
}

func startNewGameSession() string {
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
