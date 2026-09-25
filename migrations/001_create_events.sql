CREATE TABLE IF NOT EXISTS events (
    id                      BIGSERIAL PRIMARY KEY,
    session_id              TEXT NOT NULL,
    visitor_id              TEXT NOT NULL,
    event_type              TEXT NOT NULL,
    page                    TEXT,
    page_title              TEXT,
    referrer                TEXT,
    language                TEXT,
    timezone                TEXT,
    viewport                TEXT,
    screen                  TEXT,
    user_agent              TEXT,
    traffic_source          JSONB,
    time_spent_seconds      INTEGER,
    engaged_time_seconds    INTEGER,
    section                 TEXT,
    selector                TEXT,
    label                   TEXT,
    custom_name             TEXT,
    custom_data             JSONB,
    ip_address              TEXT,
    country                 TEXT,
    city                    TEXT,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_events_visitor_id  ON events(visitor_id);
CREATE INDEX IF NOT EXISTS idx_events_session_id  ON events(session_id);
CREATE INDEX IF NOT EXISTS idx_events_created_at  ON events(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_events_event_type  ON events(event_type);
CREATE INDEX IF NOT EXISTS idx_events_page        ON events(page);