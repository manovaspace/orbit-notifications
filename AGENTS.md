---
see_also:
  - handbook/docs/orbit/architecture/go-toolchain.md
  - handbook/docs/orbit/decisions/003-mail-strategy.md
  - handbook/docs/orbit/decisions/069-identity.md
  - handbook/docs/orbit/guides/platform-dev-quickstart.md
---

# orbit-notifications

gRPC email notifications service. Status: `beta`. MIT — `github.com/manovaspace/orbit-notifications`.

## Ports & Commands

```bash
export DEPLOYMENT_ENVIRONMENT=dev
export DATABASE_URL=postgres://notifications:<password>@127.0.0.1:10000/notifications?sslmode=disable
export SMTP_FROM=noreply@manova.space
export SMTP_HOST=localhost
export SMTP_PORT=10024
export GRPC_PORT=10012
export HEALTH_PORT=10013
go run ./cmd/notifications
go run ./cmd/notifications migrate status   # up, down, status
go test ./...
TEST_DATABASE_URL='postgres://postgres:postgres@localhost:55437/test?sslmode=disable' go test ./... -race -count=1
go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1 generate
./scripts/generate-proto.sh   # after proto changes
```

Ports: gRPC on **10012** (`GRPC_PORT`), HTTP health on **10013** (`HEALTH_PORT`, `/healthz`, `/readyz`). HTTP runs on Fiber v3; gRPC is unchanged.

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
- **Feature Flags (OpenFeature):** read from orbit-auth `/v1/flags` via `FLAGS_URL` (default `http://localhost:10011/v1/flags`; `off` disables). Flag `manova.notifications.dev_payload` enables persisting plaintext OTP/payload in `delivery_records.dev_payload` JSONB when in dev environment.
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

<!-- manova:conventions v1 — generated from manovaspace/ci conventions/core.md; edit it there -->
## Shared conventions

These rules apply to every Manova repository. They are copied from `manovaspace/ci` (`conventions/core.md`); change them there, not here.

### Delivery

1. Never push to `main`. Every change goes on a topic branch (`feat/<name>`, `fix/<name>`, `refactor/<name>`, `docs/<name>`, `chore/<name>`) and reaches `main` through a pull request.
2. One pull request does one thing. An unrelated problem you find on the way becomes an issue, not part of this change.
3. Pull request titles and commit messages follow Conventional Commits (`feat:`, `fix:`, `docs:`, `refactor:`, `chore:`, `test:`; `!` for a breaking change). Releases are computed from them.
4. Merge with squash once the checks are green. Do not use `--no-verify`; if you ever must, say so in the pull request.

### Checks

5. CI runs `manovaspace/ci@v1`. Run the same gate locally before you push; a red check is fixed, not skipped.
6. Tool versions (Go, Node, Bun, linters) come from `ci/versions.env` in `manovaspace/ci`. Do not pin a different version in this repository.
7. A behaviour change comes with a test that fails without it. Do not delete or weaken a test to make a change pass.

### Secrets

8. Never commit a secret, token, password, private key or `.env` file. Commit `.env.example` with placeholders instead.
9. Never print a secret in logs, test output, error messages or pull requests. Read secrets from the environment or a file at run time.

### Runtime

10. Containers talk to each other by name on a Docker network. Do not publish a port on the host unless a person or a host process needs it, and then bind `127.0.0.1`, never `0.0.0.0`.
11. Configuration comes from the environment. A service with a missing required setting refuses to start; it never falls back to a built-in secret or a development default in production.
12. Port numbers come from the workspace port registry. Do not invent a port.

### Documentation

13. This repository explains itself: `README.md` says how to set up, run, test and deploy it; `AGENTS.md` holds its binding rules; repository decisions live in `docs/decisions/`.
14. Write plain, short sentences. Keep exact names, paths, commands and versions. Do not document what the code already says.
<!-- /manova:conventions -->
