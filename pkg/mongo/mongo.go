package mongo

import (
	"context"
	"sync"
	"time"

	"github.com/0dev1337/tmpvault-backend/internal/config"
	"github.com/0dev1337/tmpvault-backend/pkg/logging"
	mongodriver "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	mu                sync.Mutex
	mongoClient       *mongodriver.Client
	UploadsCollection *mongodriver.Collection
)

// InitMongo returns the shared driver client, connecting on first use.
func InitMongo() *mongodriver.Client {
	mu.Lock()
	defer mu.Unlock()
	if mongoClient != nil {
		return mongoClient
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	opts := options.Client().ApplyURI(config.Configuration.Database.MongoURI)
	client, err := mongodriver.Connect(ctx, opts)
	if err != nil {
		logging.Logger.Fatal().Err(err).Msg("mongo connect failed")
	}
	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(context.Background())
		logging.Logger.Fatal().Err(err).Msg("mongo ping failed")
	}
	mongoClient = client
	logging.Logger.Info().Msg("connected to MongoDB")
	return mongoClient
}

// SetupCollections binds package-level collection handles (call after config load).
func SetupCollections() {
	c := InitMongo()
	mu.Lock()
	UploadsCollection = c.Database(config.Configuration.Database.DBName).Collection("uploads")
	mu.Unlock()
	logging.Logger.Info().Msg("mongo collections ready")
}

// Disconnect closes the shared client and clears handles.
func Disconnect(ctx context.Context) error {
	mu.Lock()
	defer mu.Unlock()
	if mongoClient == nil {
		return nil
	}
	err := mongoClient.Disconnect(ctx)
	mongoClient = nil
	UploadsCollection = nil
	if err != nil {
		logging.Logger.Error().Err(err).Msg("mongo disconnect")
		return err
	}
	logging.Logger.Info().Msg("disconnected from MongoDB")
	return nil
}
