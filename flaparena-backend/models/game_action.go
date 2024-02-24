package models

type GameActionMessage struct {
    Action    string `json:"action"`
    Timestamp int64  `json:"timestamp"`
}

type GameAction struct {
    UserID  string `bson:"userId"`
    Action    string `bson:"action"`
    Timestamp int64  `bson:"timestamp"`
}

type GameEvent struct {
    UserID  string `bson:"userId"`
    GameAction []GameAction `bson:"gameAction"`
}

// GameSession represents all actions taken in a single game session.
type GameSession struct {
    ID      string       `bson:"_id,omitempty"`
    Actions []GameAction `bson:"actions"`
}
