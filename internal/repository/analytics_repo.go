package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"github.com/ramisoul84/rami-server/internal/domain"
)

// ErrNotFound is returned when a requested row does not exist.
var ErrNotFound = errors.New("repo: not found")

// AnalyticsRepository backs /api/v1/admin/analytics/*. Definitions of every
// metric are in client/docs/ANALYTICS_API.md.
type AnalyticsRepository interface {
	KpiTotals(ctx context.Context, r domain.AnalyticsRange) (*domain.KpiTotals, error)
	Timeseries(ctx context.Context, r domain.AnalyticsRange) ([]domain.TimeseriesPoint, error)
	Sources(ctx context.Context, r domain.AnalyticsRange) ([]domain.SourceRow, error)
	Campaigns(ctx context.Context, r domain.AnalyticsRange) ([]domain.CampaignRow, error)
	Locations(ctx context.Context, r domain.AnalyticsRange) ([]domain.LocationRow, error)
	// SessionBreakdown counts sessions by a column of the session's first event
	// (device, browser, os, site_language).
	SessionBreakdown(ctx context.Context, r domain.AnalyticsRange, column string) ([]domain.CountRow, error)
	BrowserLanguages(ctx context.Context, r domain.AnalyticsRange) ([]domain.CountRow, error)
	Funnel(ctx context.Context, r domain.AnalyticsRange, sections []string) ([]domain.FunnelStep, error)
	ScrollDepth(ctx context.Context, r domain.AnalyticsRange) ([]domain.CountRow, error)
	// CustomEventBreakdown groups one custom event by a key of its data (e.g. resume_download by "lang").
	CustomEventBreakdown(ctx context.Context, r domain.AnalyticsRange, name, dataKey string) ([]domain.CountRow, error)
	Interactions(ctx context.Context, r domain.AnalyticsRange) ([]domain.CountRow, error)
	TopPages(ctx context.Context, r domain.AnalyticsRange) ([]domain.PageRow, error)
	Heatmap(ctx context.Context, r domain.AnalyticsRange) ([]domain.HeatmapCell, error)
	WebVitals(ctx context.Context, r domain.AnalyticsRange) (*domain.WebVitalsSummary, error)
	Live(ctx context.Context, windowSeconds int) ([]domain.LivePage, error)
	Sessions(ctx context.Context, q domain.SessionsQuery) (*domain.SessionsPage, error)
	Session(ctx context.Context, id string) (*domain.SessionDetail, error)
	// Export returns raw events for the CSV download, newest first.
	Export(ctx context.Context, r domain.AnalyticsRange, limit int) ([]domain.ExportRow, error)
}

type analyticsRepository struct {
	db *sqlx.DB
}

func NewAnalyticsRepository(db *sqlx.DB) AnalyticsRepository {
	return &analyticsRepository{db: db}
}

// ---------------------------------------------------------------------------
// SHARED CTEs
// ---------------------------------------------------------------------------

// inRange limits events to [from, to] by calendar day in the connection's
// TimeZone ($1 / $2 are 'YYYY-MM-DD' strings).
const inRange = `created_at >= $1::date AND created_at < $2::date + 1`

const conversionNames = `('resume_download', 'contact_click')`

// sessionCTEs builds, for the events matching evWhere:
//
//	ev    — the events themselves, bots excluded
//	s     — one row per session: visitor, start/end, pageviews, engaged seconds,
//	        converted, returning
//	first — the session's first event (source, device, location, landing page…)
func sessionCTEs(evWhere string) string {
	return `
	ev AS (
		SELECT * FROM events
		WHERE ` + evWhere + ` AND device IS DISTINCT FROM 'bot'
	),
	sess AS (
		SELECT
			session_id,
			min(visitor_id)                                         AS visitor_id,
			min(created_at)                                         AS started_at,
			max(created_at)                                         AS ended_at,
			count(*) FILTER (WHERE event_type = 'pageview')         AS pageviews,
			coalesce(bool_or(event_type = 'custom' AND custom_name IN ` + conversionNames + `), false) AS converted
		FROM ev
		GROUP BY session_id
	),
	-- Engaged time: the highest engaged_time_seconds reported per page, summed per session.
	eng AS (
		SELECT session_id, sum(m) AS engaged
		FROM (
			SELECT session_id, page, max(engaged_time_seconds) AS m
			FROM ev
			WHERE event_type IN ('leave', 'heartbeat') AND engaged_time_seconds IS NOT NULL
			GROUP BY session_id, page
		) per_page
		GROUP BY session_id
	),
	s AS (
		SELECT
			sess.*,
			coalesce(eng.engaged, 0) AS engaged,
			EXISTS (
				SELECT 1 FROM events o
				WHERE o.visitor_id = sess.visitor_id AND o.created_at < sess.started_at
			) AS is_returning
		FROM sess
		LEFT JOIN eng USING (session_id)
	),
	first AS (
		SELECT DISTINCT ON (session_id)
			session_id, page, referrer, source, traffic_source, country, city,
			device, os, browser, language, site_language
		FROM ev
		ORDER BY session_id, created_at
	)`
}

func rangeArgs(r domain.AnalyticsRange) []any { return []any{r.From, r.To} }

// ---------------------------------------------------------------------------
// OVERVIEW
// ---------------------------------------------------------------------------

func (r *analyticsRepository) KpiTotals(ctx context.Context, rg domain.AnalyticsRange) (*domain.KpiTotals, error) {
	query := `WITH ` + sessionCTEs(inRange) + `
	SELECT
		(SELECT count(DISTINCT visitor_id) FROM ev)::float8                                   AS visitors,
		(SELECT count(*) FROM s)::float8                                                      AS sessions,
		(SELECT count(*) FROM ev WHERE event_type = 'pageview')::float8                       AS pageviews,
		coalesce((SELECT avg(engaged) FROM s), 0)::float8                                     AS avg_engaged_seconds,
		coalesce((SELECT avg((pageviews <= 1 AND engaged < 10)::int) FROM s), 0)::float8      AS bounce_rate,
		coalesce((SELECT avg(is_returning::int) FROM s), 0)::float8                              AS returning_rate,
		(SELECT count(*) FROM ev WHERE event_type = 'custom' AND custom_name = 'resume_download')::float8 AS resume_downloads,
		(SELECT count(*) FROM ev WHERE event_type = 'custom' AND custom_name = 'contact_click')::float8   AS contact_clicks,
		coalesce((SELECT avg(converted::int) FROM s), 0)::float8                              AS conversion_rate`

	var out domain.KpiTotals
	if err := r.db.GetContext(ctx, &out, query, rangeArgs(rg)...); err != nil {
		return nil, fmt.Errorf("repo: kpi totals: %w", err)
	}
	return &out, nil
}

func (r *analyticsRepository) Timeseries(ctx context.Context, rg domain.AnalyticsRange) ([]domain.TimeseriesPoint, error) {
	query := `WITH ` + sessionCTEs(inRange) + `,
	days AS (
		SELECT d::date AS day FROM generate_series($1::date, $2::date, interval '1 day') d
	)
	SELECT
		to_char(days.day, 'YYYY-MM-DD')                                    AS date,
		count(DISTINCT ev.visitor_id)                                      AS visitors,
		count(DISTINCT ev.session_id)                                      AS sessions,
		count(*) FILTER (WHERE ev.event_type = 'pageview')                 AS pageviews,
		count(DISTINCT ev.session_id) FILTER (
			WHERE ev.event_type = 'custom' AND ev.custom_name IN ` + conversionNames + `
		)                                                                  AS conversions
	FROM days
	LEFT JOIN ev ON ev.created_at::date = days.day
	GROUP BY days.day
	ORDER BY days.day`

	return selectRows[domain.TimeseriesPoint](ctx, r.db, "timeseries", query, rangeArgs(rg)...)
}

func (r *analyticsRepository) Sources(ctx context.Context, rg domain.AnalyticsRange) ([]domain.SourceRow, error) {
	query := `WITH ` + sessionCTEs(inRange) + `
	SELECT
		coalesce(f.source, 'direct')       AS source,
		count(DISTINCT s.visitor_id)       AS visitors,
		count(*) FILTER (WHERE s.converted) AS conversions
	FROM s JOIN first f USING (session_id)
	GROUP BY 1
	ORDER BY visitors DESC, source
	LIMIT 20`

	return selectRows[domain.SourceRow](ctx, r.db, "sources", query, rangeArgs(rg)...)
}

func (r *analyticsRepository) Campaigns(ctx context.Context, rg domain.AnalyticsRange) ([]domain.CampaignRow, error) {
	query := `WITH ` + sessionCTEs(inRange) + `
	SELECT
		f.traffic_source->>'campaign'       AS campaign,
		f.traffic_source->>'source'         AS source,
		f.traffic_source->>'medium'         AS medium,
		count(DISTINCT s.visitor_id)        AS visitors,
		count(*) FILTER (WHERE s.converted) AS conversions
	FROM s JOIN first f USING (session_id)
	WHERE coalesce(f.traffic_source->>'campaign', '') <> ''
	GROUP BY 1, 2, 3
	ORDER BY visitors DESC
	LIMIT 20`

	return selectRows[domain.CampaignRow](ctx, r.db, "campaigns", query, rangeArgs(rg)...)
}

func (r *analyticsRepository) Locations(ctx context.Context, rg domain.AnalyticsRange) ([]domain.LocationRow, error) {
	query := `WITH ` + sessionCTEs(inRange) + `
	SELECT country, nullif(city, '') AS city, count(DISTINCT visitor_id) AS visitors
	FROM ev
	WHERE coalesce(country, '') <> ''
	GROUP BY 1, 2
	ORDER BY visitors DESC
	LIMIT 12`

	return selectRows[domain.LocationRow](ctx, r.db, "locations", query, rangeArgs(rg)...)
}

// Whitelist: the column name is interpolated into SQL.
var breakdownColumns = map[string]bool{"device": true, "browser": true, "os": true, "site_language": true}

func (r *analyticsRepository) SessionBreakdown(ctx context.Context, rg domain.AnalyticsRange, column string) ([]domain.CountRow, error) {
	if !breakdownColumns[column] {
		return nil, fmt.Errorf("repo: session breakdown: unsupported column %q", column)
	}
	query := `WITH ` + sessionCTEs(inRange) + `
	SELECT coalesce(nullif(f.` + column + `, ''), 'unknown') AS key, count(*) AS count
	FROM s JOIN first f USING (session_id)
	GROUP BY 1
	ORDER BY count DESC, key
	LIMIT 12`

	return selectRows[domain.CountRow](ctx, r.db, "breakdown "+column, query, rangeArgs(rg)...)
}

func (r *analyticsRepository) BrowserLanguages(ctx context.Context, rg domain.AnalyticsRange) ([]domain.CountRow, error) {
	query := `WITH ` + sessionCTEs(inRange) + `
	SELECT coalesce(nullif(language, ''), 'unknown') AS key, count(DISTINCT visitor_id) AS count
	FROM ev
	GROUP BY 1
	ORDER BY count DESC, key
	LIMIT 12`

	return selectRows[domain.CountRow](ctx, r.db, "browser languages", query, rangeArgs(rg)...)
}

func (r *analyticsRepository) Funnel(ctx context.Context, rg domain.AnalyticsRange, sections []string) ([]domain.FunnelStep, error) {
	query := `WITH ` + sessionCTEs(inRange) + `
	SELECT step.section, count(DISTINCT ev.session_id) AS sessions
	FROM unnest($3::text[]) WITH ORDINALITY AS step(section, ord)
	LEFT JOIN ev ON ev.event_type = 'section_view' AND ev.section = step.section
	GROUP BY step.section, step.ord
	ORDER BY step.ord`

	return selectRows[domain.FunnelStep](ctx, r.db, "funnel", query, rg.From, rg.To, pq.Array(sections))
}

func (r *analyticsRepository) ScrollDepth(ctx context.Context, rg domain.AnalyticsRange) ([]domain.CountRow, error) {
	query := `WITH ` + sessionCTEs(inRange) + `,
	deepest AS (
		SELECT session_id, max((custom_data->>'percent')::int) AS pct
		FROM ev
		WHERE event_type = 'custom' AND custom_name = 'scroll_depth'
		GROUP BY session_id
	)
	SELECT t::text AS key, count(d.session_id) AS count
	FROM unnest(ARRAY[25, 50, 75, 100]) AS t
	LEFT JOIN deepest d ON d.pct >= t
	GROUP BY t
	ORDER BY t`

	return selectRows[domain.CountRow](ctx, r.db, "scroll depth", query, rangeArgs(rg)...)
}

func (r *analyticsRepository) CustomEventBreakdown(ctx context.Context, rg domain.AnalyticsRange, name, dataKey string) ([]domain.CountRow, error) {
	query := `WITH ` + sessionCTEs(inRange) + `
	SELECT coalesce(nullif(custom_data->>$4, ''), 'unknown') AS key, count(*) AS count
	FROM ev
	WHERE event_type = 'custom' AND custom_name = $3
	GROUP BY 1
	ORDER BY count DESC, key`

	return selectRows[domain.CountRow](ctx, r.db, "custom "+name, query, rg.From, rg.To, name, dataKey)
}

func (r *analyticsRepository) Interactions(ctx context.Context, rg domain.AnalyticsRange) ([]domain.CountRow, error) {
	query := `WITH ` + sessionCTEs(inRange) + `
	SELECT custom_name AS key, count(*) AS count
	FROM ev
	WHERE event_type = 'custom'
		AND custom_name IS NOT NULL
		AND custom_name NOT IN ('resume_download', 'contact_click', 'scroll_depth', 'web_vitals')
	GROUP BY 1
	ORDER BY count DESC, key`

	return selectRows[domain.CountRow](ctx, r.db, "interactions", query, rangeArgs(rg)...)
}

func (r *analyticsRepository) TopPages(ctx context.Context, rg domain.AnalyticsRange) ([]domain.PageRow, error) {
	query := `WITH ` + sessionCTEs(inRange) + `
	SELECT
		page,
		count(*) FILTER (WHERE event_type = 'pageview')                                        AS views,
		coalesce(avg(engaged_time_seconds) FILTER (WHERE event_type = 'leave'), 0)::float8      AS avg_engaged_seconds
	FROM ev
	WHERE coalesce(page, '') <> ''
	GROUP BY page
	HAVING count(*) FILTER (WHERE event_type = 'pageview') > 0
	ORDER BY views DESC
	LIMIT 10`

	return selectRows[domain.PageRow](ctx, r.db, "top pages", query, rangeArgs(rg)...)
}

func (r *analyticsRepository) Heatmap(ctx context.Context, rg domain.AnalyticsRange) ([]domain.HeatmapCell, error) {
	query := `WITH ` + sessionCTEs(inRange) + `
	SELECT
		extract(isodow FROM started_at)::int - 1 AS weekday,
		extract(hour FROM started_at)::int       AS hour,
		count(*)                                 AS visitors
	FROM s
	GROUP BY 1, 2
	ORDER BY 1, 2`

	return selectRows[domain.HeatmapCell](ctx, r.db, "heatmap", query, rangeArgs(rg)...)
}

func (r *analyticsRepository) WebVitals(ctx context.Context, rg domain.AnalyticsRange) (*domain.WebVitalsSummary, error) {
	query := `WITH ` + sessionCTEs(inRange) + `,
	v AS (SELECT custom_data AS d FROM ev WHERE event_type = 'custom' AND custom_name = 'web_vitals')
	SELECT
		percentile_cont(0.75) WITHIN GROUP (ORDER BY (d->>'lcp_ms')::float8)  AS lcp_p75_ms,
		percentile_cont(0.75) WITHIN GROUP (ORDER BY (d->>'cls')::float8)     AS cls_p75,
		percentile_cont(0.75) WITHIN GROUP (ORDER BY (d->>'inp_ms')::float8)  AS inp_p75_ms,
		percentile_cont(0.75) WITHIN GROUP (ORDER BY (d->>'ttfb_ms')::float8) AS ttfb_p75_ms,
		count(*)                                                              AS samples
	FROM v`

	var out domain.WebVitalsSummary
	if err := r.db.GetContext(ctx, &out, query, rangeArgs(rg)...); err != nil {
		return nil, fmt.Errorf("repo: web vitals: %w", err)
	}
	return &out, nil
}

// ---------------------------------------------------------------------------
// LIVE
// ---------------------------------------------------------------------------

// Live returns, per page, the visitors whose latest event in the window is not a "leave".
func (r *analyticsRepository) Live(ctx context.Context, windowSeconds int) ([]domain.LivePage, error) {
	const query = `
	WITH latest AS (
		SELECT DISTINCT ON (visitor_id) visitor_id, page, event_type
		FROM events
		WHERE created_at > now() - make_interval(secs => $1) AND device IS DISTINCT FROM 'bot'
		ORDER BY visitor_id, created_at DESC
	)
	SELECT coalesce(nullif(page, ''), '/') AS page, count(*) AS visitors
	FROM latest
	WHERE event_type <> 'leave'
	GROUP BY 1
	ORDER BY visitors DESC, page`

	return selectRows[domain.LivePage](ctx, r.db, "live", query, windowSeconds)
}

// ---------------------------------------------------------------------------
// SESSIONS
// ---------------------------------------------------------------------------

// summarySelect turns the shared CTEs into SessionSummary columns (+ converted, for filtering).
const summarySelect = `
	summ AS (
		SELECT
			s.session_id,
			s.visitor_id,
			s.started_at,
			s.ended_at,
			extract(epoch FROM s.ended_at - s.started_at)::bigint AS duration_seconds,
			s.engaged::bigint                                     AS engaged_seconds,
			s.pageviews,
			nullif(f.country, '')                                 AS country,
			nullif(f.city, '')                                    AS city,
			coalesce(nullif(f.device, ''), 'unknown')             AS device,
			nullif(f.os, '')                                      AS os,
			nullif(f.browser, '')                                 AS browser,
			nullif(f.language, '')                                AS browser_language,
			nullif(f.site_language, '')                           AS site_language,
			coalesce(f.source, 'direct')                          AS source,
			nullif(f.referrer, '')                                AS referrer,
			nullif(f.traffic_source->>'campaign', '')             AS utm_campaign,
			coalesce(f.page, '')                                  AS landing_page,
			s.is_returning,
			s.converted,
			(
				SELECT coalesce(jsonb_agg(DISTINCT e.custom_name), '[]'::jsonb)
				FROM ev e
				WHERE e.session_id = s.session_id AND e.event_type = 'custom' AND e.custom_name IN ` + conversionNames + `
			) AS conversions,
			(
				SELECT coalesce(max((e.custom_data->>'percent')::int), 0)
				FROM ev e
				WHERE e.session_id = s.session_id AND e.event_type = 'custom' AND e.custom_name = 'scroll_depth'
			)::bigint AS max_scroll,
			(
				SELECT coalesce(jsonb_agg(x.section ORDER BY x.first_at), '[]'::jsonb)
				FROM (
					SELECT e.section, min(e.created_at) AS first_at
					FROM ev e
					WHERE e.session_id = s.session_id AND e.event_type = 'section_view' AND e.section IS NOT NULL
					GROUP BY e.section
				) x
			) AS sections
		FROM s
		JOIN first f USING (session_id)
	)`

type sessionRow struct {
	domain.SessionSummary
	Converted bool `db:"converted"`
}

func (r *analyticsRepository) Sessions(ctx context.Context, q domain.SessionsQuery) (*domain.SessionsPage, error) {
	const filters = `
		WHERE ($3::bool = false OR converted)
			AND ($4 = '' OR country = $4)
			AND ($5 = '' OR source = $5)`
	args := []any{q.From, q.To, q.Converted, q.Country, q.Source}
	base := `WITH ` + sessionCTEs(inRange) + `,` + summarySelect

	var total int64
	if err := r.db.GetContext(ctx, &total, base+` SELECT count(*) FROM summ`+filters, args...); err != nil {
		return nil, fmt.Errorf("repo: sessions count: %w", err)
	}

	rows, err := selectRows[sessionRow](ctx, r.db, "sessions",
		base+` SELECT * FROM summ`+filters+` ORDER BY started_at DESC LIMIT $6 OFFSET $7`,
		append(args, q.Limit, q.Offset)...)
	if err != nil {
		return nil, err
	}

	page := &domain.SessionsPage{Total: total, Sessions: make([]domain.SessionSummary, len(rows))}
	for i, row := range rows {
		page.Sessions[i] = row.SessionSummary
	}
	return page, nil
}

func (r *analyticsRepository) Session(ctx context.Context, id string) (*domain.SessionDetail, error) {
	var row sessionRow
	query := `WITH ` + sessionCTEs(`session_id = $1`) + `,` + summarySelect + ` SELECT * FROM summ`
	if err := r.db.GetContext(ctx, &row, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("repo: session: %w", err)
	}

	const eventsQuery = `
	SELECT
		created_at                    AS at,
		event_type                    AS type,
		nullif(page, '')              AS page,
		section,
		custom_name                   AS name,
		custom_data                   AS data,
		engaged_time_seconds::bigint  AS engaged_seconds
	FROM events
	WHERE session_id = $1 AND event_type <> 'heartbeat'
	ORDER BY created_at`

	events, err := selectRows[domain.SessionEvent](ctx, r.db, "session events", eventsQuery, id)
	if err != nil {
		return nil, err
	}
	return &domain.SessionDetail{Session: row.SessionSummary, Events: events}, nil
}

// ---------------------------------------------------------------------------
// EXPORT
// ---------------------------------------------------------------------------

func (r *analyticsRepository) Export(ctx context.Context, rg domain.AnalyticsRange, limit int) ([]domain.ExportRow, error) {
	query := `
	SELECT
		created_at, event_type,
		coalesce(page, '')       AS page,
		coalesce(page_title, '') AS page_title,
		coalesce(referrer, '')   AS referrer,
		coalesce(country, '')    AS country,
		coalesce(city, '')       AS city,
		coalesce(language, '')   AS language,
		coalesce(viewport, '')   AS viewport,
		coalesce(user_agent, '') AS user_agent,
		time_spent_seconds, engaged_time_seconds
	FROM events
	WHERE ` + inRange + `
	ORDER BY created_at DESC
	LIMIT $3`

	return selectRows[domain.ExportRow](ctx, r.db, "export", query, rg.From, rg.To, limit)
}

// ---------------------------------------------------------------------------
// HELPERS
// ---------------------------------------------------------------------------

// selectRows runs a query into a non-nil slice, so empty results encode as [] rather than null.
func selectRows[T any](ctx context.Context, db *sqlx.DB, what, query string, args ...any) ([]T, error) {
	out := []T{}
	if err := db.SelectContext(ctx, &out, query, args...); err != nil {
		return nil, fmt.Errorf("repo: %s: %w", what, err)
	}
	return out, nil
}
