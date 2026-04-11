package upload

import (
	"context"
	"os"
	"path/filepath"

	"github.com/0dev1337/tmpvault-backend/internal/config"
	"github.com/0dev1337/tmpvault-backend/internal/database"
	"github.com/0dev1337/tmpvault-backend/internal/helpers"
	"github.com/0dev1337/tmpvault-backend/pkg/logging"
	"github.com/gofiber/fiber/v3"
)

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
	file.Filename = helpers.GenFileName(file.Filename)

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

	if err = c.SaveFile(file, savePath); err != nil {
		logging.Logger.Error().Err(err).Msg("save upload")
		return c.Status(fiber.StatusInternalServerError).JSON(UploadResponse{
			Status: "error",
			Error:  "failed to save file",
		})
	}

	ctx := context.Background()
	rec := database.UploadRecord{
		FileID:   file.Filename,
		FileName: originalName,
		FilePath: savePath,
		Expiry:   expiryUnix,
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
