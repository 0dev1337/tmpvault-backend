package service

import (
	"context"
	"os"
	"time"

	"github.com/0dev1337/tmpvault-backend/internal/database"
	"github.com/0dev1337/tmpvault-backend/pkg/logging"
	"github.com/0dev1337/tmpvault-backend/pkg/mongo"
	"go.mongodb.org/mongo-driver/bson"
)

func CleanupService() {

	records, err := mongo.UploadsCollection.Find(context.Background(), bson.M{"expiry": bson.M{"$lt": time.Now().Unix()}})
	if err != nil {
		logging.Logger.Error().Str("error", err.Error()).Msg("Failed to find records")
		return
	}
	for records.Next(context.Background()) {
		var record database.UploadRecord
		if err := records.Decode(&record); err != nil {
			logging.Logger.Error().Str("error", err.Error()).Msg("Failed to decode record")
			continue
		}
		os.Remove(record.FilePath)
		mongo.UploadsCollection.DeleteOne(context.Background(), bson.M{"file_id": record.FileID})
		logging.Logger.Info().Str("file_id", record.FileID).Msg("Deleted file from database")
	}
	logging.Logger.Info().Msg("Cleanup service completed")
}

func CleanupServiceWorker() {
	for {
		CleanupService()
		time.Sleep(30 * time.Second)
	}
}
