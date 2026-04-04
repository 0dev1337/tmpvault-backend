package service

import "mime/multipart"

func UploadFile(file *multipart.FileHeader, expiry string) (string, error) {
	return "https://example.com/file", nil
}
