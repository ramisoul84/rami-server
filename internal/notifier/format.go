package notifier

import (
	"encoding/json"
	"fmt"
	"html"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/go-telegram/bot/models"

	"github.com/ramisoul84/rami-server/internal/domain"
)

// ---------------------------------------------------------------------------
// VISIT ALERT
// ---------------------------------------------------------------------------

// formatVisit renders the alert for a session's first pageview.
//
// Example output:
//
//	🆕 New visitor
//	🇩🇪 Berlin, DE
//
//	📄 /projects/rigla — Rigla case study
//	🔗 linkedin · utm spring-2026
//	💻 macOS 14 · Chrome 129 · 1440×900
//	🗣 de-DE · site in de
//	🕐 14:22 · 143.92.44.18
func formatVisit(v domain.VisitNotification, loc *time.Location) string {
	var b strings.Builder

	if v.Kind == domain.VisitKindReturning {
		fmt.Fprintf(&b, "🔁 <b>Returning visitor</b> · %s visit\n", ordinal(v.VisitNumber))
	} else {
		b.WriteString("🆕 <b>New visitor</b>\n")
	}
	if loc := formatLocation(v.Country, v.City); loc != "" {
		fmt.Fprintf(&b, "%s %s\n", flag(v.Country), escape(loc))
	}
	b.WriteString("\n")

	page := v.Page
	if page == "" {
		page = "/"
	}
	fmt.Fprintf(&b, "📄 <b>%s</b>", escape(truncate(page, 60)))
	if v.PageTitle != "" && v.PageTitle != page {
		fmt.Fprintf(&b, " — %s", escape(truncate(v.PageTitle, 60)))
	}
	b.WriteString("\n")

	source := v.Source
	if source == "" {
		source = shortReferrer(v.Referrer)
	}
	fmt.Fprintf(&b, "🔗 %s", escape(source))
	if v.Campaign != "" {
		fmt.Fprintf(&b, " · utm %s", escape(truncate(v.Campaign, 40)))
	}
	b.WriteString("\n")

	if line := formatDeviceLine(v.Device, v.OS, v.Browser, v.Viewport); line != "" {
		b.WriteString(line + "\n")
	}

	var langs []string
	if v.Language != "" {
		langs = append(langs, v.Language)
	}
	if v.SiteLanguage != "" {
		langs = append(langs, "site in "+v.SiteLanguage)
	}
	if len(langs) > 0 {
		fmt.Fprintf(&b, "🗣 %s\n", escape(strings.Join(langs, " · ")))
	}

	fmt.Fprintf(&b, "🕐 %s", time.UnixMilli(v.OccurredAt).In(loc).Format("15:04"))
	if v.IP != "" {
		fmt.Fprintf(&b, " · <code>%s</code>", escape(v.IP))
	}
	return b.String()
}

// ---------------------------------------------------------------------------
// LEAD ALERT
// ---------------------------------------------------------------------------

func formatLead(l domain.LeadNotification, loc *time.Location) string {
	var b strings.Builder

	switch l.Name {
	case domain.LeadResumeDownload:
		b.WriteString("🔥 <b>Lead · resume downloaded</b>")
		if l.Detail != "" {
			fmt.Fprintf(&b, " (%s)", escape(strings.ToUpper(l.Detail)))
		}
	default:
		b.WriteString("🔥 <b>Lead · contact click</b>")
		if l.Detail != "" {
			fmt.Fprintf(&b, " → %s", escape(l.Detail))
		}
	}
	b.WriteString("\n")

	if place := formatLocation(l.Country, l.City); place != "" {
		fmt.Fprintf(&b, "%s %s\n", flag(l.Country), escape(place))
	}
	b.WriteString("\n")

	page := l.Page
	if page == "" {
		page = "/"
	}
	fmt.Fprintf(&b, "📄 %s", escape(truncate(page, 60)))
	if l.Origin != "" {
		fmt.Fprintf(&b, " · from %s", escape(l.Origin))
	}
	b.WriteString("\n")
	if l.Source != "" {
		fmt.Fprintf(&b, "🔗 %s\n", escape(l.Source))
	}
	if line := formatDeviceLine(l.Device, l.OS, l.Browser, ""); line != "" {
		b.WriteString(line + "\n")
	}
	fmt.Fprintf(&b, "🕐 %s", time.UnixMilli(l.OccurredAt).In(loc).Format("15:04"))
	return b.String()
}

// ---------------------------------------------------------------------------
// BOT USER ALERT
// ---------------------------------------------------------------------------

func formatBotUser(u *models.User, payload string, loc *time.Location) string {
	var b strings.Builder
	b.WriteString("🤖 <b>New bot user</b>\n\n")

	name := strings.TrimSpace(u.FirstName + " " + u.LastName)
	if name == "" {
		name = "Someone"
	}
	fmt.Fprintf(&b, "👤 <a href=\"tg://user?id=%d\">%s</a>", u.ID, escape(name))
	if u.Username != "" {
		fmt.Fprintf(&b, " · @%s", escape(u.Username))
	}
	b.WriteString("\n")
	if u.LanguageCode != "" {
		fmt.Fprintf(&b, "🗣 %s\n", escape(u.LanguageCode))
	}
	if payload != "" {
		fmt.Fprintf(&b, "🔗 via <code>%s</code>\n", escape(truncate(payload, 40)))
	}
	fmt.Fprintf(&b, "🕐 %s", time.Now().In(loc).Format("15:04"))
	return b.String()
}

// ---------------------------------------------------------------------------
// ANALYTICS OVERVIEW
// ---------------------------------------------------------------------------

// formatOverview renders the stats card for a 1/7/30-day range.
func formatOverview(ov *domain.Overview, days int, loc *time.Location) string {
	k := ov.Kpis
	var b strings.Builder

	title, versus := "Today", "vs yesterday"
	if days > 1 {
		title = fmt.Sprintf("Last %d days", days)
		versus = fmt.Sprintf("vs previous %d days", days)
	}
	fmt.Fprintf(&b, "📊 <b>%s</b>\n<i>%s · %s</i>\n\n", title, rangeLabel(ov.Range.AnalyticsRange, days), versus)

	if k.Visitors.Current == 0 && k.Pageviews.Current == 0 {
		b.WriteString("No visits in this period yet. 🌙\n")
		return b.String()
	}

	fmt.Fprintf(&b, "👥 Visitors  <b>%s</b>  %s\n", formatNumber(int64(k.Visitors.Current)), delta(k.Visitors))
	fmt.Fprintf(&b, "🔄 Sessions  <b>%s</b>  %s\n", formatNumber(int64(k.Sessions.Current)), delta(k.Sessions))
	fmt.Fprintf(&b, "📄 Pageviews  <b>%s</b>  %s\n", formatNumber(int64(k.Pageviews.Current)), delta(k.Pageviews))
	fmt.Fprintf(&b, "⏱ Avg. engaged  <b>%s</b>\n", formatDuration(int64(k.AvgEngagedSeconds.Current)))
	fmt.Fprintf(&b, "↩️ Bounce  <b>%s</b>  ·  🔁 Returning  <b>%s</b>\n", percent(k.BounceRate.Current), percent(k.ReturningRate.Current))

	leads := int64(k.ResumeDownloads.Current + k.ContactClicks.Current)
	fmt.Fprintf(&b, "🎯 Leads  <b>%d</b>", leads)
	if leads > 0 {
		fmt.Fprintf(&b, "  (📄 %d resume · ✉️ %d contact · %s conv.)",
			int64(k.ResumeDownloads.Current), int64(k.ContactClicks.Current), percent(k.ConversionRate.Current))
	}
	b.WriteString("\n")

	if days > 1 && len(ov.Timeseries) > 1 {
		values := make([]int64, len(ov.Timeseries))
		for i, p := range ov.Timeseries {
			values[i] = p.Visitors
		}
		fmt.Fprintf(&b, "\n📈 <code>%s</code>\n", sparkline(values))
	}

	var details strings.Builder
	if len(ov.Sources) > 0 {
		parts := make([]string, 0, 5)
		for _, s := range topN(ov.Sources, 5) {
			parts = append(parts, fmt.Sprintf("%s <b>%d</b>", escape(s.Source), s.Visitors))
		}
		details.WriteString("🔗 <b>Sources</b>\n" + strings.Join(parts, " · ") + "\n\n")
	}
	if countries := countryTotals(ov.Locations); len(countries) > 0 {
		parts := make([]string, 0, 5)
		for _, c := range topN(countries, 5) {
			parts = append(parts, fmt.Sprintf("%s %s <b>%d</b>", flag(c.Key), escape(c.Key), c.Count))
		}
		details.WriteString("🌍 <b>Countries</b>\n" + strings.Join(parts, " · ") + "\n\n")
	}
	if len(ov.Devices) > 0 {
		var total int64
		for _, d := range ov.Devices {
			total += d.Count
		}
		parts := make([]string, 0, len(ov.Devices))
		for _, d := range ov.Devices {
			parts = append(parts, fmt.Sprintf("%s %s", deviceIcon(d.Key), percent(float64(d.Count)/float64(max(total, 1)))))
		}
		details.WriteString("📱 <b>Devices</b>\n" + strings.Join(parts, " · ") + "\n\n")
	}
	if len(ov.TopPages) > 0 {
		details.WriteString("📄 <b>Top pages</b>")
		for i, p := range topN(ov.TopPages, 5) {
			page := p.Page
			if page == "" {
				page = "/"
			}
			fmt.Fprintf(&details, "\n%d. %s — <b>%d</b>", i+1, escape(truncate(page, 36)), p.Views)
		}
	}
	if d := strings.TrimSpace(details.String()); d != "" {
		b.WriteString("\n<blockquote expandable>" + d + "</blockquote>\n")
	}
	return b.String()
}

// formatSessionLine is one entry of the "Recent sessions" list.
func formatSessionLine(s domain.SessionSummary, loc *time.Location) string {
	var b strings.Builder

	icon := "🆕"
	if s.IsReturning {
		icon = "🔁"
	}
	var conversions []string
	_ = json.Unmarshal(s.Conversions, &conversions)
	if len(conversions) > 0 {
		icon = "🔥"
	}

	place := formatLocation(deref(s.Country), deref(s.City))
	if place == "" {
		place = "Unknown location"
	}
	fmt.Fprintf(&b, "%s %s %s · <i>%s</i>\n", icon, flag(deref(s.Country)), escape(place), s.StartedAt.In(loc).Format("02 Jan 15:04"))

	landing := s.LandingPage
	if landing == "" {
		landing = "/"
	}
	meta := []string{
		fmt.Sprintf("%s %s", deviceIcon(s.Device), escape(s.Source)),
		fmt.Sprintf("%d page%s", s.Pageviews, plural(s.Pageviews)),
	}
	if s.EngagedSeconds > 0 {
		meta = append(meta, formatDuration(s.EngagedSeconds))
	}
	fmt.Fprintf(&b, "    %s · %s\n", escape(truncate(landing, 30)), strings.Join(meta, " · "))
	if len(conversions) > 0 {
		fmt.Fprintf(&b, "    🎯 %s\n", escape(strings.Join(conversions, ", ")))
	}
	return b.String()
}

// ---------------------------------------------------------------------------
// SHARED HELPERS
// ---------------------------------------------------------------------------

func escape(s string) string { return html.EscapeString(s) }

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func formatDeviceLine(device, os, browser, viewport string) string {
	parts := make([]string, 0, 4)
	for _, p := range []string{os, browser} {
		if p != "" && p != "Unknown" {
			parts = append(parts, p)
		}
	}
	if device != "" {
		parts = append(parts, device)
	}
	if viewport != "" {
		parts = append(parts, viewport)
	}
	if len(parts) == 0 {
		return ""
	}
	return deviceIcon(device) + " " + escape(strings.Join(parts, " · "))
}

// formatLocation builds "City, Country" from its parts.
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

// flag turns an ISO 3166-1 alpha-2 code into its flag emoji; anything else gets a globe.
func flag(code string) string {
	code = strings.ToUpper(strings.TrimSpace(code))
	if len(code) != 2 || code[0] < 'A' || code[0] > 'Z' || code[1] < 'A' || code[1] > 'Z' {
		return "🌐"
	}
	return string([]rune{rune(code[0]-'A') + 0x1F1E6, rune(code[1]-'A') + 0x1F1E6})
}

// shortReferrer trims the protocol, trailing slash, and truncates long URLs.
func shortReferrer(ref string) string {
	if strings.TrimSpace(ref) == "" {
		return "direct"
	}
	ref = strings.TrimPrefix(ref, "https://")
	ref = strings.TrimPrefix(ref, "http://")
	ref = strings.TrimPrefix(ref, "www.")
	ref = strings.TrimSuffix(ref, "/")
	return truncate(ref, 50)
}

func deviceIcon(device string) string {
	switch strings.ToLower(device) {
	case "mobile":
		return "📱"
	case "tablet":
		return "📟"
	case "bot":
		return "🤖"
	default:
		return "💻"
	}
}

func liveDot(n int64) string {
	if n > 0 {
		return "🟢"
	}
	return "⚪"
}

// delta renders the change against the previous period: "▲ 12%", "▼ 3%", "· 0%", or "new".
func delta(k domain.Kpi) string {
	switch {
	case k.Previous == 0 && k.Current == 0:
		return ""
	case k.Previous == 0:
		return "<i>▲ new</i>"
	}
	change := (k.Current - k.Previous) / k.Previous * 100
	switch {
	case math.Abs(change) < 0.5:
		return "<i>· 0%</i>"
	case change > 0:
		return fmt.Sprintf("<i>▲ %.0f%%</i>", change)
	default:
		return fmt.Sprintf("<i>▼ %.0f%%</i>", -change)
	}
}

func percent(fraction float64) string {
	return fmt.Sprintf("%.0f%%", fraction*100)
}

// sparkline draws values as a row of block characters.
func sparkline(values []int64) string {
	bars := []rune("▁▂▃▄▅▆▇█")
	var peak int64
	for _, v := range values {
		peak = max(peak, v)
	}
	out := make([]rune, len(values))
	for i, v := range values {
		if peak == 0 {
			out[i] = bars[0]
			continue
		}
		out[i] = bars[int(float64(v)/float64(peak)*float64(len(bars)-1)+0.5)]
	}
	return string(out)
}

func rangeLabel(r domain.AnalyticsRange, days int) string {
	from, err1 := time.Parse(dayLayout, r.From)
	to, err2 := time.Parse(dayLayout, r.To)
	if err1 != nil || err2 != nil {
		return r.From + " – " + r.To
	}
	if days == 1 {
		return to.Format("Mon 2 Jan")
	}
	return from.Format("2 Jan") + " – " + to.Format("2 Jan")
}

func countryTotals(rows []domain.LocationRow) []domain.CountRow {
	totals := map[string]int64{}
	for _, r := range rows {
		totals[r.Country] += r.Visitors
	}
	out := make([]domain.CountRow, 0, len(totals))
	for k, v := range totals {
		if k != "" {
			out = append(out, domain.CountRow{Key: k, Count: v})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Key < out[j].Key
	})
	return out
}

func topN[T any](rows []T, n int) []T {
	if len(rows) > n {
		return rows[:n]
	}
	return rows
}

func formatDuration(seconds int64) string {
	switch {
	case seconds < 60:
		return fmt.Sprintf("%ds", seconds)
	case seconds < 3600:
		return fmt.Sprintf("%dm %02ds", seconds/60, seconds%60)
	default:
		return fmt.Sprintf("%dh %02dm", seconds/3600, (seconds%3600)/60)
	}
}

func ordinal(n int) string {
	suffix := "th"
	if n%100 < 11 || n%100 > 13 {
		switch n % 10 {
		case 1:
			suffix = "st"
		case 2:
			suffix = "nd"
		case 3:
			suffix = "rd"
		}
	}
	return fmt.Sprintf("%d%s", n, suffix)
}

func plural(n int64) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// truncate shortens s to max runes, adding an ellipsis if cut.
func truncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	if max <= 1 {
		return string(r[:max])
	}
	return string(r[:max-1]) + "…"
}

// formatNumber adds a space as a thousands separator: 1234567 → "1 234 567".
func formatNumber(n int64) string {
	sign := ""
	if n < 0 {
		sign, n = "-", -n
	}
	s := fmt.Sprintf("%d", n)
	var parts []string
	for len(s) > 3 {
		parts = append([]string{s[len(s)-3:]}, parts...)
		s = s[:len(s)-3]
	}
	parts = append([]string{s}, parts...)
	return sign + strings.Join(parts, " ")
}
