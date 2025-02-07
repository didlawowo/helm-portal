// internal/api/handlers/chart_handlers.go

package handlers

import (
	"io"

	"strings"

	service "helm-portal/pkg/services"

	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
)

// ChartHandler manages chart operations
type ChartHandler struct {
	service *service.ChartService
	log     *logrus.Logger
}

type IndexHandler struct {
	service *service.IndexService
	log     *logrus.Logger
}

// NewChartHandler creates a new handler instance
func NewChartHandler(service *service.ChartService, logger *logrus.Logger) *ChartHandler {

	return &ChartHandler{
		service: service,
		log:     logger,
	}
}

func NewIndexHandler(service *service.IndexService, logger *logrus.Logger) *IndexHandler {

	return &IndexHandler{
		service: service,
		log:     logger,
	}
}

func (h *IndexHandler) GetIndex(c *fiber.Ctx) error {
	indexPath := h.service.GetIndexPath()

	h.log.WithFields(logrus.Fields{
		"path": indexPath,
		"ip":   c.IP(),
	}).Info("Requesting index.yaml")

	// D'abord s'assurer que l'index existe
	if err := h.service.EnsureIndexExists(); err != nil {
		h.log.WithError(err).Error("Failed to ensure index exists")
		return c.Status(500).SendString("Failed to create index")
	}

	// Envoyer le fichier
	return c.SendFile(indexPath)
}

func (h *ChartHandler) GetChart(c *fiber.Ctx) error {
	chartName := c.Params("name")

	h.log.WithFields(logrus.Fields{
		"chart": chartName,
		"ip":    c.IP(),
	}).Info("Requesting chart")

	if !h.service.ChartExists(chartName) {
		h.log.WithField("chart", chartName).Warn("Chart not found")
		return c.Status(404).SendString("Chart not found")
	}

	return c.SendFile(h.service.GetChartPath(chartName))
}

// UploadChart gère POST /charts
func (h *ChartHandler) UploadChart(c *fiber.Ctx) error {
	// Récupérer le fichier uploadé
	file, err := c.FormFile("chart")
	if err != nil {
		h.log.WithError(err).Error("Failed to get uploaded file")
		return c.Status(400).JSON(fiber.Map{"error": "No chart file provided"})
	}

	// Vérifier l'extension
	if !strings.HasSuffix(file.Filename, ".tgz") {
		h.log.WithField("filename", file.Filename).Error("Invalid file type")
		return c.Status(400).JSON(fiber.Map{"error": "Chart must be a .tgz file"})
	}

	// Ouvrir le fichier
	fileContent, err := file.Open()
	if err != nil {
		h.log.WithError(err).Error("Failed to open uploaded file")
		return c.Status(500).JSON(fiber.Map{"error": "Failed to process file"})
	}
	defer fileContent.Close()

	// Lire le contenu
	chartData, err := io.ReadAll(fileContent)
	if err != nil {
		h.log.WithError(err).Error("Failed to read file content")
		return c.Status(500).JSON(fiber.Map{"error": "Failed to read file"})
	}

	// Sauvegarder via le service
	if err := h.service.SaveChart(chartData, file.Filename); err != nil {
		h.log.WithError(err).Error("Failed to save chart")
		return c.Status(500).JSON(fiber.Map{"error": "Failed to save chart"})
	}

	return c.JSON(fiber.Map{
		"message": "Chart uploaded successfully",
		"name":    file.Filename,
	})
}

func (h *ChartHandler) DownloadChart(c *fiber.Ctx) error {
	name := c.Params("name")
	chart, err := h.service.GetChart(name)
	if err != nil {
		h.log.WithError(err).Error("Failed to get chart")
		return c.Status(500).JSON(fiber.Map{"error": "Failed to get chart"})
	}
	return c.Send(chart)
}

func (h *ChartHandler) DeleteChart(c *fiber.Ctx) error {
	name := c.Params("name")
	version := c.Query("version")
	if err := h.service.DeleteChart(name, version); err != nil {
		h.log.WithError(err).Error("Failed to delete chart")
		return c.Status(500).JSON(fiber.Map{"error": "Failed to delete chart"})
	}
	return c.JSON(fiber.Map{"message": "Chart deleted successfully"})
}

func (h *ChartHandler) DisplayHome(c *fiber.Ctx) error {
	charts, err := h.service.ListCharts()
	if err != nil {
		h.log.WithError(err).Error("Failed to list charts")
		return c.Status(500).Render("error", fiber.Map{
			"Error": "Failed to load charts",
		})
	}

	return c.Render("home", fiber.Map{
		"Charts": charts,
		"Title":  "Helm Charts Repository",
	})
}
