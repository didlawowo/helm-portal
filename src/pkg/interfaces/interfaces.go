package interfaces

import (
	"helm-portal/pkg/models"
	"helm-portal/pkg/storage"
)

type ChartServiceInterface interface {
	SaveChart(data []byte, filename string) error
	ListCharts() ([]models.ChartMetadata, error)
	ChartExists(name, version string) bool
	GetChart(name, version string) ([]byte, error)
	GetChartDetails(name, version string) (*models.ChartMetadata, error)
	DeleteChart(name, version string) error
	GetPathManager() *storage.PathManager
	ExtractChartMetadata(chartData []byte) (*models.ChartMetadata, error)
	CreateChartTgz(chartMetadata *models.ChartMetadata, chartData []byte) ([]byte, error)
}

type IndexServiceInterface interface {
	UpdateIndex() error
	GetIndexPath() string
	EnsureIndexExists() error
}
