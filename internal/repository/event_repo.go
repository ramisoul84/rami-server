package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/ramisoul84/rami-server/internal/domain"
)

type EventRepository interface {
	InsertEvent(ctx context.Context, event *domain.Event) error
	GetSummary(ctx context.Context, dateRange domain.DateRange) (*domain.Summary, error)
	GetTodayStats(ctx context.Context) (*domain.DayStat, error)
	GetTopPages(ctx context.Context, dateRange domain.DateRange, limit int) ([]domain.PageStat, error)
	GetTopReferrers(ctx context.Context, dateRange domain.DateRange, limit int) ([]domain.ReferrerStat, error)
	GetTopCountries(ctx context.Context, dateRange domain.DateRange, limit int) ([]domain.CountryStat, error)
	GetVisitorsByDay(ctx context.Context, days int) ([]domain.DayStat, error)
	GetRecentVisits(ctx context.Context, limit int) ([]domain.Visit, error)
	GetLiveVisitors(ctx context.Context, windowSeconds int) (int64, error)
	ExportEvents(ctx context.Context, dateRange domain.DateRange, limit int) ([]domain.ExportRow, error)
}

type eventRepository struct {
	db *sqlx.DB
}

func NewEventRepository(db *sqlx.DB) EventRepository {
	return &eventRepository{db: db}
}

// ---------------------------------------------------------------------------
// INSERT
// ---------------------------------------------------------------------------

func (r *eventRepository) InsertEvent(ctx context.Context, event *domain.Event) error {
	const query = `
		INSERT INTO events (
			session_id, visitor_id, event_type, page, page_title,
			referrer, language, timezone, viewport, screen, user_agent,
			traffic_source, time_spent_seconds, engaged_time_seconds,
			section, selector, label, custom_name, custom_data,
			ip_address, country, city
		) VALUES (
			:session_id, :visitor_id, :event_type, :page, :page_title,
			:referrer, :language, :timezone, :viewport, :screen, :user_agent,
			:traffic_source, :time_spent_seconds, :engaged_time_seconds,
			:section, :selector, :label, :custom_name, :custom_data,
			:ip_address, :country, :city
		)
	`
	if _, err := r.db.NamedExecContext(ctx, query, event); err != nil {
		return fmt.Errorf("repo: insert event: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// SUMMARY
// ---------------------------------------------------------------------------

func (r *eventRepository) GetSummary(ctx context.Context, dateRange domain.DateRange) (*domain.Summary, error) {
	query := `
		SELECT
			COUNT(DISTINCT visitor_id)                              AS total_visitors,
			COUNT(DISTINCT session_id)                              AS total_sessions,
			COUNT(*) FILTER (WHERE event_type = 'pageview')         AS total_page_views
		FROM events
	`
	args := []interface{}{}

	if !dateRange.IsZero() {
		query += ` WHERE created_at >= $1 AND created_at < $2`
		args = append(args, dateRange.From, dateRange.To.Add(24*time.Hour))
	}

	var s domain.Summary
	if err := r.db.GetContext(ctx, &s, query, args...); err != nil {
		return nil, fmt.Errorf("repo: get summary: %w", err)
	}
	return &s, nil
}

// ---------------------------------------------------------------------------
// TODAY
// ---------------------------------------------------------------------------

func (r *eventRepository) GetTodayStats(ctx context.Context) (*domain.DayStat, error) {
	const query = `
		SELECT
			CURRENT_DATE::text                                      AS date,
			COUNT(DISTINCT visitor_id)                              AS visitors,
			COUNT(*) FILTER (WHERE event_type = 'pageview')         AS page_views
		FROM events
		WHERE created_at::date = CURRENT_DATE
	`
	var s domain.DayStat
	if err := r.db.GetContext(ctx, &s, query); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &domain.DayStat{
				Date:      time.Now().Format("2006-01-02"),
				Visitors:  0,
				PageViews: 0,
			}, nil
		}
		return nil, fmt.Errorf("repo: get today: %w", err)
	}
	return &s, nil
}

// ---------------------------------------------------------------------------
// TOP PAGES
// ---------------------------------------------------------------------------

func (r *eventRepository) GetTopPages(ctx context.Context, dateRange domain.DateRange, limit int) ([]domain.PageStat, error) {
	query := `
		SELECT page, COUNT(*) AS count
		FROM events
		WHERE event_type = 'pageview' AND page IS NOT NULL AND page != ''
	`
	args := []interface{}{}

	if !dateRange.IsZero() {
		query += ` AND created_at >= $1 AND created_at < $2`
		args = append(args, dateRange.From, dateRange.To.Add(24*time.Hour))
	}

	query += fmt.Sprintf(`
		GROUP BY page
		ORDER BY count DESC
		LIMIT %d
	`, limit)

	var pages []domain.PageStat
	if err := r.db.SelectContext(ctx, &pages, query, args...); err != nil {
		return nil, fmt.Errorf("repo: top pages: %w", err)
	}
	return pages, nil
}

// ---------------------------------------------------------------------------
// TOP REFERRERS
// ---------------------------------------------------------------------------

func (r *eventRepository) GetTopReferrers(ctx context.Context, dateRange domain.DateRange, limit int) ([]domain.ReferrerStat, error) {
	query := `
		SELECT COALESCE(NULLIF(referrer, ''), 'direct') AS referrer,
		       COUNT(*) AS count
		FROM events
		WHERE event_type = 'pageview'
	`
	args := []interface{}{}

	if !dateRange.IsZero() {
		query += ` AND created_at >= $1 AND created_at < $2`
		args = append(args, dateRange.From, dateRange.To.Add(24*time.Hour))
	}

	query += fmt.Sprintf(`
		GROUP BY referrer
		ORDER BY count DESC
		LIMIT %d
	`, limit)

	var refs []domain.ReferrerStat
	if err := r.db.SelectContext(ctx, &refs, query, args...); err != nil {
		return nil, fmt.Errorf("repo: top referrers: %w", err)
	}
	return refs, nil
}

// ---------------------------------------------------------------------------
// TOP COUNTRIES
// ---------------------------------------------------------------------------

func (r *eventRepository) GetTopCountries(ctx context.Context, dateRange domain.DateRange, limit int) ([]domain.CountryStat, error) {
	query := `
		SELECT country, COUNT(DISTINCT visitor_id) AS count
		FROM events
		WHERE country IS NOT NULL AND country != ''
		  AND event_type = 'pageview'
	`
	args := []interface{}{}

	if !dateRange.IsZero() {
		query += ` AND created_at >= $1 AND created_at < $2`
		args = append(args, dateRange.From, dateRange.To.Add(24*time.Hour))
	}

	query += fmt.Sprintf(`
		GROUP BY country
		ORDER BY count DESC
		LIMIT %d
	`, limit)

	var stats []domain.CountryStat
	if err := r.db.SelectContext(ctx, &stats, query, args...); err != nil {
		return nil, fmt.Errorf("repo: top countries: %w", err)
	}
	return stats, nil
}

// ---------------------------------------------------------------------------
// VISITORS BY DAY
// ---------------------------------------------------------------------------

func (r *eventRepository) GetVisitorsByDay(ctx context.Context, days int) ([]domain.DayStat, error) {
	const query = `
		SELECT
			created_at::date::text                              AS date,
			COUNT(DISTINCT visitor_id)                          AS visitors,
			COUNT(*) FILTER (WHERE event_type = 'pageview')     AS page_views
		FROM events
		WHERE created_at >= NOW() - ($1 || ' days')::interval
		GROUP BY created_at::date
		ORDER BY created_at::date
	`
	var stats []domain.DayStat
	if err := r.db.SelectContext(ctx, &stats, query, days); err != nil {
		return nil, fmt.Errorf("repo: visitors by day: %w", err)
	}
	return stats, nil
}

// ---------------------------------------------------------------------------
// RECENT VISITS
// ---------------------------------------------------------------------------

func (r *eventRepository) GetRecentVisits(ctx context.Context, limit int) ([]domain.Visit, error) {
	const query = `
		SELECT
			page,
			COALESCE(NULLIF(referrer, ''), 'direct')    AS referrer,
			COALESCE(country, '')                       AS country,
			COALESCE(city, '')                          AS city,
			ip_address,
			created_at
		FROM events
		WHERE event_type = 'pageview'
		ORDER BY created_at DESC
		LIMIT $1
	`
	var visits []domain.Visit
	if err := r.db.SelectContext(ctx, &visits, query, limit); err != nil {
		return nil, fmt.Errorf("repo: recent visits: %w", err)
	}
	return visits, nil
}

// ---------------------------------------------------------------------------
// LIVE VISITORS
// ---------------------------------------------------------------------------

func (r *eventRepository) GetLiveVisitors(ctx context.Context, windowSeconds int) (int64, error) {
	const query = `
		SELECT COUNT(DISTINCT visitor_id)
		FROM events
		WHERE created_at > NOW() - ($1 || ' seconds')::interval
	`
	var count int64
	if err := r.db.GetContext(ctx, &count, query, windowSeconds); err != nil {
		return 0, fmt.Errorf("repo: live visitors: %w", err)
	}
	return count, nil
}

// ---------------------------------------------------------------------------
// EXPORT
// ---------------------------------------------------------------------------

func (r *eventRepository) ExportEvents(ctx context.Context, dateRange domain.DateRange, limit int) ([]domain.ExportRow, error) {
	query := `
		SELECT
			created_at, event_type, page, page_title, referrer,
			COALESCE(country, '')    AS country,
			COALESCE(city, '')       AS city,
			language, viewport, user_agent,
			time_spent_seconds, engaged_time_seconds
		FROM events
	`
	args := []interface{}{}

	if !dateRange.IsZero() {
		query += ` WHERE created_at >= $1 AND created_at < $2`
		args = append(args, dateRange.From, dateRange.To.Add(24*time.Hour))
	}

	query += fmt.Sprintf(`
		ORDER BY created_at DESC
		LIMIT %d
	`, limit)

	var rows []domain.ExportRow
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("repo: export events: %w", err)
	}
	return rows, nil
}
