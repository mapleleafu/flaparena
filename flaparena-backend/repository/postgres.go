package repository

import (
    "database/sql"
    "fmt"
    "log"
    _ "github.com/lib/pq"
    
    "github.com/mapleleafu/flaparena/flaparena-backend/config"
)

func ConnectToPostgreSQL(cfg *config.Config) *sql.DB {
    connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
        cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)

    db, err := sql.Open("postgres", connStr)
    if err != nil {
        log.Fatalln(err)
    }

    if err := db.Ping(); err != nil {
        db.Close()
        log.Fatal(err)
    }
    PostgreSQLDB = db

    log.Println("Successfully connected to PostgreSQL")
    return nil
}

var (
    PostgreSQLDB *sql.DB
)
