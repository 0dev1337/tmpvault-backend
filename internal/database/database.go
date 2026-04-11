package database

import (
	"context"
	"time"

	"github.com/0dev1337/tmpvault-backend/pkg/mongo"
	"go.mongodb.org/mongo-driver/bson"
)

// UploadRecord is the persisted shape for an uploaded file.
type UploadRecord struct {
	FileID      string `bson:"file_id"`
	FileName    string `bson:"file_name"`
	FilePath    string `bson:"file_path"`
	Expiry      int64  `bson:"expiry"`
	Fingerprint string `bson:"fingerprint"`
}

// SaveUpload inserts one upload record into MongoDB.
func SaveUpload(ctx context.Context, rec UploadRecord) error {
	_, err := mongo.UploadsCollection.InsertOne(ctx, rec)
	return err
}

// FindUpload finds one upload record by file_id.
func FindUpload(ctx context.Context, file_id string) (UploadRecord, error) {
	var rec UploadRecord
	err := mongo.UploadsCollection.FindOne(ctx, bson.M{"file_id": file_id}).Decode(&rec)
	return rec, err
}

// FindUploadByFingerprint finds an upload by SHA-256 hex fingerprint that is not yet expired.
func FindUploadByFingerprint(ctx context.Context, fingerprint string) (UploadRecord, error) {
	var rec UploadRecord
	now := time.Now().Unix()
	err := mongo.UploadsCollection.FindOne(ctx, bson.M{
		"fingerprint": fingerprint,
		"expiry":      bson.M{"$gte": now},
	}).Decode(&rec)
	return rec, err
}
