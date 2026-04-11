package download

import (
	"context"

	"github.com/0dev1337/tmpvault-backend/internal/database"
	"github.com/gofiber/fiber/v3"
)

func DownloadHandler(c fiber.Ctx) error {
	file_id := c.Params("file_id")
	record, err := database.FindUpload(context.Background(), file_id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(DownloadResponse{
			Status: "not found",
			Error:  "File not found",
		})
	}
	return c.Status(fiber.StatusOK).SendFile(record.FilePath)
}

type DownloadResponse struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}
