package mongo

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/0dev1337/tmpvault-backend/internal/config"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	clientInstance *mongo.Client
	clientOnce     sync.Once
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

		clientInstance = client
		log.Println("MongoDB connected")
	})

	return clientInstance
}

// GetCollection helper
func GetCollection(dbName, collectionName string) *mongo.Collection {
	client := InitMongo()
	return client.Database(dbName).Collection(collectionName)
}

// InsertOne document
func InsertOne(ctx context.Context, collection *mongo.Collection, doc interface{}) (*mongo.InsertOneResult, error) {
	return collection.InsertOne(ctx, doc)
}

// InsertMany documents
func InsertMany(ctx context.Context, collection *mongo.Collection, docs []interface{}) (*mongo.InsertManyResult, error) {
	return collection.InsertMany(ctx, docs)
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
	if clientInstance != nil {
		if err := clientInstance.Disconnect(ctx); err != nil {
			log.Println("Error disconnecting MongoDB:", err)
		}
	}
}
