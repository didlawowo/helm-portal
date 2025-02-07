// pkg/handlers/mocks.go
package handlers

import (
	"helm-portal/pkg/models"

	"github.com/stretchr/testify/mock"
)

// MockChartService implémente l'interface ChartService pour les tests
type MockChartService struct {
	mock.Mock
}

// Implémentation des méthodes requises
func (m *MockChartService) SaveChart(chartData []byte, filename string) error {
	args := m.Called(chartData, filename)
	return args.Error(0)
}

func (m *MockChartService) GetChart(name string) ([]byte, error) {
	args := m.Called(name)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockChartService) DeleteChart(name string, version string) error {
	args := m.Called(name, version)
	return args.Error(0)
}

func (m *MockChartService) ListCharts() ([]models.ChartMetadata, error) {
	args := m.Called()
	return args.Get(0).([]models.ChartMetadata), args.Error(1)
}

func (m *MockChartService) ChartExists(name string) bool {
	args := m.Called(name)
	return args.Bool(0)
}

// Idem pour le service Index
type MockIndexService struct {
	mock.Mock
}
