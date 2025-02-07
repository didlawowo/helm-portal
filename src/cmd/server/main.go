package main

import (
	"helm-portal/config"
	"helm-portal/pkg/handlers"
	"helm-portal/pkg/interfaces"
	service "helm-portal/pkg/services"
	"helm-portal/pkg/storage"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"
	"github.com/sirupsen/logrus"
	// ... autres imports
)

// setupServices initialise et configure tous les services
func setupServices(cfg *config.Config, log *logrus.Logger) (interfaces.ChartServiceInterface, interfaces.IndexServiceInterface) {
	// 1. Initialiser le PathManager (utilisé par les deux services)

	// 2. Créer un ChartService temporaire sans IndexService
	tmpChartService := service.NewChartService(cfg, log, nil)

	// 3. Créer l'IndexService avec le ChartService temporaire
	indexService := service.NewIndexService(cfg, log, tmpChartService)

	// 4. Créer le ChartService final avec l'IndexService
	finalChartService := service.NewChartService(cfg, log, indexService)

	return finalChartService, indexService
}

// setupHandlers initialise tous les handlers
func setupHandlers(
	chartService interfaces.ChartServiceInterface,
	indexService interfaces.IndexServiceInterface,
	pathManager *storage.PathManager,
	log *logrus.Logger,
) (*handlers.HelmHandler, *handlers.OCIHandler, *handlers.ConfigHandler, *handlers.IndexHandler) {
	helmHandler := handlers.NewHelmHandler(chartService, pathManager, log)
	ociHandler := handlers.NewOCIHandler(chartService, log)
	configHandler := handlers.NewConfigHandler(&config.Config{}, log)
	indexHandler := handlers.NewIndexHandler(chartService, pathManager, log)

	return helmHandler, ociHandler, configHandler, indexHandler
}

func main() {
	// Logger setup
	log := logrus.New()
	log.SetFormatter(&logrus.JSONFormatter{PrettyPrint: true})
	log.SetOutput(os.Stdout)
	log.SetLevel(logrus.InfoLevel)

	// Configuration
	cfg, err := config.LoadConfig("config/config.yaml")
	if err != nil {
		log.WithError(err).Fatal("Failed to load configuration")
	}

	// PathManager
	pathManager := storage.NewPathManager(cfg.Storage.Path)

	// Services
	chartService, indexService := setupServices(cfg, log)

	// Handlers
	helmHandler, ociHandler, configHandler, indexHandler := setupHandlers(
		chartService,
		indexService,
		pathManager,
		log,
	)

	// Fiber app configuration
	app := fiber.New(fiber.Config{
		AppName:       "Helm Portal",
		Prefork:       false,
		CaseSensitive: true,
		StrictRouting: true,
		ServerHeader:  "Helm Portal",
		Views:         html.New("./views", ".html"),
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			log.WithFields(logrus.Fields{
				"path":   c.Path(),
				"method": c.Method(),
				"ip":     c.IP(),
				"error":  err.Error(),
			}).Error("Error handling request")
			return c.Status(500).SendString("Internal Server Error")
		},
	})

	// Middleware pour le logging
	app.Use(func(c *fiber.Ctx) error {
		log.WithFields(logrus.Fields{
			"path":   c.Path(),
			"method": c.Method(),
			"ip":     c.IP(),
		}).Info("Incoming request")
		return c.Next()
	})

	// Routes
	app.Get("/favicon.ico", func(c *fiber.Ctx) error {
		return c.SendFile("./views/static/ico.webp")
	})
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// Routes Helm
	app.Get("/", helmHandler.DisplayHome)
	app.Delete("/chart/:name/:version", helmHandler.DeleteChart)
	app.Post("/chart", helmHandler.UploadChart)
	app.Get("/config", configHandler.GetConfig)
	app.Get("/chart/:name/:version", helmHandler.DownloadChart)
	app.Get("/index.yaml", indexHandler.GetIndex)
	app.Get("/charts", helmHandler.ListCharts)

	// Routes OCI
	app.Get("/v2/", ociHandler.HandleOCIAPI)
	app.Get("/v2/_catalog", ociHandler.HandleCatalog)
	app.Head("/v2/:name/manifests/:reference", ociHandler.HandleManifest)
	app.Put("/v2/:name/blobs/uploads/:uuid", ociHandler.PushBlob)

	// Démarrage du serveur
	port := ":3030"
	log.WithField("port", port).Info("Starting server")
	if err := app.Listen(port); err != nil {
		log.WithError(err).Fatal("Server failed to start")
	}
}
