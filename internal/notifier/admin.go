package notifier

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-telegram/bot/models"

	"github.com/ramisoul84/rami-server/internal/domain"
)

// Admin routes (callback data after "a:"):
//
//	home            panel: live count + today at a glance
//	st:<days>       analytics for today / 7 / 30 days
//	live            who is on the site right now
//	rec             latest sessions
//	set             alert settings
//	nm:<mode>       set alert mode (all / new / off)
//	dg              toggle the daily digest

const dayLayout = "2006-01-02"

func (n *TelegramNotifier) adminScreen(ctx context.Context, chatID int64, route string) (s screen, toast string) {
	view, arg, _ := strings.Cut(route, ":")
	switch view {
	case "st":
		days, err := strconv.Atoi(arg)
		if err != nil || (days != 1 && days != 7 && days != 30) {
			days = 1
		}
		return n.statsScreen(ctx, days), ""
	case "live":
		return n.liveScreen(ctx), ""
	case "rec":
		return n.recentScreen(ctx), ""
	case "set":
		return n.settingsScreen(ctx, chatID), ""
	case "nm":
		mode := domain.NotifyMode(arg)
		if mode != domain.NotifyAll && mode != domain.NotifyNew && mode != domain.NotifyOff {
			return n.settingsScreen(ctx, chatID), ""
		}
		if err := n.chats.SetNotifyMode(ctx, chatID, mode); err != nil {
			n.log.Error("telegram: set notify mode", "err", err)
			return n.settingsScreen(ctx, chatID), "⚠️ Couldn't save"
		}
		return n.settingsScreen(ctx, chatID), "🔔 " + notifyModeLabel(mode)
	case "dg":
		chat, err := n.chats.GetAdminChat(ctx, chatID)
		if err != nil || chat == nil {
			return n.settingsScreen(ctx, chatID), "⚠️ Send /start first"
		}
		if err := n.chats.SetDigest(ctx, chatID, !chat.Digest); err != nil {
			n.log.Error("telegram: set digest", "err", err)
			return n.settingsScreen(ctx, chatID), "⚠️ Couldn't save"
		}
		if chat.Digest {
			return n.settingsScreen(ctx, chatID), "🌙 Digest off"
		}
		return n.settingsScreen(ctx, chatID), "🌙 Digest on"
	default:
		return n.homeScreen(ctx), ""
	}
}

func (n *TelegramNotifier) backRow() []models.InlineKeyboardButton {
	return row(btn("⬅ Admin", "a:home"), btn("🏠 Menu", viewData(viewMenu)))
}

func (n *TelegramNotifier) updatedLine() string {
	return fmt.Sprintf("\n<i>Updated %s</i>", n.now().In(n.loc).Format("15:04:05"))
}

func (n *TelegramNotifier) errorScreen(what string, err error) screen {
	n.log.Error("telegram admin view failed", "view", what, "err", err)
	return screen{
		text: fmt.Sprintf("⚠️ Couldn't load %s. Try again in a moment.", what),
		kb:   keyboard(row(btn("🔄 Retry", "a:home")), n.backRow()),
	}
}

// dayRange is the inclusive range of the last `days` days, ending today in the bot's timezone.
func (n *TelegramNotifier) dayRange(days int) domain.AnalyticsRange {
	today := n.now().In(n.loc)
	return domain.AnalyticsRange{
		From: today.AddDate(0, 0, -(days - 1)).Format(dayLayout),
		To:   today.Format(dayLayout),
	}
}

// ---------------------------------------------------------------------------
// SCREENS
// ---------------------------------------------------------------------------

func (n *TelegramNotifier) homeScreen(ctx context.Context) screen {
	var b strings.Builder
	b.WriteString("👑 <b>Admin panel</b>\n\n")

	if live, err := n.analytics.Live(ctx); err == nil {
		fmt.Fprintf(&b, "%s <b>%d</b> on the site now\n", liveDot(live.Visitors), live.Visitors)
	}
	if ov, err := n.analytics.Overview(ctx, n.dayRange(1)); err == nil {
		k := ov.Kpis
		fmt.Fprintf(&b, "👥 <b>%s</b> visitors today %s\n", formatNumber(int64(k.Visitors.Current)), delta(k.Visitors))
		fmt.Fprintf(&b, "📄 <b>%s</b> pageviews · 🎯 <b>%s</b> leads\n",
			formatNumber(int64(k.Pageviews.Current)),
			formatNumber(int64(k.ResumeDownloads.Current+k.ContactClicks.Current)))
	} else {
		n.log.Warn("telegram admin home: overview", "err", err)
	}
	b.WriteString(n.updatedLine())

	return screen{b.String(), keyboard(
		row(btn("📊 Today", "a:st:1"), btn("📅 7 days", "a:st:7"), btn("🗓 30 days", "a:st:30")),
		row(btn("🟢 Live now", "a:live"), btn("🕒 Recent", "a:rec")),
		row(btn("🔔 Alerts", "a:set"), urlBtn("📈 Dashboard", n.siteURL+"/admin")),
		row(btn("🔄 Refresh", "a:home"), btn("🏠 Menu", viewData(viewMenu))),
	)}
}

func (n *TelegramNotifier) statsScreen(ctx context.Context, days int) screen {
	ov, err := n.analytics.Overview(ctx, n.dayRange(days))
	if err != nil {
		return n.errorScreen("stats", err)
	}

	period := func(d int, label string) models.InlineKeyboardButton {
		if d == days {
			label = "• " + label + " •"
		}
		return btn(label, "a:st:"+strconv.Itoa(d))
	}
	return screen{formatOverview(ov, days, n.loc) + n.updatedLine(), keyboard(
		row(period(1, "Today"), period(7, "7 days"), period(30, "30 days")),
		row(btn("🔄 Refresh", "a:st:"+strconv.Itoa(days)), btn("🟢 Live", "a:live")),
		n.backRow(),
	)}
}

func (n *TelegramNotifier) liveScreen(ctx context.Context) screen {
	live, err := n.analytics.Live(ctx)
	if err != nil {
		return n.errorScreen("live visitors", err)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%s <b>Live now</b> · %d visitor%s\n<i>active in the last %d s</i>\n",
		liveDot(live.Visitors), live.Visitors, plural(live.Visitors), live.WindowSeconds)
	if len(live.Pages) == 0 {
		b.WriteString("\nNobody's browsing right now. 🌙\n")
	} else {
		b.WriteString("\n")
		for _, p := range live.Pages {
			fmt.Fprintf(&b, "<code>%3d</code>  %s\n", p.Visitors, escape(truncate(p.Page, 40)))
		}
	}
	b.WriteString(n.updatedLine())

	return screen{b.String(), keyboard(
		row(btn("🔄 Refresh", "a:live"), btn("🕒 Recent", "a:rec")),
		n.backRow(),
	)}
}

func (n *TelegramNotifier) recentScreen(ctx context.Context) screen {
	page, err := n.analytics.Sessions(ctx, domain.SessionsQuery{AnalyticsRange: n.dayRange(7), Limit: 8})
	if err != nil {
		return n.errorScreen("sessions", err)
	}

	var b strings.Builder
	b.WriteString("🕒 <b>Recent sessions</b> · last 7 days\n")
	if len(page.Sessions) == 0 {
		b.WriteString("\nNo sessions yet.\n")
	}
	for _, s := range page.Sessions {
		b.WriteString("\n" + formatSessionLine(s, n.loc))
	}
	if page.Total > int64(len(page.Sessions)) {
		fmt.Fprintf(&b, "\n<i>…and %s more on the dashboard</i>\n", formatNumber(page.Total-int64(len(page.Sessions))))
	}
	b.WriteString(n.updatedLine())

	return screen{b.String(), keyboard(
		row(btn("🔄 Refresh", "a:rec"), urlBtn("🔎 All sessions", n.siteURL+"/admin/sessions")),
		n.backRow(),
	)}
}

func (n *TelegramNotifier) settingsScreen(ctx context.Context, chatID int64) screen {
	chat, err := n.chats.GetAdminChat(ctx, chatID)
	if err != nil {
		return n.errorScreen("settings", err)
	}
	if chat == nil {
		return screen{"⚠️ This chat isn't registered for alerts yet — send /start.", keyboard(n.backRow())}
	}

	digest := "off"
	if chat.Digest {
		digest = "on"
	}
	digestAt := "disabled on the server"
	if n.digestHour >= 0 {
		digestAt = fmt.Sprintf("%02d:00 %s", n.digestHour, n.loc.String())
	}

	text := fmt.Sprintf("🔔 <b>Alerts</b>\n\n"+
		"<b>Real-time:</b> %s\n"+
		"<blockquote>🔥 Leads — resume downloads and contact clicks\n"+
		"🆕 New visitors — first visit ever\n"+
		"🔁 Returning visitors — a new session from someone who's been here\n"+
		"🤖 New bot users</blockquote>\n"+
		"<b>Daily digest:</b> %s · %s",
		notifyModeLabel(chat.NotifyMode), digest, digestAt)

	mode := func(m domain.NotifyMode, label string) models.InlineKeyboardButton {
		if chat.NotifyMode == m {
			label = "✓ " + label
		}
		return btn(label, "a:nm:"+string(m))
	}
	digestBtn := "🌙 Digest: off → turn on"
	if chat.Digest {
		digestBtn = "🌙 Digest: on → turn off"
	}
	return screen{text, keyboard(
		row(mode(domain.NotifyAll, "All"), mode(domain.NotifyNew, "New only"), mode(domain.NotifyOff, "Off")),
		row(btn(digestBtn, "a:dg")),
		n.backRow(),
	)}
}

func notifyModeLabel(m domain.NotifyMode) string {
	switch m {
	case domain.NotifyNew:
		return "new visitors + leads"
	case domain.NotifyOff:
		return "off"
	default:
		return "every session + leads"
	}
}

// ---------------------------------------------------------------------------
// DAILY DIGEST
// ---------------------------------------------------------------------------

func (n *TelegramNotifier) runDigest(ctx context.Context) {
	if n.digestHour < 0 || n.digestHour > 23 || n.chats == nil {
		return
	}
	for {
		now := n.now().In(n.loc)
		next := time.Date(now.Year(), now.Month(), now.Day(), n.digestHour, 0, 0, 0, n.loc)
		if !next.After(now) {
			next = next.AddDate(0, 0, 1)
		}
		timer := time.NewTimer(time.Until(next))
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
		n.sendDigest(ctx)
	}
}

func (n *TelegramNotifier) sendDigest(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	ov, err := n.analytics.Overview(ctx, n.dayRange(1))
	if err != nil {
		n.log.Error("telegram digest: overview", "err", err)
		return
	}
	text := "🌙 <b>Daily digest</b> · " + n.now().In(n.loc).Format("Mon 2 Jan") + "\n\n" + formatOverview(ov, 1, n.loc)
	s := screen{text, keyboard(row(btn("📅 Last 7 days", "a:st:7:n"), urlBtn("📈 Dashboard", n.siteURL+"/admin")))}

	chats, err := n.chats.ListAdminChats(ctx)
	if err != nil {
		n.log.Error("telegram digest: list chats", "err", err)
		return
	}
	for _, c := range chats {
		if !c.Digest {
			continue
		}
		n.show(ctx, c.ChatID, 0, s)
	}
	n.log.Info("telegram digest sent")
}
