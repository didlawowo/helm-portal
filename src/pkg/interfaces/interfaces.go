package interfaces

import (
	"helm-portal/pkg/models"
	"helm-portal/pkg/storage"
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

type BackupServiceInterface interface {
	BackupCharts() error
	RestoreCharts() error
}

type IndexServiceInterface interface {
	UpdateIndex() error
	GetIndexPath() string
	EnsureIndexExists() error
}
