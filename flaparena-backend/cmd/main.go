package main

import (
    "log"
    "net/http"

    "github.com/joho/godotenv"
    "github.com/mapleleafu/flaparena/flaparena-backend/config"
    "github.com/mapleleafu/flaparena/flaparena-backend/handlers"
    "github.com/mapleleafu/flaparena/flaparena-backend/middleware"
    "github.com/mapleleafu/flaparena/flaparena-backend/repository"
)

func main() {
    if err := godotenv.Load(); err != nil {
        log.Fatal("Error loading .env file:", err)
    }

    repository.ConnectToPostgreSQL(config.LoadConfig())
    repository.ConnectMongoDB()

    r := handlers.NewRouter()
    corsHandler := middleware.CORSMiddleware(r)

    log.Println("Server running on http://localhost:8000")
    http.ListenAndServe(":8000", corsHandler)
}
