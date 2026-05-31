package handler

import (
	"encoding/csv"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/ramisoul84/rami-server/internal/domain"
	"github.com/ramisoul84/rami-server/internal/service"
)

// AnalyticsHandler serves /api/v1/admin/analytics/* for the admin panel.
type AnalyticsHandler struct {
	analytics service.AnalyticsService
}

func NewAnalyticsHandler(analytics service.AnalyticsService) *AnalyticsHandler {
	return &AnalyticsHandler{analytics: analytics}
}

func (h *AnalyticsHandler) Overview(c *fiber.Ctx) error {
	data, err := h.analytics.Overview(c.UserContext(), queryRange(c))
	if err != nil {
		return analyticsError(err, "failed to load overview")
	}
	return c.JSON(data)
}

func (h *AnalyticsHandler) Live(c *fiber.Ctx) error {
	data, err := h.analytics.Live(c.UserContext())
	if err != nil {
		return analyticsError(err, "failed to load live visitors")
	}
	return c.JSON(data)
}

func (h *AnalyticsHandler) Sessions(c *fiber.Ctx) error {
	q := domain.SessionsQuery{
		AnalyticsRange: queryRange(c),
		Limit:          c.QueryInt("limit", 25),
		Offset:         c.QueryInt("offset", 0),
		Converted:      c.QueryBool("converted", false),
		Country:        c.Query("country"),
		Source:         c.Query("source"),
	}
	data, err := h.analytics.Sessions(c.UserContext(), q)
	if err != nil {
		return analyticsError(err, "failed to load sessions")
	}
	return c.JSON(data)
}

func (h *AnalyticsHandler) Session(c *fiber.Ctx) error {
	data, err := h.analytics.Session(c.UserContext(), c.Params("id"))
	if err != nil {
		return analyticsError(err, "failed to load session")
	}
	return c.JSON(data)
}

// ExportCSV streams the raw events of a date range as a spreadsheet-friendly CSV.
func (h *AnalyticsHandler) ExportCSV(c *fiber.Ctx) error {
	r := queryRange(c)
	rows, err := h.analytics.Export(c.UserContext(), r)
	if err != nil {
		return analyticsError(err, "failed to export events")
	}

	c.Set(fiber.HeaderContentType, "text/csv; charset=utf-8")
	c.Set(fiber.HeaderContentDisposition, fmt.Sprintf(`attachment; filename="analytics-%s_%s.csv"`, r.From, r.To))

	// UTF-8 BOM so Excel detects the encoding.
	if _, err := c.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
		return err
	}
	w := csv.NewWriter(c)
	_ = w.Write([]string{
		"timestamp", "event_type", "page", "page_title", "referrer",
		"country", "city", "language", "viewport", "user_agent",
		"time_spent_seconds", "engaged_time_seconds",
	})
	for _, row := range rows {
		_ = w.Write([]string{
			row.CreatedAt.Format(time.RFC3339), row.EventType, row.Page, row.PageTitle, row.Referrer,
			row.Country, row.City, row.Language, row.Viewport, row.UserAgent,
			optionalInt(row.TimeSpentSeconds), optionalInt(row.EngagedTimeSeconds),
		})
	}
	w.Flush()
	return w.Error()
}

func queryRange(c *fiber.Ctx) domain.AnalyticsRange {
	return domain.AnalyticsRange{From: c.Query("from"), To: c.Query("to")}
}

func optionalInt(v *int) string {
	if v == nil {
		return ""
	}
	return strconv.Itoa(*v)
}

// analyticsError maps service errors to HTTP errors; unknown ones become a logged 500.
func analyticsError(err error, message string) error {
	switch {
	case errors.Is(err, service.ErrInvalidRange):
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrSessionNotFound):
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	default:
		return internal(message, err)
	}
}
