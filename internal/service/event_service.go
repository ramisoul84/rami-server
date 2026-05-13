package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/mssola/useragent"

	"github.com/ramisoul84/rami-server/internal/domain"
	"github.com/ramisoul84/rami-server/internal/repository"
	"github.com/ramisoul84/rami-server/pkg/logger"
)

type VisitNotifier interface {
	Enabled() bool
	NotifyVisit(ctx context.Context, n domain.VisitNotification) error
	NotifyLead(ctx context.Context, n domain.LeadNotification) error
}

// ErrInvalidEvent marks an event rejected by validation (a client bug, not a server fault).
var ErrInvalidEvent = errors.New("event: invalid")

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
	if err := sanitize(event); err != nil {
		return err
	}

	uaOS, uaBrowser, device := uaColumns(event.UserAgent)
	if device == "bot" {
		// Crawlers and link previews would skew every metric — accept and discard.
		return nil
	}

	country, city := "", ""
	if s.geo != nil {
		country, city = s.geo.Lookup(ip)
	}

	// A session's first pageview is what triggers a visit alert, so look the
	// visitor up before this event lands in the table.
	notifying := s.notifier != nil && s.notifier.Enabled()
	var history *domain.VisitorHistory
	if notifying && event.EventType == domain.EventTypePageView {
		h, err := s.repo.VisitorHistory(ctx, event.VisitorID, event.SessionID)
		if err != nil {
			s.log.Warn("visitor history lookup failed", "err", err, "session_id", event.SessionID)
		} else {
			history = h
		}
	}

	row := toEventRow(event, ip)
	row.Country = country
	row.City = city
	row.OS = uaOS
	row.Browser = uaBrowser
	row.Device = device
	row.Source = normalizeSource(event.Referrer, event.Traffic)
	row.SiteLanguage = nullable(event.SiteLanguage)

	if err := s.repo.InsertEvent(ctx, row); err != nil {
		s.log.Error("event insert failed",
			"err", err,
			"session_id", event.SessionID,
			"event_type", event.EventType,
		)
		return fmt.Errorf("event: insert: %w", err)
	}

	if !notifying {
		return nil
	}
	switch {
	case history != nil && !history.SessionSeen:
		go s.notifyVisit(event, row, history)
	case event.EventType == domain.EventTypeCustom && isLead(event.Name):
		go s.notifyLead(event, row)
	}
	return nil
}

var eventTypes = map[string]bool{
	domain.EventTypePageView: true, domain.EventTypeLeave: true, domain.EventTypeSectionView: true,
	domain.EventTypeHeartbeat: true, domain.EventTypeClick: true, domain.EventTypeCustom: true,
}

const (
	maxIDLen       = 64
	maxShortLen    = 64   // language, timezone, viewport, names…
	maxTextLen     = 512  // page, title, referrer, user agent, selector, label
	maxDataBytes   = 2048 // serialized custom data
	maxSecondsSpan = 86_400
)

// sanitize validates an incoming event and trims every field to a sane size,
// so a buggy or hostile client cannot bloat the events table.
func sanitize(e *domain.IncomingEvent) error {
	if e == nil {
		return fmt.Errorf("%w: empty body", ErrInvalidEvent)
	}
	if e.SessionID == "" || e.VisitorID == "" || len(e.SessionID) > maxIDLen || len(e.VisitorID) > maxIDLen {
		return fmt.Errorf("%w: session_id and visitor_id are required (max %d chars)", ErrInvalidEvent, maxIDLen)
	}
	if !eventTypes[e.EventType] {
		return fmt.Errorf("%w: unknown event_type %q", ErrInvalidEvent, truncate(e.EventType, maxShortLen))
	}
	if e.EventType == domain.EventTypeCustom && e.Name == "" {
		return fmt.Errorf("%w: custom events need a name", ErrInvalidEvent)
	}

	for _, f := range []*string{&e.Page, &e.PageTitle, &e.Referrer, &e.UserAgent, &e.Selector, &e.Label} {
		*f = truncate(*f, maxTextLen)
	}
	for _, f := range []*string{&e.Language, &e.Timezone, &e.Viewport, &e.Screen, &e.Section, &e.Name, &e.SiteLanguage} {
		*f = truncate(*f, maxShortLen)
	}
	e.TimeSpentSeconds = clampSeconds(e.TimeSpentSeconds)
	e.EngagedTimeSeconds = clampSeconds(e.EngagedTimeSeconds)

	if e.Data != nil {
		if b, err := json.Marshal(e.Data); err != nil || len(b) > maxDataBytes {
			e.Data = nil
		}
	}
	if t := e.Traffic; t != nil {
		for _, f := range []*string{t.Source, t.Medium, t.Campaign, t.Term, t.Content} {
			if f != nil {
				*f = truncate(*f, maxShortLen*2)
			}
		}
		t.LandingPage = truncate(t.LandingPage, maxTextLen)
	}
	return nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	// Cut on a rune boundary so the stored text stays valid UTF-8.
	for max > 0 && !utf8.RuneStart(s[max]) {
		max--
	}
	return s[:max]
}

func clampSeconds(v *int) *int {
	if v == nil {
		return nil
	}
	n := min(max(*v, 0), maxSecondsSpan)
	return &n
}

func isLead(name string) bool {
	return name == domain.LeadResumeDownload || name == domain.LeadContactClick
}

func (s *eventService) notifyVisit(event *domain.IncomingEvent, row *domain.Event, history *domain.VisitorHistory) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	os, browser, device := parseUserAgent(event.UserAgent)

	kind := domain.VisitKindNew
	if history.Sessions > 0 {
		kind = domain.VisitKindReturning
	}
	campaign := ""
	if event.Traffic != nil && event.Traffic.Campaign != nil {
		campaign = *event.Traffic.Campaign
	}

	payload := domain.VisitNotification{
		Kind:         kind,
		VisitNumber:  history.Sessions + 1,
		SessionID:    event.SessionID,
		VisitorID:    event.VisitorID,
		IP:           row.IPAddress,
		Country:      row.Country,
		City:         row.City,
		Page:         event.Page,
		PageTitle:    event.PageTitle,
		Referrer:     event.Referrer,
		Source:       row.Source,
		Campaign:     campaign,
		Language:     event.Language,
		SiteLanguage: event.SiteLanguage,
		Timezone:     event.Timezone,
		Viewport:     event.Viewport,
		Device:       device,
		OS:           os,
		Browser:      browser,
		UserAgent:    event.UserAgent,
		OccurredAt:   time.Now().UnixMilli(),
	}

	if err := s.notifier.NotifyVisit(ctx, payload); err != nil {
		s.log.Warn("visit notification failed",
			"err", err,
			"session_id", event.SessionID,
		)
	}
}

func (s *eventService) notifyLead(event *domain.IncomingEvent, row *domain.Event) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	os, browser, device := parseUserAgent(event.UserAgent)

	detailKey := "channel"
	if event.Name == domain.LeadResumeDownload {
		detailKey = "lang"
	}

	payload := domain.LeadNotification{
		Name:       event.Name,
		Detail:     dataString(event.Data, detailKey),
		Origin:     dataString(event.Data, "source"),
		SessionID:  event.SessionID,
		Country:    row.Country,
		City:       row.City,
		Device:     device,
		OS:         os,
		Browser:    browser,
		Page:       event.Page,
		Source:     row.Source,
		OccurredAt: time.Now().UnixMilli(),
	}

	if err := s.notifier.NotifyLead(ctx, payload); err != nil {
		s.log.Warn("lead notification failed",
			"err", err,
			"session_id", event.SessionID,
			"lead", event.Name,
		)
	}
}

func dataString(data map[string]interface{}, key string) string {
	if v, ok := data[key].(string); ok {
		return v
	}
	return ""
}

// parseUserAgent extracts OS, browser (with major version, for notifications)
// and device type (desktop / mobile / tablet / bot) from a UA string.
func parseUserAgent(ua string) (os string, browser string, device string) {
	u := useragent.New(ua)

	os = u.OSInfo().Name
	if os == "" {
		os = "Unknown"
	} else if v := u.OSInfo().Version; v != "" {
		os += " " + v
	}

	browserName, browserVersion := u.Browser()
	if browserName == "" {
		browser = "Unknown"
	} else {
		browser = browserName
		if major, _, _ := strings.Cut(browserVersion, "."); major != "" {
			browser += " " + major
		}
	}

	return os, browser, deviceType(u, ua)
}

func deviceType(u *useragent.UserAgent, ua string) string {
	lower := strings.ToLower(ua)
	switch {
	// HeadlessChrome is how scrapers, uptime checkers and screenshot tools identify themselves.
	case u.Bot() || strings.Contains(lower, "headless"):
		return "bot"
	case strings.Contains(lower, "ipad") || strings.Contains(lower, "tablet") ||
		(strings.Contains(lower, "android") && !strings.Contains(lower, "mobile")):
		return "tablet"
	case u.Mobile():
		return "mobile"
	default:
		return "desktop"
	}
}

// uaColumns returns the version-free values stored per event, so they group cleanly.
func uaColumns(ua string) (os, browser, device string) {
	u := useragent.New(ua)
	os = u.OSInfo().Name
	browser, _ = u.Browser()
	switch {
	case strings.Contains(ua, "iPad"):
		os = "iPadOS" // the parser reports iPads as "OS"
	case os == "iPhone OS":
		os = "iOS"
	case os == "":
		os = "Unknown"
	}
	if browser == "" {
		browser = "Unknown"
	}
	return os, browser, deviceType(u, ua)
}

// normalizeSource maps a visit to a short traffic source name. utm_source wins over the referrer.
func normalizeSource(referrer string, traffic *domain.TrafficSource) string {
	if traffic != nil && traffic.Source != nil {
		if s := strings.ToLower(strings.TrimSpace(*traffic.Source)); s != "" {
			return s
		}
	}

	ref := strings.TrimSpace(referrer)
	if ref == "" {
		return "direct"
	}
	u, err := url.Parse(ref)
	if err != nil || u.Hostname() == "" {
		return "direct"
	}
	host := strings.TrimPrefix(strings.ToLower(u.Hostname()), "www.")

	switch {
	case strings.HasPrefix(host, "google.") || strings.Contains(host, ".google."):
		return "google"
	case host == "linkedin.com" || strings.HasSuffix(host, ".linkedin.com") || host == "lnkd.in":
		return "linkedin"
	case host == "hh.ru" || strings.HasSuffix(host, ".hh.ru"):
		return "hh.ru"
	case host == "t.me" || host == "web.telegram.org":
		return "telegram"
	case host == "github.com":
		return "github"
	case strings.HasPrefix(host, "yandex.") || strings.HasPrefix(host, "ya."):
		return "yandex"
	default:
		return host
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
