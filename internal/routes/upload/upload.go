package upload

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/0dev1337/tmpvault-backend/internal/config"
	"github.com/0dev1337/tmpvault-backend/internal/database"
	"github.com/0dev1337/tmpvault-backend/internal/helpers"
	"github.com/0dev1337/tmpvault-backend/pkg/logging"
	"github.com/gofiber/fiber/v3"
	mongodriver "go.mongodb.org/mongo-driver/mongo"
)

// mongoOpTimeout bounds each Mongo call made during an upload so a slow or
// unresponsive database can't hang a request indefinitely.
const mongoOpTimeout = 10 * time.Second

// hasFreeDiskSpace reports whether dir's filesystem has at least needed bytes
// free, so a large upload fails fast with a clear error instead of partway
// through a long write.
func hasFreeDiskSpace(dir string, needed int64) (bool, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(dir, &stat); err != nil {
		return false, err
	}
	free := stat.Bavail * uint64(stat.Bsize)
	return free >= uint64(needed), nil
}

// UploadHandler accepts multipart file upload and stores metadata in MongoDB.
func UploadHandler(c fiber.Ctx) error {
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(UploadResponse{
			Status: "error",
			Error:  "file is required",
		})
	}

	originalName := file.Filename
	fileNameLength := 0
	fileNameLengthRaw := c.FormValue("file_name_length")
	if fileNameLengthRaw != "" {
		parsedLength, err := strconv.Atoi(fileNameLengthRaw)
		if err != nil || parsedLength < 1 {
			return c.Status(fiber.StatusBadRequest).JSON(UploadResponse{
				Status: "error",
				Error:  "file_name_length must be a positive integer",
			})
		}
		fileNameLength = parsedLength
	}

	generatedName, err := helpers.GenFileNameWithLength(file.Filename, fileNameLength)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(UploadResponse{
			Status: "error",
			Error:  err.Error(),
		})
	}
	file.Filename = generatedName

	expiry := c.FormValue("expiry")
	if expiry == "" {
		expiry = "24h"
	}
	expirySubdir, ok := config.Configuration.Expiry[expiry]
	if !ok {
		return c.Status(fiber.StatusBadRequest).JSON(UploadResponse{
			Status: "error",
			Error:  "unknown expiry option",
		})
	}

	expiryUnix, err := helpers.ExpiryToUnix(expiry)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(UploadResponse{
			Status: "error",
			Error:  "invalid expiry value",
		})
	}

	base := config.Configuration.Storage.Path
	saveDir := filepath.Join(base, expirySubdir)
	savePath := filepath.Join(saveDir, file.Filename)

	if err := os.MkdirAll(saveDir, 0o755); err != nil {
		logging.Logger.Error().Err(err).Msg("create storage directory")
		return c.Status(fiber.StatusInternalServerError).JSON(UploadResponse{
			Status: "error",
			Error:  "failed to create storage directory",
		})
	}

	if ok, err := hasFreeDiskSpace(saveDir, file.Size); err != nil {
		logging.Logger.Error().Err(err).Msg("check free disk space")
	} else if !ok {
		return c.Status(fiber.StatusInsufficientStorage).JSON(UploadResponse{
			Status: "error",
			Error:  "not enough storage space available",
		})
	}

	src, err := file.Open()
	if err != nil {
		logging.Logger.Error().Err(err).Msg("open upload")
		return c.Status(fiber.StatusBadRequest).JSON(UploadResponse{
			Status: "error",
			Error:  "failed to read file",
		})
	}
	defer src.Close()

	dst, err := os.Create(savePath)
	if err != nil {
		logging.Logger.Error().Err(err).Msg("create upload file")
		return c.Status(fiber.StatusInternalServerError).JSON(UploadResponse{
			Status: "error",
			Error:  "failed to save file",
		})
	}

	h := sha256.New()
	if _, err := io.Copy(io.MultiWriter(dst, h), src); err != nil {
		_ = dst.Close()
		_ = os.Remove(savePath)
		logging.Logger.Error().Err(err).Msg("save upload")
		return c.Status(fiber.StatusInternalServerError).JSON(UploadResponse{
			Status: "error",
			Error:  "failed to save file",
		})
	}
	if err := dst.Close(); err != nil {
		_ = os.Remove(savePath)
		logging.Logger.Error().Err(err).Msg("close upload file")
		return c.Status(fiber.StatusInternalServerError).JSON(UploadResponse{
			Status: "error",
			Error:  "failed to save file",
		})
	}

	fingerprint := hex.EncodeToString(h.Sum(nil))
	ctx, cancel := context.WithTimeout(context.Background(), mongoOpTimeout)
	defer cancel()

	existing, err := database.FindUploadByFingerprint(ctx, fingerprint)
	if err == nil {
		_ = os.Remove(savePath)
		return c.Status(fiber.StatusOK).JSON(UploadResponse{
			ID:     existing.FileID,
			Status: "success",
		})
	}
	if !errors.Is(err, mongodriver.ErrNoDocuments) {
		logging.Logger.Error().Err(err).Msg("fingerprint lookup")
		_ = os.Remove(savePath)
		return c.Status(fiber.StatusInternalServerError).JSON(UploadResponse{
			Status: "error",
			Error:  "failed to check existing upload",
		})
	}

	rec := database.UploadRecord{
		FileID:      file.Filename,
		FileName:    originalName,
		FilePath:    savePath,
		Expiry:      expiryUnix,
		Fingerprint: fingerprint,
	}
	if err := database.SaveUpload(ctx, rec); err != nil {
		logging.Logger.Error().Err(err).Msg("persist upload metadata")
		_ = os.Remove(savePath)
		return c.Status(fiber.StatusInternalServerError).JSON(UploadResponse{
			Status: "error",
			Error:  "failed to save upload metadata",
		})
	}

	return c.Status(fiber.StatusOK).JSON(UploadResponse{
		ID:     file.Filename,
		Expiry: expiry,
		Status: "success",
	})
}
