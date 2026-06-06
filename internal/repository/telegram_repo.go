package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/ramisoul84/rami-server/internal/domain"
)

type TelegramRepository interface {
	UpsertAdminChat(ctx context.Context, username string, chatID int64, firstName string) error
	ListAdminChats(ctx context.Context) ([]domain.AdminChat, error)
	GetAdminChat(ctx context.Context, chatID int64) (*domain.AdminChat, error)
	SetNotifyMode(ctx context.Context, chatID int64, mode domain.NotifyMode) error
	SetDigest(ctx context.Context, chatID int64, enabled bool) error
}

type telegramRepository struct {
	db *sqlx.DB
}

func NewTelegramRepository(db *sqlx.DB) TelegramRepository {
	return &telegramRepository{db: db}
}

func (r *telegramRepository) UpsertAdminChat(ctx context.Context, username string, chatID int64, firstName string) error {
	const query = `
		INSERT INTO telegram_admins (username, chat_id, first_name)
		VALUES ($1, $2, $3)
		ON CONFLICT (username) DO UPDATE
		SET chat_id = EXCLUDED.chat_id,
		    first_name = EXCLUDED.first_name,
		    updated_at = NOW()
	`
	if _, err := r.db.ExecContext(ctx, query, username, chatID, firstName); err != nil {
		return fmt.Errorf("repo: upsert telegram admin: %w", err)
	}
	return nil
}

func (r *telegramRepository) ListAdminChats(ctx context.Context) ([]domain.AdminChat, error) {
	const query = `SELECT username, chat_id, notify_mode, digest FROM telegram_admins`
	var chats []domain.AdminChat
	if err := r.db.SelectContext(ctx, &chats, query); err != nil {
		return nil, fmt.Errorf("repo: list telegram admins: %w", err)
	}
	return chats, nil
}

func (r *telegramRepository) GetAdminChat(ctx context.Context, chatID int64) (*domain.AdminChat, error) {
	const query = `SELECT username, chat_id, notify_mode, digest FROM telegram_admins WHERE chat_id = $1 LIMIT 1`
	var chat domain.AdminChat
	if err := r.db.GetContext(ctx, &chat, query, chatID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("repo: get telegram admin: %w", err)
	}
	return &chat, nil
}

func (r *telegramRepository) SetNotifyMode(ctx context.Context, chatID int64, mode domain.NotifyMode) error {
	const query = `UPDATE telegram_admins SET notify_mode = $2, updated_at = NOW() WHERE chat_id = $1`
	if _, err := r.db.ExecContext(ctx, query, chatID, string(mode)); err != nil {
		return fmt.Errorf("repo: set notify mode: %w", err)
	}
	return nil
}

func (r *telegramRepository) SetDigest(ctx context.Context, chatID int64, enabled bool) error {
	const query = `UPDATE telegram_admins SET digest = $2, updated_at = NOW() WHERE chat_id = $1`
	if _, err := r.db.ExecContext(ctx, query, chatID, enabled); err != nil {
		return fmt.Errorf("repo: set digest: %w", err)
	}
	return nil
}
