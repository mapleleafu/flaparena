package handlers

import (
	"net/http"

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
        utils.HandleError(w, responses.InternalServerError{Msg: "Failed to fetch user games."})
        return
    }
    defer rows.Close()

    if !rows.Next() {
        utils.HandleError(w, responses.NotFoundError{Msg: "No games found for this user."})
        return
    }

    for rows.Next() {
        var game models.Game
        err := rows.Scan(&game.ID, &game.CreatedAt, &game.FinishedAt, pq.Array(&game.UserIDs))
        if err != nil {
            continue
        }
        games = append(games, game)
    }

    if err = rows.Err(); err != nil {
        utils.HandleError(w, responses.InternalServerError{Msg: "Error processing user games."})
        return
    }

    utils.HandleSuccess(w, models.SuccessResponse(games))
}
