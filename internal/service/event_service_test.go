package service

import (
	"errors"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/ramisoul84/rami-server/internal/domain"
)

func TestSanitize(t *testing.T) {
	valid := func() *domain.IncomingEvent {
		return &domain.IncomingEvent{SessionID: "s", VisitorID: "v", EventType: domain.EventTypePageView}
	}

	if err := sanitize(valid()); err != nil {
		t.Fatalf("valid event rejected: %v", err)
	}

	for name, mutate := range map[string]func(*domain.IncomingEvent){
		"missing session":     func(e *domain.IncomingEvent) { e.SessionID = "" },
		"oversized visitor":   func(e *domain.IncomingEvent) { e.VisitorID = strings.Repeat("x", 65) },
		"unknown type":        func(e *domain.IncomingEvent) { e.EventType = "drop table" },
		"custom without name": func(e *domain.IncomingEvent) { e.EventType = domain.EventTypeCustom },
	} {
		e := valid()
		mutate(e)
		if err := sanitize(e); !errors.Is(err, ErrInvalidEvent) {
			t.Errorf("%s: want ErrInvalidEvent, got %v", name, err)
		}
	}

	e := valid()
	e.Page = strings.Repeat("я", 1000)
	e.Data = map[string]interface{}{"blob": strings.Repeat("x", 5000)}
	neg, huge := -5, 10_000_000
	e.TimeSpentSeconds, e.EngagedTimeSeconds = &neg, &huge
	if err := sanitize(e); err != nil {
		t.Fatal(err)
	}
	if len(e.Page) > maxTextLen || !utf8.ValidString(e.Page) {
		t.Errorf("page not truncated to valid UTF-8: %d bytes", len(e.Page))
	}
	if e.Data != nil {
		t.Error("oversized data should be dropped")
	}
	if *e.TimeSpentSeconds != 0 || *e.EngagedTimeSeconds != maxSecondsSpan {
		t.Errorf("seconds not clamped: %d, %d", *e.TimeSpentSeconds, *e.EngagedTimeSeconds)
	}
}

func TestNormalizeSource(t *testing.T) {
	src := "LinkedIn"
	cases := map[string]struct {
		ref     string
		traffic *domain.TrafficSource
		want    string
	}{
		"utm wins": {"https://google.com", &domain.TrafficSource{Source: &src}, "linkedin"},
		"google":   {"https://www.google.de/search", nil, "google"},
		"telegram": {"https://t.me/x", nil, "telegram"},
		"direct":   {"", nil, "direct"},
		"other":    {"https://news.ycombinator.com/item", nil, "news.ycombinator.com"},
	}
	for name, tc := range cases {
		if got := normalizeSource(tc.ref, tc.traffic); got != tc.want {
			t.Errorf("%s: got %q, want %q", name, got, tc.want)
		}
	}
}
