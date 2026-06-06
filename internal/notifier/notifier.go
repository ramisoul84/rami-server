package notifier

import (
	"context"

	"github.com/ramisoul84/rami-server/internal/domain"
)

// Analytics is what the admin side of the bot reads. service.AnalyticsService satisfies it.
type Analytics interface {
	Overview(ctx context.Context, r domain.AnalyticsRange) (*domain.Overview, error)
	Live(ctx context.Context) (*domain.LiveSnapshot, error)
	Sessions(ctx context.Context, q domain.SessionsQuery) (*domain.SessionsPage, error)
}

// Notifier is the sending side.
type Notifier interface {
	Enabled() bool
	NotifyVisit(ctx context.Context, n domain.VisitNotification) error
	NotifyLead(ctx context.Context, n domain.LeadNotification) error
}

// ChatStore keeps registered admin chats and their alert preferences.
type ChatStore interface {
	UpsertAdminChat(ctx context.Context, username string, chatID int64, firstName string) error
	ListAdminChats(ctx context.Context) ([]domain.AdminChat, error)
	GetAdminChat(ctx context.Context, chatID int64) (*domain.AdminChat, error)
	SetNotifyMode(ctx context.Context, chatID int64, mode domain.NotifyMode) error
	SetDigest(ctx context.Context, chatID int64, enabled bool) error
}

// Runnable is implemented by notifiers that need a background loop
// (e.g., Telegram long-polling). Optional.
type Runnable interface {
	Run(ctx context.Context) error
}
