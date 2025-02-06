// cmd/server/main.go

package main

import (
	"helm-portal/pkg/api/handlers"
	service "helm-portal/pkg/chart/services"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"

	"github.com/sirupsen/logrus"
)

func main() {
	// ✨ Setup logger
	log := logrus.New()
	log.SetFormatter(&logrus.JSONFormatter{PrettyPrint: true})
	log.SetOutput(os.Stdout)
	log.SetLevel(logrus.InfoLevel)

	// 🚀 Create Fiber app
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

	// 📝 Log all requests middleware
	app.Use(func(c *fiber.Ctx) error {
		log.WithFields(logrus.Fields{
			"path":   c.Path(),
			"method": c.Method(),
			"ip":     c.IP(),
		}).Info("Incoming request")

		return c.Next()
	})

	// Initialize handlers with logger
	chartService := service.NewChartService("./charts", log)
	chartHandler := handlers.NewChartHandler(chartService, log)

	// Setup routes
	// add route to favicon
	app.Get("/favicon.ico", func(c *fiber.Ctx) error {
		return c.SendFile("./static/ico.webp")
	})

	app.Get("/", chartHandler.Home)
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})
	app.Post("/charts", chartHandler.UploadChart)

	app.Get("/index.yaml", chartHandler.GetIndex)
	app.Get("/charts/:name", chartHandler.GetChart)

	// 🚀 Start server
	port := ":3030"
	log.WithField("port", port).Info("Starting server")
	if err := app.Listen(port); err != nil {
		log.WithError(err).Fatal("Server failed to start")
	}
}
