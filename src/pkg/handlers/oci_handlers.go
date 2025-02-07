// pkg/handlers/oci_handlers.go
package handlers

import (
	"crypto/sha256"
	"fmt"

	interfaces "helm-portal/pkg/interfaces"
	storage "helm-portal/pkg/storage"

	"os"
	"path/filepath"

	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
)

type OCIHandler struct {
	log         *logrus.Logger
	service     interfaces.ChartServiceInterface
	pathManager *storage.PathManager
}

type OCIManifest struct {
	SchemaVersion int    `json:"schemaVersion"`
	MediaType     string `json:"mediaType"`
	Config        struct {
		MediaType string `json:"mediaType"`
		Digest    string `json:"digest"`
		Size      int    `json:"size"`
	} `json:"config"`
	Layers []struct {
		MediaType string `json:"mediaType"`
		Digest    string `json:"digest"`
		Size      int    `json:"size"`
	} `json:"layers"`
}

func NewOCIHandler(service interfaces.ChartServiceInterface, log *logrus.Logger) *OCIHandler {
	return &OCIHandler{
		service:     service,
		log:         log,
		pathManager: service.GetPathManager(),
	}
}

// Vérification API - Répond avec les fonctionnalités supportées
func (h *OCIHandler) HandleOCIAPI(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"apiVersion":            "2.0",
		"docker-content-digest": true,
		"oci-distribution-spec": "v1.0",
	})
}

// Liste des repositories
func (h *OCIHandler) HandleCatalog(c *fiber.Ctx) error {
	charts, err := h.service.ListCharts()
	if err != nil {
		h.log.WithError(err).Error("Failed to list charts")
		return c.Status(500).JSON(fiber.Map{"error": "Failed to list charts"})
	}

	repositories := make([]string, 0)
	for _, chart := range charts {
		repositories = append(repositories, chart.Name)
	}

	return c.JSON(fiber.Map{
		"repositories": repositories,
	})
}

// Vérification manifest
func (h *OCIHandler) HandleManifest(c *fiber.Ctx) error {
	name := c.Params("name")
	reference := c.Params("version")

	h.log.WithFields(logrus.Fields{
		"name": name,
		"ref":  reference,
	}).Info("Checking manifest")

	if !h.service.ChartExists(name, reference) {
		return c.SendStatus(404)
	}

	// Pour un HEAD request, on renvoie juste les headers
	return c.SendStatus(200)
}

// Upload manifest
func (h *OCIHandler) PushManifest(c *fiber.Ctx) error {
	var manifest OCIManifest
	if err := c.BodyParser(&manifest); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid manifest"})
	}

	// Vérifier que c'est un manifest de chart Helm
	if manifest.Config.MediaType != "application/vnd.cncf.helm.config.v1+json" {
		return c.Status(400).JSON(fiber.Map{"error": "Not a Helm chart"})
	}
	name := c.Params("name")
	reference := c.Params("reference")
	body := c.Body()
	for _, layer := range manifest.Layers {
		if layer.MediaType == "application/vnd.cncf.helm.chart.content.v1.tar+gzip" {
			// Récupérer le contenu du chart via le digest
			chartData, error := h.getBlobByDigest(layer.Digest)
			if error != nil {
				return c.Status(400).JSON(fiber.Map{"error": "Chart data not found"})
			}
			fileName := fmt.Sprintf("%s-%s.tgz", name, reference)
			return h.service.SaveChart(chartData, fileName)
		}
	}

	h.log.WithFields(logrus.Fields{
		"name": name,
		"ref":  reference,
	}).Info("Receiving manifest")
	// Stocker le digest pour la réponse
	digest := calculateDigest(body)

	c.Set("Docker-Content-Digest", digest)
	return c.SendStatus(201)
}

// Gestion des blobs
// Dans OCIHandler
func (h *OCIHandler) getBlobByDigest(digest string) ([]byte, error) {
	// Construire le chemin du blob à partir du digest
	blobPath := h.pathManager.GetBlobPath(digest)

	// Lire le contenu du blob
	chartData, err := os.ReadFile(blobPath)
	if err != nil {
		h.log.WithError(err).Error("Failed to read blob")
		return nil, fmt.Errorf("failed to read blob: %w", err)
	}

	return chartData, nil
}

// Dans PushBlob, il faut sauvegarder le blob
func (h *OCIHandler) PushBlob(c *fiber.Ctx) error {
	digest := calculateDigest(c.Body())
	blobPath := h.pathManager.GetBlobPath(digest)

	if err := os.MkdirAll(filepath.Dir(blobPath), 0755); err != nil {
		return err
	}

	if err := os.WriteFile(blobPath, c.Body(), 0644); err != nil {
		return err
	}

	c.Set("Docker-Content-Digest", digest)
	return c.SendStatus(201)
}

// Fonction utilitaire pour calculer le digest
func calculateDigest(data []byte) string {
	hash := sha256.Sum256(data)
	return fmt.Sprintf("sha256:%x", hash)
}
