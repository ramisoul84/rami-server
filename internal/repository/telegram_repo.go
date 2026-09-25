package repository

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

type TelegramRepository interface {
	UpsertAdminChat(ctx context.Context, username string, chatID int64, firstName string) error
	ListAdminChats(ctx context.Context) ([]int64, error)
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

func (r *telegramRepository) ListAdminChats(ctx context.Context) ([]int64, error) {
	const query = `SELECT chat_id FROM telegram_admins`
	var ids []int64
	if err := r.db.SelectContext(ctx, &ids, query); err != nil {
		return nil, fmt.Errorf("repo: list telegram admins: %w", err)
	}
	return ids, nil
}
