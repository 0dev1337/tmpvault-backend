package mongo

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/0dev1337/tmpvault-backend/internal/config"
	"github.com/0dev1337/tmpvault-backend/pkg/logging"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	MongoClient       *mongo.Client
	clientOnce        sync.Once
	UploadsCollection *mongo.Collection
)

// InitMongo initializes a global MongoDB client (singleton)
func InitMongo() *mongo.Client {
	clientOnce.Do(func() {

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		clientOptions := options.Client().ApplyURI(config.Configuration.Database.MongoURI)

		client, err := mongo.Connect(ctx, clientOptions)
		if err != nil {
			log.Fatal("Mongo connect error:", err)
		}

		if err := client.Ping(ctx, nil); err != nil {
			log.Fatal("Mongo ping error:", err)
		}

		MongoClient = client
		log.Println("MongoDB connected")
	})

	return MongoClient
}

func SetupCollections() {
	UploadsCollection = GetCollection("uploads")
	logging.Logger.Info().Msg("Uploads collection setup complete")
}

// GetCollection helper
func GetCollection(collectionName string) *mongo.Collection {
	client := InitMongo()
	return client.Database(config.Configuration.Database.DB).Collection(collectionName)
}

// FindOne document
func FindOne(ctx context.Context, collection *mongo.Collection, filter interface{}, result interface{}) error {
	return collection.FindOne(ctx, filter).Decode(result)
}

// Find many documents
func Find(ctx context.Context, collection *mongo.Collection, filter interface{}) ([]bson.M, error) {
	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []bson.M
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			return nil, err
		}
		results = append(results, doc)
	}

	return results, nil
}

// UpdateOne document
func UpdateOne(ctx context.Context, collection *mongo.Collection, filter interface{}, update interface{}) (*mongo.UpdateResult, error) {
	return collection.UpdateOne(ctx, filter, update)
}

// DeleteOne document
func DeleteOne(ctx context.Context, collection *mongo.Collection, filter interface{}) (*mongo.DeleteResult, error) {
	return collection.DeleteOne(ctx, filter)
}

// Disconnect MongoDB
func Disconnect(ctx context.Context) {
	if MongoClient != nil {
		if err := MongoClient.Disconnect(ctx); err != nil {
			log.Println("Error disconnecting MongoDB:", err)
		}
	}
}
