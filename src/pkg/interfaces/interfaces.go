package interfaces

import (
	"helm-portal/pkg/models"
	storage "helm-portal/pkg/utils"
)

type ChartServiceInterface interface {
	SaveChart(data []byte, filename string) error
	ListCharts() ([]models.ChartGroup, error)
	ChartExists(name, version string) bool
	GetChart(name, version string) ([]byte, error)
	GetChartDetails(name, version string) (*models.ChartMetadata, error)
	DeleteChart(name, version string) error
	GetPathManager() *storage.PathManager
	GetChartValues(name, version string) (string, error)
	ExtractChartMetadata(chartData []byte) (*models.ChartMetadata, error)
}

type ImageServiceInterface interface {
	// SaveImage saves a Docker image manifest and metadata
	SaveImage(name, reference string, manifest *models.OCIManifest) error
	// ListImages returns all available images grouped by name
	ListImages() ([]models.ImageGroup, error)
	// ImageExists checks if an image with the given name and tag exists
	ImageExists(name, tag string) bool
	// GetImageManifest returns the manifest for a specific image
	GetImageManifest(name, reference string) (*models.OCIManifest, error)
	// GetImageMetadata returns metadata for a specific image
	GetImageMetadata(name, tag string) (*models.ImageMetadata, error)
	// DeleteImage removes an image by name and tag
	DeleteImage(name, tag string) error
	// GetImageConfig returns the parsed image configuration
	GetImageConfig(name, tag string) (*models.ImageConfig, error)
	// ListTags returns all tags for a given repository
	ListTags(name string) ([]string, error)
	// GetPathManager returns the path manager
	GetPathManager() *storage.PathManager
}

type BackupServiceInterface interface {
	BackupCharts() error
	RestoreCharts() error
}

type IndexServiceInterface interface {
	UpdateIndex() error
	GetIndexPath() string
	EnsureIndexExists() error
}
