package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ramisoul84/rami-server/internal/domain"
	"github.com/ramisoul84/rami-server/internal/repository"
	"github.com/ramisoul84/rami-server/pkg/logger"
)

const (
	dayLayout         = "2006-01-02"
	liveWindowSeconds = 45
	// maxRangeDays caps how much data a single overview scans.
	maxRangeDays = 366
	// maxExportRows caps one CSV download.
	maxExportRows = 100_000
)

var (
	ErrInvalidRange    = errors.New("analytics: 'from' and 'to' must be YYYY-MM-DD with from <= to, at most 366 days apart")
	ErrSessionNotFound = errors.New("analytics: session not found")
)

// funnelSections is the order of sections on the home page (trackSection ids).
var funnelSections = []string{"hero", "about", "skills", "experience", "projects", "contact"}

type AnalyticsService interface {
	Overview(ctx context.Context, r domain.AnalyticsRange) (*domain.Overview, error)
	Live(ctx context.Context) (*domain.LiveSnapshot, error)
	Sessions(ctx context.Context, q domain.SessionsQuery) (*domain.SessionsPage, error)
	Session(ctx context.Context, id string) (*domain.SessionDetail, error)
	Export(ctx context.Context, r domain.AnalyticsRange) ([]domain.ExportRow, error)
}

type analyticsService struct {
	repo repository.AnalyticsRepository
	log  *logger.Logger
}

func NewAnalyticsService(repo repository.AnalyticsRepository, log *logger.Logger) AnalyticsService {
	return &analyticsService{repo: repo, log: log}
}

// ValidateRange checks an inclusive day range and returns its length in days.
func ValidateRange(r domain.AnalyticsRange) (int, error) {
	from, err1 := time.Parse(dayLayout, r.From)
	to, err2 := time.Parse(dayLayout, r.To)
	if err1 != nil || err2 != nil || from.After(to) {
		return 0, ErrInvalidRange
	}
	days := int(to.Sub(from).Hours()/24) + 1
	if days > maxRangeDays {
		return 0, ErrInvalidRange
	}
	return days, nil
}

// previousRange is the same number of days ending the day before r.From.
func previousRange(r domain.AnalyticsRange, days int) domain.AnalyticsRange {
	from, _ := time.Parse(dayLayout, r.From)
	prevTo := from.AddDate(0, 0, -1)
	prevFrom := prevTo.AddDate(0, 0, -(days - 1))
	return domain.AnalyticsRange{From: prevFrom.Format(dayLayout), To: prevTo.Format(dayLayout)}
}

func (s *analyticsService) Overview(ctx context.Context, r domain.AnalyticsRange) (*domain.Overview, error) {
	days, err := ValidateRange(r)
	if err != nil {
		return nil, err
	}
	prev := previousRange(r, days)

	cur, err := s.repo.KpiTotals(ctx, r)
	if err != nil {
		return nil, err
	}
	old, err := s.repo.KpiTotals(ctx, prev)
	if err != nil {
		return nil, err
	}

	out := &domain.Overview{
		Range: domain.OverviewRange{AnalyticsRange: r, PreviousFrom: prev.From, PreviousTo: prev.To},
		Kpis: domain.OverviewKpis{
			Visitors:          domain.Kpi{Current: cur.Visitors, Previous: old.Visitors},
			Sessions:          domain.Kpi{Current: cur.Sessions, Previous: old.Sessions},
			Pageviews:         domain.Kpi{Current: cur.Pageviews, Previous: old.Pageviews},
			AvgEngagedSeconds: domain.Kpi{Current: cur.AvgEngagedSeconds, Previous: old.AvgEngagedSeconds},
			BounceRate:        domain.Kpi{Current: cur.BounceRate, Previous: old.BounceRate},
			ReturningRate:     domain.Kpi{Current: cur.ReturningRate, Previous: old.ReturningRate},
			ResumeDownloads:   domain.Kpi{Current: cur.ResumeDownloads, Previous: old.ResumeDownloads},
			ContactClicks:     domain.Kpi{Current: cur.ContactClicks, Previous: old.ContactClicks},
			ConversionRate:    domain.Kpi{Current: cur.ConversionRate, Previous: old.ConversionRate},
		},
	}

	// Each step fills one field; the first failure aborts the request.
	steps := []struct {
		name string
		run  func() error
	}{
		{"timeseries", func() (err error) { out.Timeseries, err = s.repo.Timeseries(ctx, r); return }},
		{"sources", func() (err error) { out.Sources, err = s.repo.Sources(ctx, r); return }},
		{"campaigns", func() (err error) { out.Campaigns, err = s.repo.Campaigns(ctx, r); return }},
		{"locations", func() (err error) { out.Locations, err = s.repo.Locations(ctx, r); return }},
		{"devices", func() (err error) { out.Devices, err = s.repo.SessionBreakdown(ctx, r, "device"); return }},
		{"browsers", func() (err error) { out.Browsers, err = s.repo.SessionBreakdown(ctx, r, "browser"); return }},
		{"os", func() (err error) { out.OS, err = s.repo.SessionBreakdown(ctx, r, "os"); return }},
		{"site languages", func() (err error) {
			out.SiteLanguages, err = s.repo.SessionBreakdown(ctx, r, "site_language")
			return
		}},
		{"browser languages", func() (err error) { out.BrowserLanguages, err = s.repo.BrowserLanguages(ctx, r); return }},
		{"funnel", func() (err error) { out.Funnel, err = s.repo.Funnel(ctx, r, funnelSections); return }},
		{"scroll depth", func() (err error) { out.ScrollDepth, err = s.repo.ScrollDepth(ctx, r); return }},
		{"resume by language", func() (err error) {
			out.ResumeByLanguage, err = s.repo.CustomEventBreakdown(ctx, r, "resume_download", "lang")
			return
		}},
		{"contact by channel", func() (err error) {
			out.ContactByChannel, err = s.repo.CustomEventBreakdown(ctx, r, "contact_click", "channel")
			return
		}},
		{"interactions", func() (err error) { out.Interactions, err = s.repo.Interactions(ctx, r); return }},
		{"top pages", func() (err error) { out.TopPages, err = s.repo.TopPages(ctx, r); return }},
		{"heatmap", func() (err error) { out.Heatmap, err = s.repo.Heatmap(ctx, r); return }},
		{"web vitals", func() error {
			v, err := s.repo.WebVitals(ctx, r)
			if err == nil {
				out.WebVitals = *v
			}
			return err
		}},
	}
	for _, step := range steps {
		if err := step.run(); err != nil {
			return nil, fmt.Errorf("analytics: overview %s: %w", step.name, err)
		}
	}
	return out, nil
}

func (s *analyticsService) Live(ctx context.Context) (*domain.LiveSnapshot, error) {
	pages, err := s.repo.Live(ctx, liveWindowSeconds)
	if err != nil {
		return nil, fmt.Errorf("analytics: live: %w", err)
	}
	var total int64
	for _, p := range pages {
		total += p.Visitors
	}
	return &domain.LiveSnapshot{Visitors: total, WindowSeconds: liveWindowSeconds, Pages: pages}, nil
}

func (s *analyticsService) Sessions(ctx context.Context, q domain.SessionsQuery) (*domain.SessionsPage, error) {
	if _, err := ValidateRange(q.AnalyticsRange); err != nil {
		return nil, err
	}
	q.Limit = clamp(q.Limit, 1, 100, 25)
	if q.Offset < 0 {
		q.Offset = 0
	}

	page, err := s.repo.Sessions(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("analytics: sessions: %w", err)
	}
	return page, nil
}

func (s *analyticsService) Session(ctx context.Context, id string) (*domain.SessionDetail, error) {
	detail, err := s.repo.Session(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrSessionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("analytics: session: %w", err)
	}
	return detail, nil
}

func (s *analyticsService) Export(ctx context.Context, r domain.AnalyticsRange) ([]domain.ExportRow, error) {
	if _, err := ValidateRange(r); err != nil {
		return nil, err
	}
	return s.repo.Export(ctx, r, maxExportRows)
}

// clamp returns v limited to [lo, hi], or fallback when v is not positive.
func clamp(v, lo, hi, fallback int) int {
	if v <= 0 {
		return fallback
	}
	return min(max(v, lo), hi)
}
