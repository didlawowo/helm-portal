// pkg/services/interfaces.go
package service

type IndexUpdater interface {
	UpdateIndex() error
	EnsureIndexExists() error
	GetIndexPath() string
}

type ChartExtractor interface {
	ExtractChartMetadata(chartData []byte) (*ChartMetadata, error)
}
