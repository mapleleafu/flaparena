package repository

import (
	"context"
	"log"
	"time"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func ConnectMongoDB() error {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
    if err != nil {
        return err
    }

    if err := client.Ping(ctx, nil); err != nil {
        return err
    }

    MongoDBClient = client 
    log.Println("Successfully connected to MongoDB")
    return nil
}

var (
    MongoDBClient *mongo.Client
)
