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
    Actions []GameAction
}
