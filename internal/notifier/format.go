package notifier

import (
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/ramisoul84/rami-server/internal/domain"
)

// ---------------------------------------------------------------------------
// VISIT NOTIFICATION
// ---------------------------------------------------------------------------

// formatVisit renders a visit notification with a compact, readable layout.
//
// Example output:
//
//	👀 New visit
//
//	/projects/bristol — Case study
//
//	🌍 Amsterdam, NL · 143.92.44.18
//	📱 iOS 17 · Safari 17 · mobile · 390×844
//	🔗 google.com
//	🕐 14:22:08 · 27 Sep
func formatVisit(v domain.VisitNotification) string {
	var b strings.Builder

	b.WriteString("👀 <b>New visit</b>\n\n")

	// Page — path bold, optional title after an em-dash
	page := v.Page
	if page == "" {
		page = "/"
	}
	b.WriteString(fmt.Sprintf("<b>%s</b>", html.EscapeString(page)))
	if v.PageTitle != "" && v.PageTitle != page {
		b.WriteString(fmt.Sprintf(" — %s", html.EscapeString(v.PageTitle)))
	}
	b.WriteString("\n\n")

	// Location + IP — combined into one line
	if loc := formatLocation(v.Country, v.City); loc != "" {
		b.WriteString(fmt.Sprintf("🌍 %s · <code>%s</code>\n",
			html.EscapeString(loc),
			html.EscapeString(v.IP),
		))
	} else if v.IP != "" {
		b.WriteString(fmt.Sprintf("🌐 <code>%s</code>\n", html.EscapeString(v.IP)))
	}

	// Device stack: OS · Browser · device type · viewport
	if line := formatDeviceLine(v); line != "" {
		b.WriteString(line)
	}

	// Referrer — "direct" if empty
	ref := v.Referrer
	if strings.TrimSpace(ref) == "" {
		ref = "direct"
	}
	b.WriteString(fmt.Sprintf("🔗 %s\n", html.EscapeString(shortReferrer(ref))))

	// Time
	b.WriteString(fmt.Sprintf("🕐 %s",
		time.UnixMilli(v.OccurredAt).Format("15:04:05 · 02 Jan")))

	return b.String()
}

// formatDeviceLine builds the "OS · Browser · device · viewport" line.
// Returns an empty string if there's nothing to show.
func formatDeviceLine(v domain.VisitNotification) string {
	parts := make([]string, 0, 4)

	if v.OS != "" && v.OS != "Unknown" {
		parts = append(parts, v.OS)
	}
	if v.Browser != "" && v.Browser != "Unknown" {
		parts = append(parts, v.Browser)
	}
	if v.Device != "" {
		parts = append(parts, v.Device)
	}
	if v.Viewport != "" {
		parts = append(parts, v.Viewport)
	}

	if len(parts) == 0 {
		return ""
	}

	return fmt.Sprintf("%s %s\n",
		deviceIcon(v.Device),
		html.EscapeString(strings.Join(parts, " · ")),
	)
}

// ---------------------------------------------------------------------------
// STATS
// ---------------------------------------------------------------------------

// formatStats renders the /stats reply.
func formatStats(s *domain.DashboardStats) string {
	if s == nil {
		return "No stats available."
	}

	var b strings.Builder
	b.WriteString("📊 <b>Site analytics</b>\n\n")

	if s.Summary != nil {
		b.WriteString(fmt.Sprintf(
			"👥  <b>%s</b>  visitors\n"+
				"📄  <b>%s</b>  page views\n",
			formatNumber(s.Summary.TotalVisitors),
			formatNumber(s.Summary.TotalPageViews),
		))
	}

	if s.Today != nil {
		b.WriteString(fmt.Sprintf(
			"📅  <b>%s</b>  today  ·  <b>%s</b>  views\n",
			formatNumber(s.Today.Visitors),
			formatNumber(s.Today.PageViews),
		))
	}

	if len(s.VisitorsByDay) > 0 {
		var total30 int64
		for _, d := range s.VisitorsByDay {
			total30 += d.Visitors
		}
		b.WriteString(fmt.Sprintf("🗓  <b>%s</b>  last 30 days\n",
			formatNumber(total30)))
	}

	if len(s.TopPages) > 0 {
		b.WriteString("\n<b>Top pages</b>\n")
		for i, p := range s.TopPages {
			page := p.Page
			if page == "" {
				page = "/"
			}
			page = truncate(page, 40)
			b.WriteString(fmt.Sprintf(
				"<code>%d.</code>  %s  —  <b>%s</b>\n",
				i+1,
				html.EscapeString(page),
				formatNumber(p.Count),
			))
		}
	}

	return b.String()
}

// ---------------------------------------------------------------------------
// RECENT VISITS
// ---------------------------------------------------------------------------

// formatVisits renders the /visits reply.
func formatVisits(visits []domain.Visit) string {
	if len(visits) == 0 {
		return "No visits yet."
	}

	var b strings.Builder
	b.WriteString("🕒 <b>Recent visits</b>\n\n")

	for _, v := range visits {
		page := v.Page
		if page == "" {
			page = "/"
		}
		page = truncate(page, 40)

		b.WriteString(fmt.Sprintf("<b>%s</b>\n", html.EscapeString(page)))

		meta := make([]string, 0, 3)
		if loc := formatLocation(v.Country, v.City); loc != "" {
			meta = append(meta, loc)
		}
		if v.IPAddress != "" {
			meta = append(meta, v.IPAddress)
		}
		meta = append(meta, v.CreatedAt.Format("15:04 · 02 Jan"))

		b.WriteString(fmt.Sprintf("<i>%s</i>\n\n",
			html.EscapeString(strings.Join(meta, "  ·  "))))
	}

	return b.String()
}

// ---------------------------------------------------------------------------
// SHARED HELPERS
// ---------------------------------------------------------------------------

// formatLocation builds "City, Country" from its parts.
// Handles the case where only one of the two is present.
func formatLocation(country, city string) string {
	country = strings.TrimSpace(country)
	city = strings.TrimSpace(city)

	switch {
	case country == "" && city == "":
		return ""
	case city == "":
		return country
	case country == "":
		return city
	default:
		return city + ", " + country
	}
}

// shortReferrer trims the protocol, trailing slash, and truncates long URLs.
func shortReferrer(ref string) string {
	if ref == "" {
		return "direct"
	}
	ref = strings.TrimPrefix(ref, "https://")
	ref = strings.TrimPrefix(ref, "http://")
	ref = strings.TrimPrefix(ref, "www.")
	ref = strings.TrimSuffix(ref, "/")
	return truncate(ref, 50)
}

// deviceIcon picks the emoji for a device type.
func deviceIcon(device string) string {
	switch strings.ToLower(device) {
	case "mobile":
		return "📱"
	case "bot":
		return "🤖"
	case "tablet":
		return "📱"
	default:
		return "💻"
	}
}

// truncate shortens a string to max length, adding an ellipsis if cut.
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}

// formatNumber adds a thin space as a thousands separator.
// Example: 1234567 → "1 234 567"
func formatNumber(n int64) string {
	negative := n < 0
	if negative {
		n = -n
	}

	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		if negative {
			return "-" + s
		}
		return s
	}

	var parts []string
	for len(s) > 3 {
		parts = append([]string{s[len(s)-3:]}, parts...)
		s = s[:len(s)-3]
	}
	parts = append([]string{s}, parts...)

	result := strings.Join(parts, " ")
	if negative {
		return "-" + result
	}
	return result
}
