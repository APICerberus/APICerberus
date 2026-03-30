# Changelog

All notable changes to API Cerberus will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v0.1.0] - 2026-03-31

### Added
- MCP (Model Context Protocol) Server with JSON-RPC 2.0 support
  - stdio and SSE transports
  - 25+ tools for gateway, user, credit, audit, and analytics management
  - `initialize`, `tools/list`, `tools/call`, `resources/list`, `resources/read` methods
- TLS Manager with ACME support
  - Automatic certificate provisioning via Let's Encrypt (tls-alpn-01)
  - Manual certificate loading from files
  - Automatic certificate renewal (< 30 days)
  - Disk persistence for certificates
  - SNI-based virtual hosting
- CLI Completion commands
  - `user list/create/get/update/suspend/activate`
  - `user apikey list/create/revoke`
  - `user permission list/grant/revoke`
  - `user ip list/add/remove`
  - `credit overview/balance/topup/deduct/transactions`
  - `audit search/tail/detail/export/stats/cleanup/retention`
  - `analytics overview/requests/latency`
  - `service/route/upstream list/add/get/update/delete`
  - `config export/import/diff`
  - `mcp start [--transport stdio|sse]`
- Config Export/Import endpoints
  - `GET /admin/api/v1/config/export` - Export current config as YAML
  - `POST /admin/api/v1/config/import` - Import and apply YAML config
- CLI table formatter with aligned columns and truncation
- CLI JSON output mode (`--output json`)

## [v0.0.9] - 2026-03-28

### Added
- React Flow topology visualization
  - Pipeline view - plugin execution order for routes
  - Upstream health map - traffic volume and health status
  - Service dependency graph - auto-layout with dagre
  - Cluster topology placeholder for v0.5.0
- WebSocket real-time feed
  - Server-side: broadcast metrics every 1s
  - Server-side: broadcast health changes immediately
  - Client-side: `use-realtime.ts` hook with Zustand store
  - Live request tail in Dashboard
  - Real-time chart updates
- Alert Rules Engine
  - Rule types: error_rate > X%, p99_latency > Xms, upstream_health < X%
  - Actions: log, webhook (HTTP POST)
  - Cooldown period support
  - Alert history storage
  - Admin API: `GET/POST/PUT/DELETE /admin/api/v1/alerts`
  - Admin UI: alert configuration page

## [v0.0.8] - 2026-03-27

### Added
- User Portal (React frontend)
  - Portal authentication with session-based auth
  - Portal dashboard with KPI cards and mini usage chart
  - API Keys management page
  - APIs catalog page
  - API Playground with full request builder
  - Usage page with charts
  - Logs page with search/filter
  - Credits page with balance and transactions
  - Security page with IP whitelist
  - Settings page with profile management
- Portal Backend
  - Session management with cookies
  - `POST /portal/api/v1/auth/login/logout`
  - `GET /portal/api/v1/auth/me`
  - All portal API endpoints (API keys, APIs, playground, usage, logs, credits, security, settings)
  - `POST /portal/api/v1/playground/send` - proxy test requests
  - Playground templates CRUD
- Portal embed and serve from Go binary

## [v0.0.7] - 2026-03-25

### Added
- Web Dashboard (React + Vite + TypeScript)
  - 35+ shadcn/ui components
  - Tailwind CSS v4.1
  - TanStack Query + Table
  - Zustand stores (auth, theme, realtime)
  - Recharts charts (area, bar, line, pie, heatmap)
  - CodeMirror 6 (YAML/JSON editors)
  - Geist font family
- Admin Layout
  - Collapsible sidebar
  - Top header with search and theme toggle
  - Mobile responsive (Sheet menu)
- Admin Pages
  - Dashboard with KPI cards and traffic chart
  - Services management
  - Routes management
  - Upstreams management
  - Consumers management
  - Users management with detail view
  - Credits overview
  - Audit logs with filters
  - Analytics with time-series
  - Plugins configuration
  - Config editor with validation
  - Settings
- WebSocket real-time updates
- Dashboard embedded in Go binary via `//go:embed`

## [v0.0.6] - 2026-03-22

### Added
- Audit Logging
  - Buffered channel logger (10K capacity)
  - `AuditEntry` struct with full request/response capture
  - Non-blocking log buffer
  - Batch insert every 1s or 100 items
- Sensitive Data Masking
  - Header masking (`***REDACTED***`)
  - JSON body field masking
  - Nested field support
- Audit Repository
  - `BatchInsert` with prepared statements
  - `Search` with dynamic filters (user_id, route, method, status, date range, etc.)
  - `Stats` for aggregations
  - `Export` to CSV/JSON/JSONL
- Log Retention & Cleanup
  - Per-route retention overrides
  - Batch deletion with configurable size
- Log Archival
  - Gzip compression for archives
  - Date-based filenames
- Analytics Engine
  - Ring buffer (100K metrics)
  - Time-series store with per-minute buckets
  - Real-time atomic counters
  - Latency percentiles (p50/p95/p99)
- Analytics API Endpoints
  - `/admin/api/v1/analytics/overview`
  - `/admin/api/v1/analytics/timeseries`
  - `/admin/api/v1/analytics/top-routes`
  - `/admin/api/v1/analytics/top-consumers`
  - `/admin/api/v1/analytics/errors`
  - `/admin/api/v1/analytics/latency`
  - `/admin/api/v1/analytics/throughput`
  - `/admin/api/v1/analytics/status-codes`
- Audit Log API Endpoints
  - `/admin/api/v1/audit-logs` with full search
  - `/admin/api/v1/audit-logs/{id}` detail
  - `/admin/api/v1/audit-logs/export`
  - `/admin/api/v1/audit-logs/stats`
  - `/admin/api/v1/audit-logs/cleanup`

## [v0.0.5] - 2026-03-20

### Added
- Embedded SQLite database
  - SQLite amalgamation bundled
  - WAL mode, busy timeout, foreign keys
  - Schema migrations
- User Management
  - User entity (ID, Email, Name, Company, PasswordHash, Role, Status, CreditBalance)
  - Password hashing (SHA-256 + 16-byte salt)
  - Initial admin user creation on first boot
- API Key Management
  - Key format: `ck_live_` (production) / `ck_test_` (test)
  - Key hashing with SHA-256
  - Key prefix display (first 12 chars)
  - SQLite-backed key repository
- Credit System
  - `CreditTransaction` entity
  - Atomic balance updates
  - Credit engine with route costs and method multipliers
  - Test key bypass (`ck_test_*`)
  - 402 Payment Required on zero balance
- Endpoint Permissions
  - `EndpointPermission` entity
  - Method restrictions
  - Time/day restrictions
  - Per-endpoint rate limit overrides
  - Per-endpoint credit cost overrides
- User IP Whitelist
  - CIDR range support
  - User-level enforcement
- Admin API Extensions
  - User CRUD + suspend/activate/reset-password
  - API key management per user
  - Permission management per user
  - IP whitelist management per user
  - Credit operations (topup, deduct, balance, transactions)
  - Billing configuration endpoints

## [v0.0.4] - 2026-03-15

### Added
- Request Transform Plugin
  - Header manipulation (add, remove, rename)
  - Query parameter manipulation
  - Path rewriting with regex
- Response Transform Plugin
  - Response header manipulation
  - Response body interception and transformation
- Body Template Engine
  - Variables: `$body`, `$timestamp_ms`, `$timestamp_iso`, `$upstream_latency_ms`, `$consumer_id`, `$route_name`, `$request_id`, `$remote_addr`, `$header.*`
  - JSON field operations (add, remove, rename)
  - JSON path traversal for nested fields
- URL Rewrite Plugin
  - Regex-based path rewriting with capture groups
  - Query string preservation
- Plugin Pipeline Architecture
  - Global + per-route plugin chains
  - Phase-ordered execution with abort support
  - Post-proxy phase support
  - `RequestContext` with all fields
  - `ctx.Aborted` flag with reason
  - Plugin config merging
- Request Size Limit Plugin (413 on exceed)
- Request Validator Plugin (JSON Schema validation)
- Compression Plugin (gzip with threshold)
- Correlation ID Plugin (UUID generation)
- Bot Detection Plugin (User-Agent patterns)
- Redirect Plugin (301/302/307/308)

## [v0.0.3] - 2026-03-12

### Added
- Additional Load Balancers
  - LeastConn - active connection tracking
  - IPHash - FNV hash of client IP
  - Random - math/rand/v2 selection
  - ConsistentHash - virtual node ring (CRC32)
  - LeastLatency - EWMA latency tracking
  - Adaptive - auto-switch algorithm
  - GeoAware - placeholder
  - HealthWeighted - health_pct × weight
- Passive Health Checking
  - Error tracking in proxy
  - Error window with sliding duration
  - Success recovery
- Circuit Breaker Plugin
  - States: Closed, Open, HalfOpen
  - Error rate tracking
  - Configurable thresholds and sleep window
  - Half-open trial requests
  - Returns 503 when open
- Retry Plugin
  - Retry on 502/503/504
  - Exponential backoff with jitter
  - Idempotency check (safe methods)
- Timeout Plugin
  - Per-route timeout with context
  - Returns 504 on deadline exceeded
- Sliding Window Rate Limiter
- Leaky Bucket Rate Limiter
- Rate Limit algorithm selection (4 algorithms)

## [v0.0.2] - 2026-03-08

### Added
- Consumer Entity with config struct
- API Key Authentication Plugin
  - Key extraction: header, query param, cookie
  - Configurable key header names
  - Constant-time key comparison
  - Key expiration check
- JWT Authentication
  - `internal/pkg/jwt` package
  - HS256 and RS256 verification
  - JWKS fetching with TTL caching (1h)
  - Claim validation (exp, iss, aud)
  - Clock skew tolerance
  - Claims-to-headers mapping
- Token Bucket Rate Limiter
- Fixed Window Rate Limiter
- Rate Limit Plugin
  - Scope: global, consumer, IP, route, composite
  - Response headers: X-RateLimit-*
  - Returns 429 with Retry-After
- IP Restrict Plugin
  - Whitelist and blacklist modes
  - CIDR range matching
- CORS Plugin
  - Preflight handling
  - Origin validation (exact and wildcard)
  - Configurable methods, headers, max_age
- Plugin Registry and Pipeline Integration

## [v0.0.1] - 2026-03-01

### Added
- Project scaffolding (Go module, structure)
- Custom YAML Parser
  - Tokenizer with indentation tracking
  - Node types: Map, Sequence, Scalar
  - Comment stripping
  - Quoted strings with escapes
  - Multi-line strings (literal |, folded >)
  - Unmarshal/Marshal with reflection
  - Struct tag support
  - Type coercion
- Configuration System
  - `Config` struct with all sections
  - `config.Load(path)` with validation
  - Environment variable overrides (`APICERBERUS_*`)
  - File watching with SIGHUP reload
- UUID Generator (v4)
- JSON Helpers
- Structured Logging (slog with rotation)
- Radix Tree Router
  - Wildcard (*) and param (:id) support
  - Per-method trees
  - Host-based routing
  - Route priority (exact > prefix > regex)
  - Path stripping
- Reverse Proxy
  - HTTP/HTTPS with connection pooling
  - WebSocket proxy with hijacking
  - Header filtering (hop-by-hop)
  - Response streaming with buffer pool
- Load Balancing (Round Robin, Weighted)
- Active Health Checking
  - HTTP GET health checks
  - Consecutive success/failure thresholds
  - Integration with balancer
- Gateway Server
  - Graceful shutdown
  - Hot reload via RWMutex
  - Custom error responses
- Admin REST API
  - Services CRUD
  - Routes CRUD
  - Upstreams CRUD with targets
  - Health status endpoint
  - Config reload endpoint
- CLI
  - `start`, `stop`, `version`, `config validate`
  - Graceful shutdown with signal handling

[v0.1.0]: https://github.com/APICerberus/APICerebrus/releases/tag/v0.1.0
[v0.0.9]: https://github.com/APICerberus/APICerebrus/releases/tag/v0.0.9
[v0.0.8]: https://github.com/APICerberus/APICerebrus/releases/tag/v0.0.8
[v0.0.7]: https://github.com/APICerberus/APICerebrus/releases/tag/v0.0.7
[v0.0.6]: https://github.com/APICerberus/APICerebrus/releases/tag/v0.0.6
[v0.0.5]: https://github.com/APICerberus/APICerebrus/releases/tag/v0.0.5
[v0.0.4]: https://github.com/APICerberus/APICerebrus/releases/tag/v0.0.4
[v0.0.3]: https://github.com/APICerberus/APICerebrus/releases/tag/v0.0.3
[v0.0.2]: https://github.com/APICerberus/APICerebrus/releases/tag/v0.0.2
[v0.0.1]: https://github.com/APICerberus/APICerebrus/releases/tag/v0.0.1
