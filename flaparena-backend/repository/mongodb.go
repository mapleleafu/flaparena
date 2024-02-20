package repository

import (
	"context"
	"log"
	"time"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func ConnectMongoDB() {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
    if err != nil {
        log.Fatal(err)
    }

    // Optional: Ping the primary
    if err := client.Ping(ctx, nil); err != nil {
        log.Fatal(err)
    }

    log.Println("Successfully connected to MongoDB")
}

var (
    MongoDBClient *mongo.Client
)
