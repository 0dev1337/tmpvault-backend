package upload

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/0dev1337/tmpvault-backend/internal/config"
	"github.com/0dev1337/tmpvault-backend/internal/helpers"
	"github.com/0dev1337/tmpvault-backend/pkg/logging"
	"github.com/gofiber/fiber/v3"
)

// defaultChunkSizeMB is used when chunk_size_mb isn't set in config.toml.
// Kept comfortably under reverse-proxy caps like Cloudflare's 100MB-per-request
// limit (and its ~100s edge timeout) on Free/Pro/Business plans.
const defaultChunkSizeMB = 40

func chunkSize() int64 {
	mb := config.Configuration.Server.ChunkSize
	if mb <= 0 {
		mb = defaultChunkSizeMB
	}
	return int64(mb) * 1024 * 1024
}

// InitUploadHandler starts a chunked upload: it validates the declared total
// size and expiry, pre-allocates a temp file on disk, and returns a session
// id plus the chunk size the client must use for every chunk request.
func InitUploadHandler(c fiber.Ctx) error {
	size, err := strconv.ParseInt(c.FormValue("size"), 10, 64)
	if err != nil || size <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(UploadResponse{Status: "error", Error: "size must be a positive integer"})
	}

	maxSize := int64(config.Configuration.Server.MaxUploadSize) * 1024 * 1024
	if size > maxSize {
		return c.Status(fiber.StatusBadRequest).JSON(UploadResponse{Status: "error", Error: "file exceeds maximum upload size"})
	}

	filename := c.FormValue("filename")
	if filename == "" {
		return c.Status(fiber.StatusBadRequest).JSON(UploadResponse{Status: "error", Error: "filename is required"})
	}

	fileNameLength, err := parseFileNameLength(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(UploadResponse{Status: "error", Error: err.Error()})
	}

	generatedName, err := helpers.GenFileNameWithLength(filename, fileNameLength)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(UploadResponse{Status: "error", Error: err.Error()})
	}

	expiry, expirySubdir, expiryUnix, err := resolveExpiry(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(UploadResponse{Status: "error", Error: err.Error()})
	}

	dir, err := incomingDir()
	if err != nil {
		logging.Logger.Error().Err(err).Msg("create incoming directory")
		return c.Status(fiber.StatusInternalServerError).JSON(UploadResponse{Status: "error", Error: "failed to start upload"})
	}

	if ok, err := hasFreeDiskSpace(dir, size); err != nil {
		logging.Logger.Error().Err(err).Msg("check free disk space")
	} else if !ok {
		return c.Status(fiber.StatusInsufficientStorage).JSON(UploadResponse{Status: "error", Error: "not enough storage space available"})
	}

	sessionID, err := newSessionID()
	if err != nil {
		logging.Logger.Error().Err(err).Msg("generate upload session id")
		return c.Status(fiber.StatusInternalServerError).JSON(UploadResponse{Status: "error", Error: "failed to start upload"})
	}

	tempPath := filepath.Join(dir, sessionID+".part")
	f, err := os.Create(tempPath)
	if err != nil {
		logging.Logger.Error().Err(err).Msg("create temp upload file")
		return c.Status(fiber.StatusInternalServerError).JSON(UploadResponse{Status: "error", Error: "failed to start upload"})
	}
	// Pre-size the file so chunks can be written at their final offset via
	// WriteAt regardless of arrival order.
	truncErr := f.Truncate(size)
	closeErr := f.Close()
	if truncErr != nil || closeErr != nil {
		_ = os.Remove(tempPath)
		logging.Logger.Error().Err(truncErr).Err(closeErr).Msg("allocate temp upload file")
		return c.Status(fiber.StatusInternalServerError).JSON(UploadResponse{Status: "error", Error: "failed to start upload"})
	}

	cs := chunkSize()
	totalChunks := int((size + cs - 1) / cs)

	putSession(sessionID, &uploadSession{
		TempPath:      tempPath,
		TotalSize:     size,
		ChunkSize:     cs,
		TotalChunks:   totalChunks,
		received:      make([]bool, totalChunks),
		OriginalName:  filename,
		GeneratedName: generatedName,
		Expiry:        expiry,
		ExpirySubdir:  expirySubdir,
		ExpiryUnix:    expiryUnix,
		CreatedAt:     time.Now(),
	})

	return c.Status(fiber.StatusOK).JSON(InitUploadResponse{
		Status:      "success",
		SessionID:   sessionID,
		ChunkSize:   cs,
		TotalChunks: totalChunks,
	})
}

// UploadChunkHandler stores one chunk of raw bytes (not multipart) at its
// deterministic offset within the session's temp file. index is 0-based and
// every chunk except the last must be exactly the session's chunk size.
func UploadChunkHandler(c fiber.Ctx) error {
	session, ok := getSession(c.Params("session_id"))
	if !ok {
		return c.Status(fiber.StatusNotFound).JSON(UploadResponse{Status: "error", Error: "unknown or expired upload session"})
	}

	index, err := strconv.Atoi(c.Query("index"))
	if err != nil || index < 0 || index >= session.TotalChunks {
		return c.Status(fiber.StatusBadRequest).JSON(UploadResponse{Status: "error", Error: "invalid chunk index"})
	}

	expectedSize := session.ChunkSize
	if index == session.TotalChunks-1 {
		expectedSize = session.TotalSize - int64(index)*session.ChunkSize
	}

	body := c.Body()
	if int64(len(body)) != expectedSize {
		return c.Status(fiber.StatusBadRequest).JSON(UploadResponse{Status: "error", Error: "chunk size does not match expected size"})
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	f, err := os.OpenFile(session.TempPath, os.O_WRONLY, 0o644)
	if err != nil {
		logging.Logger.Error().Err(err).Msg("open temp upload file")
		return c.Status(fiber.StatusInternalServerError).JSON(UploadResponse{Status: "error", Error: "failed to store chunk"})
	}
	_, writeErr := f.WriteAt(body, int64(index)*session.ChunkSize)
	closeErr := f.Close()
	if writeErr != nil || closeErr != nil {
		logging.Logger.Error().Err(writeErr).Err(closeErr).Msg("write chunk")
		return c.Status(fiber.StatusInternalServerError).JSON(UploadResponse{Status: "error", Error: "failed to store chunk"})
	}

	session.received[index] = true

	return c.Status(fiber.StatusOK).JSON(UploadResponse{Status: "success"})
}

// CompleteUploadHandler verifies every chunk arrived, hashes the assembled
// file, and hands off to the same dedupe/save logic as the direct upload path.
func CompleteUploadHandler(c fiber.Ctx) error {
	sessionID := c.Params("session_id")
	session, ok := getSession(sessionID)
	if !ok {
		return c.Status(fiber.StatusNotFound).JSON(UploadResponse{Status: "error", Error: "unknown or expired upload session"})
	}

	session.mu.Lock()
	for i, got := range session.received {
		if !got {
			session.mu.Unlock()
			return c.Status(fiber.StatusBadRequest).JSON(UploadResponse{Status: "error", Error: fmt.Sprintf("missing chunk %d", i)})
		}
	}
	session.mu.Unlock()

	src, err := os.Open(session.TempPath)
	if err != nil {
		logging.Logger.Error().Err(err).Msg("open completed upload")
		return c.Status(fiber.StatusInternalServerError).JSON(UploadResponse{Status: "error", Error: "failed to finalize upload"})
	}
	h := sha256.New()
	_, err = io.Copy(h, src)
	_ = src.Close()
	if err != nil {
		logging.Logger.Error().Err(err).Msg("hash completed upload")
		return c.Status(fiber.StatusInternalServerError).JSON(UploadResponse{Status: "error", Error: "failed to finalize upload"})
	}
	fingerprint := hex.EncodeToString(h.Sum(nil))

	resp, status := finalizeUpload(session.TempPath, session.OriginalName, session.GeneratedName, session.ExpirySubdir, session.Expiry, session.ExpiryUnix, fingerprint)
	deleteSession(sessionID)

	return c.Status(status).JSON(resp)
}
