package upload

import (
	"fmt"

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

	if err = c.SaveFile(file, fmt.Sprintf("%s/%s/%s", config.Configuration.Storage.Path,
		config.Configuration.Expiry.Paths[expiry], file.Filename)); err != nil {
		logging.Logger.Error().Str("error", err.Error()).Msg("Failed to save file")
		return c.Status(fiber.StatusInternalServerError).JSON(UploadResponse{
			Status: "error",
			Error:  "Failed to save file",
		})
	}

	return c.Status(200).JSON(UploadResponse{
		URL: fmt.Sprintf("%s/%s%s", config.Configuration.Storage.Path,
			config.Configuration.Expiry.Paths[expiry], file.Filename),
		Expiry: expiry,
		Status: "success",
	})

}
