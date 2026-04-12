# APICerebrus Architecture Map

**Generated:** 2026-04-09

## System Overview

APICerebrus is a production-ready API Gateway built in Go with a React-based admin dashboard. It provides routing, authentication, rate limiting, billing/credits, audit logging, GraphQL Federation, and Raft-based clustering.

## Services & Ports

| Service | Port | Protocol | Auth |
|---------|------|----------|------|
| Gateway HTTP | 8080 | HTTP/1.1, WebSocket | Plugin pipeline (configurable) |
| Gateway HTTPS | 8443 | HTTP/2, TLS | Plugin pipeline (configurable) |
| Admin API | 9876 | HTTP/JSON | X-Admin-Key / Bearer JWT |
| User Portal | 9877 | HTTP/HTML + REST | Session cookie / API key |
| gRPC | 50051 | HTTP/2 + Protobuf | gRPC metadata auth |
| Raft Consensus | 12000 | HTTP/RPC | Shared token (non-constant-time) |

## Tech Stack

### Backend (Go)
- **Language:** Go 1.25.0 (installed: 1.26.1)
- **Database:** SQLite via `modernc.org/sqlite` (pure Go, no CGO), WAL mode
- **Raft:** Custom implementation (not hashicorp/raft)
- **WebSockets:** `github.com/gorilla/websocket` + `golang.org/x/net/websocket` (deprecated)
- **GraphQL:** `github.com/graphql-go/graphql` v0.8.1 (stale)
- **Custom parsers:** JWT (`internal/pkg/jwt/`), YAML (`internal/pkg/yaml/`)

### Frontend (React)
- **Framework:** React + TypeScript + Vite
- **Styling:** Tailwind CSS v4 + shadcn/ui
- **State:** React Query for data fetching
- **WebSocket:** Custom client for real-time updates (`web/src/lib/ws.ts`)

### Infrastructure
- **Containerization:** Docker (Dockerfile, docker-compose, docker-compose.cluster, docker-compose.swarm)
- **Orchestration:** Kubernetes (base manifests, overlays for dev/staging/prod)
- **Package Management:** Helm charts
- **CI/CD:** GitHub Actions (.github/workflows/ci.yml)

## Core Modules (`internal/`)

| Module | Purpose | Key Files |
|--------|---------|-----------|
| `gateway/` | HTTP/gRPC/WebSocket servers, radix tree router, proxy engine, 10 LB algorithms | `router.go`, `optimized_proxy.go`, `server.go`, `health.go` |
| `plugin/` | 5-phase pipeline (PRE_AUTH→AUTH→PRE_PROXY→PROXY→POST_PROXY), 20+ plugins | `pipeline.go`, `auth_apikey.go`, `auth_jwt.go`, `cors.go` |
| `admin/` | REST API for management, webhook delivery, JWT token management | `server.go`, `admin_users.go`, `token.go`, `webhooks.go`, `ws.go` |
| `portal/` | User-facing web portal with playground | `server.go`, `handlers_playground_usage.go` |
| `store/` | SQLite repositories (WAL mode): users, API keys, sessions, audit logs | `user_repo.go`, `api_key_repo.go`, `session_repo.go`, `audit_repo.go` |
| `raft/` | Custom Raft consensus, FSM, transport, TLS cert manager | `node.go`, `fsm.go`, `transport.go`, `tls.go` |
| `federation/` | GraphQL Federation (schema composition, query planning, executor) | `composer.go`, `planner.go`, `executor.go`, `subgraph.go` |
| `analytics/` | Metrics with ring buffers, time-series aggregation, webhook templates | `engine.go`, `webhook_templates.go` |
| `audit/` | Async request/response logging with field masking, Kafka export | `logger.go`, `masker.go`, `retention.go`, `kafka.go` |
| `ratelimit/` | Token bucket, fixed/sliding window, leaky bucket; Redis-backed | — |
| `billing/` | Credit system with atomic SQLite transactions | — |
| `mcp/` | Model Context Protocol server (stdio + SSE transports) | — |
| `config/` | Configuration loading, env overrides, hot reload (SIGHUP) | `load.go`, `types.go`, `dynamic_reload.go` |
| `shutdown/` | LIFO shutdown hook system | `manager.go` |
| `pkg/netutil/` | Client IP extraction with trusted proxy support | `clientip.go` |

## Plugin Pipeline

```
PRE_AUTH → AUTH → PRE_PROXY → PROXY → POST_PROXY
```

| Phase | Plugins |
|-------|---------|
| PRE_AUTH | Correlation ID, IP restrictions, bot detection |
| AUTH | API key, JWT, user IP whitelist |
| PRE_PROXY | Rate limiting, request validation, transforms, CORS, JSON schema |
| PROXY | Circuit breaker, retry, timeout, caching, request coalescing |
| POST_PROXY | Response transforms, compression |

## Data Stores

| Store | Type | Location | Mode |
|-------|------|----------|------|
| Primary DB | SQLite (pure Go) | `apicerberus.db` | WAL mode, busy timeout 5s |
| Rate Limits | In-memory / Redis | Configurable | Distributed via Redis |
| Analytics | Ring buffers (in-memory) | — | Lock-free circular buffers |
| Audit Logs | SQLite (async flush) | Same DB | Buffered, batch flush |
| Sessions | SQLite | Same DB | Configurable TTL |
| Config | YAML file + in-memory | `apicerberus.yaml` | Hot reloadable via SIGHUP |

## Authentication Flow

```
Client Request
  → Trusted Proxy Check (ignore X-Forwarded-For unless configured)
  → IP Allowlist (if configured)
  → Admin API Key Rate Limit (5 failures → 15 min lockout)
  → Auth Middleware:
    1. Bearer JWT (verify against token secret)
    2. Fallback: X-Admin-Key static comparison
  → Route-specific plugins:
    - API Key Auth (header, query, cookie, or Bearer)
    - JWT Auth (RS256/HS256 with JTI replay protection)
    - IP Whitelist per user
```

## Request Flow

```
Client → Gateway (Radix Router, O(k) lookup)
  → Plugin Pipeline (5 phases)
  → Load Balancer (11 algorithms)
  → Upstream Service

Parallel:
  → Audit Logger (async ring buffer → SQLite)
  → Analytics Engine (ring buffers → time-series)
  → Webhook Dispatcher (async with retry)
```

## Security Controls

### Implemented Correctly
- Parameterized SQL queries (zero string concatenation)
- crypto/rand for all secret generation
- bcrypt cost 10 for passwords
- SHA-256 hashed API keys in database
- Constant-time comparisons for auth
- Trusted proxy: forwarding headers ignored by default
- Security headers: HSTS, CSP, X-Frame-Options, X-Content-Type-Options
- Raft TLS 1.3 with 4096-bit RSA certs
- K8s: non-root user, dropped capabilities, read-only filesystem

### Known Weaknesses
- Custom JWT and YAML parsers (not independently audited)
- Raft RPC endpoints lack authentication
- Homegrown request coalescing without auth identity in key
- CORS credentials with wildcard origins possible
- Portal playground acts as SSRF proxy
- Secrets accepted in URL query parameters
- No body size limits on Raft RPC endpoints
- YAML parser has no depth/node limits

## Entry Points

| Entry Point | File | Authentication |
|-------------|------|----------------|
| `cmd/apicerberus/main.go` | Main binary | — |
| Gateway HTTP | `internal/gateway/server.go` | Plugin pipeline |
| Admin API | `internal/admin/server.go` | X-Admin-Key / JWT |
| User Portal | `internal/portal/server.go` | Session cookie |
| gRPC | `internal/grpc/server.go` | gRPC metadata |
| Raft RPC | `internal/raft/transport.go` | **None** (CRITICAL) |
| MCP Server | `internal/mcp/` | stdio/SSE |
| WebSocket (Admin) | `internal/admin/ws.go` | Cookie / Bearer / Query token / Static key |
| WebSocket (GraphQL) | `internal/graphql/subscription.go` | Via gateway auth |

## Deployment Topologies

1. **Standalone:** Single binary with embedded SQLite
2. **Docker Compose:** 3-node cluster with nginx load balancer
3. **Kubernetes:** Deployment + NetworkPolicy + Service + Secrets
4. **Docker Swarm:** 3-node cluster with overlay network
5. **Helm:** Parameterized K8s deployment with configurable values
