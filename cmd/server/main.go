package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/0dev1337/tmpvault-backend/internal/config"
	"github.com/0dev1337/tmpvault-backend/internal/middleware"
	"github.com/0dev1337/tmpvault-backend/internal/routes"
	"github.com/0dev1337/tmpvault-backend/internal/service"
	"github.com/0dev1337/tmpvault-backend/pkg/logging"
	"github.com/0dev1337/tmpvault-backend/pkg/mongo"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
)

func main() {
	if err := config.LoadConfig(); err != nil {
		logging.Logger.Fatal().Err(err).Msg("failed to load config")
	}
	logging.Logger.Info().Msg("config loaded")

	mongo.SetupCollections()
	defer func() { _ = mongo.Disconnect(context.Background()) }()

	ctx, stopCleanup := context.WithCancel(context.Background())
	defer stopCleanup()
	go service.RunCleanupWorker(ctx)

	app := fiber.New(fiber.Config{
		BodyLimit: config.Configuration.Server.MaxUploadSize * 1024 * 1024,
	})
	app.Use(middleware.Logging)
	app.Use(cors.New(cors.Config{
		AllowOrigins: []string{"*"},
		AllowHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
	}))
	routes.SetupRoutes(app)

	addr := fmt.Sprintf("%s:%d", config.Configuration.Server.Host, config.Configuration.Server.Port)

	go func() {
		if err := app.Listen(addr); err != nil {
			logging.Logger.Error().Err(err).Msg("http server exited")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	stopCleanup()
	if err := app.Shutdown(); err != nil {
		logging.Logger.Error().Err(err).Msg("graceful shutdown")
	}
}
