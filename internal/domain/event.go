package domain

import (
	"encoding/json"
	"time"
)

const (
	EventTypePageView    = "pageview"
	EventTypeLeave       = "leave"
	EventTypeSectionView = "section_view"
	EventTypeHeartbeat   = "heartbeat"
	EventTypeClick       = "click"
	EventTypeCustom      = "custom"
)

type TrafficSource struct {
	Source      *string `json:"source"`
	Medium      *string `json:"medium"`
	Campaign    *string `json:"campaign"`
	Term        *string `json:"term"`
	Content     *string `json:"content"`
	LandingPage string  `json:"landing_page"`
}

type IncomingEvent struct {
	SessionID          string                 `json:"session_id"`
	VisitorID          string                 `json:"visitor_id"`
	EventType          string                 `json:"event_type"`
	Page               string                 `json:"page"`
	PageTitle          string                 `json:"page_title"`
	Referrer           string                 `json:"referrer"`
	Language           string                 `json:"language"`
	Timezone           string                 `json:"timezone"`
	Viewport           string                 `json:"viewport"`
	Screen             string                 `json:"screen"`
	UserAgent          string                 `json:"user_agent"`
	Timestamp          string                 `json:"timestamp"`
	Traffic            *TrafficSource         `json:"traffic"`
	TimeSpentSeconds   *int                   `json:"time_spent_seconds"`
	EngagedTimeSeconds *int                   `json:"engaged_time_seconds"`
	Section            string                 `json:"section"`
	Selector           string                 `json:"selector"`
	Label              string                 `json:"label"`
	Name               string                 `json:"name"`
	Data               map[string]interface{} `json:"data"`
}

type Event struct {
	SessionID          string          `db:"session_id"`
	VisitorID          string          `db:"visitor_id"`
	EventType          string          `db:"event_type"`
	Page               string          `db:"page"`
	PageTitle          string          `db:"page_title"`
	Referrer           string          `db:"referrer"`
	Language           string          `db:"language"`
	Timezone           string          `db:"timezone"`
	Viewport           string          `db:"viewport"`
	Screen             string          `db:"screen"`
	UserAgent          string          `db:"user_agent"`
	TrafficSource      json.RawMessage `db:"traffic_source"`
	TimeSpentSeconds   *int            `db:"time_spent_seconds"`
	EngagedTimeSeconds *int            `db:"engaged_time_seconds"`
	Section            *string         `db:"section"`
	Selector           *string         `db:"selector"`
	Label              *string         `db:"label"`
	CustomName         *string         `db:"custom_name"`
	CustomData         json.RawMessage `db:"custom_data"`
	IPAddress          string          `db:"ip_address"`
	Country            string          `db:"country"`
	City               string          `db:"city"`
}

type Summary struct {
	TotalVisitors  int64 `db:"total_visitors" json:"total_visitors"`
	TotalSessions  int64 `db:"total_sessions" json:"total_sessions"`
	TotalPageViews int64 `db:"total_page_views" json:"total_page_views"`
}

type DayStat struct {
	Date      string `db:"date" json:"date"`
	Visitors  int64  `db:"visitors" json:"visitors"`
	PageViews int64  `db:"page_views" json:"page_views"`
}

type PageStat struct {
	Page  string `db:"page" json:"page"`
	Count int64  `db:"count" json:"count"`
}

type ReferrerStat struct {
	Referrer string `db:"referrer" json:"referrer"`
	Count    int64  `db:"count" json:"count"`
}

type CountryStat struct {
	Country string `db:"country" json:"country"`
	Count   int64  `db:"count" json:"count"`
}

type Visit struct {
	Page      string    `db:"page" json:"page"`
	Referrer  string    `db:"referrer" json:"referrer"`
	Country   string    `db:"country" json:"country"`
	City      string    `db:"city" json:"city"`
	IPAddress string    `db:"ip_address" json:"ip_address"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

type DashboardStats struct {
	Summary       *Summary       `json:"summary"`
	Today         *DayStat       `json:"today"`
	TopPages      []PageStat     `json:"top_pages"`
	TopReferrers  []ReferrerStat `json:"top_referrers"`
	VisitorsByDay []DayStat      `json:"visitors_by_day"`
}

// DateRange represents a filterable window. Zero value means "no filter".
type DateRange struct {
	From time.Time
	To   time.Time
}

func (r DateRange) IsZero() bool {
	return r.From.IsZero() && r.To.IsZero()
}

// ExportRow is the projection used by the CSV export endpoint.
type ExportRow struct {
	CreatedAt          time.Time `db:"created_at"`
	EventType          string    `db:"event_type"`
	Page               string    `db:"page"`
	PageTitle          string    `db:"page_title"`
	Referrer           string    `db:"referrer"`
	Country            string    `db:"country"`
	City               string    `db:"city"`
	Language           string    `db:"language"`
	Viewport           string    `db:"viewport"`
	UserAgent          string    `db:"user_agent"`
	TimeSpentSeconds   *int      `db:"time_spent_seconds"`
	EngagedTimeSeconds *int      `db:"engaged_time_seconds"`
}
