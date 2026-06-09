package notifier

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/go-telegram/bot/models"

	"github.com/ramisoul84/rami-server/internal/domain"
)

var (
	tagRe       = regexp.MustCompile(`<(/?)([a-z-]+)[^>]*>`)
	allowedTags = map[string]bool{"b": true, "i": true, "u": true, "s": true, "code": true, "pre": true, "a": true, "blockquote": true}
)

// checkTelegramHTML fails when text would be rejected by Telegram's HTML parse mode
// or is over the 4096-character message limit.
func checkTelegramHTML(t *testing.T, name, text string) {
	t.Helper()
	var stack []string
	for _, m := range tagRe.FindAllStringSubmatch(text, -1) {
		closing, tag := m[1] == "/", m[2]
		if !allowedTags[tag] {
			t.Errorf("%s: unsupported tag <%s>", name, tag)
		}
		if !closing {
			stack = append(stack, tag)
			continue
		}
		if len(stack) == 0 || stack[len(stack)-1] != tag {
			t.Errorf("%s: unbalanced </%s>", name, tag)
			return
		}
		stack = stack[:len(stack)-1]
	}
	if len(stack) > 0 {
		t.Errorf("%s: unclosed tags %v", name, stack)
	}

	plain := tagRe.ReplaceAllString(text, "")
	if strings.ContainsAny(plain, "<>") {
		t.Errorf("%s: unescaped < or > in text", name)
	}
	if n := utf8.RuneCountInString(plain); n > 4096 {
		t.Errorf("%s: %d characters, over Telegram's 4096 limit", name, n)
	}
}

func checkKeyboard(t *testing.T, name string, kb *models.InlineKeyboardMarkup) {
	t.Helper()
	if kb == nil {
		return
	}
	for _, r := range kb.InlineKeyboard {
		for _, b := range r {
			if len(b.CallbackData) > 64 {
				t.Errorf("%s: callback data %q over 64 bytes", name, b.CallbackData)
			}
			if b.CallbackData == "" && b.URL == "" && b.CopyText == nil {
				t.Errorf("%s: button %q does nothing", name, b.Text)
			}
		}
	}
}

func testNotifier() *TelegramNotifier {
	return &TelegramNotifier{
		siteURL: "https://ramisuliman.com",
		loc:     time.UTC,
		now:     func() time.Time { return time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC) },
	}
}

func TestPublicScreensRenderInEveryLanguage(t *testing.T) {
	n := testNotifier()
	views := []string{viewMenu, viewAbout, viewSkills, viewProjects, viewContact, viewLanguage}
	for _, l := range langs {
		for _, admin := range []bool{false, true} {
			for _, v := range views {
				s, ok := n.publicScreen(l, v, "", admin)
				if !ok {
					t.Fatalf("%s/%s: unknown view", l, v)
				}
				checkTelegramHTML(t, string(l)+"/"+v, s.text)
				checkKeyboard(t, string(l)+"/"+v, s.kb)
			}
			for i := range contentFor(l).jobs {
				s, _ := n.publicScreen(l, viewExperience, string(rune('0'+i)), admin)
				checkTelegramHTML(t, string(l)+"/exp", s.text)
				checkKeyboard(t, string(l)+"/exp", s.kb)
			}
		}
		w := n.welcomeScreen(l, "<Ann & Bob>", false)
		checkTelegramHTML(t, string(l)+"/welcome", w.text)
	}
}

func TestEveryLanguageHasSameShape(t *testing.T) {
	en := contentFor(LangEN)
	for _, l := range langs {
		c := contentFor(l)
		if len(c.jobs) != len(en.jobs) || len(c.cases) != len(en.cases) || len(c.projects) != len(en.projects) || len(c.skillGroups) != len(en.skillGroups) {
			t.Errorf("%s: content shape differs from English", l)
		}
		for i := range c.jobs {
			if len(c.jobs[i].achievements) != len(en.jobs[i].achievements) {
				t.Errorf("%s: job %d has %d achievements, English has %d", l, i, len(c.jobs[i].achievements), len(en.jobs[i].achievements))
			}
		}
	}
}

func TestExperienceArgIsClamped(t *testing.T) {
	n := testNotifier()
	for _, arg := range []string{"", "-1", "99", "x"} {
		s, ok := n.publicScreen(LangEN, viewExperience, arg, false)
		if !ok || !strings.Contains(s.text, "1/3") {
			t.Errorf("arg %q: want first job, got %q", arg, s.text[:40])
		}
	}
}

func TestJobPeriod(t *testing.T) {
	now := time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC)
	c := contentFor(LangEN)
	if got, want := jobPeriod(LangEN, c, c.jobs[0], now), "May 2025 — Present · 1 yr 6 mo"; got != want {
		t.Errorf("current job: got %q, want %q", got, want)
	}
	if got, want := jobPeriod(LangEN, c, c.jobs[1], now), "Jul 2023 — Apr 2025 · 1 yr 10 mo"; got != want {
		t.Errorf("past job: got %q, want %q", got, want)
	}
}

func TestAlertsRender(t *testing.T) {
	visit := domain.VisitNotification{
		Kind: domain.VisitKindReturning, VisitNumber: 3,
		IP: "1.2.3.4", Country: "DE", City: "Berlin",
		Page: "/projects/rigla", PageTitle: "Rigla <case>", Source: "linkedin", Campaign: "a&b",
		Language: "de-DE", SiteLanguage: "de", Device: "desktop", OS: "macOS 14", Browser: "Chrome 129",
		Viewport: "1440×900", OccurredAt: time.Date(2026, 10, 9, 14, 22, 0, 0, time.UTC).UnixMilli(),
	}
	text := formatVisit(visit, time.UTC)
	checkTelegramHTML(t, "visit", text)
	for _, want := range []string{"Returning visitor", "3rd visit", "🇩🇪", "Berlin, DE", "linkedin", "14:22"} {
		if !strings.Contains(text, want) {
			t.Errorf("visit alert missing %q:\n%s", want, text)
		}
	}

	checkTelegramHTML(t, "new visit", formatVisit(domain.VisitNotification{Kind: domain.VisitKindNew, VisitNumber: 1}, time.UTC))

	lead := domain.LeadNotification{Name: domain.LeadResumeDownload, Detail: "en", Origin: "hero", Country: "RU", Page: "/"}
	checkTelegramHTML(t, "lead", formatLead(lead, time.UTC))
	checkTelegramHTML(t, "contact lead", formatLead(domain.LeadNotification{Name: domain.LeadContactClick, Detail: "telegram"}, time.UTC))

	checkTelegramHTML(t, "bot user", formatBotUser(&models.User{ID: 42, FirstName: "<x>", Username: "y", LanguageCode: "ru"}, "site", time.UTC))
}

func TestOverviewRenders(t *testing.T) {
	city := "Berlin"
	ov := &domain.Overview{
		Range: domain.OverviewRange{AnalyticsRange: domain.AnalyticsRange{From: "2026-10-03", To: "2026-10-09"}},
		Kpis: domain.OverviewKpis{
			Visitors:        domain.Kpi{Current: 128, Previous: 100},
			Sessions:        domain.Kpi{Current: 160, Previous: 170},
			Pageviews:       domain.Kpi{Current: 420},
			BounceRate:      domain.Kpi{Current: 0.38},
			ResumeDownloads: domain.Kpi{Current: 4},
			ContactClicks:   domain.Kpi{Current: 2},
			ConversionRate:  domain.Kpi{Current: 0.037},
		},
		Timeseries: []domain.TimeseriesPoint{{Visitors: 1}, {Visitors: 5}, {Visitors: 9}},
		Sources:    []domain.SourceRow{{Source: "google", Visitors: 30}},
		Locations:  []domain.LocationRow{{Country: "DE", City: &city, Visitors: 7}, {Country: "DE", Visitors: 3}, {Country: "RU", Visitors: 9}},
		Devices:    []domain.CountRow{{Key: "desktop", Count: 7}, {Key: "mobile", Count: 3}},
		TopPages:   []domain.PageRow{{Page: "/", Views: 300}},
	}
	text := formatOverview(ov, 7, time.UTC)
	checkTelegramHTML(t, "overview", text)
	for _, want := range []string{"Last 7 days", "▲ 28%", "▼ 6%", "▲ new", "🇩🇪 DE <b>10</b>", "38%", "💻 70%", "▂▅█"} {
		if !strings.Contains(text, want) {
			t.Errorf("overview missing %q:\n%s", want, text)
		}
	}

	empty := formatOverview(&domain.Overview{}, 1, time.UTC)
	checkTelegramHTML(t, "empty overview", empty)
}

func TestSessionLine(t *testing.T) {
	country := "US"
	conv, _ := json.Marshal([]string{"resume_download"})
	s := domain.SessionSummary{
		StartedAt: time.Date(2026, 10, 9, 10, 0, 0, 0, time.UTC), Country: &country,
		Device: "mobile", Source: "direct", Pageviews: 1, EngagedSeconds: 75, LandingPage: "/", Conversions: conv,
	}
	line := formatSessionLine(s, time.UTC)
	checkTelegramHTML(t, "session", line)
	if !strings.Contains(line, "🔥") || !strings.Contains(line, "1 page ·") || !strings.Contains(line, "1m 15s") {
		t.Errorf("unexpected session line:\n%s", line)
	}
}

func TestHelpers(t *testing.T) {
	cases := map[string][2]string{
		"flag":     {flag("de"), "🇩🇪"},
		"flagBad":  {flag("Germany"), "🌐"},
		"number":   {formatNumber(1234567), "1 234 567"},
		"negative": {formatNumber(-1000), "-1 000"},
		"truncate": {truncate("Привет мир", 6), "Приве…"},
		"ordinal":  {ordinal(11) + ordinal(22) + ordinal(103), "11th22nd103rd"},
		"spark":    {sparkline([]int64{0, 4, 8}), "▁▅█"},
		"lang":     {string(langFromCode("uk")) + string(langFromCode("de-AT")) + string(langFromCode("fr")), "rudeen"},
	}
	for name, c := range cases {
		if c[0] != c[1] {
			t.Errorf("%s: got %q, want %q", name, c[0], c[1])
		}
	}
}
