# orbit-notifications

[![CI](https://github.com/manovaspace/orbit-notifications/actions/workflows/ci.yml/badge.svg)](https://github.com/manovaspace/orbit-notifications/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](./LICENSE)

gRPC notifications service for templated email delivery and delivery logs.

Part of the [Manova / Orbit](https://github.com/manovaspace) open toolkit.

## Install

```bash
go get github.com/manovaspace/orbit-notifications@latest
```

Protobuf API: `github.com/manovaspace/orbit-notifications/api/notifications/v1`.

## Layout

```
cmd/notifications/                 wiring only: config, migrations, pool, servers
internal/domain/                   delivery types and ports (Store, MailSender,
                                   Flags, Clock, IDGenerator)
internal/application/              send/list use case and template rendering
internal/infrastructure/
  clock/                           system clock adapter
  featureflags/                    Unleash/OpenFeature adapter
  grpc/                            NotificationsService adapter
  httpapi/                         Fiber v3 health server (/healthz, /readyz)
  ids/                             UUID delivery ids
  postgres/                        Store adapter, sqlc queries and generated code
  postgres/migrations/             embedded goose migrations
  smtp/                            SMTP adapter
internal/testutil/                 disposable-schema integration-test helper
```

Migrations are embedded in the binary; there is no runtime `migrations/`
directory.

The domain imports no infrastructure package. The application layer receives
every dependency through `application.Deps`, and `cmd/` only wires adapters.

## Development

```bash
cp .env.example .env   # set SMTP_FROM and DATABASE_URL for local runs
go test ./...
```

The service binary is `go run ./cmd/notifications` (requires Postgres and
`SMTP_FROM`). Migrations are embedded and applied on start; see
[docs/migrations.md](./docs/migrations.md) for the `migrate` subcommand,
production adoption and rollback.

Integration tests run against a disposable database and skip without it:

```bash
docker run -d --rm --name notif-pg \
  -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=test \
  -p 55437:5432 postgres:18-alpine

TEST_DATABASE_URL='postgres://postgres:postgres@localhost:55437/test?sslmode=disable' \
  go test ./... -race -count=1

docker stop notif-pg
```

`TEST_DATABASE_URL` is accepted only when the connected database name starts
with `test`; each test creates and drops its own schema.

Regenerate the sqlc package after changing SQL:

```bash
go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1 generate
```

Generated code in `internal/infrastructure/postgres/sqlcgen` is committed and
must never be edited by hand.

## Events

This service publishes and consumes no events. `orbit-auth` calls
`notifications.Send` over gRPC, so `messagebus.Bus` is not wired here.

## Contributing

See [CONTRIBUTING.md](./CONTRIBUTING.md). Security reports: [SECURITY.md](./SECURITY.md).

## License

MIT — see [LICENSE](./LICENSE).
