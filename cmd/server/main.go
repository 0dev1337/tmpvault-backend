package main

import (
	"fmt"

	"github.com/0dev1337/tmpvault-backend/internal/config"
	"github.com/0dev1337/tmpvault-backend/internal/middleware"
	"github.com/0dev1337/tmpvault-backend/pkg/logging"
	"github.com/gofiber/fiber/v3"
)

func main() {

	if err := config.LoadConfig(); err != nil {
		logging.Logger.Fatal().Str("error", err.Error()).Msg("Failed to load config")
	}
	logging.Logger.Info().Msg("Config loaded successfully")

	app := fiber.New()
	app.Use(middleware.Logging)
	app.Listen(fmt.Sprintf("%s:%d", config.Configuration.Server.Host, config.Configuration.Server.Port))

}
