package handlers

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"helm-portal/pkg/storage"

	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupTestEnv(t *testing.T) (*fiber.App, *MockChartService, *OCIHandler, func()) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "helm-portal-test")
	assert.NoError(t, err)

	// Setup components
	log := logrus.New()
	mockService := new(MockChartService)
	pathManager := storage.NewPathManager(tempDir)

	mockService.On("GetPathManager").Return(pathManager)

	handler := NewOCIHandler(mockService, log)
	app := fiber.New()

	// Cleanup function
	cleanup := func() {
		os.RemoveAll(tempDir)
	}

	return app, mockService, handler, cleanup
}

func TestHandleManifest(t *testing.T) {
	app, mockService, handler, cleanup := setupTestEnv(t)
	defer cleanup()

	app.Get("/v2/:name/manifests/:reference", handler.HandleManifest)
	app.Head("/v2/:name/manifests/:reference", handler.HandleManifest)

	manifestContent := []byte(`{"schemaVersion": 2}`)
	manifestPath := filepath.Join(mockService.GetPathManager().GetGlobalPath(), "test-chart", "1.0.0.json")

	// Ensure directory exists
	os.MkdirAll(filepath.Dir(manifestPath), 0755)
	os.WriteFile(manifestPath, manifestContent, 0644)

	tests := []struct {
		name           string
		method         string
		chartName      string
		reference      string
		expectedStatus int
	}{
		{
			name:           "GET manifest success",
			method:         "GET",
			chartName:      "test-chart",
			reference:      "1.0.0",
			expectedStatus: 200,
		},
		{
			name:           "HEAD manifest success",
			method:         "HEAD",
			chartName:      "test-chart",
			reference:      "1.0.0",
			expectedStatus: 200,
		},
		{
			name:           "manifest not found",
			method:         "GET",
			chartName:      "missing-chart",
			reference:      "1.0.0",
			expectedStatus: 404,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService.On("ChartExists", tt.chartName, tt.reference).Return(tt.chartName == "test-chart")

			req := httptest.NewRequest(tt.method, "/v2/"+tt.chartName+"/manifests/"+tt.reference, nil)
			resp, err := app.Test(req)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
}

func TestPutManifest(t *testing.T) {
	app, mockService, handler, cleanup := setupTestEnv(t)
	defer cleanup()

	app.Put("/v2/:name/manifests/:reference", handler.PutManifest)

	manifest := OCIManifest{
		SchemaVersion: 2,
		MediaType:     "application/vnd.oci.image.manifest.v1+json",
		Config: struct {
			MediaType string `json:"mediaType"`
			Digest    string `json:"digest"`
			Size      int    `json:"size"`
		}{
			MediaType: "application/vnd.cncf.helm.config.v1+json",
			Digest:    "sha256:123",
			Size:      100,
		},
		Layers: []struct {
			MediaType string `json:"mediaType"`
			Digest    string `json:"digest"`
			Size      int    `json:"size"`
		}{
			{
				MediaType: "application/vnd.cncf.helm.chart.content.v1.tar+gzip",
				Digest:    "sha256:456",
				Size:      200,
			},
		},
	}

	manifestBytes, _ := json.Marshal(manifest)

	tests := []struct {
		name           string
		chartName      string
		reference      string
		body           []byte
		setupMocks     func()
		expectedStatus int
	}{
		{
			name:      "successful put manifest",
			chartName: "test-chart",
			reference: "1.0.0",
			body:      manifestBytes,
			setupMocks: func() {
				mockService.On("SaveChart", mock.Anything, "test-chart-1.0.0.tgz").Return(nil)
				mockService.On("GetBlobByDigest", "sha256:456").Return([]byte("chart data"), nil)
			},
			expectedStatus: 201,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupMocks != nil {
				tt.setupMocks()
			}

			req := httptest.NewRequest("PUT", "/v2/"+tt.chartName+"/manifests/"+tt.reference,
				bytes.NewReader(tt.body))
			resp, err := app.Test(req)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
}
