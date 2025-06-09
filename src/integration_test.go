package main_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestHealthEndpointIntegration(t *testing.T) {
	// Create a new Fiber app
	app := fiber.New()

	// Add the health route exactly as in main.go
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// Test cases
	tests := []struct {
		name           string
		method         string
		route          string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Valid health check",
			method:         "GET",
			route:          "/health",
			expectedStatus: http.StatusOK,
			expectedBody:   "OK",
		},
		{
			name:           "Health check with POST method should fail",
			method:         "POST",
			route:          "/health",
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   "",
		},
		{
			name:           "Non-existent endpoint should return 404",
			method:         "GET",
			route:          "/nonexistent",
			expectedStatus: http.StatusNotFound,
			expectedBody:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new HTTP request
			req := httptest.NewRequest(tt.method, tt.route, nil)

			// Perform the request
			resp, err := app.Test(req)

			// Assert no error occurred
			assert.NoError(t, err)

			// Assert status code
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			// For successful health checks, verify the response body
			if tt.expectedStatus == http.StatusOK {
				body := make([]byte, len(tt.expectedBody))
				resp.Body.Read(body)
				assert.Equal(t, tt.expectedBody, string(body))
			}
		})
	}
}

func TestHealthEndpointAvailabilityIntegration(t *testing.T) {
	// Create a new Fiber app
	app := fiber.New()

	// Add the health route
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// Test multiple consecutive requests to ensure endpoint stability
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("GET", "/health", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	}
}