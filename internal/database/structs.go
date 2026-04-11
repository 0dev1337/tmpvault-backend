package database

type UploadRecord struct {
	FileID   string `bson:"file_id"`
	FileName string `bson:"file_name"`
	FilePath string `bson:"file_path"`
	Expiry   int64  `bson:"expiry"`
}
