package domain

import (
	"encoding/json"
	"time"
)

// Types for /api/v1/admin/analytics/*. They mirror the client contract in
// client/src/app/pages/admin/data/analytics.models.ts — keep the JSON names in sync.

// AnalyticsRange is an inclusive day range in the database timezone, "YYYY-MM-DD".
type AnalyticsRange struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type Kpi struct {
	Current  float64 `json:"current"`
	Previous float64 `json:"previous"`
}

type OverviewKpis struct {
	Visitors          Kpi `json:"visitors"`
	Sessions          Kpi `json:"sessions"`
	Pageviews         Kpi `json:"pageviews"`
	AvgEngagedSeconds Kpi `json:"avg_engaged_seconds"`
	BounceRate        Kpi `json:"bounce_rate"`
	ReturningRate     Kpi `json:"returning_rate"`
	ResumeDownloads   Kpi `json:"resume_downloads"`
	ContactClicks     Kpi `json:"contact_clicks"`
	ConversionRate    Kpi `json:"conversion_rate"`
}

// KpiTotals is one period's worth of the numbers behind OverviewKpis.
type KpiTotals struct {
	Visitors          float64 `db:"visitors"`
	Sessions          float64 `db:"sessions"`
	Pageviews         float64 `db:"pageviews"`
	AvgEngagedSeconds float64 `db:"avg_engaged_seconds"`
	BounceRate        float64 `db:"bounce_rate"`
	ReturningRate     float64 `db:"returning_rate"`
	ResumeDownloads   float64 `db:"resume_downloads"`
	ContactClicks     float64 `db:"contact_clicks"`
	ConversionRate    float64 `db:"conversion_rate"`
}

type TimeseriesPoint struct {
	Date        string `db:"date" json:"date"`
	Visitors    int64  `db:"visitors" json:"visitors"`
	Sessions    int64  `db:"sessions" json:"sessions"`
	Pageviews   int64  `db:"pageviews" json:"pageviews"`
	Conversions int64  `db:"conversions" json:"conversions"`
}

type CountRow struct {
	Key   string `db:"key" json:"key"`
	Count int64  `db:"count" json:"count"`
}

type SourceRow struct {
	Source      string `db:"source" json:"source"`
	Visitors    int64  `db:"visitors" json:"visitors"`
	Conversions int64  `db:"conversions" json:"conversions"`
}

type CampaignRow struct {
	Campaign    string  `db:"campaign" json:"campaign"`
	Source      *string `db:"source" json:"source"`
	Medium      *string `db:"medium" json:"medium"`
	Visitors    int64   `db:"visitors" json:"visitors"`
	Conversions int64   `db:"conversions" json:"conversions"`
}

type LocationRow struct {
	Country  string  `db:"country" json:"country"`
	City     *string `db:"city" json:"city"`
	Visitors int64   `db:"visitors" json:"visitors"`
}

type FunnelStep struct {
	Section  string `db:"section" json:"section"`
	Sessions int64  `db:"sessions" json:"sessions"`
}

type HeatmapCell struct {
	Weekday  int   `db:"weekday" json:"weekday"`
	Hour     int   `db:"hour" json:"hour"`
	Visitors int64 `db:"visitors" json:"visitors"`
}

type PageRow struct {
	Page              string  `db:"page" json:"page"`
	Views             int64   `db:"views" json:"views"`
	AvgEngagedSeconds float64 `db:"avg_engaged_seconds" json:"avg_engaged_seconds"`
}

type WebVitalsSummary struct {
	LcpP75Ms  *float64 `db:"lcp_p75_ms" json:"lcp_p75_ms"`
	ClsP75    *float64 `db:"cls_p75" json:"cls_p75"`
	InpP75Ms  *float64 `db:"inp_p75_ms" json:"inp_p75_ms"`
	TtfbP75Ms *float64 `db:"ttfb_p75_ms" json:"ttfb_p75_ms"`
	Samples   int64    `db:"samples" json:"samples"`
}

type OverviewRange struct {
	AnalyticsRange
	PreviousFrom string `json:"previous_from"`
	PreviousTo   string `json:"previous_to"`
}

type Overview struct {
	Range            OverviewRange     `json:"range"`
	Kpis             OverviewKpis      `json:"kpis"`
	Timeseries       []TimeseriesPoint `json:"timeseries"`
	Sources          []SourceRow       `json:"sources"`
	Campaigns        []CampaignRow     `json:"campaigns"`
	Locations        []LocationRow     `json:"locations"`
	Devices          []CountRow        `json:"devices"`
	Browsers         []CountRow        `json:"browsers"`
	OS               []CountRow        `json:"os"`
	SiteLanguages    []CountRow        `json:"site_languages"`
	BrowserLanguages []CountRow        `json:"browser_languages"`
	Funnel           []FunnelStep      `json:"funnel"`
	ScrollDepth      []CountRow        `json:"scroll_depth"`
	ResumeByLanguage []CountRow        `json:"resume_by_language"`
	ContactByChannel []CountRow        `json:"contact_by_channel"`
	Interactions     []CountRow        `json:"interactions"`
	TopPages         []PageRow         `json:"top_pages"`
	Heatmap          []HeatmapCell     `json:"heatmap"`
	WebVitals        WebVitalsSummary  `json:"web_vitals"`
}

type LivePage struct {
	Page     string `db:"page" json:"page"`
	Visitors int64  `db:"visitors" json:"visitors"`
}

type LiveSnapshot struct {
	Visitors      int64      `json:"visitors"`
	WindowSeconds int        `json:"window_seconds"`
	Pages         []LivePage `json:"pages"`
}

type SessionSummary struct {
	SessionID       string          `db:"session_id" json:"session_id"`
	VisitorID       string          `db:"visitor_id" json:"visitor_id"`
	StartedAt       time.Time       `db:"started_at" json:"started_at"`
	EndedAt         time.Time       `db:"ended_at" json:"ended_at"`
	DurationSeconds int64           `db:"duration_seconds" json:"duration_seconds"`
	EngagedSeconds  int64           `db:"engaged_seconds" json:"engaged_seconds"`
	Pageviews       int64           `db:"pageviews" json:"pageviews"`
	Country         *string         `db:"country" json:"country"`
	City            *string         `db:"city" json:"city"`
	Device          string          `db:"device" json:"device"`
	OS              *string         `db:"os" json:"os"`
	Browser         *string         `db:"browser" json:"browser"`
	BrowserLanguage *string         `db:"browser_language" json:"browser_language"`
	SiteLanguage    *string         `db:"site_language" json:"site_language"`
	Source          string          `db:"source" json:"source"`
	Referrer        *string         `db:"referrer" json:"referrer"`
	UtmCampaign     *string         `db:"utm_campaign" json:"utm_campaign"`
	LandingPage     string          `db:"landing_page" json:"landing_page"`
	IsReturning     bool            `db:"is_returning" json:"is_returning"`
	Conversions     json.RawMessage `db:"conversions" json:"conversions"`
	MaxScroll       int64           `db:"max_scroll" json:"max_scroll"`
	Sections        json.RawMessage `db:"sections" json:"sections"`
}

type SessionsQuery struct {
	AnalyticsRange
	Limit     int
	Offset    int
	Converted bool
	Country   string
	Source    string
}

type SessionsPage struct {
	Total    int64            `json:"total"`
	Sessions []SessionSummary `json:"sessions"`
}

type SessionEvent struct {
	At             time.Time       `db:"at" json:"at"`
	Type           string          `db:"type" json:"type"`
	Page           *string         `db:"page" json:"page"`
	Section        *string         `db:"section" json:"section"`
	Name           *string         `db:"name" json:"name"`
	Data           json.RawMessage `db:"data" json:"data"`
	EngagedSeconds *int64          `db:"engaged_seconds" json:"engaged_seconds"`
}

type SessionDetail struct {
	Session SessionSummary `json:"session"`
	Events  []SessionEvent `json:"events"`
}
