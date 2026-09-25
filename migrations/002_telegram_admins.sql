CREATE TABLE IF NOT EXISTS telegram_admins (
    username    TEXT PRIMARY KEY,
    chat_id     BIGINT NOT NULL,
    first_name  TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_telegram_admins_chat_id ON telegram_admins(chat_id);