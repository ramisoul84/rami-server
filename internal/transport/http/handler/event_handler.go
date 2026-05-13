package handler

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"github.com/ramisoul84/rami-server/internal/domain"
	"github.com/ramisoul84/rami-server/internal/service"
	"github.com/ramisoul84/rami-server/internal/transport/http/middleware"
)

type EventHandler struct {
	events service.EventService
}

func NewEventHandler(events service.EventService) *EventHandler {
	return &EventHandler{events: events}
}

// Track stores one analytics event. Storage failures are logged by the service
// but still answered with 204: the tracker is fire-and-forget and must not retry.
func (h *EventHandler) Track(c *fiber.Ctx) error {
	var event domain.IncomingEvent
	if err := c.BodyParser(&event); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid payload")
	}
	if err := h.events.TrackEvent(c.UserContext(), &event, middleware.ClientIP(c)); errors.Is(err, service.ErrInvalidEvent) {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	return c.SendStatus(fiber.StatusNoContent)
}
