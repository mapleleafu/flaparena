package handlers

import (
	"net/http"
    "context"
    "log"
    "github.com/gorilla/mux"
    "go.mongodb.org/mongo-driver/bson/primitive"

    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/bson"
	"github.com/lib/pq"
	"github.com/mapleleafu/flaparena/flaparena-backend/models"
	"github.com/mapleleafu/flaparena/flaparena-backend/repository"
	"github.com/mapleleafu/flaparena/flaparena-backend/responses"
	"github.com/mapleleafu/flaparena/flaparena-backend/utils"
)

func FetchUserGames(w http.ResponseWriter, r *http.Request) {
    authInfo, ok := r.Context().Value("authInfo").(*models.CustomClaims)
    if !ok {
        utils.HandleError(w, responses.InternalServerError{Msg: "Error processing request."})
        return
    }
    
    userID := authInfo.ID
    db := repository.PostgreSQLDB

    var games []models.Game
    query := "SELECT id, created_at, finished_at, user_ids FROM games WHERE $1 = ANY(user_ids)"
    rows, err := db.Query(query, userID)

    if err != nil {
        log.Printf("Error fetching games: %v", err)
        utils.HandleError(w, responses.InternalServerError{Msg: "Failed to fetch user games."})
        return
    }
    defer rows.Close()

    for rows.Next() {
        var game models.Game
        err := rows.Scan(&game.ID, &game.CreatedAt, &game.FinishedAt, pq.Array(&game.UserIDs))
        if err != nil {
            utils.HandleError(w, responses.InternalServerError{Msg: "Error processing user games."})
        }
        games = append(games, game)
    }

    if err = rows.Err(); err != nil {
        log.Printf("Error iterating games rows: %v", err)
        utils.HandleError(w, responses.InternalServerError{Msg: "Error processing user games."})
        return
    }

    if len(games) == 0 {
        log.Printf("No games found for user %s", userID)
        utils.HandleSuccess(w, models.SuccessResponse(models.Game{})) // Return an empty array for consistency
        return
    }

    utils.HandleSuccess(w, models.SuccessResponse(games))
}

func FetchGameActions(w http.ResponseWriter, r *http.Request) {
    authInfo, ok := r.Context().Value("authInfo").(*models.CustomClaims)
    if !ok {
        utils.HandleError(w, responses.InternalServerError{Msg: "Error processing request."})
        return
    }
    
    userID := authInfo.ID

    // Get gameID from the path parameters
    vars := mux.Vars(r)
    gameIDStr := vars["gameID"]
    if gameIDStr == "" {
        utils.HandleError(w, responses.BadRequestError{Msg: "gameID is required."})
        return
    }

    gameID, err := primitive.ObjectIDFromHex(gameIDStr)
    if err != nil {
        log.Printf("Error converting gameID to ObjectID: %v", err)
        utils.HandleError(w, responses.BadRequestError{Msg: "Invalid gameID format."})
        return
    }

    // Fetch game session from MongoDB
    collection := repository.MongoDBClient.Database("flaparena").Collection("game_sessions")
    var gameSession models.GameSession
    err = collection.FindOne(context.Background(), bson.M{"_id": gameID}).Decode(&gameSession)
    if err != nil {
        if err == mongo.ErrNoDocuments {
            utils.HandleError(w, responses.NotFoundError{Msg: "Game session not found."})
            return
        }
        log.Printf("Error fetching game session: %v", err)
        utils.HandleError(w, responses.InternalServerError{Msg: "Error fetching game session."})
        return
    }

    // Check if the user is part of the game actions
    userInGame := false
    for _, action := range gameSession.Actions {
        if action.UserID == userID || action.UserID == "server" {
            userInGame = true
            break
        }
    }

    if !userInGame {
        utils.HandleError(w, responses.BadRequestError{Msg: "User is not part of the game."})
        return
    }

    utils.HandleSuccess(w, models.SuccessResponse(gameSession))
}