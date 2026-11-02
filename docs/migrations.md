# Migrations

This service uses [goose](https://github.com/pressly/goose) with SQL migrations
embedded in the binary (`internal/infrastructure/postgres/migrations`). There is
no hand-written migration runner.

## Commands

```bash
# apply every pending migration (also the default at start-up)
go run ./cmd/notifications migrate
go run ./cmd/notifications migrate up

# roll back the most recent migration
go run ./cmd/notifications migrate down

# show every embedded migration and the current version
go run ./cmd/notifications migrate status
```

`MIGRATE_ON_START` defaults to `true`, which keeps the behaviour of the previous
release: the binary applies pending migrations before it opens the pool. Set
`MIGRATE_ON_START=false` when a deploy step runs `migrate` explicitly and the
service must not touch the schema.

`migrate down` rolls back exactly one migration. Rolling back `001` drops
`delivery_records` and deletes its rows: that is a data-destroying rollback, not
a deployment rollback.

## Adding a migration

1. Add the next numbered file, for example
   `internal/infrastructure/postgres/migrations/002_add_status_index.sql`.
   The file must contain `-- +goose Up` first and `-- +goose Down` second.
2. Keep the Down section reversible. If a change cannot be reversed safely
   (dropping a column that holds data), write the Down to restore the object and
   document the manual step in this file.
3. Regenerate the sqlc package when a query or the schema changes:
   `go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1 generate`. Never edit
   `internal/infrastructure/postgres/sqlcgen` by hand.
4. Run the migration tests against a disposable database (below).
5. Fix a defect in a new migration; never edit a migration that has already run
   in production.

## Production adoption (existing database)

The previous release used a small custom runner. It created a
`goose_db_version` table with the columns goose v3 expects and applied
`001_delivery_records.sql`, but it never recorded a version row. `migrations.ensureBaseline` handles that
state: when `goose_db_version` exists and is empty, it inserts goose's zero row
(the row goose itself inserts when it creates the table). `migrate up` then runs
the first migration, which is written with `CREATE TABLE IF NOT EXISTS` and
`CREATE INDEX IF NOT EXISTS`, so it changes nothing on a database that already
has the objects, and goose records version 1.

Exact adoption on a production database:

```bash
# 1. Back up first; this also gives the rollback point.
pg_dump "$DATABASE_URL" --format=custom --file notifications-before-goose.dump

# 2. Inspect the legacy state (goose_db_version exists and has no rows).
psql "$DATABASE_URL" -c '\d delivery_records'
psql "$DATABASE_URL" -c 'SELECT count(*) FROM goose_db_version'

# 3. Apply. The first migration is a no-op on the existing objects.
/app/notifications migrate up

# 4. Verify the recorded version and that the objects are unchanged.
psql "$DATABASE_URL" -c 'SELECT version_id, is_applied FROM goose_db_version ORDER BY id'
psql "$DATABASE_URL" -c '\d delivery_records'
```

The adoption is covered by
`TestAdoptionOnLegacyDatabaseChangesNothing`: it builds the exact legacy state
(including a row), snapshots table and index `relfilenode`s, runs `migrate up`,
and asserts the snapshot and the row are unchanged while version 1 is recorded.
Run it with:

```bash
docker run -d --rm --name notif-pg \
  -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=test \
  -p 55437:5432 postgres:18-alpine
TEST_DATABASE_URL='postgres://postgres:postgres@localhost:55437/test?sslmode=disable' \
  go test ./internal/infrastructure/postgres/migrations/ -race -count=1 -v
```

## Rolling back

Rollback is migration-by-migration:

```bash
# one migration back (here: drop delivery_records; destructive)
/app/notifications migrate down

# then re-apply
/app/notifications migrate up
```

An application rollback (deploying the previous image) does not need a schema
rollback when the change followed expand-then-contract. The previous binary
ignores new nullable columns and new tables.

To restore a full database from the adoption backup:

```bash
pg_restore --clean --if-exists --dbname "$DATABASE_URL" notifications-before-goose.dump
```

## Expand, then contract

A breaking schema change takes two releases. Both releases must run against the
same schema, and the schema must be compatible with the release before them.

1. **Expand** (release N): add the new column/table, nullable or with a
   default; write both the old and new shapes; keep the old read path.
2. **Migrate data** (release N or a job): backfill in batches. Do not hold a
   long transaction.
3. **Switch reads** (release N+1): read the new shape, still write both if the
   previous release may still run.
4. **Contract** (release N+2): drop the old column/table in a migration whose
   Down restores it.

A single release never drops an object that the previous release still reads.

## One schema per service

`orbit-notifications` owns the dedicated `notifications` logical database and
its `public` schema. No other service reads these tables; `orbit-auth` calls the
gRPC API instead. That is the "one schema per service" rule for this service.

No data moves in this change. If the platform later gives every service a named
schema inside a shared database, the move is:

1. `CREATE SCHEMA notifications;`
2. Set `search_path=notifications,public` for this service's role/DSN.
3. `ALTER TABLE delivery_records SET SCHEMA notifications;` (indexes and the
   sequence follow the table; `goose_db_version` moves the same way).
4. Apply the same DSN change in every environment, then remove `public` from
   the service's search path in a later release.

That move is a deployment task with a maintenance window, not a migration.
