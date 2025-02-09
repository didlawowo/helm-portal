// pkg/handlers/oci_handlers.go
package handlers

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	interfaces "helm-portal/pkg/interfaces"
	// "helm-portal/pkg/models"
	storage "helm-portal/pkg/storage"

	"os"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
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

func (h *OCIHandler) HandleManifest(c *fiber.Ctx) error {
	name := c.Params("name")
	reference := c.Params("reference")

	var manifestPath string
	if strings.HasPrefix(reference, "sha256:") {
		var err error
		manifestPath = h.pathManager.FindManifestByDigest(name, reference)
		if err != nil {
			h.log.WithError(err).Error("Failed to find manifest by digest")
			return c.SendStatus(404)
		}
	} else {
		manifestPath = h.pathManager.GetManifestPath(name, reference)
		if !h.service.ChartExists(name, reference) {
			return c.SendStatus(404)
		}
	}

	// Pour un HEAD request, on s'arrête là
	if c.Method() == "HEAD" {
		return c.SendStatus(200)
	}

	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		h.log.WithError(err).Error("Failed to read manifest")
		return c.SendStatus(500)
	}

	c.Set("Content-Type", "application/vnd.oci.image.manifest.v1+json")

	// Ajout important du Docker-Content-Digest header
	digest := sha256.Sum256(manifestData)
	digestStr := fmt.Sprintf("sha256:%x", digest)
	c.Set("Docker-Content-Digest", digestStr)

	return c.Send(manifestData)
}

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

func calculateDigest(data []byte) string {
	hash := sha256.Sum256(data)
	return fmt.Sprintf("sha256:%x", hash)
}

func generateUUID() string {
	uuid := uuid.New().String()
	return uuid
}

// Fonction pour gérer la réception d'un blob
func (h *OCIHandler) PutBlob(c *fiber.Ctx) error {
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

func (h *OCIHandler) PostUpload(c *fiber.Ctx) error {
	name := c.Params("name")
	uuid := generateUUID()
	location := fmt.Sprintf("/v2/%s/blobs/uploads/%s", name, uuid)
	c.Set("Location", location)
	c.Set("Docker-Upload-UUID", uuid)
	return c.SendStatus(202)
}

func (h *OCIHandler) PatchBlob(c *fiber.Ctx) error {
	uuid := c.Params("uuid")
	tempPath := h.pathManager.GetTempPath(uuid)

	h.log.WithFields(logrus.Fields{
		"uuid": uuid,
		"size": len(c.Body()),
		"path": tempPath,
	}).Info("📝 Receiving PATCH data")

	// Créer le dossier temp s'il n'existe pas
	if err := os.MkdirAll(filepath.Dir(tempPath), 0755); err != nil {
		h.log.WithError(err).Error("Failed to create temp directory")
		return c.SendStatus(500)
	}

	// Vérifier que les données sont bien reçues
	if len(c.Body()) == 0 {
		h.log.Error("Empty body received")
		return c.Status(400).JSON(fiber.Map{"error": "Empty body"})
	}
	// Sauvegarder dans le dossier temp
	if err := os.WriteFile(tempPath, c.Body(), 0644); err != nil {
		h.log.WithError(err).Error("Failed to write temp file")
		return c.SendStatus(500)
	}
	h.log.Info("✅ PATCH data written successfully")

	c.Set("Range", fmt.Sprintf("0-%d", len(c.Body())-1))
	return c.SendStatus(202)
}

func (h *OCIHandler) CompleteUpload(c *fiber.Ctx) error {
	name := c.Params("name")
	uuid := c.Params("uuid")
	digest := c.Query("digest") // SHA256 du contenu

	// Déplacer le fichier temporaire vers son emplacement final
	tempPath := h.pathManager.GetTempPath(uuid)
	finalPath := h.pathManager.GetBlobPath(digest)
	// version := c.Query("version")
	if len(c.Body()) > 0 {

		h.log.WithFields(logrus.Fields{
			"uuid": uuid,
			"size": len(c.Body()),
		}).Info("📤 Receiving PUT data directly")

		// Écrire les données reçues
		if err := os.WriteFile(tempPath, c.Body(), 0644); err != nil {
			h.log.WithError(err).Error("Failed to write PUT data")
			return c.SendStatus(500)
		}
	}
	h.log.WithFields(logrus.Fields{
		"tempPath":  tempPath,
		"finalPath": finalPath,
	}).Info("Chemins utilisés")

	if err := os.Rename(tempPath, finalPath); err != nil {
		h.log.Error("❌ Erreur finalisation upload: ", err)
		return c.SendStatus(500)
	}
	// h.service.CreateChartTgz(&models.ChartMetadata{
	// 	Name:    name,
	// 	Version: version,
	// }, c.Body())

	c.Set("Docker-Content-Digest", digest) // Ajoutez cette ligne

	h.log.Info("✅ Upload terminé pour ", name)
	return c.SendStatus(201) // 201 = Created
}

func (h *OCIHandler) HeadBlob(c *fiber.Ctx) error {

	digest := c.Params("digest")
	name := c.Params("name")

	h.log.WithFields(logrus.Fields{
		"chart":  name,
		"digest": digest,
		"path":   c.Path(),
	}).Info("🔍 Head Blob Request")

	blobPath := h.pathManager.GetBlobPath(digest)

	if _, err := os.Stat(blobPath); err != nil {
		if os.IsNotExist(err) {
			h.log.WithError(err).Error("Blob not found")
			return c.SendStatus(404)
		}
		h.log.WithError(err).Error("Failed to check blob")
		return c.SendStatus(500)
	}

	info, err := os.Stat(blobPath)
	if err != nil {
		h.log.WithError(err).Error("Failed to get blob info")
		return c.SendStatus(500)
	}

	c.Set("Content-Length", fmt.Sprintf("%d", info.Size()))
	c.Set("Docker-Content-Digest", digest)
	c.Set("Content-Type", "application/octet-stream")

	return c.SendStatus(200)
}

func (h *OCIHandler) PutManifest(c *fiber.Ctx) error {
	name := c.Params("name")
	reference := c.Params("reference")

	// Décoder le manifest pour extraire les infos du chart
	var manifest OCIManifest
	if err := json.Unmarshal(c.Body(), &manifest); err != nil {
		h.log.WithError(err).Error("❌ Erreur parsing manifest")
		return c.SendStatus(500)
	}

	// Trouver le layer qui contient le chart
	var chartLayer *struct {
		MediaType string `json:"mediaType"`
		Digest    string `json:"digest"`
		Size      int    `json:"size"`
	}

	for _, layer := range manifest.Layers {
		if layer.MediaType == "application/vnd.cncf.helm.chart.content.v1.tar+gzip" {
			chartLayer = &layer
			break
		}
	}

	if chartLayer != nil {
		// Récupérer le contenu du chart depuis le blob
		chartData, err := h.getBlobByDigest(chartLayer.Digest)
		if err != nil {
			h.log.WithError(err).Error("❌ Erreur lecture chart data")
			return c.SendStatus(500)
		}

		// Sauvegarder le chart
		fileName := fmt.Sprintf("%s-%s.tgz", name, reference)
		if err := h.service.SaveChart(chartData, fileName); err != nil {
			h.log.WithError(err).Error("❌ Erreur sauvegarde chart")
			return c.SendStatus(500)
		}
	} else {
		h.log.Error("❌ Layer du chart non trouvé")
		return c.SendStatus(500)
	}
	manifestData := c.Body()
	manifestPath := h.pathManager.GetManifestPath(name, reference)
	h.log.WithFields(logrus.Fields{
		"path": manifestPath,
		"size": len(manifestData),
	}).Info("📝  manifest")

	manifestDir := filepath.Dir(manifestPath)
	if err := os.MkdirAll(manifestDir, 0755); err != nil {
		h.log.WithError(err).Error("❌ Erreur création dossier manifest")
		return c.SendStatus(500)
	}

	if err := os.WriteFile(manifestPath, manifestData, 0644); err != nil {
		h.log.Error("❌ Erreur sauvegarde manifest: ", err)
		return c.SendStatus(500)
	}

	digest := sha256.Sum256(manifestData)
	digestStr := fmt.Sprintf("sha256:%x", digest)
	c.Set("Docker-Content-Digest", digestStr)

	h.log.Info("✅ Manifest sauvegardé pour ", name)
	return c.SendStatus(201)
}
