package main

import (
	"fmt"
	"helm-portal/config"
	"helm-portal/pkg/handlers"
	"helm-portal/pkg/interfaces"
	middleware "helm-portal/pkg/middlewares"
	service "helm-portal/pkg/services"
	"helm-portal/pkg/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"
	"github.com/sirupsen/logrus"
)

// setupServices initialise et configure tous les services
func setupServices(cfg *config.Config, log *utils.Logger) (interfaces.ChartServiceInterface, interfaces.IndexServiceInterface, *service.BackupService) {

	tmpChartService := service.NewChartService(cfg, log, nil)
	indexService := service.NewIndexService(cfg, log, tmpChartService)
	finalChartService := service.NewChartService(cfg, log, indexService)
	backupService, err := service.NewBackupService(cfg, log)
	if err != nil {
		log.WithFunc().WithError(err).Fatal("Failed to initialize backup service")
		// Gérer l'erreur selon votre logique d'application
	}
	return finalChartService, indexService, backupService
}

// setupHandlers initialise tous les handlers
func setupHandlers(
	chartService interfaces.ChartServiceInterface,
	_ interfaces.IndexServiceInterface,
	pathManager *utils.PathManager,
	cfg *config.Config,
	backupService *service.BackupService,
	log *utils.Logger,

) (*handlers.HelmHandler, *handlers.OCIHandler, *handlers.ConfigHandler, *handlers.IndexHandler, *handlers.BackupHandler) {
	helmHandler := handlers.NewHelmHandler(chartService, pathManager, log)
	ociHandler := handlers.NewOCIHandler(chartService, log)
	configHandler := handlers.NewConfigHandler(cfg, log)
	indexHandler := handlers.NewIndexHandler(chartService, pathManager, log)
	backupHandler := handlers.NewBackupHandler(backupService, log, cfg)

	return helmHandler, ociHandler, configHandler, indexHandler, backupHandler
}

func setupHTTPServer(app *fiber.App, log *utils.Logger) {

	log.WithFunc().Info("🚀 Application starting")

	if err := app.Listen(":3030"); err != nil {
		log.WithFunc().Fatal("HTTP Server failed")
	}
}

func main() {

	fmt.Println("Application starting...")

	// Logger setup
	logConfig := utils.Config{
		LogLevel:  "debug", // ou depuis votre config
		LogFormat: "json",  // ou "text"
		Pretty:    true,
	}
	log := utils.NewLogger(logConfig)

	// Configuration
	cfg, err := config.LoadConfig("config/config.yaml")
	if err != nil {
		log.WithError(err).Fatal("Failed to load configuration")
	}
	if err := config.LoadAuthFromFile(cfg); err != nil {
		log.WithError(err).Fatal("Failed to load auth configuration")
	}

	// PathManager
	pathManager := utils.NewPathManager(cfg.Storage.Path, log)

	// Services
	chartService, indexService, backupService := setupServices(cfg, log)

	// Handlers
	helmHandler, ociHandler, configHandler, indexHandler, backupHandler := setupHandlers(
		chartService,
		indexService,
		pathManager,
		cfg,
		backupService,
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
			"route":  c.Route().Path,
			"params": c.AllParams(),

			// "headers": c.GetReqHeaders(),
		}).Info("Incoming request")

		return c.Next()
	})
	// app.Use(middleware.HTTPSRedirect(log))

	app.Static("/static", "./views/static")

	// Routes
	app.Get("/favicon.ico", func(c *fiber.Ctx) error {
		return c.SendFile("./views/static/ico.png")
	})
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})
	// Créer le middleware d'authentification
	authMiddleware := middleware.NewAuthMiddleware(cfg, log)

	// Appliquer le middleware aux routes OCI qui nécessitent une authentification
	ociGroup := app.Group("/v2")
	ociGroup.Use(authMiddleware.Authenticate())
	// log.WithField("config", *cfg).Info("Configuration loaded")
	log.WithField("backup", cfg.Backup).Info("Backup configuration")
	// Routes Portal Interface
	app.Get("/", helmHandler.DisplayHome)
	app.Get("/backup/status", backupHandler.GetBackupStatus)

	app.Get("/chart/:name/:version/details", helmHandler.DisplayChartDetails)
	app.Delete("/chart/:name/:version", helmHandler.DeleteChart)
	app.Post("/chart", helmHandler.UploadChart)
	app.Get("/config", configHandler.GetConfig)
	app.Get("/chart/:name/:version", helmHandler.DownloadChart)
	app.Get("/index.yaml", indexHandler.GetIndex)
	app.Get("/charts", helmHandler.ListCharts)
	app.Get("/chart/:name/versions", helmHandler.GetChartVersions)

	// Routes Backup
	app.Post("/backup", backupHandler.HandleBackup)
	app.Post("/restore", backupHandler.HandleRestore)

	// Routes OCI
	ociGroup.Get("/", ociHandler.HandleOCIAPI)
	ociGroup.Get("/_catalog", ociHandler.HandleCatalog)
	ociGroup.Head("/:name/manifests/:reference", ociHandler.HandleManifest)
	ociGroup.Get("/:name/manifests/:reference", ociHandler.HandleManifest)
	ociGroup.Put("/:name/manifests/:reference", ociHandler.PutManifest)
	ociGroup.Put("/:name/blobs/:digest", ociHandler.PutBlob)
	ociGroup.Post("/:name/blobs/uploads/", ociHandler.PostUpload)
	ociGroup.Patch("/:name/blobs/uploads/:uuid", ociHandler.PatchBlob)
	ociGroup.Put("/:name/blobs/uploads/:uuid", ociHandler.CompleteUpload)
	ociGroup.Head("/:name/blobs/:digest", ociHandler.HeadBlob)
	ociGroup.Get("/:name/blobs/:digest", ociHandler.GetBlob)

	// Démarrage du serveur
	port := ":3030"
	log.WithField("port", port).Info("Starting server")

	setupHTTPServer(app, log)
}
