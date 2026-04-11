package upload

import (
	"fmt"
	"os"

	"github.com/0dev1337/tmpvault-backend/internal/config"
	"github.com/0dev1337/tmpvault-backend/internal/database"
	"github.com/0dev1337/tmpvault-backend/internal/helpers"
	"github.com/0dev1337/tmpvault-backend/pkg/logging"
	"github.com/gofiber/fiber/v3"
)

func UploadHandler(c fiber.Ctx) error {

	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(UploadResponse{
			Status: "error",
			Error:  "File is required",
		})
	}
	originalName := file.Filename
	file.Filename = helpers.GenFileName(file.Filename)

	expiry := c.FormValue("expiry")
	if expiry == "" {
		expiry = "24h"
	}
	expiryUnix, err := helpers.ExpiryToUnix(expiry)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(UploadResponse{
			Status: "error",
			Error:  "Invalid expiry value",
		})
	}

	saveDir := fmt.Sprintf("%s/%s", config.Configuration.Storage.Path, config.Configuration.Expiry[expiry])
	savePath := fmt.Sprintf("%s/%s", saveDir, file.Filename)
	if err := os.MkdirAll(saveDir, os.ModePerm); err != nil {
		logging.Logger.Error().Str("error", err.Error()).Msg("Failed to create directory")
		return c.Status(fiber.StatusInternalServerError).JSON(UploadResponse{
			Status: "error",
			Error:  "Failed to create storage directory",
		})
	}

	if err = c.SaveFile(file, savePath); err != nil {
		logging.Logger.Error().Str("error", err.Error()).Msg("Failed to save file")
		return c.Status(fiber.StatusInternalServerError).JSON(UploadResponse{
			Status: "error",
			Error:  "Failed to save file",
		})
	}
	err = database.UploadDatabase(file.Filename, originalName, savePath, expiryUnix)
	if err != nil {
		logging.Logger.Error().Str("error", err.Error()).Msg("Failed to upload file to database")
		return c.Status(fiber.StatusInternalServerError).JSON(UploadResponse{
			Status: "error",
			Error:  "Failed to upload file to database",
		})
	}
	return c.Status(200).JSON(UploadResponse{
		URL: fmt.Sprintf("%s/%s%s", config.Configuration.Storage.Path,
			config.Configuration.Expiry[expiry], file.Filename),
		Expiry: expiry,
		Status: "success",
	})

}
