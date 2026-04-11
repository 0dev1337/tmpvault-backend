package main

import (
	"fmt"

	"github.com/0dev1337/tmpvault-backend/internal/config"
	"github.com/0dev1337/tmpvault-backend/internal/middleware"
	"github.com/0dev1337/tmpvault-backend/internal/routes"
	"github.com/0dev1337/tmpvault-backend/internal/service"
	"github.com/0dev1337/tmpvault-backend/pkg/logging"
	"github.com/0dev1337/tmpvault-backend/pkg/mongo"
	"github.com/gofiber/fiber/v3"
)

func main() {

	if err := config.LoadConfig(); err != nil {
		logging.Logger.Fatal().Str("error", err.Error()).Msg("Failed to load config")
	}
	logging.Logger.Info().Msg("Config loaded successfully")
	// fmt.Println(helpers.GetStoragePath("1h"))
	mongo.SetupCollections()
	go service.CleanupServiceWorker()
	app := fiber.New(fiber.Config{
		BodyLimit: config.Configuration.Server.MaxUploadSize * 1024 * 1024,
	})
	app.Use(middleware.Logging)
	routes.SetupRoutes(app)
	app.Listen(fmt.Sprintf("%s:%d", config.Configuration.Server.Host, config.Configuration.Server.Port))

}
