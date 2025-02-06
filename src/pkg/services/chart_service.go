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

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// ChartMetadata represents the Chart.yaml file structure
type ChartMetadata struct {
	Name         string `yaml:"name"`
	Version      string `yaml:"version"`
	Description  string `yaml:"description"`
	ApiVersion   string `yaml:"apiVersion"`
	Type         string `yaml:"type,omitempty"`
	AppVersion   string `yaml:"appVersion,omitempty"`
	Dependencies []struct {
		Name       string `yaml:"name"`
		Version    string `yaml:"version"`
		Repository string `yaml:"repository"`
	} `yaml:"dependencies,omitempty"`
}

// ChartService handles chart operations
type ChartService struct {
	storagePath string
	config      *config.Config
	log         *logrus.Logger
	baseURL     string
	// maxChartSize int64
}

// NewChartService creates a new chart service
func NewChartService(config *config.Config, log *logrus.Logger) *ChartService {
	return &ChartService{
		storagePath: config.Storage.Path,
		config:      config,
		log:         log,
		baseURL:     config.Helm.BaseURL,
		// maxChartSize: config.Helm.MaxChartSize,
	}
}

// SaveChart saves an uploaded chart file
func (s *ChartService) SaveChart(chartData []byte, filename string, baseURL string) error {
	// ✨ Create charts directory if not exists
	chartsDir := filepath.Join(s.storagePath, "charts")
	if err := os.MkdirAll(chartsDir, 0755); err != nil {
		return fmt.Errorf("❌ failed to create charts directory: %w", err)
	}

	// 💾 Save chart file
	chartPath := filepath.Join(chartsDir, filename)
	if err := os.WriteFile(chartPath, chartData, 0644); err != nil {
		return fmt.Errorf("❌ failed to save chart: %w", err)
	}

	// 📝 Extract and validate metadata
	metadata, err := s.extractChartMetadata(chartData)
	if err != nil {
		return fmt.Errorf("❌ failed to extract chart metadata: %w", err)
	}

	if err := s.UpdateIndex(baseURL); err != nil {
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
func (s *ChartService) extractChartMetadata(chartData []byte) (*ChartMetadata, error) {
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
			var metadata ChartMetadata
			if err := yaml.Unmarshal(content, &metadata); err != nil {
				return nil, err
			}

			return &metadata, nil
		}
	}

	return nil, fmt.Errorf("Chart.yaml not found in chart archive")
}

// updateIndex rebuilds the index.yaml file
func (s *ChartService) updateIndex() error {
	// 🏗️ Implement index.yaml generation
	// This is a placeholder for now
	index := map[string]interface{}{
		"apiVersion": "v1",
		"entries":    make(map[string]interface{}),
	}

	// Convert to YAML
	indexYaml, err := yaml.Marshal(index)
	if err != nil {
		return err
	}

	// Save index file
	indexPath := filepath.Join(s.storagePath, "index.yaml")
	if err := os.WriteFile(indexPath, indexYaml, 0644); err != nil {
		return err
	}

	return nil
}

// ListCharts returns all available charts
func (s *ChartService) ListCharts() ([]ChartMetadata, error) {
	chartsDir := filepath.Join(s.storagePath, "charts")
	var charts []ChartMetadata

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
		metadata, err := s.extractChartMetadata(chartData)
		if err != nil {
			s.log.WithError(err).WithField("file", file.Name()).Error("Failed to extract metadata")
			continue
		}

		charts = append(charts, *metadata)
	}

	return charts, nil
}

func (s *ChartService) RegenerateIndex(baseURL string) error {
	s.log.Info("🔄 Démarrage régénération index")
	return s.UpdateIndex(baseURL)
}

func (s *ChartService) EnsureIndexExists(baseURL string) error {
	indexPath := filepath.Join(s.storagePath, "index.yaml")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		return s.UpdateIndex(baseURL)
	}
	return nil
}

func (s *ChartService) GetIndexPath() string {
	return filepath.Join(s.storagePath, "index.yaml")
}

func (s *ChartService) GetChartPath(chartName string) string {
	return filepath.Join(s.storagePath, "charts", chartName)
}

func (s *ChartService) ChartExists(chartName string) bool {
	_, err := os.Stat(s.GetChartPath(chartName))
	return !os.IsNotExist(err)
}

func (s *ChartService) IndexExists() bool {
	_, err := os.Stat(s.GetIndexPath())
	return !os.IsNotExist(err)
}

func (s *ChartService) GetChart(chartName string) ([]byte, error) {
	chartPath := s.GetChartPath(chartName)
	return os.ReadFile(chartPath)
}

func (s *ChartService) DeleteChart(chartName string, version string) error {
	// Construire le nom du fichier avec la version
	chartFileName := fmt.Sprintf("%s-%s.tgz", chartName, version)
	chartPath := s.GetChartPath(chartFileName)

	// Vérifier si le chart existe
	if !s.ChartExists(chartFileName) {
		return fmt.Errorf("chart %s version %s not found", chartName, version)
	}

	// Supprimer le fichier
	if err := os.Remove(chartPath); err != nil {
		return fmt.Errorf("failed to delete chart: %w", err)
	}

	// Mettre à jour l'index
	return s.UpdateIndex(s.baseURL)
}
