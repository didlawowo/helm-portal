// internal/api/handlers/chart_handlers.go

package handlers

import (
	config "helm-portal/config"

	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
)

// ChartHandler manages chart operations
type ConfigHandler struct {
	log    *logrus.Logger
	config *config.Config
}

func NewConfigHandler(config *config.Config, logger *logrus.Logger) *ConfigHandler {

	return &ConfigHandler{
		config: config,
		log:    logger,
	}
}

func (h *ConfigHandler) GetConfig(c *fiber.Ctx) error {
	return c.JSON(h.config)
}
