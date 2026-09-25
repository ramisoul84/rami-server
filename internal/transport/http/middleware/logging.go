package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/ramisoul84/rami-server/pkg/logger"
)

func Logging(log *logger.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		latency := time.Since(start).Milliseconds()
		requestID, _ := c.Locals("request_id").(string)

		// If the handler returned an error, extract its status code.
		// Otherwise use what the response has already written.
		status := c.Response().StatusCode()
		if err != nil {
			if e, ok := err.(*fiber.Error); ok {
				status = e.Code
			} else {
				status = fiber.StatusInternalServerError
			}
		}

		attrs := []any{
			"method", c.Method(),
			"path", c.Path(),
			"status", status,
			"latency_ms", latency,
			"request_id", requestID,
		}
		if err != nil {
			attrs = append(attrs, "err", err.Error())
		}

		switch {
		case status >= 500:
			log.Error("request", attrs...)
		case status >= 400:
			log.Warn("request", attrs...)
		default:
			log.Info("request", attrs...)
		}
		return err
	}
}
