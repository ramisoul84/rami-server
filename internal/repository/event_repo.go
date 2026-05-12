package repository

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/ramisoul84/rami-server/internal/domain"
)

// EventRepository is the write side of analytics: storing tracked events.
type EventRepository interface {
	InsertEvent(ctx context.Context, event *domain.Event) error
	VisitorHistory(ctx context.Context, visitorID, sessionID string) (*domain.VisitorHistory, error)
}

type eventRepository struct {
	db *sqlx.DB
}

func NewEventRepository(db *sqlx.DB) EventRepository {
	return &eventRepository{db: db}
}

func (r *eventRepository) InsertEvent(ctx context.Context, event *domain.Event) error {
	const query = `
		INSERT INTO events (
			session_id, visitor_id, event_type, page, page_title,
			referrer, language, timezone, viewport, screen, user_agent,
			traffic_source, time_spent_seconds, engaged_time_seconds,
			section, selector, label, custom_name, custom_data,
			ip_address, country, city,
			site_language, device, os, browser, source
		) VALUES (
			:session_id, :visitor_id, :event_type, :page, :page_title,
			:referrer, :language, :timezone, :viewport, :screen, :user_agent,
			:traffic_source, :time_spent_seconds, :engaged_time_seconds,
			:section, :selector, :label, :custom_name, :custom_data,
			:ip_address, :country, :city,
			:site_language, :device, :os, :browser, :source
		)
	`
	if _, err := r.db.NamedExecContext(ctx, query, event); err != nil {
		return fmt.Errorf("repo: insert event: %w", err)
	}
	return nil
}

// VisitorHistory reports what is already stored about a visitor before their
// next event: how many sessions they had and whether this session has events.
func (r *eventRepository) VisitorHistory(ctx context.Context, visitorID, sessionID string) (*domain.VisitorHistory, error) {
	const query = `
		SELECT
			COUNT(DISTINCT session_id)                AS sessions,
			COALESCE(BOOL_OR(session_id = $2), FALSE) AS session_seen
		FROM events
		WHERE visitor_id = $1
	`
	var h domain.VisitorHistory
	if err := r.db.GetContext(ctx, &h, query, visitorID, sessionID); err != nil {
		return nil, fmt.Errorf("repo: visitor history: %w", err)
	}
	return &h, nil
}
