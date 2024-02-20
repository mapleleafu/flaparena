package main

import (
    "log"
    "net/http"

    "github.com/mapleleafu/flaparena/flaparena-backend/config"
    "github.com/mapleleafu/flaparena/flaparena-backend/repository"
    "github.com/gorilla/mux"
    "github.com/joho/godotenv"
    "github.com/mapleleafu/flaparena/flaparena-backend/handlers"
)

func main() {
    if err := godotenv.Load(); err != nil {
        log.Fatal("Error loading .env file:", err)
    }

    repository.ConnectToPostgreSQL(config.LoadConfig())
    repository.ConnectMongoDB()

    r := mux.NewRouter()
    r.HandleFunc("/api/register", handlers.Register).Methods("POST")
    r.HandleFunc("/api/login", handlers.Login).Methods("POST")
    r.HandleFunc("/api/logout", handlers.Logout).Methods("POST")
    r.HandleFunc("/api/refresh/token", handlers.RefreshToken).Methods("POST")
    r.HandleFunc("/ws/{token}", handlers.WsHandler)

    log.Println("Server running on http://localhost:8000")
    http.ListenAndServe(":8000", r)
}
