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

## Development

```bash
cp .env.example .env   # set SMTP_FROM and DATABASE_URL for local runs
go test ./...
```

Service binary: `go run ./cmd/notifications` (requires Postgres and `SMTP_FROM`).

## Contributing

See [CONTRIBUTING.md](./CONTRIBUTING.md). Security reports: [SECURITY.md](./SECURITY.md).

## License

MIT — see [LICENSE](./LICENSE).
