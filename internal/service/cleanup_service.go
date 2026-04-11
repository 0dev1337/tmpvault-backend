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

// RunCleanupOnce deletes expired uploads from disk and removes their MongoDB documents.
func RunCleanupOnce(ctx context.Context) {
	if mongo.UploadsCollection == nil {
		return
	}
	now := time.Now().Unix()
	cur, err := mongo.UploadsCollection.Find(ctx, bson.M{"expiry": bson.M{"$lt": now}})
	if err != nil {
		logging.Logger.Error().Err(err).Msg("cleanup: find expired uploads")
		return
	}
	defer func() { _ = cur.Close(ctx) }()

	for cur.Next(ctx) {
		var rec database.UploadRecord
		if err := cur.Decode(&rec); err != nil {
			logging.Logger.Error().Err(err).Msg("cleanup: decode record")
			continue
		}
		if err := os.Remove(rec.FilePath); err != nil && !os.IsNotExist(err) {
			logging.Logger.Error().Err(err).Str("path", rec.FilePath).Msg("cleanup: remove file")
		}
		if _, err := mongo.UploadsCollection.DeleteOne(ctx, bson.M{"file_id": rec.FileID}); err != nil {
			logging.Logger.Error().Err(err).Str("file_id", rec.FileID).Msg("cleanup: delete document")
			continue
		}
		logging.Logger.Info().Str("file_id", rec.FileID).Msg("cleanup: removed expired upload")
	}
	if err := cur.Err(); err != nil {
		logging.Logger.Error().Err(err).Msg("cleanup: cursor")
	}
}

// RunCleanupWorker periodically runs RunCleanupOnce until ctx is cancelled.
func RunCleanupWorker(ctx context.Context) {
	RunCleanupOnce(ctx)
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			RunCleanupOnce(ctx)
		}
	}
}
