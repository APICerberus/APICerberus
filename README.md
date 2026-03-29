# API Cerberus

[![Go](https://img.shields.io/badge/go-1.26%2B-00ADD8.svg)](https://go.dev/)
[![Release](https://img.shields.io/badge/release-v0.0.5-blue.svg)](#release-status)
[![License: MIT](https://img.shields.io/badge/license-MIT-green.svg)](./LICENSE)

API Cerberus is an API gateway and API management platform written in Go.
It combines gateway routing/proxy features with authentication, rate limiting,
user management, credits/billing, and an admin REST API.

## Release Status

- Current tagged release: `v0.0.5`
- Implemented milestones: `v0.0.1` to `v0.0.5`
- Next milestone: `v0.0.6` (Audit Logging and Analytics)

Progress is tracked in [`.project/TASKS.md`](./.project/TASKS.md).

## What Is Implemented (v0.0.1 - v0.0.5)

- Core gateway: routing, reverse proxy, websocket proxy
- Load balancing: round robin, weighted, least_conn, ip_hash, consistent_hash, adaptive, and more
- Health checks: active and passive
- Plugin pipeline with route/global plugin configuration
- Authentication: API key and JWT (HS256 and RS256)
- Rate limiting: token bucket, fixed window, sliding window, leaky bucket
- Traffic controls: circuit breaker, retry, timeout, IP restrict, CORS
- Transform plugins: request/response transform, validation, request size limits, correlation IDs
- Embedded SQLite-backed data model for users, API keys, credits, and endpoint permissions
- User-level IP whitelist enforcement
- Extended admin API for users, keys, permissions, IP whitelist, credits, and billing config
- End-to-end test coverage for v0.0.5 scenarios

## Documentation

- Product specification: [`.project/SPECIFICATION.md`](./.project/SPECIFICATION.md)
- Implementation guide: [`.project/IMPLEMENTATION.md`](./.project/IMPLEMENTATION.md)
- Task roadmap and milestones: [`.project/TASKS.md`](./.project/TASKS.md)
- Example config: [`apicerberus.example.yaml`](./apicerberus.example.yaml)

## Requirements

- Go `1.26+`
- Make (optional, for convenience commands)

## Quick Start

1. Copy the example configuration:

```bash
cp apicerberus.example.yaml apicerberus.yaml
```

PowerShell:

```powershell
Copy-Item apicerberus.example.yaml apicerberus.yaml
```

2. Build:

```bash
make build
```

3. Validate config:

```bash
./bin/apicerberus config validate apicerberus.yaml
```

4. Edit `apicerberus.yaml` for your local environment:

- Set `admin.api_key` to a secure value.
- Update upstream targets (`upstreams[].targets[].address`) to reachable services.
- If you keep route host filters from the example config, send matching `Host` headers in requests.

5. Start gateway and admin API:

```bash
./bin/apicerberus start --config apicerberus.yaml
```

6. Check admin status:

```bash
curl -H "X-Admin-Key: change-me" http://127.0.0.1:9876/admin/api/v1/status
```

7. Stop process (from another terminal):

```bash
./bin/apicerberus stop
```

## Local Request Example

After configuring a reachable upstream and starting the server:

```bash
curl \
  -H "Host: api.example.com" \
  -H "X-API-Key: ck_live_mobile_abc123" \
  http://127.0.0.1:8080/api/v1/users
```

## CLI Commands

```text
apicerberus start [--config path] [--pid-file path]
apicerberus stop [--pid-file path]
apicerberus version
apicerberus config validate <path>
```

## Admin API Overview

The admin server is protected by `X-Admin-Key`.

Main groups currently available:

- System: `/admin/api/v1/status`, `/info`, `/config/reload`
- Services CRUD: `/admin/api/v1/services`
- Routes CRUD: `/admin/api/v1/routes`
- Upstreams CRUD + targets + health: `/admin/api/v1/upstreams`
- Users CRUD + suspend/activate/reset-password: `/admin/api/v1/users`
- User API keys: `/admin/api/v1/users/{id}/api-keys`
- User permissions: `/admin/api/v1/users/{id}/permissions`
- User IP whitelist: `/admin/api/v1/users/{id}/ip-whitelist`
- Credits and billing: `/admin/api/v1/credits/overview`, `/admin/api/v1/users/{id}/credits/*`, `/admin/api/v1/billing/*`

## Tests

Run the full test suite:

```bash
go test ./...
```

Run only end-to-end tests:

```bash
go test ./test
```

## Docker

Build and run using Docker:

```bash
docker build -t apicerberus:local .
docker run --rm -p 8080:8080 -p 9876:9876 apicerberus:local
```

## Repository Layout

- `cmd/apicerberus` - application entrypoint
- `internal` - gateway, plugins, admin API, billing, store, config
- `test` - E2E and integration tests
- `web` - dashboard assets
- `.project` - product docs, roadmap, and task breakdown

## Roadmap

Upcoming high-level milestones:

- `v0.0.6`: audit logging and analytics
- `v0.0.7`: web dashboard
- `v0.0.8`: user portal and API playground
- `v0.1.0`: MCP server and expanded CLI

See the full plan in [`.project/TASKS.md`](./.project/TASKS.md).
