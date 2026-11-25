package database

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// DB is the global MongoDB database handle.
var DB *mongo.Database

// Connect opens a connection to MongoDB and verifies it with a ping.
func Connect(uri string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	c, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return err
	}
	if err := c.Ping(ctx, nil); err != nil {
		return err
	}

	DB = c.Database("whatsappgo")
	log.Println("✅ MongoDB connected")
	return nil
}

// Disconnect cleanly closes the MongoDB connection.
func Disconnect() {
	if DB == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = DB.Client().Disconnect(ctx)
}
