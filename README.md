# Rami Server

Backend for [ramisuliman.com](https://ramisuliman.com): privacy-friendly analytics, the admin API, and the Telegram bot.

- **Go 1.26**, Fiber, PostgreSQL (sqlx), GeoLite2
- `POST /api/v1/analytics/event` — the site's tracker (rate-limited, validated)
- `/api/v1/admin/analytics/*` — admin panel data, JWT-protected (contract: `client/docs/ANALYTICS_API.md`)
- Telegram bot — public portfolio menu (EN/RU/DE) + admin alerts, stats and a daily digest

## Layout

```
cmd/server            entry point
internal/app          wiring and lifecycle
internal/config       env configuration and validation
internal/domain       shared types
internal/repository   SQL
internal/service      business logic (events, analytics, auth, geo)
internal/notifier     Telegram bot
internal/transport    HTTP server, handlers, middleware
migrations            SQL schema, embedded and applied at startup
pkg/                  database, jwt, logger
```

## Run locally

```sh
cp .env.example .env   # fill in DB_*, AUTH_* (and TELEGRAM_* for the bot)
make run
```

## Deploy

```sh
docker compose up -d --build
docker compose logs -f rami-server
```

Put the GeoLite2 database in `./data/` (Country or City edition). The container
listens on `127.0.0.1:8000`; the reverse proxy forwards `/api` to it and should
set `X-Real-IP` or `X-Forwarded-For` so visits get the right country.

## Migrations

`migrations/NNNN_name.sql` files are embedded in the binary and run at startup,
in order, each once (tracked in `schema_migrations`, guarded by an advisory lock).
To change the schema, add a new numbered file — never edit one that has shipped.

## Tests

```sh
make test
# migration test against a throwaway Postgres:
docker run --rm -d -p 55432:5432 -e POSTGRES_PASSWORD=test postgres:17-alpine
TEST_DATABASE_URL='postgres://postgres:test@localhost:55432/postgres?sslmode=disable' go test ./migrations
```
