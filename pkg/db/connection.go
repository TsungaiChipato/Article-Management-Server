package db

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Connection represents a connection to the MongoDB database.
type Connection struct {
	client   *mongo.Client
	Database *mongo.Database
	Close    func()
}

// Connect initializes the MongoDB connection and sets up the database.
func (c *Connection) Connect(mongoUri string) error {
	// Configure MongoDB client options.
	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	opts := options.Client().ApplyURI(mongoUri).SetServerAPIOptions(serverAPI)

	// Connect to the MongoDB server.
	var err error
	c.client, err = mongo.Connect(context.TODO(), opts)
	if err != nil {
		return err
	}

	// Send a ping to confirm a successful connection.
	var result bson.M
	if err := c.client.Database("admin").RunCommand(context.TODO(), bson.D{{Key: "ping", Value: 1}}).Decode(&result); err != nil {
		return err
	}
	fmt.Println("Pinged your deployment. You successfully connected to MongoDB!")

	// Define the function to close the connection and stop the server.
	c.Close = func() {
		c.client.Disconnect(context.TODO())
	}

	// Set the database to use.
	c.Database = c.client.Database("ArticleManagement")
	return nil
}
