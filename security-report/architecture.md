# Architecture Report — APICerebrus

**Scan Date:** 2026-04-25
**Project:** APICerebrus — Production API Gateway (Go + React)

---

## 1. Technology Stack Detection

### Languages (by LOC weight)
| Language | LOC | % | Notes |
|----------|-----|---|-------|
| Go | ~152,646 | ~85% | Primary application logic |
| TypeScript/TSX | ~10,000 est. | ~12% | React web dashboard |
| YAML | ~2,000 est. | ~2% | Config files, proto definitions |
| SQL | ~1,000 est. | ~1% | SQLite migrations |

### Frameworks & Libraries

**Go (core):**
- `net/http` — HTTP server foundation
- `google.golang.org/grpc` — gRPC server + HTTP transcoding
- `github.com/graphql-go/graphql` — GraphQL execution engine
- `modernc.org/sqlite` — Pure Go SQLite (WAL mode, no CGO)
- `github.com/redis/go-redis/v9` — Redis client for distributed rate limiting
- `github.com/golang-jwt/jwt/v5` — JWT verification (RS256, ES256)
- `github.com/coreos/go-oidc/v3` — OIDC/OAuth2 provider support
- `go.opentelemetry.io/otel` — Distributed tracing
- `golang.org/x/crypto` — Cryptography (Salsa20, Argon2, bcrypt)
- `github.com/tetratelabs/wazero` — WASM plugin runtime
- `hashicorp/raft` — Consensus (via direct integration patterns)

**TypeScript/React (web dashboard):**
- React 19 + React Router 7 (SPA)
- Tailwind v4 + shadcn/ui (component library)
- Zustand (state management)
- TanStack Query (data fetching)
- Recharts (analytics charts)
- CodeMirror 6 (YAML/JSON editors)

### Build Tools
| Tool | Usage |
|------|-------|
| Go 1.26 | Backend binary compilation |
| Vite 8 | React frontend bundling |
| TypeScript 5.9 | Type checking |
| Playwright | E2E testing |

---

## 2. Application Type Classification

**Primary Type:** API Gateway / Reverse Proxy

**Key capabilities:**
- Radix tree HTTP router (O(k) path matching)
- gRPC/WebSocket/HTTP/2 proxy
- Plugin-based request pipeline (5 phases)
- Raft-based distributed clustering
- GraphQL Federation (Apollo-compatible)
- ACME/Let's Encrypt certificate auto-provisioning
- Multi-tenant billing with credit system

**Secondary Type:** Security & Observability Platform
- Rate limiting (token bucket, sliding window, leaky bucket)
- Audit logging with GZIP archival + Kafka export
- OpenTelemetry tracing (multiple exporters)
- Alert engine with webhook notifications
- RBAC-based access control

---

## 3. Entry Points Mapping

### HTTP Servers

| Service | Port | Handler File |
|---------|------|--------------|
| Gateway HTTP | 8080 | `internal/gateway/server.go` |
| Gateway HTTPS | 8443 | `internal/gateway/server.go` (TLS) |
| Admin API | 9876 | `internal/admin/server.go` |
| User Portal | 9877 | `internal/portal/server.go` |
| gRPC | 50051 | `internal/grpc/proxy.go` |
| Raft | 12000 | `internal/raft/node.go` |

### HTTP Route Registration

**Gateway Routes** (`internal/gateway/router.go`):
- Method-scoped radix trees per host
- Supports: static paths, wildcard params (`*id`), regex routes
- Routes mapped: service → upstream targets with load balancing

**Admin API Routes** (`internal/admin/admin_routes.go`):
```
/admin/api/v1/users          — User management
/admin/api/v1/services       — Service CRUD
/admin/api/v1/routes         — Route management
/admin/api/v1/upstreams      — Upstream management
/admin/api/v1/billing         — Billing/credits
/admin/api/v1/audit          — Audit log search
/admin/api/v1/analytics       — Analytics/metrics
/admin/api/v1/cluster        — Raft cluster management
/admin/api/v1/webhooks       — Webhook configuration
/admin/api/v1/ws            — WebSocket for real-time updates
/admin/api/v1/oidc           — OIDC provider endpoints
```

**Portal Routes** (`internal/portal/server.go`):
- User-facing API key management, usage dashboard, playground

### gRPC Services
- Protobuf definitions (`.proto` files implied by `google.golang.org/protobuf`)
- HTTP transcoding for REST → gRPC bridging
- gRPC-Web support

### CLI Entry Points
- `apicerberus start` — Gateway startup
- `apicerberus user create/list/apikey/credit` — User management
- `apicerberus service/route/upstream add/list` — Entity management
- `apicerberus audit search/tail` — Audit log queries
- `apicerberus cluster join/leave/status` — Cluster management
- `apicerberus mcp start` — MCP server startup
- `apicerberus config validate/reload` — Configuration management

---

## 4. Data Flow Map

### Gateway Request Flow
```
Client → net/http Server (serve_http/serve_https)
       → Router.ServeHTTP (radix tree lookup)
       → Plugin Pipeline (5 phases):
           PRE_AUTH  → Correlation ID, IP restrictions, Bot detection
           AUTH      → API Key, JWT, User IP whitelist
           PRE_PROXY → Rate limiting, Request validation, Transforms, CORS
           PROXY     → Circuit breaker, Retry, Timeout, Caching
           POST_PROXY → Response transforms, Compression
       → Load Balancer (11 algorithms)
       → Upstream Target (HTTP/2, keep-alive, connection pooling)
       → Response through pipeline in reverse
       → Audit logging (async ring buffer)
```

### Admin API Data Flow
```
Admin request → Validate X-Admin-Key header
             → Rate limiting (per-key, 30 ops/min)
             → RBAC permission check
             → Handler (users/services/routes/upstreams/billing/audit)
             → SQLite (WAL mode, atomic transactions)
             → Webhook dispatch (async, retry with backoff)
             → WebSocket broadcast (real-time updates)
```

### Authentication Flow
```
OIDC Provider Flow:
  Authorization Endpoint → /admin/api/v1/oidc/authorize
  Token Endpoint → /admin/api/v1/oidc/token
  JWKS Endpoint → /admin/api/v1/oidc/jwks
  UserInfo Endpoint → /admin/api/v1/oidc/userinfo

API Key Flow:
  Header: X-API-Key: ck_live_* / ck_test_*
  → auth_apikey.go plugin

JWT Flow:
  Header: Authorization: Bearer <jwt>
  → auth_jwt.go plugin (RS256, ES256, OIDC自动发现)
```

---

## 5. Trust Boundaries

### Authentication Boundaries
| Boundary | Mechanism |
|----------|-----------|
| Admin API | `X-Admin-Key` header (32+ char secret) |
| Gateway API Keys | `ck_live_*` / `ck_test_*` prefix, stored hashed in SQLite |
| Gateway JWT | RS256/ES256 signature verification, OIDC auto-discovery |
| OIDC Auth | Authorization code flow, state/CORS validation |
| Raft Cluster | mTLS with auto-generated or manual certificates |

### Rate Limiting Boundaries
| Level | Mechanism |
|-------|-----------|
| Admin API | Per-key rate limiting (30 ops/min) |
| Gateway | Per-route and per-user limits (Redis-backed distributed) |
| Credit operations | Per-key rate limiting (30 credits/min) |

### Network Security
- Trusted proxies: configurable CIDR allowlist for X-Forwarded-For
- TLS: Minimum TLS 1.2, auto-provisioned via ACME
- Inter-node: mTLS for Raft communication
- No plaintext gRPC in production (TLS implied by configs)

### Input Validation
- Request size limits per plugin
- JSON body validation with size limits
- SQL parameterization (no raw query concatenation)
- Path traversal prevention via `filepath.Join` + prefix validation
- Header injection prevention (Go 1.22+ sanitization)

---

## 6. External Integrations

| Service | Integration | Security Config |
|---------|-------------|-----------------|
| SQLite | `modernc.org/sqlite` | WAL mode, busy timeout, atomic transactions |
| Redis | `go-redis/v9` | Used for distributed rate limiting, optional |
| Kafka | `internal/audit/kafka.go` | Optional async audit export |
| Let's Encrypt | ACME protocol via `internal/certmanager/acme.go` | Auto-renewal |
| OIDC Providers | `coreos/go-oidc/v3` | Auto-discovery + JWKS caching |

---

## 7. Authentication Architecture

| Method | Implementation | Location |
|--------|----------------|-----------|
| Admin API Key | `X-Admin-Key` header validation | `internal/admin/server.go` |
| API Key (live) | `ck_live_*` prefix → SQLite lookup | `internal/plugin/auth_apikey.go` |
| API Key (test) | `ck_test_*` prefix → bypass credits | `internal/plugin/auth_apikey.go` |
| JWT (RS256/ES256) | `jwt.go` + `rs256.go`/`es256.go` | `internal/pkg/jwt/` |
| OIDC | Authorization code flow, state validation | `internal/admin/oidc_provider.go` |
| IP Whitelist | `user_ip_whitelist.go` plugin | `internal/plugin/` |

---

## 8. File Structure Analysis

```
D:\CODEBOX\PROJECTS\APICerebrus\
├── cmd/apicerberus/main.go          # Entry point, CLI dispatcher
├── embed.go                         # Web dashboard embedding
├── internal/
│   ├── admin/                       # Admin REST API (port 9876)
│   ├── analytics/                   # Alert engine, metrics
│   ├── audit/                       # Request/response logging
│   ├── billing/                     # Credit engine
│   ├── certmanager/                 # ACME/Let's Encrypt
│   ├── cli/                         # 40+ CLI commands
│   ├── config/                      # YAML config loading, hot reload
│   ├── federation/                  # GraphQL Federation
│   ├── gateway/                     # HTTP/GRPC proxy, router, balancer
│   ├── graphql/                     # GraphQL execution
│   ├── grpc/                        # gRPC server, transcoding
│   ├── loadbalancer/                # 11 load balancing algorithms
│   ├── logging/                     # Structured logging
│   ├── mcp/                         # Model Context Protocol server
│   ├── metrics/                     # Prometheus exporter
│   ├── migrations/                 # SQLite schema migrations
│   ├── plugin/                      # 5-phase plugin pipeline, 20+ plugins
│   ├── portal/                      # User-facing portal (port 9877)
│   ├── raft/                       # Distributed consensus, mTLS
│   ├── ratelimit/                   # Token bucket, sliding window, etc.
│   ├── store/                       # SQLite repositories
│   └── tracing/                     # OpenTelemetry
├── web/                             # React + TypeScript dashboard
│   ├── src/
│   │   ├── components/             # UI components
│   │   ├── hooks/                  # React Query hooks
│   │   ├── lib/                    # API client, WebSocket
│   │   ├── pages/                  # Route pages
│   │   └── stores/                 # Zustand state
│   └── package.json               # React 19, Tailwind v4, Vite
├── deployments/docker/             # Docker build
├── Dockerfile                      # Multi-stage Go build
├── apicerberus.yaml               # Runtime config
└── apicerberus.example.yaml       # Config documentation
```

### Sensitive Files
| File | Risk |
|------|------|
| `apicerberus.yaml` | Contains admin API key, OIDC secrets, TLS certs |
| `internal/raft/tls.go` | mTLS certificate generation logic |
| `internal/admin/oidc_provider.go` | OIDC state management, signing keys |
| `internal/store/` | SQLite DB with user data, API keys, audit logs |

---

## 9. Detected Security Controls

| Control | Location | Implementation |
|---------|----------|----------------|
| Input validation | `internal/plugin/request_validator.go` | Schema-based validation |
| Rate limiting | `internal/plugin/rate_limit.go` | Token bucket + Redis backend |
| IP restrictions | `internal/plugin/ip_restrict.go` | CIDR-based allow/deny |
| CORS | `internal/plugin/cors.go` | Configurable origin allowlist |
| Bot detection | `internal/plugin/bot_detect.go` | Header-based detection |
| Circuit breaker | `internal/plugin/circuit_breaker.go` | Failure threshold + recovery |
| Request size limit | `internal/plugin/request_size_limit.go` | Max body bytes config |
| Panic recovery | `internal/plugin/pipeline.go` | Middleware-level recover |
| Panic middleware | Implicit in HTTP handlers | Prevents crashes |
| Graceful shutdown | `internal/shutdown/manager.go` | LIFO hook execution |
| Client IP extraction | `internal/pkg/netutil/clientip.go` | XFF parsing with trusted proxy logic |
| Audit logging | `internal/audit/logger.go` | Ring buffer, GZIP, Kafka export |
| Field masking | `internal/audit/masker.go` | PII redaction |
| Retention policy | `internal/audit/retention.go` | Time-based cleanup |
| Webhook signatures | `internal/admin/webhooks.go` | HMAC-SHA256 |

---

## 10. Language Detection Summary

- **Go (95% of codebase)** → activates `sc-lang-go`
- **TypeScript (5% of codebase)** → activates `sc-lang-typescript`
- **Python** → None detected (no Python scripts/tools)
- **No other languages detected**

---

## Detected Entry Points (Summary)

| Entry Point Type | Count | Examples |
|------------------|-------|----------|
| HTTP handlers | 50+ | Gateway proxy, Admin API, Portal |
| gRPC services | 5+ | Health, Proxy, Stream |
| CLI commands | 40+ | User, Service, Route, Cluster, MCP |
| WebSocket | 2 | Admin WS hub, Portal WS |
| OIDC endpoints | 4 | Authorize, Token, JWKS, UserInfo |

---

## Architecture Notes

1. **Monolithic Go binary** with embedded React dashboard — single deployable artifact
2. **SQLite as single source of truth** — WAL mode, atomic transactions for billing consistency
3. **Raft clustering** — leader-based consensus with mTLS inter-node encryption
4. **Plugin architecture** — WASM support with hot-reload, 5-phase pipeline
5. **Security-first design** — AuthZ/RBAC, rate limiting, audit logging, TLS everywhere
6. **No external ORM** — raw `database/sql` with repository pattern

---

*Generated by sc-recon (Phase 1 of security-check pipeline)*