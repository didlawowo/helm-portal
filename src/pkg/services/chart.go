// internal/chart/service/chart_service.go

package service

import (
	"archive/tar"
	"bytes"
	"compress/gzip"

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
	fileName := fmt.Sprintf("%s-%s.tgz", chartName, version)
	chartPath := filepath.Join(s.pathManager.GetGlobalPath(), fileName)

	return os.ReadFile(chartPath)
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
