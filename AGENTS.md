---
read_with:
  - handbook/docs/orbit/architecture/go-toolchain.md
  - handbook/docs/orbit/decisions/003-mail-strategy.md
  - handbook/docs/orbit/decisions/011-platform-auth-notifications.md
  - handbook/docs/orbit/guides/platform-dev-quickstart.md
---

# orbit-notifications

gRPC email notifications service. Status: `beta`. MIT — `github.com/manovaspace/orbit-notifications`.

## Ports & Commands

```bash
export DEPLOYMENT_ENVIRONMENT=dev
export DATABASE_URL=postgres://notifications:<password>@127.0.0.1:10333/notifications?sslmode=disable
export SMTP_FROM=noreply@manova.space
export SMTP_HOST=localhost
export SMTP_PORT=10024
export GRPC_PORT=10110
export HEALTH_PORT=10111
go run ./cmd/notifications
go run ./cmd/notifications migrate status   # up, down, status
go test ./...
TEST_DATABASE_URL='postgres://postgres:postgres@localhost:55437/test?sslmode=disable' go test ./... -race -count=1
go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1 generate
./scripts/generate-proto.sh   # after proto changes
```

Ports: gRPC on **10110** (`GRPC_PORT`), HTTP health on **10111** (`HEALTH_PORT`, `/healthz`, `/readyz`). HTTP runs on Fiber v3; gRPC is unchanged.

## Architecture

- `internal/domain` — delivery types plus the ports: `Store`, `MailSender`, `Flags`, `Clock`, `IDGenerator`. It imports no infrastructure, driver, HTTP framework or process package.
- `internal/application` — the send/list use case. It receives `application.Deps` and reads no environment variable.
- `internal/infrastructure` — adapters: `postgres` (sqlc-generated queries), `postgres/migrations` (embedded goose), `grpc`, `httpapi` (Fiber v3), `smtp`, `featureflags`, `clock`, `ids`.
- `cmd/notifications` — wiring only.
- `.golangci.yml` `depguard` rules enforce both boundaries. Do not weaken them to land a change.

## Migrations and SQL

- Embedded goose migrations live in `internal/infrastructure/postgres/migrations/*.sql`. There is no runtime `migrations/` directory.
- `MIGRATE_ON_START` (default `true`) applies them at start; set `false` when the deploy runs `migrate` first. `go run ./cmd/notifications migrate down` is destructive for `001`.
- Production adoption, rollback, expand-then-contract and the one-schema rule: [docs/migrations.md](./docs/migrations.md).
- Queries: `internal/infrastructure/postgres/queries/*.sql`; generated package `internal/infrastructure/postgres/sqlcgen` is committed and never edited by hand.

## Configuration

- **SMTP Adapter:** Standard library `net/smtp` (`internal/infrastructure/smtp/`) connecting to Stalwart in production or Mailpit in dev (`SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASS`, `SMTP_FROM`).
- **Feature Flags (Unleash):** `UNLEASH_URL`, `UNLEASH_API_TOKEN`, `UNLEASH_APP_NAME`. Flag `manova.notifications.dev_payload` enables persisting plaintext OTP/payload in `delivery_records.dev_payload` JSONB when in dev environment.
- **Internal Auth:** Secured via `ORBIT_INTERNAL_TOKEN` (`x-orbit-internal-token`).
- **Migrations:** `MIGRATE_ON_START` (default `true`).

## Events

None. The service publishes and consumes no events; `orbit-auth` calls the gRPC `Send` API. Do not wire `messagebus.Bus` without an ADR change.

## Mail catalog

- **Catalog:** `pkg/mailtemplates` — `Render(name, vars)` → Subject, Text, HTML.
- **Source:** `templates/email/*.mjml` + `*.txt`. Compile: `./scripts/compile-email.sh` (requires bun; `bunx mjml@4`).
- Do not hand-edit compiled HTML; regenerate from MJML.
- Do not put OTP/code in subjects.

## Docs

- Public README in this repo
- Staff handbook: [ADR-003](/decisions/003-mail-strategy), [ADR-011](/decisions/011-platform-auth-notifications), [Contracts](/architecture/contracts#7-orbit-notifications-go--tier-2-platform)
