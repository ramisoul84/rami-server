package notifier

import (
	"context"
	"errors"
	"fmt"
	"html"
	"net"
	"net/http"
	"strings"
	"time"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"github.com/ramisoul84/rami-server/internal/domain"
	"github.com/ramisoul84/rami-server/pkg/logger"
)

const (
	pollTimeout       = 60 * time.Second
	sendAttempts      = 3
	sendAttemptBudget = 20 * time.Second
)

var (
	ErrNotConfigured = errors.New("telegram: not configured")
	ErrSendFailed    = errors.New("telegram: send failed")
)

var (
	_ Notifier = (*TelegramNotifier)(nil)
	_ Runnable = (*TelegramNotifier)(nil)
)

type TelegramNotifier struct {
	bot            *tgbot.Bot
	log            *logger.Logger
	statsProvider  StatsProvider
	visitsProvider RecentVisitsProvider
	chats          ChatStore
	admins         map[string]bool
}

func NewTelegramNotifier(
	botToken string,
	adminUsernames []string,
	log *logger.Logger,
	statsProvider StatsProvider,
	visitsProvider RecentVisitsProvider,
	chats ChatStore,
) (*TelegramNotifier, error) {
	if botToken == "" {
		return nil, nil
	}

	transport := &http.Transport{
		Proxy:               http.ProxyFromEnvironment,
		DialContext:         (&net.Dialer{Timeout: 15 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		ForceAttemptHTTP2:   true,
		TLSHandshakeTimeout: 15 * time.Second,
		IdleConnTimeout:     90 * time.Second,
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 4,
	}
	httpClient := &http.Client{
		Timeout:   pollTimeout + 10*time.Second,
		Transport: transport,
	}

	bot, err := tgbot.New(botToken,
		tgbot.WithSkipGetMe(),
		tgbot.WithHTTPClient(pollTimeout, httpClient),
	)
	if err != nil {
		return nil, fmt.Errorf("telegram: create bot: %w", err)
	}

	admins := make(map[string]bool, len(adminUsernames))
	for _, a := range adminUsernames {
		if name := strings.ToLower(strings.TrimSpace(a)); name != "" {
			admins[name] = true
		}
	}

	n := &TelegramNotifier{
		bot:            bot,
		log:            log,
		statsProvider:  statsProvider,
		visitsProvider: visitsProvider,
		chats:          chats,
		admins:         admins,
	}

	bot.RegisterHandler(tgbot.HandlerTypeMessageText, "start", tgbot.MatchTypeCommand, n.handleStart)
	bot.RegisterHandler(tgbot.HandlerTypeMessageText, "menu", tgbot.MatchTypeCommand, n.handleMenu)
	bot.RegisterHandler(tgbot.HandlerTypeMessageText, "stats", tgbot.MatchTypeCommand, n.handleStats)
	bot.RegisterHandler(tgbot.HandlerTypeMessageText, "visits", tgbot.MatchTypeCommand, n.handleRecentVisits)
	bot.RegisterHandler(tgbot.HandlerTypeMessageText, "", tgbot.MatchTypeContains, n.handleDefault)

	go n.setCommands()

	return n, nil
}

func (n *TelegramNotifier) Enabled() bool { return n != nil && n.bot != nil }

func (n *TelegramNotifier) Run(ctx context.Context) error {
	if !n.Enabled() {
		return nil
	}
	n.log.Info("telegram bot polling", "admins", len(n.admins))
	n.bot.Start(ctx)
	return nil
}

func (n *TelegramNotifier) NotifyVisit(ctx context.Context, v domain.VisitNotification) error {
	if !n.Enabled() {
		return ErrNotConfigured
	}

	chatIDs, err := n.chats.ListAdminChats(ctx)
	if err != nil {
		return fmt.Errorf("telegram: list chats: %w", err)
	}
	if len(chatIDs) == 0 {
		return errors.New("telegram: no admin chats registered — send /start to the bot")
	}

	text := formatVisit(v)
	var errs []error

	for _, chatID := range chatIDs {
		if err := n.sendMessage(ctx, &tgbot.SendMessageParams{
			ChatID:    chatID,
			Text:      text,
			ParseMode: models.ParseModeHTML,
		}); err != nil {
			errs = append(errs, fmt.Errorf("chat %d: %w", chatID, err))
		}
	}
	return errors.Join(errs...)
}

// ---------------------------------------------------------------------------
// COMMANDS
// ---------------------------------------------------------------------------

func (n *TelegramNotifier) setCommands() {
	commands := []models.BotCommand{
		{Command: "start", Description: "Welcome and menu"},
		{Command: "menu", Description: "Show the menu"},
		{Command: "stats", Description: "Site traffic stats (admin)"},
		{Command: "visits", Description: "Recent visits (admin)"},
	}

	for attempt := 0; attempt < 3; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		_, err := n.bot.SetMyCommands(ctx, &tgbot.SetMyCommandsParams{Commands: commands})
		cancel()
		if err == nil {
			n.log.Info("telegram commands registered")
			return
		}
		n.log.Warn("failed to register telegram commands", "attempt", attempt+1, "err", err)
		time.Sleep(time.Duration(attempt+1) * time.Second)
	}
}

func (n *TelegramNotifier) handleStart(ctx context.Context, b *tgbot.Bot, u *models.Update) {
	msg := u.Message
	if msg == nil || msg.From == nil {
		return
	}
	isAdmin := n.isAdmin(msg.From.Username)

	// Save this admin's chat ID so we can send notifications to them.
	if isAdmin && n.chats != nil {
		if err := n.chats.UpsertAdminChat(ctx, msg.From.Username, msg.Chat.ID, msg.From.FirstName); err != nil {
			n.log.Error("failed to store admin chat", "err", err, "username", msg.From.Username)
		} else {
			n.log.Info("admin chat registered", "username", msg.From.Username, "chat_id", msg.Chat.ID)
		}
	}

	name := msg.From.FirstName
	if name == "" {
		name = msg.From.Username
	}
	text := fmt.Sprintf(
		"👋 Hey <b>%s</b>, welcome to Rami Suliman's portfolio bot.\n\n"+
			"Use the menu below to explore the site.",
		html.EscapeString(name),
	)
	if isAdmin {
		text += "\n\n👑 Admin commands: /stats · /visits"
	}
	n.sendReply(ctx, msg.Chat.ID, text, n.menuKeyboard(isAdmin))
}

func (n *TelegramNotifier) handleMenu(ctx context.Context, b *tgbot.Bot, u *models.Update) {
	msg := u.Message
	if msg == nil || msg.From == nil {
		return
	}
	n.sendReply(ctx, msg.Chat.ID, "Menu:", n.menuKeyboard(n.isAdmin(msg.From.Username)))
}

func (n *TelegramNotifier) handleStats(ctx context.Context, b *tgbot.Bot, u *models.Update) {
	msg := u.Message
	if msg == nil || msg.From == nil {
		return
	}
	if !n.isAdmin(msg.From.Username) {
		n.sendReply(ctx, msg.Chat.ID, "⛔ Access denied.", nil)
		return
	}
	if n.statsProvider == nil {
		n.sendReply(ctx, msg.Chat.ID, "Stats not available.", nil)
		return
	}

	stats, err := n.statsProvider(ctx)
	if err != nil {
		n.log.Error("stats provider failed", "err", err, "username", msg.From.Username)
		n.sendReply(ctx, msg.Chat.ID, "Failed to load stats.", nil)
		return
	}

	n.sendReply(ctx, msg.Chat.ID, formatStats(stats), nil)
}

func (n *TelegramNotifier) handleRecentVisits(ctx context.Context, b *tgbot.Bot, u *models.Update) {
	msg := u.Message
	if msg == nil || msg.From == nil {
		return
	}
	if !n.isAdmin(msg.From.Username) {
		n.sendReply(ctx, msg.Chat.ID, "⛔ Access denied.", nil)
		return
	}
	if n.visitsProvider == nil {
		n.sendReply(ctx, msg.Chat.ID, "Visits not available.", nil)
		return
	}

	visits, err := n.visitsProvider(ctx, 10)
	if err != nil {
		n.log.Error("visits provider failed", "err", err, "username", msg.From.Username)
		n.sendReply(ctx, msg.Chat.ID, "Failed to load visits.", nil)
		return
	}
	if len(visits) == 0 {
		n.sendReply(ctx, msg.Chat.ID, "No visits yet.", nil)
		return
	}

	n.sendReply(ctx, msg.Chat.ID, formatVisits(visits), nil)
}

func (n *TelegramNotifier) handleDefault(ctx context.Context, b *tgbot.Bot, u *models.Update) {
	msg := u.Message
	if msg == nil || msg.From == nil {
		return
	}
	if key, ok := resolveSection(msg.Text); ok {
		n.sendReply(ctx, msg.Chat.ID, sectionContent(key), nil)
		return
	}
	if strings.HasPrefix(strings.TrimSpace(msg.Text), "/") {
		n.sendReply(ctx, msg.Chat.ID, "Use the menu below.", n.menuKeyboard(n.isAdmin(msg.From.Username)))
		return
	}
	n.sendReply(ctx, msg.Chat.ID, "Tap a section to learn more:", n.menuKeyboard(n.isAdmin(msg.From.Username)))
}

// ---------------------------------------------------------------------------
// INTERNALS
// ---------------------------------------------------------------------------

func (n *TelegramNotifier) isAdmin(username string) bool {
	if username == "" {
		return false
	}
	return n.admins[strings.ToLower(username)]
}

func (n *TelegramNotifier) sendReply(ctx context.Context, chatID int64, text string, kb *models.ReplyKeyboardMarkup) {
	params := &tgbot.SendMessageParams{
		ChatID:    chatID,
		Text:      text,
		ParseMode: models.ParseModeHTML,
	}
	if kb != nil {
		params.ReplyMarkup = kb
	}
	if err := n.sendMessage(ctx, params); err != nil {
		n.log.Error("telegram reply failed", "err", err, "chat_id", chatID)
	}
}

func (n *TelegramNotifier) sendMessage(ctx context.Context, params *tgbot.SendMessageParams) error {
	var lastErr error
	for attempt := 0; attempt < sendAttempts; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(attempt) * 2 * time.Second):
			}
		}
		attemptCtx, cancel := context.WithTimeout(ctx, sendAttemptBudget)
		_, lastErr = n.bot.SendMessage(attemptCtx, params)
		cancel()
		if lastErr == nil {
			return nil
		}
	}
	return fmt.Errorf("%w: %v", ErrSendFailed, lastErr)
}
