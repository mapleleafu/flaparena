package models

type GameActionMessage struct {
    Action    string `json:"action"`
    Timestamp int64  `json:"timestamp"`
}

type GameAction struct {
    GameID    string `bson:"gameId"`
    PlayerID  string `bson:"playerId"`
    Action    string `bson:"action"`
    Timestamp int64  `bson:"timestamp"`
}

// GameSession represents all actions taken in a single game session.
type GameSession struct {
    ID      string       `bson:"_id,omitempty"`
    Actions []GameAction `bson:"actions"`
    // StartingTimestamp int64 `bson:"startingTimestamp"`
    // EndingTimestamp   int64 `bson:"endingTimestamp"`
}