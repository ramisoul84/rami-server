package migrations

import (
	"context"
	"os"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// TestUp needs a throwaway database, e.g.:
//
//	docker run --rm -d -p 55432:5432 -e POSTGRES_PASSWORD=test postgres:17-alpine
//	TEST_DATABASE_URL='postgres://postgres:test@localhost:55432/postgres?sslmode=disable' go test ./migrations
func TestUp(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()

	ran, err := Up(ctx, db)
	if err != nil {
		t.Fatalf("first run: %v", err)
	}
	t.Logf("applied %v", ran)

	again, err := Up(ctx, db)
	if err != nil || len(again) != 0 {
		t.Fatalf("second run should be a no-op, got %v, %v", again, err)
	}

	// The schema must accept what the server writes.
	if _, err := db.ExecContext(ctx, `INSERT INTO events (session_id, visitor_id, event_type, site_language, device, os, browser, source)
		VALUES ('s', 'v', 'pageview', 'en', 'desktop', 'Linux', 'Firefox', 'direct')`); err != nil {
		t.Fatalf("insert event: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO telegram_admins (username, chat_id, notify_mode) VALUES ('x', 1, 'bogus')`); err == nil {
		t.Fatal("notify_mode check constraint missing")
	}
}
