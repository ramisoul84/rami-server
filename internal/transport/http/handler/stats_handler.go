package handler

import (
	"encoding/csv"
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/ramisoul84/rami-server/internal/domain"
	"github.com/ramisoul84/rami-server/internal/service"
)

type StatsHandler struct {
	stats service.StatsService
}

func NewStatsHandler(stats service.StatsService) *StatsHandler {
	return &StatsHandler{stats: stats}
}

func (h *StatsHandler) Stats(c *fiber.Ctx) error {
	dateRange, err := parseDateRange(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	data, err := h.stats.Overview(c.Context(), dateRange)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to load stats")
	}
	return c.JSON(data)
}

func (h *StatsHandler) RecentVisits(c *fiber.Ctx) error {
	limit := c.QueryInt("limit", 10)
	visits, err := h.stats.RecentVisits(c.Context(), limit)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to load visits")
	}
	return c.JSON(fiber.Map{"visits": visits})
}

func (h *StatsHandler) LiveVisitors(c *fiber.Ctx) error {
	count, err := h.stats.LiveVisitors(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to load live visitors")
	}
	return c.JSON(fiber.Map{
		"live_visitors":  count,
		"window_seconds": 45,
	})
}

func (h *StatsHandler) TopCountries(c *fiber.Ctx) error {
	dateRange, err := parseDateRange(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	limit := c.QueryInt("limit", 10)
	countries, err := h.stats.TopCountries(c.Context(), dateRange, limit)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to load countries")
	}
	return c.JSON(fiber.Map{"countries": countries})
}

func (h *StatsHandler) ExportCSV(c *fiber.Ctx) error {
	dateRange, err := parseDateRange(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	rows, err := h.stats.ExportRows(c.Context(), dateRange)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to export")
	}

	c.Set("Content-Type", "text/csv; charset=utf-8")
	c.Set("Content-Disposition",
		fmt.Sprintf(`attachment; filename="analytics-%s.csv"`,
			time.Now().Format("2006-01-02")))

	// UTF-8 BOM so Excel reads it correctly.
	if _, err := c.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
		return err
	}

	w := csv.NewWriter(c)
	defer w.Flush()

	_ = w.Write([]string{
		"timestamp", "event_type", "page", "page_title", "referrer",
		"country", "city", "language", "viewport", "user_agent",
		"time_spent_seconds", "engaged_time_seconds",
	})

	for _, r := range rows {
		ts := ""
		spent := ""
		engaged := ""

		if !r.CreatedAt.IsZero() {
			ts = r.CreatedAt.Format(time.RFC3339)
		}
		if r.TimeSpentSeconds != nil {
			spent = strconv.Itoa(*r.TimeSpentSeconds)
		}
		if r.EngagedTimeSeconds != nil {
			engaged = strconv.Itoa(*r.EngagedTimeSeconds)
		}

		if err := w.Write([]string{
			ts, r.EventType, r.Page, r.PageTitle, r.Referrer,
			r.Country, r.City, r.Language, r.Viewport, r.UserAgent,
			spent, engaged,
		}); err != nil {
			return err
		}
	}

	return nil
}

// ---------------------------------------------------------------------------
// HELPERS
// ---------------------------------------------------------------------------

func parseDateRange(c *fiber.Ctx) (domain.DateRange, error) {
	from := c.Query("from")
	to := c.Query("to")

	if from == "" && to == "" {
		return domain.DateRange{}, nil
	}

	if from == "" || to == "" {
		return domain.DateRange{}, fmt.Errorf("both 'from' and 'to' must be provided")
	}

	f, err := time.Parse("2006-01-02", from)
	if err != nil {
		return domain.DateRange{}, fmt.Errorf("invalid 'from' date, expected YYYY-MM-DD")
	}

	t, err := time.Parse("2006-01-02", to)
	if err != nil {
		return domain.DateRange{}, fmt.Errorf("invalid 'to' date, expected YYYY-MM-DD")
	}

	if f.After(t) {
		return domain.DateRange{}, fmt.Errorf("'from' must be before or equal to 'to'")
	}

	return domain.DateRange{From: f, To: t}, nil
}
