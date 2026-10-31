package routes

import (
	"github.com/0dev1337/tmpvault-backend/internal/routes/download"
	"github.com/0dev1337/tmpvault-backend/internal/routes/upload"
	"github.com/gofiber/fiber/v3"
)

func SetupRoutes(app *fiber.App) {
	api := app.Group("/api/v1")
	api.Post("/upload", upload.UploadHandler)
	api.Post("/upload/init", upload.InitUploadHandler)
	api.Post("/upload/:session_id/chunk", upload.UploadChunkHandler)
	api.Post("/upload/:session_id/complete", upload.CompleteUploadHandler)
	app.Get("/download/:file_id", download.DownloadHandler)
}
