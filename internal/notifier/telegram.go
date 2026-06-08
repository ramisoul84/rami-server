package notifier

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
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
	handlerTimeout    = 30 * time.Second
)

var (
	ErrNotConfigured = errors.New("telegram: not configured")
	ErrSendFailed    = errors.New("telegram: send failed")
)

var (
	_ Notifier = (*TelegramNotifier)(nil)
	_ Runnable = (*TelegramNotifier)(nil)
)

// TelegramConfig is everything the bot needs besides its collaborators.
type TelegramConfig struct {
	BotToken string
	Admins   []string // Telegram usernames, without @
	SiteURL  string   // public site root, no trailing slash
	// DigestHour is the local hour (0–23) of the daily summary; negative disables it.
	DigestHour int
	Location   *time.Location
}

type TelegramNotifier struct {
	bot        *tgbot.Bot
	log        *logger.Logger
	analytics  Analytics
	chats      ChatStore
	admins     map[string]bool
	siteURL    string
	digestHour int
	loc        *time.Location
	now        func() time.Time

	langs       sync.Map // user ID → Lang chosen in the bot
	resumeFiles sync.Map // Lang → Telegram file_id, so the PDF is uploaded once
	adminScopes sync.Map // chat ID → struct{}: admin command list already set
}

func NewTelegramNotifier(
	cfg TelegramConfig,
	log *logger.Logger,
	analytics Analytics,
	chats ChatStore,
) (*TelegramNotifier, error) {
	if cfg.BotToken == "" {
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

	admins := make(map[string]bool, len(cfg.Admins))
	for _, a := range cfg.Admins {
		if name := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(a), "@")); name != "" {
			admins[name] = true
		}
	}

	loc := cfg.Location
	if loc == nil {
		loc = time.UTC
	}

	n := &TelegramNotifier{
		log:        log,
		analytics:  analytics,
		chats:      chats,
		admins:     admins,
		siteURL:    strings.TrimSuffix(cfg.SiteURL, "/"),
		digestHour: cfg.DigestHour,
		loc:        loc,
		now:        time.Now,
	}

	bot, err := tgbot.New(cfg.BotToken,
		tgbot.WithSkipGetMe(),
		tgbot.WithHTTPClient(pollTimeout, httpClient),
		tgbot.WithDefaultHandler(n.handleText),
		tgbot.WithErrorsHandler(func(err error) { log.Warn("telegram polling error", "err", err) }),
		tgbot.WithAllowedUpdates(tgbot.AllowedUpdates{models.AllowedUpdateMessage, models.AllowedUpdateCallbackQuery}),
	)
	if err != nil {
		return nil, fmt.Errorf("telegram: create bot: %w", err)
	}
	n.bot = bot

	commands := map[string]tgbot.HandlerFunc{
		"start":    n.handleStart,
		"menu":     n.handleCommandView(viewMenu),
		"about":    n.handleCommandView(viewAbout),
		"projects": n.handleCommandView(viewProjects),
		"contact":  n.handleCommandView(viewContact),
		"language": n.handleCommandView(viewLanguage),
		"resume":   n.handleResumeCommand,
		"admin":    n.handleAdminCommand("home"),
		"stats":    n.handleAdminCommand("st:1"),
		"live":     n.handleAdminCommand("live"),
		"visits":   n.handleAdminCommand("rec"),
	}
	for name, h := range commands {
		bot.RegisterHandler(tgbot.HandlerTypeMessageText, name, tgbot.MatchTypeCommand, h)
	}
	bot.RegisterHandler(tgbot.HandlerTypeCallbackQueryData, "", tgbot.MatchTypePrefix, n.handleCallback)

	return n, nil
}

func (n *TelegramNotifier) Enabled() bool { return n != nil && n.bot != nil }

func (n *TelegramNotifier) Run(ctx context.Context) error {
	if !n.Enabled() {
		return nil
	}
	go n.setupProfile(ctx)
	go n.runDigest(ctx)

	n.log.Info("telegram bot polling", "admins", len(n.admins), "digest_hour", n.digestHour)
	n.bot.Start(ctx)
	return nil
}

// ---------------------------------------------------------------------------
// PROFILE: commands and descriptions, per language
// ---------------------------------------------------------------------------

func (n *TelegramNotifier) publicCommands(l Lang) []models.BotCommand {
	u := uiFor(l)
	return []models.BotCommand{
		{Command: "start", Description: u.cmdStart},
		{Command: "menu", Description: u.cmdMenu},
		{Command: "resume", Description: u.cmdResume},
		{Command: "contact", Description: u.cmdContact},
		{Command: "language", Description: u.cmdLanguage},
	}
}

func adminCommands() []models.BotCommand {
	return []models.BotCommand{
		{Command: "admin", Description: "👑 Admin panel"},
		{Command: "stats", Description: "📊 Traffic stats"},
		{Command: "live", Description: "🟢 Who's on the site now"},
		{Command: "visits", Description: "🕒 Recent sessions"},
	}
}

func (n *TelegramNotifier) setupProfile(ctx context.Context) {
	for _, l := range langs {
		code := string(l)
		if l == LangEN {
			code = "" // default for every language without its own entry
		}
		u := uiFor(l)
		n.retry(ctx, "set commands", func(ctx context.Context) error {
			_, err := n.bot.SetMyCommands(ctx, &tgbot.SetMyCommandsParams{Commands: n.publicCommands(l), LanguageCode: code})
			return err
		})
		n.retry(ctx, "set description", func(ctx context.Context) error {
			_, err := n.bot.SetMyDescription(ctx, &tgbot.SetMyDescriptionParams{Description: u.description, LanguageCode: code})
			return err
		})
		n.retry(ctx, "set short description", func(ctx context.Context) error {
			_, err := n.bot.SetMyShortDescription(ctx, &tgbot.SetMyShortDescriptionParams{ShortDescription: u.shortDescription, LanguageCode: code})
			return err
		})
	}

	// Admins see the admin commands in their own chat's "/" menu.
	if n.chats == nil {
		return
	}
	chats, err := n.chats.ListAdminChats(ctx)
	if err != nil {
		n.log.Warn("telegram: list admin chats for commands", "err", err)
		return
	}
	for _, c := range chats {
		n.ensureAdminCommands(ctx, c.ChatID)
	}
	n.log.Info("telegram profile configured")
}

func (n *TelegramNotifier) ensureAdminCommands(ctx context.Context, chatID int64) {
	if _, done := n.adminScopes.Load(chatID); done {
		return
	}
	n.retry(ctx, "set admin commands", func(ctx context.Context) error {
		_, err := n.bot.SetMyCommands(ctx, &tgbot.SetMyCommandsParams{
			Commands: append(n.publicCommands(LangEN), adminCommands()...),
			Scope:    &models.BotCommandScopeChat{ChatID: chatID},
		})
		if err == nil {
			n.adminScopes.Store(chatID, struct{}{})
		}
		return err
	})
}

func (n *TelegramNotifier) retry(ctx context.Context, what string, fn func(context.Context) error) {
	for attempt := 0; attempt < 3; attempt++ {
		callCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		err := fn(callCtx)
		cancel()
		if err == nil {
			return
		}
		n.log.Warn("telegram: "+what+" failed", "attempt", attempt+1, "err", err)
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Duration(attempt+1) * 2 * time.Second):
		}
	}
}

// ---------------------------------------------------------------------------
// MESSAGE HANDLERS
// ---------------------------------------------------------------------------

func (n *TelegramNotifier) handleStart(ctx context.Context, _ *tgbot.Bot, u *models.Update) {
	msg := u.Message
	if msg == nil || msg.From == nil {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, handlerTimeout)
	defer cancel()

	from := msg.From
	isAdmin := n.isAdmin(from.Username)
	l := n.langOf(from)

	if isAdmin && n.chats != nil {
		if err := n.chats.UpsertAdminChat(ctx, from.Username, msg.Chat.ID, from.FirstName); err != nil {
			n.log.Error("failed to store admin chat", "err", err, "username", from.Username)
		} else {
			n.log.Info("admin chat registered", "username", from.Username, "chat_id", msg.Chat.ID)
			go n.ensureAdminCommands(context.Background(), msg.Chat.ID)
		}
	}

	n.clearLegacyKeyboard(ctx, msg.Chat.ID)

	// Deep links: t.me/<bot>?start=projects opens that section directly.
	payload := strings.TrimSpace(strings.TrimPrefix(msg.Text, "/start"))
	if !isAdmin {
		go n.notifyBotUser(from, payload)
	}
	if payload == "resume" {
		n.sendResume(ctx, msg.Chat.ID, l)
		return
	}
	if view := deepLinkView(payload); view != "" {
		if s, ok := n.publicScreen(l, view, "0", isAdmin); ok {
			n.show(ctx, msg.Chat.ID, 0, s)
			return
		}
	}

	name := from.FirstName
	if name == "" {
		name = from.Username
	}
	n.show(ctx, msg.Chat.ID, 0, n.welcomeScreen(l, name, isAdmin))
}

func deepLinkView(payload string) string {
	switch strings.ToLower(payload) {
	case "about":
		return viewAbout
	case "skills":
		return viewSkills
	case "experience":
		return viewExperience
	case "projects":
		return viewProjects
	case "contact":
		return viewContact
	}
	return ""
}

func (n *TelegramNotifier) handleCommandView(view string) tgbot.HandlerFunc {
	return func(ctx context.Context, _ *tgbot.Bot, u *models.Update) {
		msg := u.Message
		if msg == nil || msg.From == nil {
			return
		}
		if s, ok := n.publicScreen(n.langOf(msg.From), view, "0", n.isAdmin(msg.From.Username)); ok {
			n.show(ctx, msg.Chat.ID, 0, s)
		}
	}
}

func (n *TelegramNotifier) handleResumeCommand(ctx context.Context, _ *tgbot.Bot, u *models.Update) {
	msg := u.Message
	if msg == nil || msg.From == nil {
		return
	}
	n.sendResume(ctx, msg.Chat.ID, n.langOf(msg.From))
}

func (n *TelegramNotifier) handleAdminCommand(route string) tgbot.HandlerFunc {
	return func(ctx context.Context, _ *tgbot.Bot, u *models.Update) {
		msg := u.Message
		if msg == nil || msg.From == nil {
			return
		}
		if !n.isAdmin(msg.From.Username) {
			n.show(ctx, msg.Chat.ID, 0, n.welcomeScreen(n.langOf(msg.From), msg.From.FirstName, false))
			return
		}
		s, _ := n.adminScreen(ctx, msg.Chat.ID, route)
		n.show(ctx, msg.Chat.ID, 0, s)
	}
}

// handleText answers free text: the old reply-keyboard labels still open their
// section; anything else gets the menu.
func (n *TelegramNotifier) handleText(ctx context.Context, _ *tgbot.Bot, u *models.Update) {
	msg := u.Message
	if msg == nil || msg.From == nil || msg.Chat.Type != models.ChatTypePrivate {
		return
	}
	l := n.langOf(msg.From)
	isAdmin := n.isAdmin(msg.From.Username)

	if view := deepLinkView(strings.TrimSpace(msg.Text)); view != "" {
		if s, ok := n.publicScreen(l, view, "0", isAdmin); ok {
			n.show(ctx, msg.Chat.ID, 0, s)
			return
		}
	}
	n.show(ctx, msg.Chat.ID, 0, screen{uiFor(l).unknown, n.menuKeyboard(l, isAdmin)})
}

// ---------------------------------------------------------------------------
// CALLBACKS
// ---------------------------------------------------------------------------

func (n *TelegramNotifier) handleCallback(ctx context.Context, _ *tgbot.Bot, u *models.Update) {
	cq := u.CallbackQuery
	if cq == nil {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, handlerTimeout)
	defer cancel()

	toast := ""
	defer func() { n.answer(ctx, cq.ID, toast) }()

	msg := cq.Message.Message
	if msg == nil {
		return // too old to edit; the button no longer works
	}
	chatID, msgID := msg.Chat.ID, msg.ID

	data := cq.Data
	if strings.HasSuffix(data, ":n") {
		data, msgID = strings.TrimSuffix(data, ":n"), 0
	}

	isAdmin := n.isAdmin(cq.From.Username)
	l := n.langOf(&cq.From)

	switch {
	case data == cbNoop:

	case data == cbResume:
		n.sendResume(ctx, chatID, l)

	case strings.HasPrefix(data, "lang:"):
		if picked, ok := parseLang(strings.TrimPrefix(data, "lang:")); ok {
			n.langs.Store(cq.From.ID, picked)
			l = picked
			toast = uiFor(l).languageSet
		}
		s, _ := n.publicScreen(l, viewMenu, "", isAdmin)
		n.show(ctx, chatID, msgID, s)

	case strings.HasPrefix(data, "v:"):
		parts := strings.SplitN(strings.TrimPrefix(data, "v:"), ":", 2)
		arg := ""
		if len(parts) == 2 {
			arg = parts[1]
		}
		if s, ok := n.publicScreen(l, parts[0], arg, isAdmin); ok {
			n.show(ctx, chatID, msgID, s)
		}

	case strings.HasPrefix(data, "a:"):
		if !isAdmin {
			toast = "⛔"
			return
		}
		s, t := n.adminScreen(ctx, chatID, strings.TrimPrefix(data, "a:"))
		toast = t
		if !n.show(ctx, chatID, msgID, s) && toast == "" {
			toast = "✓ Up to date"
		}
	}
}

// ---------------------------------------------------------------------------
// RENDERING
// ---------------------------------------------------------------------------

var noPreview = &models.LinkPreviewOptions{IsDisabled: tgbot.True()}

// show edits message msgID into s, or sends s as a new message when msgID is 0
// or the edit is impossible. It returns false when the message already showed s.
func (n *TelegramNotifier) show(ctx context.Context, chatID int64, msgID int, s screen) bool {
	if msgID != 0 {
		params := &tgbot.EditMessageTextParams{
			ChatID:             chatID,
			MessageID:          msgID,
			Text:               s.text,
			ParseMode:          models.ParseModeHTML,
			LinkPreviewOptions: noPreview,
		}
		if s.kb != nil {
			params.ReplyMarkup = s.kb
		}
		_, err := n.bot.EditMessageText(ctx, params)
		switch {
		case err == nil:
			return true
		case strings.Contains(err.Error(), "message is not modified"):
			return false
		default:
			n.log.Debug("telegram edit failed, sending new message", "err", err, "chat_id", chatID)
		}
	}

	params := &tgbot.SendMessageParams{
		ChatID:             chatID,
		Text:               s.text,
		ParseMode:          models.ParseModeHTML,
		LinkPreviewOptions: noPreview,
	}
	if s.kb != nil {
		params.ReplyMarkup = s.kb
	}
	if err := n.sendMessage(ctx, params); err != nil {
		n.log.Error("telegram reply failed", "err", err, "chat_id", chatID)
	}
	return true
}

func (n *TelegramNotifier) answer(ctx context.Context, callbackID, text string) {
	if _, err := n.bot.AnswerCallbackQuery(ctx, &tgbot.AnswerCallbackQueryParams{
		CallbackQueryID: callbackID,
		Text:            text,
	}); err != nil {
		n.log.Debug("telegram answer callback failed", "err", err)
	}
}

// clearLegacyKeyboard removes the reply keyboard the previous bot version pinned
// under the input field: Telegram only drops it with a message, so send one and delete it.
func (n *TelegramNotifier) clearLegacyKeyboard(ctx context.Context, chatID int64) {
	m, err := n.bot.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID:              chatID,
		Text:                "⏳",
		DisableNotification: true,
		ReplyMarkup:         &models.ReplyKeyboardRemove{RemoveKeyboard: true},
	})
	if err != nil {
		return
	}
	_, _ = n.bot.DeleteMessage(ctx, &tgbot.DeleteMessageParams{ChatID: chatID, MessageID: m.ID})
}

// sendResume sends the PDF in the reader's language. Telegram fetches it from
// the site the first time; after that the cached file_id is reused.
func (n *TelegramNotifier) sendResume(ctx context.Context, chatID int64, l Lang) {
	u := uiFor(l)
	url := fmt.Sprintf("%s/assets/resume/rami_suliman_%s.pdf", n.siteURL, l)

	_, _ = n.bot.SendChatAction(ctx, &tgbot.SendChatActionParams{ChatID: chatID, Action: models.ChatActionUploadDocument})

	var file models.InputFile = &models.InputFileString{Data: url}
	if id, ok := n.resumeFiles.Load(l); ok {
		file = &models.InputFileString{Data: id.(string)}
	}

	m, err := n.bot.SendDocument(ctx, &tgbot.SendDocumentParams{
		ChatID:    chatID,
		Document:  file,
		Caption:   fmt.Sprintf(u.resumeCaption, escape(contentFor(l).role)),
		ParseMode: models.ParseModeHTML,
		ReplyMarkup: keyboard(
			row(urlBtn(u.btnTelegram, contactTelegram)),
			n.menuRow(l),
		),
	})
	if err != nil {
		n.resumeFiles.Delete(l)
		n.log.Warn("telegram resume send failed", "err", err, "url", url)
		n.show(ctx, chatID, 0, screen{fmt.Sprintf(u.resumeFailed, escape(url)), keyboard(n.menuRow(l))})
		return
	}
	if m != nil && m.Document != nil {
		n.resumeFiles.Store(l, m.Document.FileID)
	}
}

// ---------------------------------------------------------------------------
// OUTGOING ALERTS
// ---------------------------------------------------------------------------

func (n *TelegramNotifier) NotifyVisit(ctx context.Context, v domain.VisitNotification) error {
	want := func(m domain.NotifyMode) bool {
		return m == domain.NotifyAll || (m == domain.NotifyNew && v.Kind == domain.VisitKindNew)
	}
	return n.broadcast(ctx, want, screen{formatVisit(v, n.loc), n.alertKeyboard()})
}

func (n *TelegramNotifier) NotifyLead(ctx context.Context, l domain.LeadNotification) error {
	want := func(m domain.NotifyMode) bool { return m != domain.NotifyOff }
	return n.broadcast(ctx, want, screen{formatLead(l, n.loc), n.alertKeyboard()})
}

func (n *TelegramNotifier) notifyBotUser(from *models.User, payload string) {
	ctx, cancel := context.WithTimeout(context.Background(), handlerTimeout)
	defer cancel()
	want := func(m domain.NotifyMode) bool { return m != domain.NotifyOff }
	if err := n.broadcast(ctx, want, screen{formatBotUser(from, payload, n.loc), nil}); err != nil {
		n.log.Warn("bot user notification failed", "err", err)
	}
}

func (n *TelegramNotifier) alertKeyboard() *models.InlineKeyboardMarkup {
	return keyboard(row(
		btn("📊 Today", "a:st:1:n"),
		btn("🟢 Live", "a:live:n"),
		urlBtn("🔎 Sessions", n.siteURL+"/admin/sessions"),
	))
}

// broadcast sends s to every registered admin whose alert mode passes want.
func (n *TelegramNotifier) broadcast(ctx context.Context, want func(domain.NotifyMode) bool, s screen) error {
	if !n.Enabled() {
		return ErrNotConfigured
	}
	chats, err := n.chats.ListAdminChats(ctx)
	if err != nil {
		return fmt.Errorf("telegram: list chats: %w", err)
	}
	if len(chats) == 0 {
		return errors.New("telegram: no admin chats registered — send /start to the bot")
	}

	var errs []error
	sent := 0
	for _, c := range chats {
		if !want(c.NotifyMode) {
			continue
		}
		params := &tgbot.SendMessageParams{
			ChatID:             c.ChatID,
			Text:               s.text,
			ParseMode:          models.ParseModeHTML,
			LinkPreviewOptions: noPreview,
		}
		if s.kb != nil {
			params.ReplyMarkup = s.kb
		}
		if err := n.sendMessage(ctx, params); err != nil {
			errs = append(errs, fmt.Errorf("chat %d: %w", c.ChatID, err))
			continue
		}
		sent++
	}
	if sent > 0 {
		n.log.Info("telegram alert sent", "chats", sent, "alert", firstLine(s.text))
	}
	return errors.Join(errs...)
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

// langOf is the language picked in the bot, else the one Telegram reports for the user.
func (n *TelegramNotifier) langOf(u *models.User) Lang {
	if v, ok := n.langs.Load(u.ID); ok {
		return v.(Lang)
	}
	return langFromCode(u.LanguageCode)
}

// sendMessage retries transient failures. Bad requests and blocked chats are final.
func (n *TelegramNotifier) sendMessage(ctx context.Context, params *tgbot.SendMessageParams) error {
	var lastErr error
	for attempt := 0; attempt < sendAttempts; attempt++ {
		if attempt > 0 {
			wait := time.Duration(attempt) * 2 * time.Second
			var tooMany *tgbot.TooManyRequestsError
			if errors.As(lastErr, &tooMany) {
				wait = time.Duration(tooMany.RetryAfter) * time.Second
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(wait):
			}
		}
		attemptCtx, cancel := context.WithTimeout(ctx, sendAttemptBudget)
		_, lastErr = n.bot.SendMessage(attemptCtx, params)
		cancel()
		if lastErr == nil {
			return nil
		}
		if errors.Is(lastErr, tgbot.ErrorBadRequest) || errors.Is(lastErr, tgbot.ErrorForbidden) {
			break
		}
	}
	return fmt.Errorf("%w: %v", ErrSendFailed, lastErr)
}

// firstLine is an alert's headline without markup, for the logs.
func firstLine(text string) string {
	line, _, _ := strings.Cut(text, "\n")
	return strings.NewReplacer("<b>", "", "</b>", "", "<i>", "", "</i>", "").Replace(line)
}
