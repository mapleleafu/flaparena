package handlers

import (
    "github.com/gorilla/mux"
    "github.com/mapleleafu/flaparena/flaparena-backend/middleware"
)

func NewRouter() *mux.Router {
    r := mux.NewRouter()

    // Public routes
    r.HandleFunc("/api/register", Register).Methods("POST")
    r.HandleFunc("/api/login", Login).Methods("POST")
    r.HandleFunc("/api/refresh/token", RefreshToken).Methods("POST")
    r.HandleFunc("/ws/{token}", WsHandler)

    // Secured routes
    secured := r.PathPrefix("/api").Subrouter()
    secured.Use(middleware.JWTValidationMiddleware)
    secured.HandleFunc("/games", FetchUserGames).Methods("GET")
    secured.HandleFunc("/game/{gameID}", FetchGameActions).Methods("GET")
	secured.HandleFunc("/logout", Logout).Methods("POST")
    return r
}
