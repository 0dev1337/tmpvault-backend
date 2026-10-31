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

// incomingDir returns (creating if needed) the scratch directory used for
// in-progress uploads (both the direct and chunked paths), so finalizeUpload
// can always os.Rename out of it into place on the same filesystem.
func incomingDir() (string, error) {
	dir := filepath.Join(config.Configuration.Storage.Path, ".incoming")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

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

// finalizeUpload dedupes a fully-written temp file against existing uploads
// by fingerprint, moves it into its expiry directory (or discards it as a
// duplicate), and persists its metadata. Shared by the direct and chunked
// upload paths once each has produced a complete temp file and its hash.
func finalizeUpload(tempPath, originalName, generatedName, expirySubdir, expiry string, expiryUnix int64, fingerprint string) (UploadResponse, int) {
	ctx, cancel := context.WithTimeout(context.Background(), mongoOpTimeout)
	defer cancel()

	existing, err := database.FindUploadByFingerprint(ctx, fingerprint)
	if err == nil {
		_ = os.Remove(tempPath)
		return UploadResponse{ID: existing.FileID, Status: "success"}, fiber.StatusOK
	}
	if !errors.Is(err, mongodriver.ErrNoDocuments) {
		logging.Logger.Error().Err(err).Msg("fingerprint lookup")
		_ = os.Remove(tempPath)
		return UploadResponse{Status: "error", Error: "failed to check existing upload"}, fiber.StatusInternalServerError
	}

	finalDir := filepath.Join(config.Configuration.Storage.Path, expirySubdir)
	if err := os.MkdirAll(finalDir, 0o755); err != nil {
		logging.Logger.Error().Err(err).Msg("create storage directory")
		_ = os.Remove(tempPath)
		return UploadResponse{Status: "error", Error: "failed to create storage directory"}, fiber.StatusInternalServerError
	}

	finalPath := filepath.Join(finalDir, generatedName)
	if err := os.Rename(tempPath, finalPath); err != nil {
		logging.Logger.Error().Err(err).Msg("move upload into place")
		_ = os.Remove(tempPath)
		return UploadResponse{Status: "error", Error: "failed to save file"}, fiber.StatusInternalServerError
	}

	rec := database.UploadRecord{
		FileID:      generatedName,
		FileName:    originalName,
		FilePath:    finalPath,
		Expiry:      expiryUnix,
		Fingerprint: fingerprint,
	}
	if err := database.SaveUpload(ctx, rec); err != nil {
		logging.Logger.Error().Err(err).Msg("persist upload metadata")
		_ = os.Remove(finalPath)
		return UploadResponse{Status: "error", Error: "failed to save upload metadata"}, fiber.StatusInternalServerError
	}

	return UploadResponse{ID: generatedName, Expiry: expiry, Status: "success"}, fiber.StatusOK
}

// parseFileNameLength reads and validates the optional file_name_length form value.
func parseFileNameLength(c fiber.Ctx) (int, error) {
	raw := c.FormValue("file_name_length")
	if raw == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return 0, errors.New("file_name_length must be a positive integer")
	}
	return n, nil
}

// resolveExpiry reads the expiry form value (defaulting to 24h) and resolves
// it to its storage subdirectory and absolute expiry timestamp.
func resolveExpiry(c fiber.Ctx) (expiry, subdir string, expiryUnix int64, err error) {
	expiry = c.FormValue("expiry")
	if expiry == "" {
		expiry = "24h"
	}
	subdir, ok := config.Configuration.Expiry[expiry]
	if !ok {
		return "", "", 0, errors.New("unknown expiry option")
	}
	expiryUnix, err = helpers.ExpiryToUnix(expiry)
	if err != nil {
		return "", "", 0, errors.New("invalid expiry value")
	}
	return expiry, subdir, expiryUnix, nil
}

// UploadHandler accepts a single-request multipart file upload and stores
// metadata in MongoDB. Suitable for files small enough to fit in one HTTP
// request (notably under any reverse proxy's max body size, e.g. Cloudflare's
// plan-based cap); larger files should use the chunked init/chunk/complete
// flow instead.
func UploadHandler(c fiber.Ctx) error {
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(UploadResponse{
			Status: "error",
			Error:  "file is required",
		})
	}

	originalName := file.Filename
	fileNameLength, err := parseFileNameLength(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(UploadResponse{Status: "error", Error: err.Error()})
	}

	generatedName, err := helpers.GenFileNameWithLength(file.Filename, fileNameLength)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(UploadResponse{
			Status: "error",
			Error:  err.Error(),
		})
	}

	expiry, expirySubdir, expiryUnix, err := resolveExpiry(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(UploadResponse{Status: "error", Error: err.Error()})
	}

	dir, err := incomingDir()
	if err != nil {
		logging.Logger.Error().Err(err).Msg("create incoming directory")
		return c.Status(fiber.StatusInternalServerError).JSON(UploadResponse{
			Status: "error",
			Error:  "failed to create storage directory",
		})
	}

	if ok, err := hasFreeDiskSpace(dir, file.Size); err != nil {
		logging.Logger.Error().Err(err).Msg("check free disk space")
	} else if !ok {
		return c.Status(fiber.StatusInsufficientStorage).JSON(UploadResponse{
			Status: "error",
			Error:  "not enough storage space available",
		})
	}

	tempPath := filepath.Join(dir, generatedName+".part")

	src, err := file.Open()
	if err != nil {
		logging.Logger.Error().Err(err).Msg("open upload")
		return c.Status(fiber.StatusBadRequest).JSON(UploadResponse{
			Status: "error",
			Error:  "failed to read file",
		})
	}
	defer src.Close()

	dst, err := os.Create(tempPath)
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
		_ = os.Remove(tempPath)
		logging.Logger.Error().Err(err).Msg("save upload")
		return c.Status(fiber.StatusInternalServerError).JSON(UploadResponse{
			Status: "error",
			Error:  "failed to save file",
		})
	}
	if err := dst.Close(); err != nil {
		_ = os.Remove(tempPath)
		logging.Logger.Error().Err(err).Msg("close upload file")
		return c.Status(fiber.StatusInternalServerError).JSON(UploadResponse{
			Status: "error",
			Error:  "failed to save file",
		})
	}

	fingerprint := hex.EncodeToString(h.Sum(nil))
	resp, status := finalizeUpload(tempPath, originalName, generatedName, expirySubdir, expiry, expiryUnix, fingerprint)
	return c.Status(status).JSON(resp)
}
