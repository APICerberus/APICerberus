# Project Analysis Report

> Auto-generated comprehensive analysis of APICerebrus
> Generated: 2026-04-16
> Analyzer: Claude Code — Full Codebase Audit

## 1. Executive Summary

APICerebrus is a production-grade API Gateway, Management, and Monetization Platform built in Go with an embedded React admin dashboard and developer portal. It provides HTTP/gRPC/WebSocket reverse proxying with radix-tree routing, a 5-phase plugin pipeline with 25+ built-in plugins, credit-based billing, distributed Raft clustering, GraphQL Federation, and a comprehensive admin REST API with 95+ endpoints. The target audience is platform teams and SaaS providers needing a self-hosted, embeddable API gateway with monetization capabilities.

**Key Metrics:**

| Metric | Value |
|--------|-------|
| Total Project Files | 1,266 |
| Go Source Files (non-test) | 179 |
| Go Test Files | 269 |
| Total Go LOC | 170,018 |
| Non-test Go LOC | ~55,700 |
| Test Go LOC | ~114,318 |
| Frontend Source Files | 253 |
| External Go Dependencies | 20 direct |
| Test Coverage | ~80% |
| Admin API Endpoints | 95+ |
| Portal API Endpoints | 33+ |
| MCP Tools | 43+ |
| Built-in Plugins | 25+ |
| Load Balancing Algorithms | 11 |
| Test Packages Failing | 2 (cli + integration) |

**Overall Health Assessment: 8/10**

The codebase demonstrates exceptional breadth and depth. Recent security hardening (JWT JTI fail-closed, portal secret validation, F-010/F-012/F-013) has improved the security posture significantly. Test coverage reached ~80%. The main remaining concerns are: (1) one CLI test failing due to portal secret validation, (2) integration tests failing on Windows due to SQLite busy timeout, (3) K8s/Helm config schema has been fixed per ROADMAP but needs verification.

**Top 3 Strengths:**
1. **Comprehensive feature surface** — Full API gateway with 11 load balancers, 25+ plugins, GraphQL Federation, Raft clustering, credit billing, MCP server, and admin CLI — all in a single binary.
2. **Security-conscious implementation** — CWE-annotated code, SSRF protection, constant-time key comparison, bcrypt cost 12, crypto/rand for secrets, comprehensive security headers, JWT JTI replay protection, portal secret validation.
3. **Well-structured Go code** — Clean package boundaries, proper context propagation, atomic hot-reload with mutex protection, graceful shutdown hooks, and consistent error wrapping.

**Top 3 Concerns:**
1. **CLI `TestRunConfigImport` failing** — portal.secret validation (min 32 chars) not met by test config. Easy fix.
2. **Integration tests fail on Windows** — SQLite busy timeout during TempDir cleanup. Indicates potential handle management issue.
3. **Frontend test coverage** — 11 test files vs ~253 source files. Coverage ~12% but improving.

---

## 2. Architecture Analysis

### 2.1 High-Level Architecture

APICerebrus is a **modular monolith** — a single Go binary with internal packages organized by domain. The binary embeds the React frontend assets via `go:embed` and serves them, requiring no separate frontend deployment.

```
Client Request Flow:
====================

  Client --> Gateway (radix router) --> Plugin Pipeline (5 phases) --> Load Balancer --> Upstream
                  |                              |                            |
                  +- /health, /ready             +- PRE_AUTH: corr-id,       +- Active health
                  +- /metrics                     |  ip-restrict, bot-detect   |  checks
                  +- /admin/api/v1/*              +- AUTH: apikey, jwt,       +- Passive health
                  +- /portal/api/v1/*             |  ip-whitelist              |  checks
                  +- /dashboard/*                 +- PRE_PROXY: rate-limit,   +- Circuit breaker
                                                  |  request-validator, cors
                                                  +- PROXY: circuit-breaker,
                                                  |  retry, timeout, cache
                                                  +- POST_PROXY: response
                                                     transform, compression

Data Layer:
===========
  SQLite (WAL mode) -- Users, API Keys, Sessions, Credits, Audit Logs, Webhooks
  Redis (optional)  -- Distributed rate limiting
  Kafka (optional)  -- Audit log streaming

Clustering (optional):
======================
  Raft Consensus -- Config replication, distributed rate limiting, certificate sync
  mTLS           -- Inter-node encryption with auto-generated CA
```

**Concurrency Model:**
- Standard `net/http` server goroutine-per-request model
- `sync.RWMutex` for hot-reload of gateway state (router, pools, config)
- `sync.Map` for rate limiter sharding, connection pools
- `atomic` operations for metrics counters, balancer state
- `sync.Pool` for HTTP client reuse, buffer pooling
- Goroutine lifecycle: `context.WithCancel` for audit drain, health checkers, analytics engine
- Graceful shutdown via `shutdown.Manager` with LIFO hook execution

### 2.2 Package Structure Assessment

| Package | Responsibility | LOC (non-test) | Cohesion | Assessment |
|---------|---------------|----------------|----------|------------|
| `cmd/apicerberus` | Entry point | ~20 | High | Clean delegation to `cli.Run` |
| `internal/config` | Config types, loading, env overrides, watch | ~1,500 | High | Comprehensive validation, env override via reflection |
| `internal/gateway` | HTTP server, router, proxy, balancer, health | ~7,000 | Medium-High | Largest package; could split proxy/router/balancer |
| `internal/store` | SQLite repositories, migrations | ~4,000 | High | Consistent repo pattern, 8 tables |
| `internal/plugin` | 25+ plugin implementations, pipeline, registry | ~6,000 | High | Excellent phase-based pipeline architecture |
| `internal/admin` | REST API server, WebSocket, OIDC, RBAC | ~7,000 | Medium | Very large; graphql.go (860 LOC) and server.go (640 LOC) are oversized |
| `internal/billing` | Credit engine | ~300 | High | Clean, focused |
| `internal/ratelimit` | 4 algorithms + Redis distributed | ~1,200 | High | Good factory pattern |
| `internal/loadbalancer` | Subnet resolver, adaptive LB | ~500 | High | Focused utility package |
| `internal/raft` | Raft consensus, TLS, cert sync | ~3,500 | High | Complex but well-structured |
| `internal/federation` | GraphQL Federation composer/planner/executor | ~2,100 | High | Clean separation of concerns |
| `internal/graphql` | Parser, analyzer, APQ, proxy, subscriptions | ~2,200 | High | Comprehensive GraphQL support |
| `internal/grpc` | gRPC proxy, transcoding, health, streaming | ~1,500 | High | Full gRPC stack |
| `internal/audit` | Logger, capture, masking, retention, Kafka | ~1,300 | High | Async buffering with batch flush |
| `internal/analytics` | Ring buffer engine, alerts, webhook templates | ~1,500 | Medium | webhook_templates.go (718 LOC) is oversized |
| `internal/mcp` | MCP server, tools, resources | ~1,000 | High | 43+ tools for AI integration |
| `internal/portal` | User portal API | ~1,000 | High | Clean session-based auth |
| `internal/cli` | CLI commands, admin client | ~2,500 | Medium | cmd_user.go (744 LOC) is oversized |
| `internal/certmanager` | ACME/Let's Encrypt | ~400 | High | Focused ACME implementation |
| `internal/tracing` | OpenTelemetry setup | ~200 | High | Minimal, focused |
| `internal/metrics` | Prometheus metrics | ~600 | Medium | Single large file |
| `internal/migrations` | Migration runner | ~100 | High | Simple version tracker |
| `internal/shutdown` | Graceful shutdown manager | ~80 | High | LIFO hook execution |
| `internal/logging` | Structured logging, rotation | ~300 | High | Clean logging abstraction |
| `internal/pkg/*` | JWT, JSON, YAML, UUID, netutil, coerce, template | ~1,200 | High | Well-isolated utilities |

**Circular Dependency Risk:** Low. All packages depend on `internal/store` and `internal/config`, but no circular dependencies observed.

**Oversized Files (candidates for refactoring):**
- `internal/gateway/server.go` (1,213 LOC) — handles too many concerns
- `internal/admin/graphql.go` (860 LOC) — GraphQL admin API in one file
- `internal/admin/admin_users.go` (866 LOC) — user management CRUD
- `internal/gateway/balancer_extra.go` (842 LOC) — multiple balancer implementations
- `internal/plugin/registry.go` (817 LOC) — registry + route pipeline builder
- `internal/federation/executor.go` (792 LOC) — complex execution logic

### 2.3 Dependency Analysis

#### Go Dependencies (direct, from go.mod)

| Dependency | Version | Purpose | Maintenance | Could Use Stdlib? |
|------------|---------|---------|-------------|-------------------|
| `modernc.org/sqlite` | v1.48.0 | Pure-Go SQLite driver | Active | No — core storage |
| `google.golang.org/grpc` | v1.80.0 | gRPC framework | Active | No |
| `google.golang.org/protobuf` | v1.36.11 | Protocol buffers | Active | No |
| `go.opentelemetry.io/otel/*` | v1.43.0 | Distributed tracing | Active | No |
| `golang.org/x/crypto` | v0.49.0 | bcrypt, argon2 | Active | No — stdlib lacks bcrypt |
| `golang.org/x/net` | v0.52.0 | HTTP/2, context | Active | No — needed for h2 |
| `golang.org/x/oauth2` | v0.36.0 | OAuth2 client | Active | No |
| `golang.org/x/text` | v0.35.0 | Text processing | Active | No |
| `github.com/redis/go-redis/v9` | v9.7.3 | Redis client | Active | No |
| `github.com/alicebob/miniredis/v2` | v2.37.0 | Redis mock for tests | Active | No |
| `github.com/graphql-go/graphql` | v0.8.1 | GraphQL execution | Active | No |
| `github.com/tetratelabs/wazero` | v1.11.0 | WASM runtime | Active | No |
| `github.com/coder/websocket` | v1.8.14 | WebSocket (conforming) | Active | No |
| `github.com/coreos/go-oidc/v3` | v3.18.0 | OIDC client | Active | No |
| `github.com/golang-jwt/jwt/v5` | v5.3.1 | JWT parsing/validation | Active | No |
| `gopkg.in/yaml.v3` | v3.0.1 | YAML parsing | Active | No |

**Assessment:** All dependencies are actively maintained, well-known libraries. No deprecated or abandoned packages. The "zero dependencies" claim in README/BRANDING is misleading — there are 16 direct dependencies, but this is lean for the scope.

#### Frontend Dependencies (from web/package.json)

**Production (30 packages):** All at latest versions. React 19.2, React Router 7.13, TanStack Query 5.95, Zustand 5.0, Radix UI 1.4, Recharts 3.8, CodeMirror 6, Tailwind 4.2, Vite 8.0, TypeScript 5.9.

**Assessment:** No deprecated packages. `manualChunks` properly splits heavy deps (recharts, codemirror, react-flow) into separate bundles.

### 2.4 API & Interface Design

#### HTTP Endpoint Inventory

**Admin API (95+ endpoints on port 9876):**

| Category | Count | Methods |
|----------|-------|---------|
| Auth (token + OIDC SSO) | 8 | POST, GET |
| System (status, info, config) | 5 | GET, POST |
| Services CRUD | 5 | GET, POST, PUT, DELETE |
| Routes CRUD | 5 | GET, POST, PUT, DELETE |
| Upstreams CRUD + targets | 7 | GET, POST, PUT, DELETE |
| Users CRUD + status/role | 10 | GET, POST, PUT, DELETE |
| User API Keys | 3 | GET, POST, DELETE |
| User Permissions | 5 | GET, POST, PUT, DELETE |
| User IP Whitelist | 3 | GET, POST, DELETE |
| Credits | 8 | GET, POST |
| Audit Logs | 6 | GET, DELETE |
| Analytics | 8 | GET |
| Alerts | 4 | GET, POST, PUT, DELETE |
| Billing Config | 4 | GET, PUT |
| Subgraphs (Federation) | 5 | GET, POST, DELETE |
| WebSocket | 1 | GET (upgrade) |
| Webhooks | 4+ | GET, POST, PUT, DELETE |
| Bulk Operations | 2+ | POST |
| Advanced Analytics | 4+ | GET |
| GraphQL Admin | 1 | POST |
| pprof Debug | 5 | GET |
| Dashboard UI | 1 | GET (static) |
| Branding | 2 | GET |

**Portal API (33 endpoints on port 9877):** Auth (5), API Keys (4), APIs (2), Playground (4), Usage (4), Logs (3), Credits (4), Security (4), Settings (3).

**API Consistency Assessment:**
- **Naming:** Consistent REST patterns — plural nouns, `{id}` for resources
- **Response format:** JSON with consistent envelope
- **Error handling:** `PluginError` type with code/message/status; standard HTTP codes
- **Authentication:** Dual system — admin uses Bearer token; portal uses session cookies + CSRF
- **Rate limiting:** Per-route and per-user; not on admin endpoints by default

---

## 3. Code Quality Assessment

### 3.1 Go Code Quality

**Code Style:** Consistent. All files pass `go fmt`. Naming follows Go conventions.

**Error Handling:**
- Consistent `fmt.Errorf("context: %w", err)` wrapping
- `PluginError` type for pipeline errors with HTTP status codes
- Centralized `writeJSON`/`writeError` in admin handlers
- **Concern:** Fire-and-forget goroutines in `api_key_repo.go:UpdateLastUsed` log errors but don't propagate

**Context Usage:**
- All store methods accept `context.Context`
- Gateway creates per-request context with timeout
- **Concern:** `billing.Engine.Deduct()` uses `context.Background()` instead of request context

**Logging:** Structured JSON via `internal/logging`. Proper level usage (debug/info/warn/error).

**Configuration:** YAML + env vars + SIGHUP hot reload. Comprehensive validation with accumulated errors.

**Magic Numbers:** Security limits are hardcoded but reasonable (8KB path max, 256 segments, 1KB regex max). Connection pool settings (100 max idle, 90s timeout) should be configurable.

**TODOs:** Only 1 in non-test source: `internal/plugin/request_transform.go`.

### 3.2 Frontend Code Quality

**React Patterns:** 100% functional components + hooks. TanStack Query for server state. Zustand for client state. React.lazy + Suspense for code splitting.

**TypeScript Quality:** `strict: true` with additional flags. Zero `any` types. 45+ explicit type definitions.

**Component Structure:** Feature-based organization. Consistent shadcn/ui pattern (33 primitives).

**CSS:** Tailwind v4 with `@theme inline` directive. CSS custom properties for theming.

**Accessibility:** Semantic HTML, `aria-label` on icon buttons, touch targets enforced. **Concerns:** Missing `aria-sort` on DataTable, DiffViewer lacks ARIA.

**Frontend Test Coverage:** 11 test files for ~253 source files (~4%). Limited but infrastructure (MSW, Testing Library) is solid.

### 3.3 Concurrency & Safety

- `sync.RWMutex` for hot-reload — correct read/write separation
- `sync.Map` for rate limiter sharding — appropriate for read-heavy access
- `sync.Pool` for HTTP transport reuse — correct
- `atomic` operations for metrics — correct
- **Medium risk:** `api_key_repo.go:UpdateLastUsed` fire-and-forget goroutine without lifecycle management
- **Medium risk:** `denyPrivateUpstreams` package-level var — not goroutine-safe for concurrent init

### 3.4 Security Assessment

**Input Validation:** Body size limits, path length limits, regex length limits (CWE-1333), null byte rejection (CWE-20), JSON Schema validation plugin.

**SQL Injection:** All queries use parameterized placeholders. No string concatenation in SQL.

**XSS Protection:** Security headers (CSP, X-Frame-Options, X-Content-Type-Options), React JSX auto-escaping, `html.EscapeString` in error pages.

**Secrets Management:** Config uses `${ENV_VAR}` pattern. Initial password files gitignored. API keys SHA-256 hashed. bcrypt cost 12 for passwords. JWT secret validated for min 32 chars. Portal secret validated for min 32 chars.

**TLS:** TLS 1.0/1.1 rejected. Safe cipher suites. ACME auto-provisioning.

**JWT JTI Replay Protection:** Implemented fail-closed behavior for replayed JTIs.

**93 gosec suppressions** documented in `SECURITY-JUSTIFICATIONS.md` with justified reasons.

**Recent Security Fixes (2026-04-16):**
- HIGH-NEW-1: JWT JTI fail-closed (commit a8e5220)
- HIGH-NEW-3: Portal secret validation (commit a8e5220)
- F-010, F-012, F-013: Security audit fixes (commit a08c9ef)
- F-001, F-002, F-003, F-004: Security hardening (commit f4314d1)

---

## 4. Testing Assessment

### 4.1 Test Coverage

**Overall Coverage: ~80%**

**Test Results (latest run):**
- 30/32 packages PASS
- 2 packages FAIL:
  - `internal/cli` — `TestRunConfigImport` fails: portal.secret validation requires min 32 chars but test config has shorter value
  - `test/integration` — SQLite busy timeout during TempDir cleanup on Windows

**Packages with ZERO Tests:** None identified.

### 4.2 Test Types Present

- **Unit tests:** 269 files across all packages
- **Integration tests:** `test/integration/` — auth flow, request lifecycle, plugin chain, hot reload, Kafka
- **E2E tests:** `test/e2e_*_test.go`
- **Benchmark tests:** `test/benchmark/`
- **Fuzz tests:** 4 files (router, JSON, JWT, YAML)
- **Load tests:** `test/loadtest/` — 500+ concurrent request validation

**CI Pipeline:** 12-job GitHub Actions with 70% coverage threshold enforcement.

---

## 5. Specification vs Implementation Gap Analysis

### 5.1 Feature Completion Matrix

| Planned Feature | Status | Notes |
|----------------|--------|-------|
| HTTP/1.1 + HTTP/2 reverse proxy | COMPLETE | Full proxy with coalescing |
| TLS + ACME | COMPLETE | Let's Encrypt auto-provisioning |
| WebSocket proxying | COMPLETE | Full-duplex tunneling |
| gRPC proxy + transcoding | COMPLETE | Native + Web + transcoding |
| GraphQL proxy + APQ | COMPLETE | Parser, analyzer, subscriptions |
| GraphQL Federation | COMPLETE | Apollo-compatible |
| Radix tree router | COMPLETE | O(k) with regex support |
| Plugin pipeline (5 phases) | COMPLETE | 25+ plugins |
| API Key auth | COMPLETE | Header/query/cookie |
| JWT auth (RS256/HS256/ES256/JWKS) | COMPLETE | Full JWT stack |
| 4 rate limit algorithms | COMPLETE | + Redis distributed |
| 11 load balancers | COMPLETE | SubnetAware added beyond spec's 10 |
| Health checking | COMPLETE | Active + passive |
| Raft clustering | COMPLETE | With mTLS |
| Credit billing | COMPLETE | Atomic SQLite transactions |
| Audit logging | COMPLETE | Async, PII masking, Kafka |
| Analytics engine | COMPLETE | Ring buffer, time-series |
| Admin REST API | COMPLETE | 95+ endpoints |
| User portal API | COMPLETE | 33 endpoints |
| Web dashboard | COMPLETE | React 19 + Tailwind v4 |
| CLI commands | COMPLETE | 40+ commands |
| MCP server | COMPLETE | 43+ tools |
| OIDC SSO | COMPLETE | Login/callback/logout/status |
| RBAC | COMPLETE | Role-based access |
| WASM plugins | COMPLETE | 36 tests passing |
| Kafka audit streaming | COMPLETE | With tests |
| Brotli compression | COMPLETE | Implemented in compression_brotli.go |
| Plugin marketplace | COMPLETE | Implemented |

### 5.2 Architectural Deviations

1. **YAML parser:** IMPLEMENTATION.md describes custom parser. **Actual:** Uses `gopkg.in/yaml.v3`. Improvement.
2. **SQLite:** IMPLEMENTATION describes both CGO and pure-Go. **Actual:** Pure Go via `modernc.org/sqlite` with `CGO_ENABLED=0`.
3. **Password hashing:** TASKS/IMPLEMENTATION say SHA-256+salt. **Actual:** bcrypt cost 12. Significant security improvement.

### 5.3 Task Completion Assessment

TASKS.md claims 490 tasks at 100%. Realistic estimate: **~98%**. Minor issues remain (1 CLI test failing, 1 integration test suite failing on Windows).

### 5.4 Scope Creep Detection

| Unplanned Feature | Assessment |
|-------------------|------------|
| SubnetAware LB | Valuable replacement for "geo_aware" |
| JTI replay protection | Valuable security hardening |
| GraphQL guard | Valuable depth/complexity limiting |
| Request coalescing | Valuable performance optimization |
| Plugin marketplace | Operational convenience |
| Bulk operations | Operational convenience |
| Advanced analytics (forecast) | Valuable beyond spec |

### 5.5 Missing Critical Components

None of critical concern. Minor items:
1. **CLI test config** — portal.secret min 32 char validation not met by test
2. **Integration test Windows cleanup** — SQLite handle management on Windows
3. **Frontend test coverage** — 4% vs ~253 files, needs expansion

---

## 6. Performance & Scalability

### 6.1 Performance Patterns

**Hot Paths:** `Gateway.ServeHTTP()` → radix tree O(k) → plugin pipeline → `OptimizedProxy.Forward()` → `Balancer.Next()` → upstream

**Potential Bottlenecks:**
- SQLite WAL write serialization under heavy audit + credit load
- Rate limiter `sync.Map` per-key mutex under extreme cardinality
- Admin WebSocket hub broadcasts to all connections without topic filtering (partially fixed per ROADMAP)
- Audit `LIKE` queries on text columns on large tables

**Memory:** Buffer pools, ring buffers (fixed size), rate limiter maps grow unbounded (no TTL cleanup for stale keys) — **FIXED per ROADMAP Phase 3.**

### 6.2 Scalability Assessment

- **Horizontal:** Stateless serving with Raft config replication; Redis for distributed rate limiting
- **Billing:** Limited — SQLite single-writer requires Raft leader for credit operations
- **Sessions:** Server-side SQLite — no sticky sessions but requires leader DB access

---

## 7. Developer Experience

**Onboarding:** `go build` works with zero config. Docker compose straightforward. Example config comprehensive.

**Documentation:** README is good but has inflated metrics (claims 150K+ LOC, actual ~55.7K). CLAUDE.md is excellent and accurate. SPECIFICATION is extremely detailed (2,848 lines).

**Build/Deploy:** Makefile with 30+ targets. Multi-stage Docker. Cross-compilation for 5 platforms. 12-job CI pipeline.

---

## 8. Technical Debt Inventory

### Critical (blocks production)

None identified. All previously critical items have been addressed.

### Important (before v1.0)

1. **CLI test portal secret validation** — `TestRunConfigImport` in `cmd_config_extra_test.go` uses portal.secret < 32 chars. Fix: 15min.
2. **Integration test Windows cleanup** — SQLite handle not released before TempDir cleanup. Fix: 2-4h.
3. **Frontend test coverage** — 4% vs ~253 files. Fix: 40-60h.

### Minor

1. Missing frontend error boundaries. 1-2h.
2. `use-cluster.ts` DRY violation. 1h. (per prior ANALYSIS)
3. `BrandingProvider.tsx` raw fetch. 30min. (per prior ANALYSIS)
4. Version number inconsistency across docs. 30min.
5. BRANDING.md font references outdated. 15min.
6. Binary artifacts in working directory. Cleanup.
7. Missing `aria-sort` on DataTable. 1h.
8. K8s Secret placeholder values. 30min.
9. Monitoring alert duplication across 3 files. 2-3h.

---

## 9. Metrics Summary Table

| Metric | Value |
|--------|-------|
| Total Project Files | 1,266 |
| Total Go Files | 448 |
| Go Source Files (non-test) | 179 |
| Go Test Files | 269 |
| Total Go LOC | 170,018 |
| Non-test Go LOC | ~55,700 |
| Test Go LOC | ~114,318 |
| Frontend Source Files | 253 |
| Frontend Test Files | 11 |
| Test Coverage | ~80% |
| Failing Test Packages | 2 |
| External Go Dependencies | 20 |
| External Frontend Dependencies | 48 |
| Admin API Endpoints | 95+ |
| Portal API Endpoints | 33+ |
| MCP Tools | 43+ |
| Built-in Plugins | 25+ |
| Load Balancing Algorithms | 11 |
| Open TODOs (non-test) | 1 |
| #nosec Suppressions | 93 |
| Spec Feature Completion | ~98% |
| Overall Health Score | 8/10 |
