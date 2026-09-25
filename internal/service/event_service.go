package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ramisoul84/rami-server/internal/domain"
	"github.com/ramisoul84/rami-server/internal/repository"
	"github.com/ramisoul84/rami-server/pkg/logger"
)

type VisitNotifier interface {
	Enabled() bool
	NotifyVisit(ctx context.Context, n domain.VisitNotification) error
}

type EventService interface {
	TrackEvent(ctx context.Context, event *domain.IncomingEvent, ip string) error
}

type eventService struct {
	repo     repository.EventRepository
	notifier VisitNotifier
	geo      GeoService
	log      *logger.Logger
}

func NewEventService(
	repo repository.EventRepository,
	n VisitNotifier,
	geo GeoService,
	log *logger.Logger,
) EventService {
	return &eventService{
		repo:     repo,
		notifier: n,
		geo:      geo,
		log:      log,
	}
}

func (s *eventService) TrackEvent(ctx context.Context, event *domain.IncomingEvent, ip string) error {
	if event == nil {
		return errors.New("event: nil event")
	}
	if event.SessionID == "" || event.VisitorID == "" {
		return errors.New("event: session_id and visitor_id are required")
	}
	if event.EventType == "" {
		return errors.New("event: event_type is required")
	}

	country, city := "", ""
	if s.geo != nil {
		country, city = s.geo.Lookup(ip)
	}

	row := toEventRow(event, ip)
	row.Country = country
	row.City = city

	if err := s.repo.InsertEvent(ctx, row); err != nil {
		s.log.Error("event insert failed",
			"err", err,
			"session_id", event.SessionID,
			"event_type", event.EventType,
		)
		return fmt.Errorf("event: insert: %w", err)
	}

	if event.EventType == domain.EventTypePageView &&
		s.notifier != nil && s.notifier.Enabled() {
		go s.notifyVisit(event, ip, country, city)
	}
	return nil
}

func (s *eventService) notifyVisit(event *domain.IncomingEvent, ip, country, city string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	payload := domain.VisitNotification{
		SessionID:  event.SessionID,
		IP:         ip,
		Country:    country,
		City:       city,
		Page:       event.Page,
		PageTitle:  event.PageTitle,
		Referrer:   event.Referrer,
		Language:   event.Language,
		Timezone:   event.Timezone,
		Viewport:   event.Viewport,
		Device:     detectDevice(event.UserAgent),
		UserAgent:  event.UserAgent,
		OccurredAt: time.Now().UnixMilli(),
	}

	if err := s.notifier.NotifyVisit(ctx, payload); err != nil {
		s.log.Warn("visit notification failed",
			"err", err,
			"session_id", event.SessionID,
		)
	}
}

func toEventRow(in *domain.IncomingEvent, ip string) *domain.Event {
	return &domain.Event{
		SessionID:          in.SessionID,
		VisitorID:          in.VisitorID,
		EventType:          in.EventType,
		Page:               in.Page,
		PageTitle:          in.PageTitle,
		Referrer:           in.Referrer,
		Language:           in.Language,
		Timezone:           in.Timezone,
		Viewport:           in.Viewport,
		Screen:             in.Screen,
		UserAgent:          in.UserAgent,
		TrafficSource:      marshalJSON(in.Traffic),
		TimeSpentSeconds:   in.TimeSpentSeconds,
		EngagedTimeSeconds: in.EngagedTimeSeconds,
		Section:            nullable(in.Section),
		Selector:           nullable(in.Selector),
		Label:              nullable(in.Label),
		CustomName:         nullable(in.Name),
		CustomData:         marshalJSON(in.Data),
		IPAddress:          ip,
	}
}

func nullable(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func marshalJSON(v any) json.RawMessage {
	if v == nil {
		return nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	return b
}

func detectDevice(ua string) string {
	lower := strings.ToLower(ua)
	for _, kw := range []string{"mobile", "android", "iphone", "ipad"} {
		if strings.Contains(lower, kw) {
			return "mobile"
		}
	}
	return "desktop"
}
