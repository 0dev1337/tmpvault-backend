package upload

import (
	"fmt"
	"os"

	"github.com/0dev1337/tmpvault-backend/internal/config"
	"github.com/0dev1337/tmpvault-backend/internal/helpers"
	"github.com/0dev1337/tmpvault-backend/pkg/logging"
	"github.com/gofiber/fiber/v3"
)

func UploadHandler(c fiber.Ctx) error {

	file, err := c.FormFile("file")
	file.Filename = helpers.GenFileName(file.Filename)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(UploadResponse{
			Status: "error",
			Error:  "File is required",
		})
	}

	expiry := c.FormValue("expiry")
	if expiry == "" {
		expiry = "24h"
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

	return c.Status(200).JSON(UploadResponse{
		URL: fmt.Sprintf("%s/%s%s", config.Configuration.Storage.Path,
			config.Configuration.Expiry[expiry], file.Filename),
		Expiry: expiry,
		Status: "success",
	})

}
