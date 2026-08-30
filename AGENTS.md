# orbit-notifications

gRPC email notifications service. Status: `beta`. MIT — `github.com/manovaspace/orbit-notifications`.

## Ports & Commands

```bash
export DEPLOYMENT_ENVIRONMENT=dev
export DATABASE_URL=postgres://orbit:orbit@localhost:10332/notifications?sslmode=disable
export SMTP_FROM=noreply@manova.space
export SMTP_HOST=localhost
export SMTP_PORT=10725
export GRPC_PORT=10110
export HEALTH_PORT=10111
go run ./cmd/notifications
go test ./...
./scripts/generate-proto.sh   # after proto changes
```

Ports: gRPC on **10110** (`GRPC_PORT`), HTTP health on **10111** (`HEALTH_PORT`, `/healthz`, `/readyz`).

## Configuration

- **SMTP Adapter:** Standard library `net/smtp` (`internal/infrastructure/smtp/`) connecting to Stalwart in production or Mailpit in dev (`SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASS`, `SMTP_FROM`).
- **Feature Flags (Unleash):** `UNLEASH_URL`, `UNLEASH_API_TOKEN`, `UNLEASH_APP_NAME`. Flag `manova.notifications.dev_payload` enables persisting plaintext OTP/payload in `delivery_records.dev_payload` JSONB when in dev environment.
- **Internal Auth:** Secured via `ORBIT_INTERNAL_TOKEN` (`x-orbit-internal-token`).

## Mail catalog

- **Catalog:** `pkg/mailtemplates` — `Render(name, vars)` → Subject, Text, HTML.
- **Source:** `templates/email/*.mjml` + `*.txt`. Compile: `./scripts/compile-email.sh` (requires bun; `bunx mjml@4`).
- Do not hand-edit compiled HTML; regenerate from MJML.
- Do not put OTP/code in subjects.

## Docs

- Public README in this repo
- Staff handbook: [ADR-003](/decisions/003-mail-strategy), [ADR-011](/decisions/011-platform-auth-notifications), [Contracts](/architecture/contracts#7-orbit-notifications-go--tier-2-platform)
