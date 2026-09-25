package service

import (
	"context"
	"fmt"

	"github.com/ramisoul84/rami-server/internal/domain"
	"github.com/ramisoul84/rami-server/internal/repository"
	"github.com/ramisoul84/rami-server/pkg/logger"
)

type StatsService interface {
	Overview(ctx context.Context, dateRange domain.DateRange) (*domain.DashboardStats, error)
	Summary(ctx context.Context, dateRange domain.DateRange) (*domain.Summary, error)
	Today(ctx context.Context) (*domain.DayStat, error)
	RecentVisits(ctx context.Context, limit int) ([]domain.Visit, error)
	TopPages(ctx context.Context, dateRange domain.DateRange, limit int) ([]domain.PageStat, error)
	TopCountries(ctx context.Context, dateRange domain.DateRange, limit int) ([]domain.CountryStat, error)
	LiveVisitors(ctx context.Context) (int64, error)
	ExportRows(ctx context.Context, dateRange domain.DateRange) ([]domain.ExportRow, error)
}

type statsService struct {
	repo repository.EventRepository
	log  *logger.Logger
}

func NewStatsService(repo repository.EventRepository, log *logger.Logger) StatsService {
	return &statsService{repo: repo, log: log}
}

func (s *statsService) Overview(ctx context.Context, dateRange domain.DateRange) (*domain.DashboardStats, error) {
	summary, err := s.repo.GetSummary(ctx, dateRange)
	if err != nil {
		return nil, fmt.Errorf("stats: summary: %w", err)
	}
	today, err := s.repo.GetTodayStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("stats: today: %w", err)
	}
	pages, err := s.repo.GetTopPages(ctx, dateRange, 10)
	if err != nil {
		return nil, fmt.Errorf("stats: top pages: %w", err)
	}
	referrers, err := s.repo.GetTopReferrers(ctx, dateRange, 10)
	if err != nil {
		return nil, fmt.Errorf("stats: top referrers: %w", err)
	}
	byDay, err := s.repo.GetVisitorsByDay(ctx, 30)
	if err != nil {
		return nil, fmt.Errorf("stats: visitors by day: %w", err)
	}
	return &domain.DashboardStats{
		Summary:       summary,
		Today:         today,
		TopPages:      pages,
		TopReferrers:  referrers,
		VisitorsByDay: byDay,
	}, nil
}

func (s *statsService) Summary(ctx context.Context, dateRange domain.DateRange) (*domain.Summary, error) {
	v, err := s.repo.GetSummary(ctx, dateRange)
	if err != nil {
		return nil, fmt.Errorf("stats: summary: %w", err)
	}
	return v, nil
}

func (s *statsService) Today(ctx context.Context) (*domain.DayStat, error) {
	v, err := s.repo.GetTodayStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("stats: today: %w", err)
	}
	return v, nil
}

func (s *statsService) RecentVisits(ctx context.Context, limit int) ([]domain.Visit, error) {
	v, err := s.repo.GetRecentVisits(ctx, clamp(limit, 1, 100, 10))
	if err != nil {
		return nil, fmt.Errorf("stats: recent visits: %w", err)
	}
	return v, nil
}

func (s *statsService) TopPages(ctx context.Context, dateRange domain.DateRange, limit int) ([]domain.PageStat, error) {
	v, err := s.repo.GetTopPages(ctx, dateRange, clamp(limit, 1, 100, 10))
	if err != nil {
		return nil, fmt.Errorf("stats: top pages: %w", err)
	}
	return v, nil
}

func (s *statsService) TopCountries(ctx context.Context, dateRange domain.DateRange, limit int) ([]domain.CountryStat, error) {
	v, err := s.repo.GetTopCountries(ctx, dateRange, clamp(limit, 1, 100, 10))
	if err != nil {
		return nil, fmt.Errorf("stats: top countries: %w", err)
	}
	return v, nil
}

func (s *statsService) LiveVisitors(ctx context.Context) (int64, error) {
	const windowSeconds = 45
	v, err := s.repo.GetLiveVisitors(ctx, windowSeconds)
	if err != nil {
		return 0, fmt.Errorf("stats: live visitors: %w", err)
	}
	return v, nil
}

func (s *statsService) ExportRows(ctx context.Context, dateRange domain.DateRange) ([]domain.ExportRow, error) {
	v, err := s.repo.ExportEvents(ctx, dateRange, 10000)
	if err != nil {
		return nil, fmt.Errorf("stats: export rows: %w", err)
	}
	return v, nil
}

func clamp(v, min, max, fallback int) int {
	if v < min || v > max {
		return fallback
	}
	return v
}
