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
	log          *utils.Logger
	chartService interfaces.ChartServiceInterface
	imageService interfaces.ImageServiceInterface
	pathManager  *utils.PathManager
}

func NewOCIHandler(chartService interfaces.ChartServiceInterface, imageService interfaces.ImageServiceInterface, log *utils.Logger) *OCIHandler {
	return &OCIHandler{
		chartService: chartService,
		imageService: imageService,
		log:          log,
		pathManager:  chartService.GetPathManager(),
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

	repositories := make([]string, 0)

	// List Helm charts
	charts, err := h.chartService.ListCharts()
	if err != nil {
		h.log.WithFunc().WithError(err).Warn("Failed to list charts")
	} else {
		for _, chart := range charts {
			repositories = append(repositories, chart.Name)
		}
	}

	// List Docker images
	if h.imageService != nil {
		images, err := h.imageService.ListImages()
		if err != nil {
			h.log.WithFunc().WithError(err).Warn("Failed to list images")
		} else {
			for _, image := range images {
				// Avoid duplicates
				found := false
				for _, r := range repositories {
					if r == image.Name {
						found = true
						break
					}
				}
				if !found {
					repositories = append(repositories, image.Name)
				}
			}
		}
	}

	return c.JSON(fiber.Map{
		"repositories": repositories,
	})
}

// HandleListTags returns all tags for a repository (OCI Distribution Spec)
func (h *OCIHandler) HandleListTags(c *fiber.Ctx) error {
	name := c.Params("name")

	h.log.WithFunc().WithField("name", name).Debug("Processing tags list request")

	tags := make([]string, 0)

	// Try to get tags from image service first
	if h.imageService != nil {
		imageTags, err := h.imageService.ListTags(name)
		if err == nil && len(imageTags) > 0 {
			tags = append(tags, imageTags...)
		}
	}

	// Also check for chart versions as tags
	charts, err := h.chartService.ListCharts()
	if err == nil {
		for _, chart := range charts {
			if chart.Name == name {
				for _, version := range chart.Versions {
					tags = append(tags, version.Version)
				}
				break
			}
		}
	}

	return c.JSON(fiber.Map{
		"name": name,
		"tags": tags,
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

	// Try to find manifest in multiple locations
	manifestData, manifestPath, err := h.findManifest(name, reference)
	if err != nil {
		h.log.WithFunc().WithError(err).Debug("Manifest not found")
		return c.SendStatus(404)
	}

	h.log.WithFunc().WithFields(logrus.Fields{
		"manifestPath": manifestPath,
	}).Debug("Found manifest")

	// For HEAD requests, just verify existence
	if c.Method() == "HEAD" {
		digest := fmt.Sprintf("sha256:%x", sha256.Sum256(manifestData))
		c.Set("Content-Type", "application/vnd.oci.image.manifest.v1+json")
		c.Set("Docker-Content-Digest", digest)
		c.Set("Content-Length", fmt.Sprintf("%d", len(manifestData)))
		return c.SendStatus(200)
	}

	// Verify digest if reference is a digest
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

	// Determine content type from manifest
	var manifest models.OCIManifest
	if err := json.Unmarshal(manifestData, &manifest); err == nil && manifest.MediaType != "" {
		c.Set("Content-Type", manifest.MediaType)
	} else {
		c.Set("Content-Type", "application/vnd.oci.image.manifest.v1+json")
	}

	c.Set("Docker-Content-Digest", fmt.Sprintf("sha256:%x", sha256.Sum256(manifestData)))
	return c.Send(manifestData)
}

// findManifest searches for a manifest in all possible locations
func (h *OCIHandler) findManifest(name, reference string) ([]byte, string, error) {
	var searchPaths []string

	if strings.HasPrefix(reference, "sha256:") {
		// For digest references, we need to search through all manifests
		return h.findManifestByDigest(name, reference)
	}

	// Build list of paths to check (tag-based reference)
	searchPaths = []string{
		// Helm charts manifests
		filepath.Join(h.pathManager.GetBasePath(), "manifests", name, reference+".json"),
		// Docker images manifests
		h.pathManager.GetImageManifestPath(name, reference),
	}

	for _, path := range searchPaths {
		if data, err := os.ReadFile(path); err == nil {
			return data, path, nil
		}
	}

	return nil, "", fmt.Errorf("manifest not found for %s:%s", name, reference)
}

// findManifestByDigest searches for a manifest by its digest
func (h *OCIHandler) findManifestByDigest(name, digest string) ([]byte, string, error) {
	// Search in Helm manifests directory
	helmManifestsDir := filepath.Join(h.pathManager.GetBasePath(), "manifests", name)
	if data, path, err := h.searchDirForDigest(helmManifestsDir, digest); err == nil {
		return data, path, nil
	}

	// Search in Docker images manifests directory
	imageManifestsDir := filepath.Join(h.pathManager.GetBasePath(), "images", name, "manifests")
	if data, path, err := h.searchDirForDigest(imageManifestsDir, digest); err == nil {
		return data, path, nil
	}

	return nil, "", fmt.Errorf("manifest with digest %s not found", digest)
}

// searchDirForDigest searches a directory for a file matching the given digest
func (h *OCIHandler) searchDirForDigest(dir, targetDigest string) ([]byte, string, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, "", err
	}

	for _, f := range files {
		if f.IsDir() {
			continue
		}
		filePath := filepath.Join(dir, f.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		currentDigest := fmt.Sprintf("sha256:%x", sha256.Sum256(data))
		if currentDigest == targetDigest {
			return data, filePath, nil
		}
	}

	return nil, "", fmt.Errorf("digest not found in %s", dir)
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

	// Detect artifact type
	artifactType := models.DetectArtifactType(&manifest)

	h.log.WithFunc().WithFields(logrus.Fields{
		"name":         name,
		"reference":    reference,
		"artifactType": artifactType,
		"configType":   manifest.Config.MediaType,
	}).Debug("Detected artifact type")

	manifestData := c.Body()
	digest := sha256.Sum256(manifestData)
	digestStr := fmt.Sprintf("sha256:%x", digest)

	switch artifactType {
	case models.ArtifactTypeHelmChart:
		// Handle Helm chart
		if err := h.handleHelmChartManifest(name, reference, &manifest); err != nil {
			h.log.WithFunc().WithError(err).Error("Failed to handle Helm chart")
			return c.SendStatus(500)
		}
		// Save manifest to Helm manifests directory
		manifestPath := h.pathManager.GetManifestPath(name, reference)
		if err := h.saveManifestFile(manifestPath, manifestData); err != nil {
			return c.SendStatus(500)
		}

	case models.ArtifactTypeDockerImage:
		// Handle Docker image
		if h.imageService != nil {
			if err := h.imageService.SaveImage(name, reference, &manifest); err != nil {
				h.log.WithFunc().WithError(err).Error("Failed to save Docker image")
				return c.SendStatus(500)
			}
		} else {
			h.log.WithFunc().Warn("Image service not configured, saving manifest only")
			// Fall back to saving manifest in images directory
			manifestPath := h.pathManager.GetImageManifestPath(name, reference)
			if err := h.saveManifestFile(manifestPath, manifestData); err != nil {
				return c.SendStatus(500)
			}
		}

	default:
		// Unknown artifact type - save as generic OCI artifact
		h.log.WithFunc().WithFields(logrus.Fields{
			"configMediaType": manifest.Config.MediaType,
		}).Warn("Unknown artifact type, saving as generic manifest")
		manifestPath := h.pathManager.GetManifestPath(name, reference)
		if err := h.saveManifestFile(manifestPath, manifestData); err != nil {
			return c.SendStatus(500)
		}
	}

	c.Set("Docker-Content-Digest", digestStr)
	c.Set("Location", fmt.Sprintf("/v2/%s/manifests/%s", name, digestStr))

	h.log.WithFunc().WithFields(logrus.Fields{
		"name":         name,
		"reference":    reference,
		"artifactType": artifactType,
		"digest":       digestStr,
	}).Info("Manifest saved successfully")

	return c.SendStatus(201)
}

// handleHelmChartManifest processes a Helm chart manifest
func (h *OCIHandler) handleHelmChartManifest(name, reference string, manifest *models.OCIManifest) error {
	// Find the chart layer
	var chartDigest string
	for _, layer := range manifest.Layers {
		if layer.MediaType == models.MediaTypeHelmChart {
			chartDigest = layer.Digest
			break
		}
	}

	if chartDigest == "" {
		return fmt.Errorf("Helm chart layer not found in manifest")
	}

	// Get the chart data from blob storage
	chartData, err := h.getBlobByDigest(chartDigest)
	if err != nil {
		return fmt.Errorf("failed to read chart data: %w", err)
	}

	// Save the chart
	fileName := fmt.Sprintf("%s-%s.tgz", name, reference)
	if err := h.chartService.SaveChart(chartData, fileName); err != nil {
		return fmt.Errorf("failed to save chart: %w", err)
	}

	return nil
}

// saveManifestFile saves a manifest to the specified path
func (h *OCIHandler) saveManifestFile(manifestPath string, data []byte) error {
	manifestDir := filepath.Dir(manifestPath)
	if err := os.MkdirAll(manifestDir, 0755); err != nil {
		h.log.WithFunc().WithError(err).Error("Failed to create manifest directory")
		return err
	}

	if err := os.WriteFile(manifestPath, data, 0644); err != nil {
		h.log.WithFunc().WithError(err).Error("Failed to save manifest")
		return err
	}

	return nil
}
