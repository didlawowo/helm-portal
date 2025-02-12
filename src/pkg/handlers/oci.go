package handlers

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	interfaces "helm-portal/pkg/interfaces"
	"helm-portal/pkg/models"
	utils "helm-portal/pkg/utils"
	"os"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type OCIHandler struct {
	log         *utils.Logger
	service     interfaces.ChartServiceInterface
	pathManager *utils.PathManager
}

func NewOCIHandler(service interfaces.ChartServiceInterface, log *utils.Logger) *OCIHandler {
	return &OCIHandler{
		service:     service,
		log:         log,
		pathManager: service.GetPathManager(),
	}
}

func (h *OCIHandler) HandleOCIAPI(c *fiber.Ctx) error {
	h.log.WithFunc().Debug("Processing API request")
	return c.JSON(fiber.Map{
		"apiVersion":            "2.0",
		"docker-content-digest": true,
		"oci-distribution-spec": "v1.0",
	})
}

func (h *OCIHandler) GetBlob(c *fiber.Ctx) error {
	digest := c.Params("digest")
	name := c.Params("name")

	h.log.WithFunc().WithFields(logrus.Fields{
		"chart":  name,
		"digest": digest,
	}).Debug("Processing blob download request")

	blobData, err := h.getBlobByDigest(digest)
	if err != nil {
		if os.IsNotExist(err) {
			h.log.WithFunc().WithError(err).Debug("Blob not found")
			return c.SendStatus(404)
		}
		h.log.WithFunc().WithError(err).Error("Failed to retrieve blob")
		return c.SendStatus(500)
	}

	c.Set("Docker-Content-Digest", digest)
	c.Set("Content-Type", "application/octet-stream")
	return c.Send(blobData)
}

func (h *OCIHandler) HandleCatalog(c *fiber.Ctx) error {
	h.log.WithFunc().Debug("Processing catalog request")

	charts, err := h.service.ListCharts()
	if err != nil {
		h.log.WithFunc().WithError(err).Error("Failed to list charts")
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

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (h *OCIHandler) HandleManifest(c *fiber.Ctx) error {
	name := c.Params("name")
	reference := c.Params("reference")

	h.log.WithFunc().WithFields(logrus.Fields{
		"name":      name,
		"reference": reference,
	}).Debug("Processing manifest request")

	var manifestPath string
	if strings.HasPrefix(reference, "sha256:") {
		manifestsDir := filepath.Join(h.pathManager.GetBasePath(), "manifests", name)
		files, err := os.ReadDir(manifestsDir)
		if err != nil {
			h.log.WithFunc().WithError(err).Debug("Manifest directory not found")
			return c.SendStatus(404)
		}

		for _, f := range files {
			if f.IsDir() {
				continue
			}
			currentPath := filepath.Join(manifestsDir, f.Name())
			data, err := os.ReadFile(currentPath)
			if err != nil {
				continue
			}

			currentDigest := fmt.Sprintf("sha256:%x", sha256.Sum256(data))
			if currentDigest == reference {
				manifestPath = currentPath
				break
			}
		}
	} else {
		manifestPath = filepath.Join(h.pathManager.GetBasePath(), "manifests", name, reference+".json")
	}

	h.log.WithFunc().WithFields(logrus.Fields{
		"manifestPath": manifestPath,
		"exists":       fileExists(manifestPath),
	}).Debug("Checking manifest existence")

	if c.Method() == "HEAD" {
		if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
			h.log.WithFunc().WithError(err).Debug("Manifest not found for HEAD request")
			return c.SendStatus(404)
		}
		return c.SendStatus(200)
	}

	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		h.log.WithFunc().WithError(err).WithField("path", manifestPath).Error("Failed to read manifest")
		return c.SendStatus(500)
	}

	if strings.HasPrefix(reference, "sha256:") {
		currentDigest := fmt.Sprintf("sha256:%x", sha256.Sum256(manifestData))
		if currentDigest != reference {
			h.log.WithFunc().WithFields(logrus.Fields{
				"expected": reference,
				"got":      currentDigest,
			}).Error("Manifest digest mismatch")
			return c.SendStatus(404)
		}
	}

	c.Set("Content-Type", "application/vnd.oci.image.manifest.v1+json")
	c.Set("Docker-Content-Digest", fmt.Sprintf("sha256:%x", sha256.Sum256(manifestData)))
	return c.Send(manifestData)
}

func (h *OCIHandler) getBlobByDigest(digest string) ([]byte, error) {
	blobPath := h.pathManager.GetBlobPath(digest)
	h.log.WithFunc().WithField("path", blobPath).Debug("Retrieving blob")

	chartData, err := os.ReadFile(blobPath)
	if err != nil {
		h.log.WithFunc().WithError(err).Error("Failed to read blob data")
		return nil, fmt.Errorf("failed to read blob: %w", err)
	}

	return chartData, nil
}

func calculateDigest(data []byte) string {
	hash := sha256.Sum256(data)
	return fmt.Sprintf("sha256:%x", hash)
}

func generateUUID() string {
	return uuid.New().String()
}

func (h *OCIHandler) PutBlob(c *fiber.Ctx) error {
	digest := calculateDigest(c.Body())
	blobPath := h.pathManager.GetBlobPath(digest)

	h.log.WithFunc().WithFields(logrus.Fields{
		"digest": digest,
		"path":   blobPath,
	}).Debug("Processing blob upload")

	if err := os.MkdirAll(filepath.Dir(blobPath), 0755); err != nil {
		h.log.WithFunc().WithError(err).Error("Failed to create blob directory")
		return err
	}

	if err := os.WriteFile(blobPath, c.Body(), 0644); err != nil {
		h.log.WithFunc().WithError(err).Error("Failed to write blob")
		return err
	}

	c.Set("Docker-Content-Digest", digest)
	return c.SendStatus(201)
}

func (h *OCIHandler) PostUpload(c *fiber.Ctx) error {
	name := c.Params("name")
	uuid := generateUUID()

	h.log.WithFunc().WithFields(logrus.Fields{
		"name": name,
		"uuid": uuid,
	}).Debug("Initializing upload")

	location := fmt.Sprintf("/v2/%s/blobs/uploads/%s", name, uuid)
	c.Set("Location", location)
	c.Set("Docker-Upload-UUID", uuid)
	return c.SendStatus(202)
}

func (h *OCIHandler) PatchBlob(c *fiber.Ctx) error {
	uuid := c.Params("uuid")
	tempPath := h.pathManager.GetTempPath(uuid)

	h.log.WithFunc().WithFields(logrus.Fields{
		"uuid": uuid,
		"size": len(c.Body()),
		"path": tempPath,
	}).Debug("Processing PATCH request")

	if err := os.MkdirAll(filepath.Dir(tempPath), 0755); err != nil {
		h.log.WithFunc().WithError(err).Error("Failed to create temp directory")
		return c.SendStatus(500)
	}

	if len(c.Body()) == 0 {
		h.log.WithFunc().Error("Received empty body")
		return c.Status(400).JSON(fiber.Map{"error": "Empty body"})
	}

	if err := os.WriteFile(tempPath, c.Body(), 0644); err != nil {
		h.log.WithFunc().WithError(err).Error("Failed to write temp file")
		return c.SendStatus(500)
	}

	h.log.WithFunc().Info("Successfully processed PATCH data")
	c.Set("Range", fmt.Sprintf("0-%d", len(c.Body())-1))
	return c.SendStatus(202)
}

func (h *OCIHandler) CompleteUpload(c *fiber.Ctx) error {
	name := c.Params("name")
	uuid := c.Params("uuid")
	digest := c.Query("digest")

	tempPath := h.pathManager.GetTempPath(uuid)
	finalPath := h.pathManager.GetBlobPath(digest)

	h.log.WithFunc().WithFields(logrus.Fields{
		"name":      name,
		"uuid":      uuid,
		"digest":    digest,
		"tempPath":  tempPath,
		"finalPath": finalPath,
	}).Debug("Completing upload")

	if len(c.Body()) > 0 {
		if err := os.WriteFile(tempPath, c.Body(), 0644); err != nil {
			h.log.WithFunc().WithError(err).Error("Failed to write final data")
			return c.SendStatus(500)
		}
	}

	if err := os.Rename(tempPath, finalPath); err != nil {
		h.log.WithFunc().WithError(err).Error("Failed to finalize upload")
		return c.SendStatus(500)
	}

	c.Set("Docker-Content-Digest", digest)
	h.log.WithFunc().WithField("name", name).Info("Upload completed successfully")
	return c.SendStatus(201)
}

func (h *OCIHandler) HeadBlob(c *fiber.Ctx) error {
	digest := c.Params("digest")
	name := c.Params("name")
	blobPath := h.pathManager.GetBlobPath(digest)

	h.log.WithFunc().WithFields(logrus.Fields{
		"chart":  name,
		"digest": digest,
		"path":   blobPath,
	}).Debug("Processing HEAD request")

	if _, err := os.Stat(blobPath); err != nil {
		if os.IsNotExist(err) {
			h.log.WithFunc().WithError(err).Debug("Blob not found")
			return c.SendStatus(404)
		}
		h.log.WithFunc().WithError(err).Error("Failed to check blob")
		return c.SendStatus(500)
	}

	info, err := os.Stat(blobPath)
	if err != nil {
		h.log.WithFunc().WithError(err).Error("Failed to get blob info")
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

	h.log.WithFunc().WithFields(logrus.Fields{
		"name":      name,
		"reference": reference,
	}).Debug("Processing manifest upload")

	var manifest models.OCIManifest
	if err := json.Unmarshal(c.Body(), &manifest); err != nil {
		h.log.WithFunc().WithError(err).Error("Failed to parse manifest")
		return c.SendStatus(500)
	}

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
		chartData, err := h.getBlobByDigest(chartLayer.Digest)
		if err != nil {
			h.log.WithFunc().WithError(err).Error("Failed to read chart data")
			return c.SendStatus(500)
		}

		fileName := fmt.Sprintf("%s-%s.tgz", name, reference)
		if err := h.service.SaveChart(chartData, fileName); err != nil {
			h.log.WithFunc().WithError(err).Error("Failed to save chart")
			return c.SendStatus(500)
		}
	} else {
		h.log.WithFunc().Error("Chart layer not found in manifest")
		return c.SendStatus(500)
	}

	manifestData := c.Body()
	manifestPath := h.pathManager.GetManifestPath(name, reference)

	manifestDir := filepath.Dir(manifestPath)
	if err := os.MkdirAll(manifestDir, 0755); err != nil {
		h.log.WithFunc().WithError(err).Error("Failed to create manifest directory")
		return c.SendStatus(500)
	}

	if err := os.WriteFile(manifestPath, manifestData, 0644); err != nil {
		h.log.WithFunc().WithError(err).Error("Failed to save manifest")
		return c.SendStatus(500)
	}

	digest := sha256.Sum256(manifestData)
	digestStr := fmt.Sprintf("sha256:%x", digest)
	c.Set("Docker-Content-Digest", digestStr)

	h.log.WithFunc().WithField("name", name).Info("Manifest saved successfully")
	return c.SendStatus(201)
}
