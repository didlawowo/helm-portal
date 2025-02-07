// internal/api/handlers/chart_handlers.go

package handlers

import (
	"io"

	"strings"

	"helm-portal/pkg/interfaces"

	storage "helm-portal/pkg/storage"

	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
)

// ChartHandler manages chart operations
type HelmHandler struct {
	service     interfaces.ChartServiceInterface
	log         *logrus.Logger
	pathManager *storage.PathManager
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func sendError(c *fiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(ErrorResponse{Error: message})
}

// NewChartHandler creates a new handler instance
func NewHelmHandler(service interfaces.ChartServiceInterface, pathManager *storage.PathManager, logger *logrus.Logger) *HelmHandler {

	return &HelmHandler{
		service:     service,
		log:         logger,
		pathManager: pathManager,
	}
}

// pkg/middleware/validation.go
func ValidateChartUpload() fiber.Handler {
	return func(c *fiber.Ctx) error {
		file, err := c.FormFile("chart")
		if err != nil {
			return c.Status(400).JSON(ErrorResponse{Error: "No chart file provided"})
		}

		if !strings.HasSuffix(file.Filename, ".tgz") {
			return c.Status(400).JSON(ErrorResponse{Error: "Chart must be a .tgz file"})
		}

		c.Locals("chartFile", file)
		return c.Next()
	}
}

func (h *HelmHandler) GetIndex(c *fiber.Ctx) error {
	indexPath := h.pathManager.GetIndexPath()

	h.log.WithFields(logrus.Fields{
		"path": indexPath,
		"ip":   c.IP(),
	}).Info("Requesting index.yaml")

	// Envoyer le fichier
	return c.SendFile(indexPath)
}

func (h *HelmHandler) GetChart(c *fiber.Ctx) error {
	chartName := c.Params("name")
	version := c.Params("version")

	h.log.WithFields(logrus.Fields{
		"chart": chartName,
		"ip":    c.IP(),
	}).Info("Requesting chart")

	if !h.service.ChartExists(chartName, version) {
		h.log.WithField("chart", chartName).Warn("Chart not found")
		return c.Status(404).SendString("Chart not found")
	}

	return c.SendFile(h.pathManager.GetChartPath(chartName, version))
}

func (h *HelmHandler) ListCharts(c *fiber.Ctx) error {
	charts, err := h.service.ListCharts()
	if err != nil {
		h.log.WithError(err).Error("Failed to list charts")
		return c.Status(500).SendString("Failed to list charts")
	}
	return c.JSON(charts)
}

// UploadChart gère POST /charts
func (h *HelmHandler) UploadChart(c *fiber.Ctx) error {
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

func (h *HelmHandler) DownloadChart(c *fiber.Ctx) error {
	name := c.Params("name")
	version := c.Params("version")
	chart, err := h.service.GetChart(name, version)
	h.log.WithField("name", name).WithField("version", version).Info("Downloading chart")
	if err != nil {
		h.log.WithError(err).Error("Failed to get chart")
		return c.Status(500).JSON(fiber.Map{"error": "Failed to get chart"})
	}
	return c.Send(chart)
}

func (h *HelmHandler) DeleteChart(c *fiber.Ctx) error {
	name := c.Params("name")
	version := c.Query("version")
	if err := h.service.DeleteChart(name, version); err != nil {
		h.log.WithError(err).Error("Failed to delete chart")
		return c.Status(500).JSON(fiber.Map{"error": "Failed to delete chart"})
	}
	return c.JSON(fiber.Map{"message": "Chart deleted successfully"})
}

func (h *HelmHandler) DisplayHome(c *fiber.Ctx) error {
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
