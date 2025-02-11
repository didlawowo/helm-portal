// pkg/middlewares/https.go
package middleware

import (
    "github.com/gofiber/fiber/v2"
    "github.com/sirupsen/logrus"
)

func HTTPSRedirect(log *logrus.Logger) fiber.Handler {
    return func(c *fiber.Ctx) error {
        log.WithFields(logrus.Fields{
            "protocol": c.Protocol(),
            "secure":   c.Secure(),
            "path":     c.Path(),
            "method":   c.Method(),
        }).Debug("🔐 HTTPS Check")

        if !c.Secure() {
            httpsURL := "https://" + c.Hostname() + c.OriginalURL()
            log.WithField("redirect_to", httpsURL).Info("Redirecting to HTTPS")
            return c.Redirect(httpsURL)
        }
        return c.Next()
    }
}