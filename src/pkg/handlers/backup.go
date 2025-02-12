package handlers

import (
	services "helm-portal/pkg/services"

	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
)

type BackupHandler struct {
	backupService *services.BackupService
	log           *logrus.Logger
}

func NewBackupHandler(backupService *services.BackupService, log *logrus.Logger) *BackupHandler {
	return &BackupHandler{
		backupService: backupService,
		log:           log,
	}
}

func (h *BackupHandler) HandleBackup(c *fiber.Ctx) error {
	if err := h.backupService.Backup(); err != nil {
		h.log.WithError(err).Error("❌ Backup failed")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	h.log.Info("✅ Backup successful")
	return c.JSON(fiber.Map{
		"message": "Backup completed successfully",
	})
}

func (h *BackupHandler) HandleRestore(c *fiber.Ctx) error {
	if err := h.backupService.Restore(); err != nil {
		h.log.WithError(err).Error("❌ Restore failed")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	h.log.Info("✅ Restore successful")
	return c.JSON(fiber.Map{
		"message": "Restore completed successfully",
	})
}
