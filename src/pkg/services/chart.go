// internal/chart/service/chart_service.go

package service

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os/exec"

	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"helm-portal/config"
	"helm-portal/pkg/models"
	"helm-portal/pkg/storage"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

type IndexUpdater interface {
	UpdateIndex() error
	EnsureIndexExists() error
	GetIndexPath() string
}

// ChartService handles chart operations
type ChartService struct {
	pathManager  *storage.PathManager
	config       *config.Config
	log          *logrus.Logger
	baseURL      string
	indexUpdater IndexUpdater
}

// NewChartService creates a new chart service
func NewChartService(config *config.Config, log *logrus.Logger, indexUpdater IndexUpdater) *ChartService {
	if err := os.MkdirAll(config.Storage.Path, 0755); err != nil {
		log.WithError(err).Error("❌ Impossible de créer le dossier de stockage")
	}
	return &ChartService{
		pathManager:  storage.NewPathManager(config.Storage.Path),
		config:       config,
		log:          log,
		baseURL:      config.Helm.BaseURL,
		indexUpdater: indexUpdater,
	}
}
func (s *ChartService) GetPathManager() *storage.PathManager {
	return s.pathManager
}

// SaveChart saves an uploaded chart file
func (s *ChartService) SaveChart(chartData []byte, filename string) error {
	// ✨ Create charts directory if not exists
	chartsDir := s.pathManager.GetGlobalPath()
	if err := os.MkdirAll(chartsDir, 0755); err != nil {
		return fmt.Errorf("❌ failed to create charts directory: %w", err)
	}

	// 💾 Save chart file
	chartPath := filepath.Join(chartsDir, filename)
	if err := os.WriteFile(chartPath, chartData, 0644); err != nil {
		return fmt.Errorf("❌ failed to save chart: %w", err)
	}

	// 📝 Extract and validate metadata
	metadata, err := s.ExtractChartMetadata(chartData)
	if err != nil {
		return fmt.Errorf("❌ failed to extract chart metadata: %w", err)
	}

	if err := s.indexUpdater.UpdateIndex(); err != nil {
		s.log.WithError(err).Error("❌ Échec mise à jour index")
		return fmt.Errorf("échec mise à jour index: %w", err)
	}

	s.log.WithFields(logrus.Fields{
		"name":    metadata.Name,
		"version": metadata.Version,
		"file":    filename,
	}).Info("✅ Chart saved successfully")

	return nil
}

// extractChartMetadata extracts Chart.yaml from the tgz file
func (s *ChartService) ExtractChartMetadata(chartData []byte) (*models.ChartMetadata, error) {
	// 📦 Read the gzip file
	gr, err := gzip.NewReader(bytes.NewReader(chartData))
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gr.Close()

	// 📂 Read the tar archive
	tr := tar.NewReader(gr)

	// 🔍 Look for Chart.yaml
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		// Find Chart.yaml in the root directory of the chart
		if filepath.Base(header.Name) == "Chart.yaml" {
			// Read the Chart.yaml content
			content, err := io.ReadAll(tr)
			if err != nil {
				return nil, err
			}

			// Parse YAML
			var metadata models.ChartMetadata
			if err := yaml.Unmarshal(content, &metadata); err != nil {
				return nil, err
			}

			return &metadata, nil
		}
	}

	return nil, fmt.Errorf("Chart.yaml not found in chart archive")
}

// ListCharts returns all available charts
func (s *ChartService) ListCharts() ([]models.ChartMetadata, error) {
	chartsDir := s.pathManager.GetGlobalPath()
	var charts []models.ChartMetadata

	// Read charts directory
	files, err := os.ReadDir(chartsDir)
	if err != nil {
		return nil, err
	}

	// Process each .tgz file
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".tgz") {
			continue
		}

		// Read chart data
		chartData, err := os.ReadFile(filepath.Join(chartsDir, file.Name()))
		if err != nil {
			s.log.WithError(err).WithField("file", file.Name()).Error("Failed to read chart")
			continue
		}

		// Extract metadata
		metadata, err := s.ExtractChartMetadata(chartData)
		if err != nil {
			s.log.WithError(err).WithField("file", file.Name()).Error("Failed to extract metadata")
			continue
		}

		charts = append(charts, *metadata)
	}

	return charts, nil
}

func (s *ChartService) ChartExists(chartName string, version string) bool {
	_, err := os.Stat(s.pathManager.GetChartPath(chartName, version))
	return !os.IsNotExist(err)
}

func (s *ChartService) GetChart(chartName string, version string) ([]byte, error) {
	chartPath := s.pathManager.GetChartPath(chartName, version)
	// Vérifier si le chart existe
	if !s.ChartExists(chartName, version) {
		return nil, fmt.Errorf("chart %s version %s not found", chartName, version)
	}
	// Lire le fichier
	return os.ReadFile(chartPath)
}

func (s *ChartService) GetChartDetails(chartName string, version string) (*models.ChartMetadata, error) {
	chartPath := s.pathManager.GetChartPath(chartName, version)
	// Vérifier si le chart existe
	if !s.ChartExists(chartName, version) {
		return nil, fmt.Errorf("chart %s version %s not found", chartName, version)
	}
	// Lire le fichier
	chartData, err := os.ReadFile(chartPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read chart: %w", err)
	}
	// Extract metadata
	metadata, err := s.ExtractChartMetadata(chartData)
	if err != nil {
		return nil, fmt.Errorf("failed to extract metadata: %w", err)
	}
	return metadata, nil
}

func (s *ChartService) DeleteChart(chartName string, version string) error {
	chartPath := s.pathManager.GetChartPath(chartName, version)

	// Vérifier si le chart existe
	if !s.ChartExists(chartName, version) {
		return fmt.Errorf("chart %s version %s not found", chartName, version)
	}

	// Supprimer le fichier
	if err := os.Remove(chartPath); err != nil {
		return fmt.Errorf("failed to delete chart: %w", err)
	}

	// Mettre à jour l'index
	return s.indexUpdater.UpdateIndex()
}

func (s *ChartService) CreateChartTgz(chartName string, version string) error {
	// Chemin où seront assemblés les blobs
	chartDir := s.pathManager.GetChartPath(chartName, version)
	blobsDir := s.pathManager.GetBlobPath(chartName)

	// Créer un répertoire temporaire pour l'assemblage
	tempDir, err := os.MkdirTemp("", "chart-assemble-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	// Lister tous les blobs pour ce chart
	blobFiles, err := filepath.Glob(filepath.Join(blobsDir, "sha256:*"))
	if err != nil {
		return err
	}

	// Créer la structure de chart
	if err := os.MkdirAll(filepath.Join(tempDir, "templates"), 0755); err != nil {
		return err
	}

	// Copier les blobs dans le répertoire temporaire
	for _, blobPath := range blobFiles {
		destPath := filepath.Join(tempDir, "templates", filepath.Base(blobPath))
		if err := copyFile(blobPath, destPath); err != nil {
			return err
		}
	}

	// Créer le fichier Chart.yaml
	chartYamlContent := fmt.Sprintf(`apiVersion: v2
name: %s
version: %s
description: Automatically generated chart
`, chartName, version)

	if err := os.WriteFile(filepath.Join(tempDir, "Chart.yaml"), []byte(chartYamlContent), 0644); err != nil {
		return err
	}

	// Créer le .tgz
	outputTgzPath := filepath.Join(chartDir, fmt.Sprintf("%s-%s.tgz", chartName, version))
	cmd := exec.Command("tar", "-czf", outputTgzPath, "-C", tempDir, ".")
	if err := cmd.Run(); err != nil {
		return err
	}

	s.log.WithFields(logrus.Fields{
		"chartName": chartName,
		"version":   version,
		"tgzPath":   outputTgzPath,
	}).Info("Chart .tgz created successfully")

	return nil
}

// Fonction utilitaire pour copier des fichiers
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}
