package upload

import (
	"github.com/0dev1337/tmpvault-backend/internal/service"
	"github.com/gofiber/fiber/v3"
)

func UploadHandler(c fiber.Ctx) error {

	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Failed to get file",
		})
	}

	expiry := c.FormValue("expiry")
	if expiry == "" {
		expiry = "24h"
	}

	result, err := service.UploadFile(file, expiry)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to upload file",
		})
	}

	return c.JSON(fiber.Map{
		"url": result,
	})

}
