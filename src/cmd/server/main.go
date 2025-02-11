package main

import (
	"helm-portal/config"
	"helm-portal/pkg/handlers"
	"helm-portal/pkg/interfaces"
	middleware "helm-portal/pkg/middlewares"
	service "helm-portal/pkg/services"
	"helm-portal/pkg/storage"

	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"
	"github.com/sirupsen/logrus"
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
	_ interfaces.IndexServiceInterface,
	pathManager *storage.PathManager,
	log *logrus.Logger,
) (*handlers.HelmHandler, *handlers.OCIHandler, *handlers.ConfigHandler, *handlers.IndexHandler) {
	helmHandler := handlers.NewHelmHandler(chartService, pathManager, log)
	ociHandler := handlers.NewOCIHandler(chartService, log)
	configHandler := handlers.NewConfigHandler(&config.Config{}, log)
	indexHandler := handlers.NewIndexHandler(chartService, pathManager, log)

	return helmHandler, ociHandler, configHandler, indexHandler
}

// Dans main.go
func setupHTTPServer(app *fiber.App, log *logrus.Logger) {
	// go func() {
	// 	// Serveur HTTP qui redirige vers HTTPS
	// 	httpApp := fiber.New(fiber.Config{
	// 		DisableStartupMessage: true,
	// 	})

	// 	// Middleware de redirection
	// 	httpApp.Use(func(c *fiber.Ctx) error {
	// 		httpsURL := "https://" + c.Hostname() + c.OriginalURL()
	// 		log.WithFields(logrus.Fields{
	// 			"from": c.OriginalURL(),
	// 			"to":   httpsURL,
	// 		}).Info("🔄 Redirecting HTTP to HTTPS")
	// 		return c.Redirect(httpsURL, 301)
	// 	})

	// 	// Écouter sur le port HTTP
	// 	if err := httpApp.Listen(":3030"); err != nil {
	// 		log.WithError(err).Error("HTTP Server failed")
	// 	}
	// }()

	// Serveur HTTPS principal
	// log.Info("🔒 Starting HTTPS server on :3031")
	// if err := app.ListenTLS(":3031", "certs/ca.crt", "certs/ca.key"); err != nil {
	// 	log.WithError(err).Fatal("HTTPS Server failed")
	// }
	log.Info("🔒 Starting HTTP server on :3030")

	if err := app.Listen(":3030"); err != nil {
		log.WithError(err).Fatal("HTTP Server failed")
	}
}

func main() {
	// Logger setup
	log := logrus.New()
	log.SetFormatter(&logrus.JSONFormatter{PrettyPrint: true})
	log.SetOutput(os.Stdout)
	// log.SetLevel(logrus.InfoLevel)
	log.SetLevel(logrus.DebugLevel) // <-- Modifier cette ligne

	// Configuration
	cfg, err := config.LoadConfig("config/config.yaml")
	if err != nil {
		log.WithError(err).Fatal("Failed to load configuration")
	}

	// PathManager
	pathManager := storage.NewPathManager(cfg.Storage.Path, log)

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

	// Routes Helm
	app.Get("/", helmHandler.DisplayHome)
	app.Get("/chart/:name/:version/details", helmHandler.DisplayChartDetails)
	app.Delete("/chart/:name/:version", helmHandler.DeleteChart)
	app.Post("/chart", helmHandler.UploadChart)
	app.Get("/config", configHandler.GetConfig)
	app.Get("/chart/:name/:version", helmHandler.DownloadChart)
	app.Get("/index.yaml", indexHandler.GetIndex)
	app.Get("/charts", helmHandler.ListCharts)
	app.Get("/chart/:name/versions", helmHandler.GetChartVersions)

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
