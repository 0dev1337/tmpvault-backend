package routes

import (
	"github.com/0dev1337/tmpvault-backend/internal/routes/upload"
	"github.com/gofiber/fiber/v3"
)

func SetupRoutes(app *fiber.App) {
	api := app.Group("/api/v1")
	api.Post("/upload", upload.UploadHandler)
}
