package migrations

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/manovaspace/orbit-notifications/internal/testutil"
)

// legacyStatements reproduce the exact state the previous custom runner left
// behind: an empty goose_db_version table and the 001 Up migration applied.
var legacyStatements = []string{
	`CREATE TABLE IF NOT EXISTS goose_db_version (
		id serial PRIMARY KEY,
		version_id bigint NOT NULL,
		is_applied bool NOT NULL,
		tstamp timestamptz DEFAULT NOW()
	)`,
	`CREATE TABLE IF NOT EXISTS delivery_records (
		id UUID PRIMARY KEY,
		template TEXT NOT NULL,
		channel TEXT NOT NULL,
		recipient_hash TEXT NOT NULL,
		status TEXT NOT NULL,
		dev_payload JSONB,
		correlation_id TEXT,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	)`,
	`CREATE INDEX IF NOT EXISTS delivery_records_created_at_idx ON delivery_records (created_at DESC)`,
}

func TestFreshDatabaseAppliesEmbeddedMigrations(t *testing.T) {
	db := testutil.OpenPostgres(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	m, err := Open(db.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = m.Close() }()

	results, err := m.Up(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Source.Version != 1 {
		t.Fatalf("applied %d migrations, want version 1 only", len(results))
	}
	if !tableExists(t, ctx, db.Pool, db.Schema, "delivery_records") {
		t.Fatal("delivery_records was not created")
	}
	if current, target, err := m.Version(ctx); err != nil || current != 1 || target != 1 {
		t.Fatalf("version = %d/%d, err=%v, want 1/1", current, target, err)
	}
	assertGooseRow(t, ctx, db.Pool, db.Schema, 1)
}

func TestAdoptionOnLegacyDatabaseChangesNothing(t *testing.T) {
	db := testutil.OpenPostgres(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for _, statement := range legacyStatements {
		if _, err := db.Pool.Exec(ctx, statement); err != nil {
			t.Fatalf("apply legacy statement: %v", err)
		}
	}
	// A row proves the adoption does not drop or recreate the table.
	if _, err := db.Pool.Exec(ctx, `
		INSERT INTO delivery_records (id, template, channel, recipient_hash, status, correlation_id)
		VALUES ('11111111-1111-1111-1111-111111111111', 'otp_login', 'email', 'hash', 'sent', 'legacy')
	`); err != nil {
		t.Fatalf("insert legacy row: %v", err)
	}
	before := schemaSnapshot(t, ctx, db.Pool, db.Schema)

	m, err := Open(db.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = m.Close() }()

	if _, err := m.Up(ctx); err != nil {
		t.Fatalf("adopt legacy database: %v", err)
	}
	after := schemaSnapshot(t, ctx, db.Pool, db.Schema)

	if before != after {
		t.Fatalf("adoption changed the schema:\nbefore: %s\nafter:  %s", before, after)
	}
	var correlationID string
	if err := db.Pool.QueryRow(ctx, `
		SELECT correlation_id FROM delivery_records WHERE id = '11111111-1111-1111-1111-111111111111'
	`).Scan(&correlationID); err != nil {
		t.Fatalf("read legacy row after adoption: %v", err)
	}
	if correlationID != "legacy" {
		t.Fatalf("legacy row correlation_id = %q, want legacy", correlationID)
	}
	if current, target, err := m.Version(ctx); err != nil || current != 1 || target != 1 {
		t.Fatalf("version = %d/%d, err=%v, want 1/1", current, target, err)
	}
	assertGooseRow(t, ctx, db.Pool, db.Schema, 1)
}

func TestDownRemovesTheMigrationAndUpReappliesIt(t *testing.T) {
	db := testutil.OpenPostgres(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	m, err := Open(db.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = m.Close() }()

	if _, err := m.Up(ctx); err != nil {
		t.Fatal(err)
	}
	result, err := m.Down(ctx)
	if err != nil {
		t.Fatalf("down: %v", err)
	}
	if result.Source.Version != 1 {
		t.Fatalf("rolled back version %d, want 1", result.Source.Version)
	}
	if tableExists(t, ctx, db.Pool, db.Schema, "delivery_records") {
		t.Fatal("down left delivery_records in place")
	}
	if _, err := m.Up(ctx); err != nil {
		t.Fatalf("reapply after down: %v", err)
	}
	if !tableExists(t, ctx, db.Pool, db.Schema, "delivery_records") {
		t.Fatal("up after down did not recreate delivery_records")
	}
}

// schemaSnapshot is stable text for every object the migration owns. A
// relfilenode changes when a table or index is dropped and recreated, so equal
// snapshots prove IF NOT EXISTS was a no-op.
func schemaSnapshot(t *testing.T, ctx context.Context, pool *pgxpool.Pool, schema string) string {
	t.Helper()
	const query = `
		SELECT c.relname || ':' || c.relfilenode::text || ':' || c.relkind::text
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = $1
		  AND c.relname IN ('delivery_records', 'delivery_records_created_at_idx', 'goose_db_version')
		ORDER BY c.relname`
	rows, err := pool.Query(ctx, query, schema)
	if err != nil {
		t.Fatalf("read schema snapshot: %v", err)
	}
	defer rows.Close()
	var out string
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			t.Fatal(err)
		}
		out += line + "\n"
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	const columnsQuery = `
		SELECT c.relname || '.' || a.attname || ':' || format_type(a.atttypid, a.atttypmod)
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		JOIN pg_attribute a ON a.attrelid = c.oid
		WHERE n.nspname = $1 AND c.relname = 'delivery_records' AND a.attnum > 0 AND NOT a.attisdropped
		ORDER BY a.attnum`
	columnRows, err := pool.Query(ctx, columnsQuery, schema)
	if err != nil {
		t.Fatalf("read column snapshot: %v", err)
	}
	defer columnRows.Close()
	for columnRows.Next() {
		var line string
		if err := columnRows.Scan(&line); err != nil {
			t.Fatal(err)
		}
		out += line + "\n"
	}
	if err := columnRows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

func tableExists(t *testing.T, ctx context.Context, pool *pgxpool.Pool, schema, name string) bool {
	t.Helper()
	var exists bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM pg_class c
			JOIN pg_namespace n ON n.oid = c.relnamespace
			WHERE n.nspname = $1 AND c.relname = $2 AND c.relkind IN ('r', 'p')
		)`, schema, name).Scan(&exists); err != nil {
		t.Fatal(err)
	}
	return exists
}

func assertGooseRow(t *testing.T, ctx context.Context, pool *pgxpool.Pool, schema string, wantVersion int64) {
	t.Helper()
	var version int64
	var applied bool
	if err := pool.QueryRow(ctx, `
		SELECT version_id, is_applied FROM "`+schema+`".goose_db_version ORDER BY id DESC LIMIT 1
	`).Scan(&version, &applied); err != nil {
		t.Fatalf("read goose_db_version: %v", err)
	}
	if version != wantVersion || !applied {
		t.Fatalf("goose row = version %d applied %v, want %d/true", version, applied, wantVersion)
	}
}
