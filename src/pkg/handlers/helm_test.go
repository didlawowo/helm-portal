package handlers

import (
	"bytes"
	"mime/multipart"
	"net/http/httptest"
	"testing"

	"helm-portal/pkg/utils"

	"github.com/gofiber/fiber/v2"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestUploadChart(t *testing.T) {
	// Setup
	logger := utils.NewLogger(utils.Config{
		LogLevel:  "debug", // ou le niveau souhaité
		LogFormat: "json",  // ou "text"
		Pretty:    true,
	})
	log := utils.NewLogger(utils.Config{})

	mockService := new(MockChartService)
	pathManager := utils.NewPathManager("./charts", logger)
	handler := NewHelmHandler(mockService, pathManager, log)

	app := fiber.New()
	app.Post("/chart", handler.UploadChart)

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("chart", "test-chart.tgz")
	assert.NoError(t, err)
	part.Write([]byte("test content"))
	writer.Close()

	// Test
	req := httptest.NewRequest("POST", "/chart", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	mockService.On("SaveChart", mock.Anything, "test-chart.tgz").Return(nil)

	resp, err := app.Test(req)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	mockService.AssertExpectations(t)
}
