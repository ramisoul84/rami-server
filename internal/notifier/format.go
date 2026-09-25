package notifier

import (
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/ramisoul84/rami-server/internal/domain"
)

// formatVisit renders a single visit notification.
//
// Layout:
//
//	👀 New visit
//
//	/  · Rami Suliman — Home
//	🌐 127.0.0.1  ·  Local
//	💻 desktop  ·  1366×651
//	🔗 direct  ·  18:50:06
func formatVisit(v domain.VisitNotification) string {
	var b strings.Builder

	// Header
	b.WriteString("👀 <b>New visit</b>\n\n")

	// Page title (bold) + path
	page := v.Page
	if page == "" {
		page = "/"
	}
	b.WriteString(fmt.Sprintf("<b>%s</b>", html.EscapeString(page)))
	if v.PageTitle != "" && v.PageTitle != page {
		b.WriteString(fmt.Sprintf("\n<i>%s</i>", html.EscapeString(v.PageTitle)))
	}
	b.WriteString("\n\n")

	// Meta lines — one fact per line, aligned by icon
	if loc := formatLocation(v.Country, v.City); loc != "" {
		b.WriteString(fmt.Sprintf("🌐 %s  ·  %s\n",
			html.EscapeString(v.IP),
			html.EscapeString(loc),
		))
	} else if v.IP != "" {
		b.WriteString(fmt.Sprintf("🌐 %s\n", html.EscapeString(v.IP)))
	}

	if v.Device != "" {
		device := v.Device
		if v.Viewport != "" {
			device += "  ·  " + v.Viewport
		}
		b.WriteString(fmt.Sprintf("%s %s\n", deviceIcon(v.Device), html.EscapeString(device)))
	}

	if v.Referrer != "" {
		b.WriteString(fmt.Sprintf("🔁 %s\n", html.EscapeString(shortReferrer(v.Referrer))))
	} else {
		b.WriteString("🔁 direct\n")
	}

	b.WriteString(fmt.Sprintf("🕐 %s\n",
		time.UnixMilli(v.OccurredAt).Format("15:04:05 · 02 Jan")))

	return b.String()
}

// formatStats renders the stats summary.
func formatStats(s *domain.DashboardStats) string {
	if s == nil {
		return "No stats available."
	}

	var b strings.Builder

	// Header
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
		b.WriteString(fmt.Sprintf("🗓  <b>%s</b>  last 30 days\n", formatNumber(total30)))
	}

	// Top pages
	if len(s.TopPages) > 0 {
		b.WriteString("\n<b>Top pages</b>\n")
		for i, p := range s.TopPages {
			page := p.Page
			if page == "" {
				page = "/"
			}
			// Give the path a fixed visual width; truncate long ones
			if len(page) > 40 {
				page = page[:37] + "..."
			}
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

// formatVisits renders the recent-visits reply.
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
		if len(page) > 40 {
			page = page[:37] + "..."
		}

		b.WriteString(fmt.Sprintf("<b>%s</b>\n", html.EscapeString(page)))

		meta := []string{}
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
// HELPERS
// ---------------------------------------------------------------------------

func formatLocation(country, city string) string {
	switch {
	case country == "" && city == "":
		return ""
	case city == "":
		return country
	case country == "":
		return city
	default:
		return country + " · " + city
	}
}

func shortReferrer(ref string) string {
	if ref == "" {
		return "direct"
	}
	// Strip protocol
	ref = strings.TrimPrefix(ref, "https://")
	ref = strings.TrimPrefix(ref, "http://")
	// Strip trailing slash
	ref = strings.TrimSuffix(ref, "/")
	// Truncate
	if len(ref) > 50 {
		ref = ref[:47] + "..."
	}
	return ref
}

func deviceIcon(device string) string {
	switch strings.ToLower(device) {
	case "mobile":
		return "📱"
	default:
		return "💻"
	}
}

func formatNumber(n int64) string {
	// Simple thousands separator using a string reverse approach.
	// 1234 → "1 234"
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var parts []string
	for len(s) > 3 {
		parts = append([]string{s[len(s)-3:]}, parts...)
		s = s[:len(s)-3]
	}
	parts = append([]string{s}, parts...)
	return strings.Join(parts, " ")
}
