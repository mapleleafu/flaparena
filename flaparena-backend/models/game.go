package models

import "time"

type Game struct {
    ID        string    `json:"id"`
    CreatedAt time.Time `json:"created_at"`
    FinishedAt time.Time `json:"finished_at"`
    UserIDs   []string  `json:"user_ids"`
}
