# APICerebrus Security Architecture Report

**Phase 1: Recon - Architecture Map**
**Date:** 2026-04-16
**Project:** APICerebrus - Production API Gateway
**Classification:** INTERNAL

---

## 1. Tech Stack Detection

### 1.1 Backend (Go 1.26.2)

**Core Dependencies:**
| Library | Version | Purpose | Risk Profile |
|---------|---------|---------|--------------|
| `modernc.org/sqlite` | v1.48.0 | SQLite database (pure Go, no CGO) | Low - WAL mode, BoltDB-backed |
| `github.com/redis/go-redis/v9` | v9.7.3 | Distributed rate limiting | Medium - network access |
| `google.golang.org/grpc` | v1.80.0 | gRPC server, HTTP transcoding | Low - protobuf, h2c |
| `google.golang.org/protobuf` | v1.36.11 | Protocol buffers | Low |
| `github.com/golang-jwt/jwt/v5` | v5.3.1 | JWT parsing/validation | Medium - crypto dependency |
| `github.com/tetratelabs/wazero` | v1.11.0 | WASM runtime (sandboxed) | Medium - sandbox escape risk |
| `go.opentelemetry.io/otel/*` | v1.43.0 | Distributed tracing | Low - metadata only |
| `golang.org/x/crypto` | v0.49.0 | Cryptographic operations | Low - stdlib complement |
| `golang.org/x/oauth2` | v0.36.0 | OAuth2/OIDC integration | Medium - external calls |
| `github.com/coreos/go-oidc/v3` | v3.18.0 | OIDC provider | Medium - external calls |
| `gopkg.in/yaml.v3` | v3.0.1 | Config parsing | Low |
| `github.com/coder/websocket` | v1.8.14 | WebSocket support | Low |
| `github.com/andybalholm/brotli` | v1.2.1 | Brotli compression | Low |

**Indirect Dependencies (Notable):**
- `github.com/yuin/gopher-lua` - Lua scripting (potential sandbox)
- `github.com/graphql-go/graphql` - GraphQL execution
- `github.com/jackc/pgx/v5` - PostgreSQL driver (future use)

### 1.2 Frontend (Web Dashboard)

**Runtime:** React 19.2.4 + TypeScript 5.9.3
**Build:** Vite 8.0.1
**Styling:** Tailwind CSS 4.2.2 + shadcn/ui + Radix UI
**State:** Zustand 5.0.12, TanStack Query 5.95.2
**Charts:** Recharts 3.8.1

**Key Frontend Dependencies:**
| Library | Version | Purpose |
|---------|---------|---------|
| `react-router-dom` | v7.13.2 | Routing |
| `@tanstack/react-query` | v5.95.2 | Server state |
| `zustand` | v5.0.12 | Client state |
| `react-hook-form` | v7.72.0 | Form handling |

**Dev Dependencies:**
- Playwright 1.59.1 (E2E testing)
- Vitest 3.0.0 (Unit testing)
- MSW 2.7.0 (API mocking)

### 1.3 Infrastructure

- **Database:** SQLite (WAL mode) / PostgreSQL (future)
- **Cache/RateLimit:** Redis 9.x
- **Message Queue:** Kafka (optional, audit export)
- **Tracing:** OpenTelemetry (Jaeger, Zipkin, OTLP, stdout)
- **Certificates:** ACME/Let's Encrypt, mTLS for Raft

---

## 2. Architecture Overview

### 2.1 Core Components

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              APICerebrus                                     │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌──────────────────┐    ┌──────────────────┐    ┌──────────────────┐       │
│  │  Gateway (8080)  │    │  Admin API (9876) │    │  Portal (9877)    │       │
│  │  ───────────────  │    │  ───────────────  │    │  ───────────────  │       │
│  │  • Radix Router  │    │  • REST API       │    │  • User-facing    │       │
│  │  • Plugin Pipeline│    │  • OIDC Provider  │    │  • Sessions      │       │
│  │  • Load Balancer  │    │  • Webhooks       │    │  • API Keys       │       │
│  │  • Proxy Engine  │    │  • GraphQL Fed.   │    │                  │       │
│  └────────┬─────────┘    └────────┬─────────┘    └──────────────────┘       │
│           │                       │                                           │
│  ┌────────┴─────────┐    ┌────────┴─────────┐                               │
│  │ Plugin Pipeline  │    │    Store Layer  │                               │
│  │ ────────────────  │    │  ───────────────  │                               │
│  │ PRE_AUTH → AUTH   │    │  • SQLite (WAL)  │                               │
│  │ → PRE_PROXY →     │    │  • Repositories  │                               │
│  │ PROXY → POST_PROXY│    │  • Migrations    │                               │
│  └──────────────────┘    └──────────────────┘                               │
│                                                                              │
│  ┌──────────────────────────────────────────────────────────────────┐        │
│  │                     Supporting Systems                             │        │
│  │  ┌────────────┐  ┌────────────┐  ┌────────────┐  ┌───────────┐ │        │
│  │  │  Billing   │  │   Audit    │  │  Analytics │  │   Raft    │ │        │
│  │  │  Engine    │  │   Logger   │  │   Engine   │  │  Cluster  │ │        │
│  │  └────────────┘  └────────────┘  └────────────┘  └───────────┘ │        │
│  │  ┌────────────┐  ┌────────────┐  ┌────────────┐  ┌───────────┐ │        │
│  │  │    MCP     │  │   GraphQL  │  │   gRPC     │  │  Open     │ │        │
│  │  │   Server   │  │ Federation │  │  Server    │  │Telemetry  │ │        │
│  │  └────────────┘  └────────────┘  └────────────┘  └───────────┘ │        │
│  └──────────────────────────────────────────────────────────────────┘        │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 2.2 Entry Points

| Port | Service | Protocol | Auth Required |
|------|---------|----------|---------------|
| 8080 | Gateway HTTP | HTTP/1.1, HTTP/2 | Per-route (API key, JWT) |
| 8443 | Gateway HTTPS | TLS | Per-route (API key, JWT) |
| 9876 | Admin API | REST, WebSocket | X-Admin-Key header |
| 9877 | User Portal | HTTP | Session-based |
| 50051 | gRPC | HTTP/2 | Per-method |
| 12000 | Raft | Custom RPC | mTLS (optional) |
| 4317/4318 | OTLP | gRPC/HTTP | No (internal) |

### 2.3 Data Flows

#### Request Flow (Gateway)
```
Client
  │
  ▼
┌─────────────────────┐
│ Security Headers     │ ← X-Content-Type-Options, X-Frame-Options, CSP
│ MaxBodyBytes Check  │
└─────────────────────┘
  │
  ▼
┌─────────────────────┐
│ Health Endpoints     │ ← /health, /ready, /metrics (bypass routing)
└─────────────────────┘
  │
  ▼
┌─────────────────────┐
│ Radix Tree Router    │ ← O(k) path matching, method-based trees
│ Route Match         │
└─────────────────────┘
  │
  ▼
┌─────────────────────┐
│ Plugin Pipeline      │ ← PRE_AUTH → AUTH → PRE_PROXY
│ (per-route chain)   │
└─────────────────────┘
  │
  ├──[Auth Check]─────┼── No Auth ──► Billing Pre-Check ──► Proxy ──► Response
  │                              │                           │
  │                         Auth Failed               ┌──────┴──────┐
  │                              │                      │             │
  │                         401 Error            POST_PROXY    Analytics
  │                                              (transform)     Record
  │
  ▼
┌─────────────────────┐
│ Billing Pre-Check   │ ← Deduct credits before proxy
│ (ck_live_ keys)     │   ck_test_ keys bypass
└─────────────────────┘
  │
  ▼
┌─────────────────────┐
│ Upstream Selection   │ ← 11 load balancing algorithms
│ Target Health Check  │   Health-weighted, adaptive
└─────────────────────┘
  │
  ▼
┌─────────────────────┐
│ Proxy Engine        │ ← Connection pooling, retry, circuit breaker
│                     │   Timeout, caching
└─────────────────────┘
  │
  ▼
┌─────────────────────┐
│ Response Capture     │ ← Audit logging, analytics
│ Body + Headers      │
└─────────────────────┘
  │
  ▼
Client
```

#### Admin API Flow
```
External Client
  │
  ▼
┌─────────────────────┐
│ X-Admin-Key Header  │ ← Static API key validation
│ OR Bearer Token     │ ← JWT session token (after login)
└─────────────────────┘
  │
  ▼
┌─────────────────────┐
│ Rate Limit Check    │ ← Per-IP failed auth tracking
│ Auth Backoff        │   Exponential backoff on failures
└─────────────────────┘
  │
  ▼
┌─────────────────────┐
│ RBAC Middleware     │ ← Role-based access control
│ (future)            │
└─────────────────────┘
  │
  ▼
┌─────────────────────┐
│ REST Handlers       │ ← CRUD for routes, services, users
│ GraphQL Federation  │   Subgraph management
│ Webhook Management  │
└─────────────────────┘
  │
  ▼
┌─────────────────────┐
│ Store Layer         │ ← SQLite WAL writes
│ Audit Logging       │
└─────────────────────┘
```

---

## 3. Attack Surface Analysis

### 3.1 Public Endpoints (Gateway)

| Endpoint | Purpose | Auth | Risk |
|----------|---------|------|------|
| `/*` | Proxied routes | Per-route | High - user traffic |
| `/health` | Health check | None | Low - read-only |
| `/ready` | Readiness probe | None | Medium - DB ping |
| `/metrics` | Prometheus metrics | None | Medium - exposes stats |
| `/graphql` | Federation endpoint | Per-route | High - complex parsing |
| `/graphql/batch` | Batch queries | Per-route | High - resource exhaustion |

**Built-in Endpoints (bypass plugin pipeline):**
- `/health` - Always accessible, cannot be rate-limited by standard plugins
- `/ready` - Database connectivity exposed
- `/metrics` - Full metrics exposure
- `/health/audit-drops` - Audit buffer drop counter

### 3.2 Admin API Endpoints (Port 9876)

**Authentication Endpoints:**
| Endpoint | Method | Auth | Risk |
|----------|--------|------|------|
| `/admin/api/v1/auth/token` | POST | X-Admin-Key | Medium - token exchange |
| `/admin/api/v1/auth/logout` | POST | Bearer | Low |
| `/admin/login` | POST | Form | Medium - password login |
| `/oidc/*` | GET/POST | None | High - OIDC flows |

**OIDC Provider Endpoints:**
| Endpoint | Purpose |
|----------|---------|
| `/.well-known/openid-configuration` | OIDC discovery |
| `/oidc/jwks` | JSON Web Key Set |
| `/oidc/authorize` | Authorization endpoint |
| `/oidc/token` | Token endpoint |
| `/oidc/userinfo` | User info endpoint |
| `/oidc/revoke` | Token revocation |
| `/oidc/introspect` | Token introspection |

**Management Endpoints (Bearer auth required):**
- Routes/Services/Upstreams CRUD
- User management + API keys
- Credit operations
- Audit log search/export
- Analytics queries
- Webhook management
- Subgraph/Federation config
- Config import/export

### 3.3 Plugin System Attack Surface

**5-Phase Pipeline:**
```
PRE_AUTH (5 plugins)
├── correlation_id    - Header injection
├── ip_restrict      - IP-based access control
└── bot_detect       - Bot detection

AUTH (3 plugins)
├── auth_apikey      - API key validation ← CRITICAL
├── auth_jwt         - JWT validation
└── endpoint_permission - Route-level ACLs

PRE_PROXY (10+ plugins)
├── rate_limit       - DoS protection
├── request_validator - Input validation
├── request_transform - Header/path manipulation
├── url_rewrite      - Path rewriting
├── cors             - Cross-origin control
├── user_ip_whitelist - Per-user IP allowlist
├── graphql_guard    - GraphQL query depth/complexity
├── request_size_limit - Body size limits
├── caching          - Response caching
└── redirect         - HTTP redirects

PROXY (4 plugins)
├── circuit_breaker  - Fault isolation
├── retry            - Automatic retry
├── timeout          - Upstream timeout
└── [WASM modules]   - Custom logic ← SANDBOX RISK

POST_PROXY (3 plugins)
├── response_transform - Response manipulation
├── compression       - Brotli/gzip
└── [WASM modules]
```

### 3.4 WASM Plugin Sandbox

**Attack Vectors:**
1. **Path Traversal** - Module files outside `module_dir`
2. **Memory Exhaustion** - Large `MaxMemory` limits
3. **CPU Exhaustion** - Long `MaxExecution` timeouts
4. **Syscall Access** - WASI filesystem access (when enabled)
5. **Host Memory Read** - Malicious memory read via pointer arithmetic

**Security Controls:**
- `AllowFilesystem: false` (default) - WASI unavailable
- `MaxMemory: 128MB` default
- `MaxExecution: 30s` default
- `maxWASMModuleSize: 100MB` hard cap
- Magic header validation (`\x00asm`)
- Path traversal prevention via `filepath.Rel`
- 64MB `maxWASMReadSize` hard cap on memory reads

**Code References:**
- `internal/plugin/wasm.go` lines 22-25 (constants)
- `internal/plugin/wasm.go` lines 137-167 (validation)
- `internal/plugin/wasm.go` lines 169-190 (path safety)
- `internal/plugin/wasm.go` lines 399-404 (memory read cap)

### 3.5 Raft Clustering Attack Surface

**Network Exposure:**
- Port 12000 (configurable) - Inter-node RPC
- No encryption by default (mTLS optional)
- No authentication by default

**Attack Vectors:**
1. **Leader Election Manipulation** - Fake heartbeats
2. **Log Injection** - Malicious Raft entries
3. **Split Brain** - Network partitioning
4. **Certificate Spoofing** - Fake node identity (if mTLS disabled)

**Security Controls:**
- `cluster.mtls.enabled: true` (default: false)
- `cluster.mtls.auto_generate: true` (default: true)
- Optional CA/node cert import

**Certificate Manager (`internal/raft/tls.go`):**
- RSA 4096-bit keys
- 1-year cert validity
- CA + node cert hierarchy
- `tls.VersionTLS13` minimum

### 3.6 GraphQL Federation Attack Surface

**Endpoints:**
- `/graphql` - Single query endpoint
- `/graphql/batch` - Batch queries (max 100 per batch)

**Attack Vectors:**
1. **Query Complexity** - Deep nested queries
2. **Alias Abuse** - Multiple aliases for same field
3. **Introspection** - Schema disclosure
4. **Batch Exhaustion** - 100 query limit (M-012)
5. **Subgraph Injection** - Malicious subgraph URLs

**Protections:**
- `graphql_guard` plugin - depth/complexity limits
- `maxBatchSize: 100` constant
- Subgraph URL validation (user-provided)
- Query planning with entity resolution

### 3.7 MCP Server Attack Surface

**Transports:**
| Transport | Auth | Risk |
|-----------|------|------|
| stdio | None (local) | Low - subprocess only |
| SSE (HTTP) | X-Admin-Key header | Medium - network exposed |

**SSE Endpoints:**
- `POST /mcp` - JSON-RPC requests
- `GET /sse` - Server-Sent Events stream

**Tools (25+):**
- Gateway inspection (routes, services, upstreams)
- User/credit management
- Audit log access
- Config read/modify
- Analytics queries

**Security:**
- SSE requires `X-Admin-Key` matching admin key
- Constant-time comparison for key validation
- Token obtained via admin token exchange

### 3.8 Client IP Extraction Attack Surface

**Spoofing Vectors:**
1. `X-Forwarded-For` header injection
2. `X-Real-IP` header spoofing
3. Direct connection with forged headers

**Trust Model:**
- `trusted_proxies: []` (default) = Secure, forwarding headers ignored
- `trusted_proxies: ["10.0.0.0/8"]` = XFF parsed right-to-left, untrusted IPs extracted

**Security Controls:**
- Default: No trust for forwarding headers
- Right-to-left XFF parsing skips trusted proxies
- `X-Real-IP` validation (M-003): Must be valid IP format
- RemoteAddr always used as fallback

---

## 4. Trust Boundaries

### 4.1 Component Trust Map

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           TRUST BOUNDARIES                                   │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐     │
│  │                    HIGH-TRUST ZONE (Internal)                        │     │
│  │                                                                      │     │
│  │   ┌──────────────┐    ┌──────────────┐    ┌──────────────┐          │     │
│  │   │   Gateway    │◄───│  Admin API   │◄───│  MCP Server │          │     │
│  │   │   Process    │    │   Process    │    │   Process    │          │     │
│  │   └──────┬───────┘    └──────┬───────┘    └──────────────┘          │     │
│  │          │                   │                                      │     │
│  │          │         ┌─────────┴─────────┐                            │     │
│  │          │         │                   │                            │     │
│  │          ▼         ▼                   ▼                            │     │
│  │   ┌──────────────┐ ┌──────────────┐ ┌──────────────┐              │     │
│  │   │    Store     │ │   Billing    │ │    Audit     │              │     │
│  │   │   (SQLite)   │ │   Engine    │ │   Logger     │              │     │
│  │   └──────────────┘ └──────────────┘ └──────────────┘              │     │
│  │                                                                      │     │
│  │   ADMIN API KEY ────────────────────────────────────────────────────│     │
│  │   Portal Session Secret ─────────────────────────────────────────────│     │
│  │   JWT Token Secret ──────────────────────────────────────────────────│     │
│  │                                                                      │     │
│  └─────────────────────────────────────────────────────────────────────┘     │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐     │
│  │                   LOW-TRUST ZONE (External)                         │     │
│  │                                                                      │     │
│  │   ┌──────────────┐    ┌──────────────┐    ┌──────────────┐          │     │
│  │   │    Client    │    │  Upstream    │    │    Redis     │          │     │
│  │   │   Requests   │    │   Servers   │    │   Server     │          │     │
│  │   └──────────────┘    └──────────────┘    └──────────────┘          │     │
│  │                                                                      │     │
│  │   ┌──────────────┐    ┌──────────────┐    ┌──────────────┐          │     │
│  │   │   Kafka      │    │  OTLP       │    │   ACME/LE    │          │     │
│  │   │  (Optional)  │    │  Exporters   │    │   Servers    │          │     │
│  │   └──────────────┘    └──────────────┘    └──────────────┘          │     │
│  │                                                                      │     │
│  └─────────────────────────────────────────────────────────────────────┘     │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 4.2 Admin API Trust

**Trust Level:** VERY HIGH
**Boundary:** X-Admin-Key header or Bearer token

**What's trusted:**
- Full config read/write (secrets visible)
- User management (passwords, API keys)
- Credit manipulation
- Audit log access
- All gateway operations

**What's NOT trusted:**
- Direct SQL (parameterized only)
- File system (except config import temp files)
- Process execution

### 4.3 Gateway Trust

**Trust Level:** MEDIUM-HIGH
**Boundary:** Per-route authentication

**What's trusted:**
- Authenticated consumer identity
- Route configuration
- Plugin chain execution

**What's NOT trusted:**
- Client-provided headers (X-Forwarded-For, etc.)
- Request body (validated by plugins)
- Query parameters

### 4.4 WASM Plugin Trust

**Trust Level:** LOW (sandboxed)
**Boundary:** wazero runtime sandbox

**What's trusted:**
- Request context serialization (read-only view)
- Config (read-only)
- Memory within allocation limits

**What's NOT trusted:**
- Host filesystem (when AllowFilesystem=false)
- Host network
- Host process memory
- System calls beyond WASI subset

### 4.5 Store Layer Trust

**Trust Level:** HIGHEST
**Boundary:** SQLite WAL / PostgreSQL

**What's protected:**
- User credentials (password_hash)
- API key hashes (not raw keys)
- Credit balances (atomic transactions)
- Audit logs (tamper-evident via retention)

**Schema Tables:**
- `users` - Password hashes, roles
- `api_keys` - Key hashes (SHA-256), not raw keys
- `credit_transactions` - Immutable ledger
- `sessions` - Token hashes
- `audit_logs` - Tamper-evident

---

## 5. Sensitive Data Flows

### 5.1 API Key Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         API KEY LIFECYCLE                                    │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  1. KEY GENERATION                                                           │
│  ┌──────────────┐                                                            │
│  │ Admin CLI    │ ← apicerberus user apikey create                          │
│  │ (offline)   │    Generates: ck_live_xxx or ck_test_xxx                   │
│  └──────┬───────┘                                                            │
│         │ Raw key displayed ONCE to user                                     │
│         ▼                                                                    │
│  2. KEY STORAGE (Store Layer)                                                │
│  ┌──────────────┐                                                            │
│  │ api_keys     │ ← key_hash = SHA256(raw_key)  ★ HASH ONLY ★              │
│  │ table        │    key_prefix = "ck_live_" or "ck_test_"                 │
│  └──────┬───────┘    Raw key NEVER stored                                   │
│         │                                                                    │
│         ▼                                                                    │
│  3. KEY LOOKUP (Authentication)                                              │
│  ┌──────────────┐     ┌──────────────┐                                       │
│  │ auth_apikey  │ ──► │   Store     │                                       │
│  │ plugin       │     │  ResolveUser │                                       │
│  └──────┬───────┘     │  ByRawKey()  │                                       │
│         │             └──────────────┘                                       │
│         │             SHA256(provided_key) ──► Compare with key_hash         │
│         │                                       (constant-time)             │
│         ▼                                                                    │
│  4. CONSUMER IDENTITY                                                        │
│  ┌──────────────┐     ┌──────────────┐                                       │
│  │   Billing    │ ──► │   Credits   │                                       │
│  │   Pre-Check  │     │   Deducted  │                                       │
│  └──────────────┘     └──────────────┘                                       │
│                                                                              │
│  ★ KEY PREFIX CONVENTION ★                                                   │
│  ck_live_xxx = Production key (credit deducted)                              │
│  ck_test_xxx = Test key (bypasses credit check)                             │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 5.2 Admin Session Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                       ADMIN SESSION FLOW                                     │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  1. TOKEN ISSUE                                                             │
│  POST /admin/api/v1/auth/token                                              │
│  Header: X-Admin-Key: <admin_api_key>                                        │
│                                                                              │
│  ┌──────────────┐     ┌──────────────┐     ┌──────────────┐                  │
│  │   Validate   │ ──► │  Generate    │ ──► │   Store     │                  │
│  │  Admin Key   │     │  JWT Token   │     │  Session    │                  │
│  │  (constant   │     │  (24h TTL)   │     │  (SQLite)   │                  │
│  │   time cmp)  │     └──────────────┘     └──────────────┘                  │
│  └──────────────┘                                                            │
│                                                                              │
│  Response: Set-Cookie: apicerberus_admin_session=<token>                     │
│                      ★ HTTP-ONLY, SECURE FLAG ★                            │
│                                                                              │
│  2. SUBSEQUENT REQUESTS                                                     │
│  Cookie: apicerberus_admin_session=<token>                                  │
│  OR Authorization: Bearer <token>                                            │
│                                                                              │
│  ┌──────────────┐     ┌──────────────┐     ┌──────────────┐                  │
│  │   Validate   │ ──► │   Check      │ ──► │   RBAC      │                  │
│  │  JWT Token   │     │  Expiry      │     │  (future)   │                  │
│  │              │     │  24h max     │     │             │                  │
│  └──────────────┘     └──────────────┘     └──────────────┘                  │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 5.3 Credit/Billing Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                       BILLING FLOW                                           │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  1. PRE-PROXY CREDIT CHECK                                                   │
│  ┌──────────────┐     ┌──────────────┐     ┌──────────────┐                  │
│  │ Get Consumer │ ──► │  Load User  │ ──► │  Get Credit  │                  │
│  │ from Auth    │     │  from DB    │     │  Balance     │                  │
│  └──────────────┘     └──────────────┘     └──────────────┘                  │
│                                                       │                      │
│                          ┌────────────────────────────┘                      │
│                          │                                                   │
│                          ▼                                                   │
│  ┌──────────────┐     ┌──────────────┐     ┌──────────────┐                  │
│  │ Check Test  │ ──► │ Calc Route  │ ──► │  Sufficient?  │                  │
│  │ Key Flag    │     │ Cost        │     │  Balance >=   │                  │
│  │ ck_test_*   │     │ (route +    │     │  Cost?        │                  │
│  │ bypasses     │     │  method)    │     │               │                  │
│  └──────────────┘     └──────────────┘     └──────────────┘                  │
│                                                        │                      │
│                      ┌────────────────────────────────┘                      │
│                      │                                    │                   │
│                      ▼ No                                 ▼ Yes              │
│  ┌──────────────────────────┐              ┌──────────────────────────┐     │
│  │ 402 Payment Required      │              │  Continue to Proxy        │     │
│  │ "insufficient_credits"   │              │  Deducted atomically    │     │
│  └──────────────────────────┘              └──────────────────────────┘     │
│                                                                              │
│  2. CREDIT TRANSACTION (Atomic SQLite)                                       │
│  ┌──────────────────────────────────────────────────────────────────────┐   │
│  │ BEGIN TRANSACTION;                                                    │   │
│  │   INSERT INTO credit_transactions (...) VALUES (...);                │   │
│  │   UPDATE users SET credit_balance = credit_balance - cost             │   │
│  │   WHERE id = ? AND credit_balance >= cost;  -- atomic check          │   │
│  │ COMMIT;                                                               │   │
│  └──────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 5.4 Audit Log Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                       AUDIT LOG FLOW                                         │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  Request Complete                                                           │
│       │                                                                     │
│       ▼                                                                     │
│  ┌─────────────────┐                                                       │
│  │ Response Capture │ ← Status, headers, body (optional)                    │
│  │ Writer          │                                                        │
│  └────────┬────────┘                                                        │
│           │                                                                 │
│           ▼                                                                 │
│  ┌─────────────────┐     ┌─────────────────┐                                │
│  │ Field Masking   │ ──► │ MaskHeaders()   │ ← Authorization, X-API-Key    │
│  │                 │     │ MaskBody()      │ ← password, token fields      │
│  └────────┬────────┘     └─────────────────┘                                │
│           │                                                                 │
│           ▼                                                                 │
│  ┌─────────────────┐     ┌─────────────────┐                                │
│  │ Non-blocking    │ ──► │ entries channel │ ← 10k buffer                   │
│  │ Queue           │     │ (buffered)      │                                │
│  └────────┬────────┘     └─────────────────┘                                │
│           │                                                                 │
│           │     ┌─────────────────────────────────────────────────┐        │
│           │     │ Background Goroutine (batch flush)                │        │
│           │     │                                                  │        │
│           └────►│  Every 1s OR 100 entries:                        │        │
│                 │  ┌──────────────┐  ┌──────────────┐              │        │
│                 │  │ BatchInsert  │  │ KafkaWriter  │ (optional)    │        │
│                 │  │ (SQLite WAL) │  │ (async)      │              │        │
│                 │  └──────────────┘  └──────────────┘              │        │
│                 │                                                  │        │
│                 │  Retry: SQLITE_BUSY → exponential backoff 3x     │        │
│                 │  Drop: buffer full → l.dropped.Add(1)            │        │
│                 └─────────────────────────────────────────────────┘        │
│                                                                              │
│  ┌─────────────────┐     ┌─────────────────┐                                │
│  │ Retention       │ ──► │ Cleanup Every   │                                │
│  │ Scheduler       │     │ 1h: DELETE old │                                │
│  │                 │     │ entries         │                                │
│  └─────────────────┘     └─────────────────┘                                │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 6. External Integrations

### 6.1 Redis Integration

**Purpose:** Distributed rate limiting

**Configuration:**
```yaml
redis:
  enabled: false
  address: "localhost:6379"
  password: ""
  database: 0
  key_prefix: "ratelimit:"
  fallback_to_local: true   # Graceful degradation
```

**Security Considerations:**
- No TLS by default (LAN assumption)
- No authentication by default (password: "")
- Key prefix prevents collision
- Fallback to local on Redis failure

### 6.2 SQLite Integration

**Purpose:** Primary data store

**Configuration:**
```yaml
store:
  path: "apicerberus.db"
  journal_mode: "WAL"        # Concurrent reads, durability
  foreign_keys: true         # Referential integrity
  busy_timeout: "5s"
  synchronous: "NORMAL"      # Safe with WAL
  wal_autocheckpoint: 5000   # Performance tuning
```

**Security Considerations:**
- File permissions (0o600 on create)
- WAL mode = concurrent access
- No network exposure (local file)
- Foreign keys prevent data corruption

### 6.3 Kafka Integration (Audit Export)

**Purpose:** SIEM integration for audit logs

**Configuration:**
```yaml
kafka:
  enabled: false
  brokers: []
  topic: "apicerberus-audit"
  tls:
    enabled: false
    skip_verify: false  # MUST be false in production
  sasl:
    mechanism: ""
    username: ""
    password: ""
```

**Security Considerations:**
- TLS configurable but off by default
- `skip_verify: false` enforced in validation (CWE-295)
- SASL credentials in config (redacted on export)
- No Zookeeper dependency

### 6.4 OpenTelemetry Integration

**Purpose:** Distributed tracing

**Exporters:**
- `stdout` - Development logging
- `otlp` - Collector (gRPC or HTTP)
- `jaeger` - Direct Jaeger export
- `zipkin` - Direct Zipkin export

**Security Considerations:**
- OTLP headers may contain auth tokens
- Tokens redacted in config export
- No sensitive data in span attributes (by design)
- Sampling rate configurable (0.0-1.0)

### 6.5 ACME/Let's Encrypt Integration

**Purpose:** Automatic TLS certificate management

**Configuration:**
```yaml
gateway:
  tls:
    auto: true
    acme_email: "admin@example.com"
    acme_dir: "acme-certs"
```

**Security Considerations:**
- Email for certificate expiry notices
- Local certificate storage
- Automatic renewal before expiry
- HTTP-01 or TLS-ALPN-01 challenges

### 6.6 OIDC/External Identity Providers

**Purpose:** SSO integration

**Flow:**
1. User → `/admin/api/v1/auth/sso/login` → IdP
2. IdP → `/admin/api/v1/auth/sso/callback` → JWT session

**Security Considerations:**
- State parameter CSRF protection
- PKCE for public clients
- JWT validation with JWKS
- Token stored in HTTP-only cookie

---

## 7. Key Security Findings Summary

### 7.1 High-Risk Areas

| Component | Risk | Reason |
|-----------|------|--------|
| Admin API | HIGH | Full system control, static key only |
| API Key Auth | HIGH | Consumer identity, SHA256 hash comparison |
| WASM Sandbox | MEDIUM | Sandbox escape potential |
| Raft Clustering | MEDIUM | No auth by default, mTLS optional |
| GraphQL Federation | MEDIUM | Query complexity, batch limits |
| Client IP Extraction | MEDIUM | Header spoofing if trusted_proxies misconfigured |
| Config Import | MEDIUM | Temp file creation, YAML parsing |

### 7.2 Security Controls in Place

| Control | Location | Maturity |
|---------|----------|----------|
| API Key hashing (SHA256) | `internal/plugin/auth_apikey.go` | High |
| Constant-time key comparison | `internal/plugin/auth_apikey.go:186` | High |
| Auth backoff (DoS protection) | `internal/plugin/auth_backoff.go` | High |
| WASM sandbox (wazero) | `internal/plugin/wasm.go` | Medium |
| XFF right-to-left parsing | `internal/pkg/netutil/clientip.go:106` | High |
| X-Real-IP validation (M-003) | `internal/pkg/netutil/clientip.go:132` | High |
| Credit atomic transactions | `internal/billing/billing.go` | High |
| SQL parameterization | `internal/store/*.go` | High |
| Admin key placeholder check | `internal/config/load.go:319` | Medium |
| Kafka TLS skip_verify check | `internal/config/load.go:439` | High |
| Batch size limits (M-012) | `internal/gateway/server.go:1130` | Medium |
| Config secret redaction | `internal/admin/server.go:393` | High |
| Temp file permissions | `internal/admin/server.go:454` | High |

### 7.3 Areas Requiring Review

1. **Admin API Key Rotation** - No automatic rotation mechanism
2. **WASM Memory Limits** - Default 128MB may be excessive
3. **Raft mTLS** - Disabled by default
4. **Redis TLS** - No TLS support
5. **OIDC State CSRF** - Needs verification
6. **GraphQL Introspection** - Should be disabled in production
7. **Portal Session Lifetime** - 24h default, no refresh

---

## Appendix: File Reference Map

### Core Entry Points
- `cmd/apicerberus/main.go` - Application entrypoint
- `internal/cli/` - 40+ CLI commands

### Gateway
- `internal/gateway/server.go` - Main HTTP server, routing
- `internal/gateway/router.go` - Radix tree router
- `internal/gateway/proxy.go` - Proxy engine
- `internal/gateway/balancer.go` - Load balancing algorithms
- `internal/gateway/health.go` - Health checking

### Admin API
- `internal/admin/server.go` - REST API server
- `internal/admin/rbac.go` - RBAC middleware
- `internal/admin/token.go` - JWT session management
- `internal/admin/webhooks.go` - Webhook delivery

### Plugin System
- `internal/plugin/types.go` - Plugin interface, phases
- `internal/plugin/pipeline.go` - Pipeline execution
- `internal/plugin/registry.go` - Plugin registry
- `internal/plugin/auth_apikey.go` - API key auth
- `internal/plugin/wasm.go` - WASM sandbox

### Store Layer
- `internal/store/store.go` - Store initialization
- `internal/store/user_repo.go` - User repository
- `internal/store/api_key_repo.go` - API key repository
- `internal/store/credit_repo.go` - Credit transactions
- `internal/store/audit_repo.go` - Audit logging

### Billing
- `internal/billing/billing.go` - Billing engine

### Audit
- `internal/audit/logger.go` - Audit logger
- `internal/audit/masker.go` - PII masking

### Federation
- `internal/federation/` - GraphQL Federation

### Raft
- `internal/raft/node.go` - Raft node implementation
- `internal/raft/tls.go` - mTLS certificate management
- `internal/raft/transport.go` - RPC transport

### MCP
- `internal/mcp/server.go` - MCP server

### Config
- `internal/config/load.go` - Config loading/validation

### Networking
- `internal/pkg/netutil/clientip.go` - Client IP extraction

---

*Report generated for Phase 1: Recon - Architecture Map*
*Next: Phase 2 - Vulnerability Scan*
