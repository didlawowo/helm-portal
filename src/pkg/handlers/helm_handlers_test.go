// pkg/handlers/helm_handlers_test.go
package handlers

import (
	"bytes"

	"mime/multipart"
	"net/http/httptest"

	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockChartService est un mock du service
type MockChartService struct {
	mock.Mock
}

// Implémentation des méthodes mock...

func TestUploadChart(t *testing.T) {
	// Setup
	log := logrus.New()
	mockService := new(MockChartService)
	handler := NewChartHandler(mockService, log)
	app := fiber.New()
	app.Post("/charts", handler.UploadChart)

	// Créer un fichier test
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("chart", "test-chart.tgz")
	part.Write([]byte("test content"))
	writer.Close()

	// Test
	req := httptest.NewRequest("POST", "/charts", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	mockService.On("SaveChart", mock.Anything, "test-chart.tgz").Return(nil)

	resp, err := app.Test(req)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	mockService.AssertExpectations(t)
}
