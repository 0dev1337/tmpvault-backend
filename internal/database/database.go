package database

import (
	"context"

	"github.com/0dev1337/tmpvault-backend/pkg/mongo"
)

func UploadDatabase(fileID, fileName, filePath string, expiryUnix int64) error {
	_, err := mongo.UploadsCollection.InsertOne(context.Background(), UploadRecord{
		FileID:   fileID,
		FileName: fileName,
		FilePath: filePath,
		Expiry:   expiryUnix,
	})
	return err
}
