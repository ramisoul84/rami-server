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
	// Language the site was read in (en / ru / de), as opposed to the browser's Language.
	SiteLanguage string `json:"site_language"`
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

	// Derived at insert time so analytics queries can group on them.
	SiteLanguage *string `db:"site_language"`
	Device       string  `db:"device"`
	OS           string  `db:"os"`
	Browser      string  `db:"browser"`
	Source       string  `db:"source"`
}

// ExportRow is one line of the admin CSV export.
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

// VisitorHistory is what is stored about a visitor before their current event.
type VisitorHistory struct {
	Sessions    int  `db:"sessions"`
	SessionSeen bool `db:"session_seen"`
}
