package repository

import (
    "database/sql"
    "fmt"
    "log"
    "github.com/mapleleafu/flaparena/flaparena-backend/pkg/config"

    _ "github.com/lib/pq"
)

func ConnectToDB(cfg *config.Config) *sql.DB {
    connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
        cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)

    db, err := sql.Open("postgres", connStr)
    if err != nil {
        log.Fatalln(err)
    }

    if err := db.Ping(); err != nil {
        db.Close()
        log.Fatal(err)
    } else {
        log.Println("Successfully Connected to the Database")
    }

    return db
}
