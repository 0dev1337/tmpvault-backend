package database

import (
	"context"

	"github.com/0dev1337/tmpvault-backend/pkg/mongo"
)

// UploadRecord is the persisted shape for an uploaded file.
type UploadRecord struct {
	FileID   string `bson:"file_id"`
	FileName string `bson:"file_name"`
	FilePath string `bson:"file_path"`
	Expiry   int64  `bson:"expiry"`
}

// SaveUpload inserts one upload record into MongoDB.
func SaveUpload(ctx context.Context, rec UploadRecord) error {
	_, err := mongo.UploadsCollection.InsertOne(ctx, rec)
	return err
}
