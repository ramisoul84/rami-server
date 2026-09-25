package notifier

import (
	"context"

	"github.com/ramisoul84/rami-server/internal/domain"
)

// StatsProvider is a function the notifier calls when the user asks for stats.
// The caller decides what to provide — a service, a cache, anything.
type StatsProvider func(ctx context.Context) (*domain.DashboardStats, error)

// RecentVisitsProvider is a function the notifier calls for /visits.
type RecentVisitsProvider func(ctx context.Context, limit int) ([]domain.Visit, error)

// Notifier is the sending side.
type Notifier interface {
	Enabled() bool
	NotifyVisit(ctx context.Context, n domain.VisitNotification) error
}

// internal/notifier/notifier.go
type ChatStore interface {
	UpsertAdminChat(ctx context.Context, username string, chatID int64, firstName string) error
	ListAdminChats(ctx context.Context) ([]int64, error)
}

// Runnable is implemented by notifiers that need a background loop
// (e.g., Telegram long-polling). Optional.
type Runnable interface {
	Run(ctx context.Context) error
}
