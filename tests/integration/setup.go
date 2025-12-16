package integration

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	mongoClient *mongo.Client
	mongoURI    = "mongodb://localhost:27017"
	baseURL     = "http://localhost:8080"
)

func TestMain(m *testing.M) {
	// Wait for MongoDB to be ready
	if err := waitForMongoDB(10); err != nil {
		log.Printf("MongoDB not available: %v. Skipping integration tests.", err)
		os.Exit(0)
	}

	// Connect to MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var err error
	mongoClient, err = mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Printf("Failed to connect to MongoDB: %v. Skipping integration tests.", err)
		os.Exit(0)
	}

	// Verify connection
	if err := mongoClient.Ping(ctx, nil); err != nil {
		log.Printf("MongoDB ping failed: %v. Skipping integration tests.", err)
		os.Exit(0)
	}

	// Run tests
	code := m.Run()

	// Cleanup
	if err := mongoClient.Disconnect(context.Background()); err != nil {
		log.Printf("Error disconnecting from MongoDB: %v", err)
	}

	os.Exit(code)
}

// waitForMongoDB waits for MongoDB to be available
func waitForMongoDB(maxRetries int) error {
	for i := 0; i < maxRetries; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
		cancel()

		if err == nil {
			if err := client.Ping(context.Background(), nil); err == nil {
				client.Disconnect(context.Background())
				return nil
			}
			client.Disconnect(context.Background())
		}

		time.Sleep(1 * time.Second)
	}

	return errors.New("MongoDB did not become available in time")
}

// WaitForService waits for the service to be available
func WaitForService(t *testing.T, maxRetries int) {
	for i := 0; i < maxRetries; i++ {
		resp, err := http.Get(baseURL + "/api/accounts")
		if err == nil {
			resp.Body.Close()
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatal("Service did not become available in time")
}

// GetBaseURL returns the base URL of the test service
func GetBaseURL() string {
	return baseURL
}

// CleanupDatabase clears all data from the test database
func CleanupDatabase(t *testing.T) {
	if mongoClient == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Drop the accounts collection to clear all data
	collection := mongoClient.Database("bulwarkauth").Collection("accounts")
	if err := collection.Drop(ctx); err != nil {
		// It's okay if the collection doesn't exist
		t.Logf("Note: Could not drop collection: %v", err)
	}
}
