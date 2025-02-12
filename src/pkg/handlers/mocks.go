// pkg/handlers/mocks.go
package handlers

import (
	"helm-portal/pkg/models"
	utils "helm-portal/pkg/utils"

	"github.com/stretchr/testify/mock"
)

type MockChartService struct {
	mock.Mock
}

func (m *MockChartService) SaveChart(chartData []byte, filename string) error {
	args := m.Called(chartData, filename)
	return args.Error(0)
}

func (m *MockChartService) GetChart(name string, version string) ([]byte, error) {
	args := m.Called(name, version)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockChartService) DeleteChart(name string, version string) error {
	args := m.Called(name, version)
	return args.Error(0)
}

func (m *MockChartService) ListCharts() ([]models.ChartGroup, error) { // Correction ici
	args := m.Called()
	return args.Get(0).([]models.ChartGroup), args.Error(1)
}

func (m *MockChartService) ChartExists(name string, version string) bool {
	args := m.Called(name, version)
	return args.Bool(0)
}

func (m *MockChartService) GetPathManager() *utils.PathManager {
	args := m.Called()
	return args.Get(0).(*utils.PathManager)
}

func (m *MockChartService) GetChartValues(name, version string) (string, error) {
	args := m.Called(name, version)
	return args.String(0), args.Error(1)
}

// Nouvelles méthodes
func (m *MockChartService) GetChartDetails(name, version string) (*models.ChartMetadata, error) {
	args := m.Called(name, version)
	return args.Get(0).(*models.ChartMetadata), args.Error(1)
}

func (m *MockChartService) ExtractChartMetadata(chartData []byte) (*models.ChartMetadata, error) {
	args := m.Called(chartData)
	return args.Get(0).(*models.ChartMetadata), args.Error(1)
}
