-- 0001_init: the complete schema.
--
-- Every statement is idempotent, so this runs safely on a fresh database and on
-- one that already had the old hand-applied files (001–004) — it only adds what
-- is missing.

-- ---------------------------------------------------------------------------
-- events: one row per tracked browser event
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS events (
    id                   BIGSERIAL PRIMARY KEY,
    session_id           TEXT        NOT NULL,
    visitor_id           TEXT        NOT NULL,
    event_type           TEXT        NOT NULL,
    page                 TEXT,
    page_title           TEXT,
    referrer             TEXT,
    language             TEXT,
    timezone             TEXT,
    viewport             TEXT,
    screen               TEXT,
    user_agent           TEXT,
    traffic_source       JSONB,
    time_spent_seconds   INTEGER,
    engaged_time_seconds INTEGER,
    section              TEXT,
    selector             TEXT,
    label                TEXT,
    custom_name          TEXT,
    custom_data          JSONB,
    ip_address           TEXT,
    country              TEXT,
    city                 TEXT,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Derived once at insert time so analytics queries can group on them directly.
ALTER TABLE events
    ADD COLUMN IF NOT EXISTS site_language TEXT,
    ADD COLUMN IF NOT EXISTS device        TEXT,
    ADD COLUMN IF NOT EXISTS os            TEXT,
    ADD COLUMN IF NOT EXISTS browser       TEXT,
    ADD COLUMN IF NOT EXISTS source        TEXT;

-- Date-range scans (every analytics query).
CREATE INDEX IF NOT EXISTS idx_events_created_at        ON events (created_at DESC);
-- Per-session and per-visitor lookups (sessions list, returning-visitor checks, visit alerts).
CREATE INDEX IF NOT EXISTS idx_events_session_created   ON events (session_id, created_at);
CREATE INDEX IF NOT EXISTS idx_events_visitor_created   ON events (visitor_id, created_at);
-- Custom-event breakdowns (resume downloads, contact clicks, scroll depth…).
CREATE INDEX IF NOT EXISTS idx_events_custom_name       ON events (custom_name) WHERE event_type = 'custom';

-- Superseded by the composite indexes above, or never selective enough to be used.
DROP INDEX IF EXISTS idx_events_visitor_id;
DROP INDEX IF EXISTS idx_events_session_id;
DROP INDEX IF EXISTS idx_events_event_type;
DROP INDEX IF EXISTS idx_events_page;

-- ---------------------------------------------------------------------------
-- telegram_admins: admin chats that receive bot alerts
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS telegram_admins (
    username    TEXT PRIMARY KEY,
    chat_id     BIGINT      NOT NULL,
    first_name  TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- notify_mode: 'all' — every new session + leads; 'new' — first-time visitors + leads; 'off'.
-- digest:      daily traffic summary at TELEGRAM_DIGEST_HOUR.
ALTER TABLE telegram_admins
    ADD COLUMN IF NOT EXISTS notify_mode TEXT    NOT NULL DEFAULT 'all',
    ADD COLUMN IF NOT EXISTS digest      BOOLEAN NOT NULL DEFAULT TRUE;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'telegram_admins_notify_mode_check') THEN
        ALTER TABLE telegram_admins
            ADD CONSTRAINT telegram_admins_notify_mode_check CHECK (notify_mode IN ('all', 'new', 'off'));
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_telegram_admins_chat_id ON telegram_admins (chat_id);
