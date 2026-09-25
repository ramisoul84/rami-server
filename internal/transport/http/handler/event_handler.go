package handler

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/ramisoul84/rami-server/internal/domain"
	"github.com/ramisoul84/rami-server/internal/service"
)

type EventHandler struct {
	events service.EventService
}

func NewEventHandler(events service.EventService) *EventHandler {
	return &EventHandler{events: events}
}

func (h *EventHandler) Track(c *fiber.Ctx) error {
	var event domain.IncomingEvent
	if err := c.BodyParser(&event); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid payload")
	}
	if event.SessionID == "" || event.VisitorID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "session_id and visitor_id are required")
	}

	_ = h.events.TrackEvent(c.Context(), &event, clientIP(c))

	return c.SendStatus(fiber.StatusNoContent)
}

func clientIP(c *fiber.Ctx) string {
	if forwarded := c.Get("X-Forwarded-For"); forwarded != "" {
		if i := strings.IndexByte(forwarded, ','); i >= 0 {
			return strings.TrimSpace(forwarded[:i])
		}
		return strings.TrimSpace(forwarded)
	}
	return c.IP()
}
