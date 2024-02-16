package main

import (
    "log"
    "net/http"
    "github.com/gorilla/mux"
    "github.com/joho/godotenv"
    "github.com/mapleleafu/flaparena/flaparena-backend/pkg/handlers"
)

func main() {
    // Load .env file
    if err := godotenv.Load(); err != nil {
        log.Println("No .env file found")
    }

    r := mux.NewRouter()
    r.HandleFunc("/api/register", handlers.Register).Methods("POST")
    r.HandleFunc("/api/login", handlers.Login).Methods("POST")

    log.Println("Server running on http://localhost:8000")
    http.ListenAndServe(":8000", r)
}
