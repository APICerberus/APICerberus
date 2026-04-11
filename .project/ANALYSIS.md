# Project Analysis

> Generated: 2026-04-11
> Based on: Full codebase read (all 233 Go source files, 399 test files, all documentation)

---

## 1. Executive Summary

APICerebrus is an API Gateway built in Go 1.26.2 with a React 19 admin dashboard. It implements HTTP/HTTPS reverse proxy, gRPC proxying with transcoding, GraphQL Federation, Raft clustering, a 5-phase plugin pipeline, credit-based billing, audit logging with Kafka streaming, OpenTelemetry tracing, and an MCP server for AI integration.

**Overall status**: Feature-complete at v1.0.0-rc.1. All 37 test packages pass. Overall statement coverage ~81-85% depending on which packages are included. The core proxy path (gateway → router → plugin pipeline → load balancer → upstream) is solid. The main remaining concerns are code complexity in large files, gaps in coverage for admin/gateway hot paths, and a few architectural decisions that could cause problems at scale.

**Strengths**:
- Pure Go SQLite — no CGO, single binary deployment
- Radix tree router with O(k) matching and per-method trees
- 11 load balancing algorithms including adaptive and subnet-aware
- Comprehensive plugin system (30+ files, WASM + Lua + 20+ built-in plugins)
- All test packages pass (37 packages, 0 failures)
- Good coverage distribution — most packages 80-95%

**Weaknesses**:
- `internal/gateway/server.go` at 1,638 lines — `ServeHTTP` alone is ~400 lines of sequential logic
- Admin package coverage at 75.1% — management API hot paths under-tested
- No rate limiting actually wired into the gateway proxy (rate limit plugin exists but isn't applied per-route in `ServeHTTP`)
- `internal/federation/executor.go` at 892 lines — subscription + circuit breaker + caching + batching all in one file
- Mixed quality in helper utilities (some packages duplicate the same type-coercion patterns 4-5 times)

---

## 2. Architecture

### 2.1 High-Level Structure

```
cmd/apicerberus/main.go          → entry point → cli.Run()
internal/cli/run.go              → CLI dispatch (start, stop, config, user, credit, audit, etc.)
internal/gateway/server.go       → main HTTP handler, the heart of the system
internal/admin/server.go         → admin REST API (70+ endpoints)
internal/portal/server.go        → user-facing portal (port 9877)
internal/plugin/registry.go      → plugin registration and pipeline construction
internal/store/store.go          → SQLite database with 6 migration versions
internal/raft/node.go            → Raft consensus node
internal/federation/composer.go  → GraphQL Federation schema composition
internal/mcp/server.go           → Model Context Protocol server
```

### 2.2 Request Flow (Critical Path)

The critical path through `internal/gateway/server.go:ServeHTTP` (lines 191-597):

1. **Security headers** — `addSecurityHeaders()` (line 1619)
2. **Health/metrics endpoints** — bypass routing
3. **Route matching** — `router.Match()` → radix tree lookup
4. **Plugin pipeline pre-auth** — correlation ID, bot detection
5. **Plugin pipeline auth** — API key, JWT authentication
6. **Plugin pipeline pre-proxy** — rate limiting, CORS, transforms
7. **Billing pre-check** — credit balance verification
8. **GraphQL federation routing** — if route is GraphQL
9. **Load balancer selection** — upstream pool
10. **Proxy forwarding** — `optimized_proxy.go` or `proxy.go`
11. **Billing post-proxy** — credit deduction with SQLITE_BUSY retry
12. **Audit/analytics recording** — async buffered write
13. **Response** — back to client

**Observation**: This is a lot of sequential logic in a single method. The plugin pipeline execution happens through `internal/plugin/pipeline.go` (56 lines, simple) and `internal/plugin/optimized_pipeline.go` (667 lines, with caching and parallel execution). The optimized pipeline is the more complex one but handles route-specific plugin selection well.

### 2.3 Key Architectural Decisions

| Decision | Impact | Verdict |
|----------|--------|---------|
| Pure Go SQLite (`modernc.org/sqlite`) | Single binary, no CGO, but single-writer bottleneck | Good for single-node, problematic for cluster |
| WAL mode + busy timeout 5s | Mitigates write contention but doesn't eliminate it | Acceptable for moderate throughput |
| Ring buffer analytics (100K entries) | Lock-free, bounded memory | Good |
| Async buffered audit logging (channel + batch flush) | Non-blocking, but entries dropped on full channel | Risky under load — need monitoring |
| Raft FSM for config distribution | Consistent across cluster, but config also in-memory in gateway | Dual source of truth — potential drift |
| Embedded React dashboard (`embed.FS`) | Single binary, no separate deployment | Good |
| Per-method radix trees | O(k) lookup, avoids method check in tree traversal | Good |
| Connection pooling with sync.Pool | Reduces GC pressure, reusable http.Client instances | Good |

### 2.4 Data Flow Diagrams

**Config hot reload**: `config/watch.go` (fsnotify + SIGHUP) → `config/dynamic_reload.go` (debounce, version history) → `gateway/server.go:Reload()` (lines 722-858, full subsystem rebuild) → atomic config swap via `mutateConfig()` in `admin/admin_helpers.go:282`

**Audit pipeline**: `gateway.ServeHTTP` captures request/response → `audit/logger.go:Log()` (non-blocking channel push) → `Start()` goroutine batches to SQLite → optional Kafka export via `audit/kafka.go`

**Rate limiting**: Plugin-based (`internal/plugin/rate_limit.go`, 546 lines) with Redis distributed fallback (`internal/ratelimit/redis.go`, 483 lines). However, the rate limit plugin is registered in the default registry but **is not automatically applied to routes** — it must be explicitly configured per-route in the plugin chain.

---

## 3. Code Quality

### 3.1 File Size Distribution

| File | Lines | Concern |
|------|-------|---------|
| `internal/gateway/server.go` | 1,638 | Too large — ServeHTTP is 400+ lines |
| `internal/plugin/optimized_pipeline.go` | 667 | Complex but well-structured |
| `internal/mcp/server.go` | 1,284 | 39 tool implementations in one file |
| `internal/federation/executor.go` | 892 | Too many responsibilities |
| `internal/plugin/registry.go` | 1,019 | Many build*Plugin factory functions |
| `internal/plugin/marketplace.go` | 741 | Large but cohesive |
| `internal/plugin/cache.go` | 993 | Caching is complex by nature |
| `internal/raft/node.go` | 1,021 | Raft implementation, expected size |
| `internal/admin/webhooks.go` | 771 | Reasonable for webhook system |
| `internal/analytics/webhook_templates.go` | 717 | Template engine, acceptable |
| `internal/admin/analytics.go` | 717 | Many analytics endpoints |
| `internal/admin/graphql.go` | 905 | GraphQL resolvers are verbose |
| `internal/admin/bulk.go` | 881 | Bulk operations, many types |
| `internal/admin/admin_users.go` | 790 | User CRUD with permissions |
| `internal/store/audit_repo.go` | 938 | Export, search, retention — too many responsibilities |
| `internal/store/user_repo.go` | 729 | User management, password hashing, validation |
| `internal/plugin/wasm.go` | 841 | WASM runtime with wazero — complex but expected |
| `internal/gateway/balancer_extra.go` | 843 | 8 balancer algorithms — each is ~100 lines |
| `internal/gateway/optimized_proxy.go` | 730 | Connection pooling, coalescing, buffering |
| `internal/gateway/router.go` | 665 | Radix tree with regex support |
| `internal/cli/cmd_user.go` | 745 | Many CLI subcommands |
| `internal/cli/cmd_audit.go` | 518 | Many CLI subcommands |

**Verdict**: 15 files exceed 700 lines. The top offenders (`server.go`, `mcp/server.go`, `federation/executor.go`) would benefit from extraction. However, the internal structure within these files is generally clean — functions are well-named and not excessively nested.

### 3.2 Code Patterns

**Good patterns**:
- Repository pattern in store layer (`user_repo.go`, `api_key_repo.go`, `credit_repo.go`, etc.)
- Table-driven tests throughout the codebase
- Context-aware methods on all repositories
- Atomic config swap via `mutateConfig()` pattern
- sync.Pool for header map reuse in audit hot path
- LIFO shutdown hook execution
- Constant-time comparison for auth tokens
- Right-to-left XFF parsing for trusted proxy extraction

**Inconsistent patterns**:
- Type coercion helpers duplicated across 5+ files: `admin/admin_helpers.go` (asAnyMap, asStringSlice, asIntSlice, asBool, asString, asInt, asInt64, asFloat64), `internal/graphql/proxy.go`, `internal/admin/bulk.go`, `internal/portal/helpers.go`, `internal/mcp/server.go`. These should be a shared utility.
- Error types: some use custom error structs (`AuthError`, `RateLimitError`, `BotDetectError`), some use plain `fmt.Errorf`. Inconsistent.
- Config cloning: `cloneConfig` in admin, `clonePluginConfigs` in admin, `clonePluginConfigsBulk` in bulk.go, `cloneConfig`/`clonePluginConfigs`/`cloneBillingConfig` in MCP server — 5 implementations of the same pattern.

### 3.3 Concurrency Safety

| Component | Safety Mechanism | Verdict |
|-----------|-----------------|---------|
| Gateway config | `sync.RWMutex` on config field | Safe |
| Router rebuild | Atomic pointer swap of snapshot | Safe |
| Analytics ring buffer | `atomic.Pointer` for lock-free access | Safe |
| Admin rate limiting | `sync.Mutex` on IP map | Safe |
| WebSocket hub | Channel-based event loop | Safe |
| Raft FSM | Single-threaded event loop | Safe |
| Audit logger | Buffered channel, single consumer | Safe |
| JWT replay cache | `sync.Mutex` on map | Safe but could be a bottleneck |
| Connection pool | `sync.Pool` | Safe |
| Optimized pipeline | `sync.Map` + atomic counters | Safe |

### 3.4 Error Handling

Error handling is generally good with sentinel errors and custom error types in most packages. However:

- `internal/gateway/server.go:ServeHTTP` — many errors are logged but silently returned to the client as 500. More granular error mapping would help.
- `internal/plugin/rate_limit.go` — rate limit decisions return errors that are handled, but the plugin doesn't properly integrate with the billing system for rate-limited requests.
- `internal/store/` — SQL errors are wrapped with context (`fmt.Errorf("create user: %w", err)`), which is good.

---

## 4. Testing

### 4.1 Test Results (2026-04-11)

**All 37 packages pass. Zero failures.**

```
internal/admin        75.1%
internal/analytics    89.9%
internal/audit        86.7%
internal/billing      93.2%
internal/certmanager  91.3%
internal/cli          80.5%
internal/config       94.8%
internal/federation   88.5%
internal/gateway      84.5%
internal/graphql      91.0%
internal/grpc         94.1%
internal/loadbalancer 91.3%
internal/logging      80.7%
internal/mcp          88.6%
internal/metrics      95.9%
internal/pkg/json     100.0%
internal/pkg/jwt      93.0%
internal/pkg/netutil  90.9%
internal/pkg/template 97.4%
internal/pkg/uuid     83.3%
internal/pkg/yaml     100.0%
internal/plugin       86.3%
internal/portal       80.0%
internal/raft         84.3%
internal/shutdown     84.8%
internal/store        85.2%
internal/tracing      84.0%
test/helpers          48.0%
test/integration      [no statements]
test/loadtest         59.4%
```

### 4.2 Test Coverage Analysis

**Well-covered (90%+)**: billing, certmanager, config, graphql, grpc, loadbalancer, metrics, pkg/json, pkg/jwt, pkg/template, pkg/yaml

**Adequately covered (80-90%)**: analytics, audit, cli, federation, gateway, mcp, plugin, raft, shutdown, store, tracing

**Needs attention (<80%)**: admin (75.1%), portal (80.0%), logging (80.7%), test/helpers (48.0%)

**Coverage gaps by package**:

- **admin (75.1%)**: OAuth SSO callback paths, bulk import edge cases, advanced analytics (forecasting, anomaly detection, correlation), GraphQL mutation resolvers, config import/export with secret redaction
- **gateway (84.5%)**: WebSocket upgrade edge cases, gRPC transcoding error paths, optimized proxy request coalescing, health check passive failure detection
- **plugin (86.3%)**: WASM plugin execution with complex host functions, marketplace signature verification, cache warm-up with stale entries
- **portal (80.0%)**: CSRF validation edge cases, IP whitelist management, usage analytics with empty data
- **raft (84.3%)**: Multi-region latency measurement, certificate ACME renewal lock, InstallSnapshot with large snapshots

### 4.3 Test Quality

**Strengths**:
- Table-driven tests with parallel subtests (consistent pattern)
- Integration tests covering full request flows
- E2E tests for admin configuration and proxy, hot reload, chaos scenarios
- Benchmark tests for router and balancer
- In-memory SQLite for fast test execution
- Mock Redis via miniredis

**Weaknesses**:
- E2E tests use real admin API calls but mock the gateway — not full end-to-end
- No load testing with realistic concurrency (the load tests are minimal)
- No property-based testing for the radix tree router
- No fuzzing for JWT parsing, YAML bomb detection, or regex ReDoS

---

## 5. Spec-vs-Implementation Gap Analysis

### 5.1 Implemented and Matching Spec

| Spec Requirement | Implementation | Status |
|-----------------|----------------|--------|
| HTTP/HTTPS reverse proxy | `gateway/server.go`, `optimized_proxy.go` | Implemented |
| gRPC proxy + transcoding | `grpc/proxy.go`, `grpc/transcoder.go` | Implemented |
| gRPC-Web | `grpc/proxy.go:handleGRPCWeb` | Implemented |
| gRPC streaming (all 4 types) | `grpc/stream.go` | Implemented |
| GraphQL Federation | `federation/composer.go`, `federation/planner.go` | Implemented |
| GraphQL subscriptions | `graphql/subscription.go` | Implemented |
| APQ (persisted queries) | `graphql/apq.go` | Implemented |
| Query depth/complexity | `graphql/analyzer.go` | Implemented |
| Radix tree router | `gateway/router.go` | Implemented |
| 11 load balancing algorithms | `gateway/balancer.go`, `balancer_extra.go`, `loadbalancer/` | Implemented |
| 5-phase plugin pipeline | `plugin/pipeline.go`, `optimized_pipeline.go` | Implemented |
| Rate limiting (4 algorithms) | `ratelimit/` | Implemented |
| Distributed rate limiting | `ratelimit/redis.go` | Implemented |
| API key auth | `plugin/auth_apikey.go` | Implemented |
| JWT auth (HS256, RS256, ES256) | `plugin/auth_jwt.go`, `pkg/jwt/` | Implemented |
| Credit billing | `billing/engine.go`, `store/credit_repo.go` | Implemented |
| Audit logging with masking | `audit/logger.go`, `audit/masker.go` | Implemented |
| Kafka audit streaming | `audit/kafka.go` | Implemented |
| Raft clustering | `raft/node.go`, `raft/cluster.go` | Implemented |
| Raft mTLS | `raft/tls.go` | Implemented |
| Multi-region | `raft/multiregion.go` | Implemented |
| MCP server | `mcp/server.go` | Implemented |
| WASM plugins | `plugin/wasm.go` | Implemented |
| Plugin marketplace | `plugin/marketplace.go` | Implemented |
| ACME/Let's Encrypt | `certmanager/acme.go`, `gateway/tls.go` | Implemented |
| OpenTelemetry tracing | `tracing/tracing.go` | Implemented |
| WebSocket proxy | `gateway/proxy.go` | Implemented |
| Admin REST API (70+ endpoints) | `admin/` (20 files) | Implemented |
| User portal | `portal/` (6 files) | Implemented |
| React dashboard | `web/` | Implemented |
| CLI (40+ commands) | `cli/` | Implemented |
| Hot config reload | `config/dynamic_reload.go`, `config/watch.go` | Implemented |
| CORS | `plugin/cors.go` | Implemented |
| Bot detection | `plugin/bot_detect.go` | Implemented |
| Circuit breaker | `plugin/circuit_breaker.go` | Implemented |
| Request/response transforms | `plugin/request_transform.go`, `plugin/response_transform.go` | Implemented |
| URL rewriting | `plugin/url_rewrite.go` | Implemented |
| Compression | `plugin/compression.go` | Implemented |
| Request validation | `plugin/request_validator.go` | Implemented |
| Caching | `plugin/cache.go` | Implemented |
| Webhooks | `admin/webhooks.go` | Implemented |
| GraphQL admin API | `admin/graphql.go` | Implemented |
| Analytics with alerts | `admin/analytics.go`, `analytics/alerts.go` | Implemented |
| Bulk operations | `admin/bulk.go` | Implemented |
| RBAC | `admin/rbac.go` | Implemented |
| OIDC SSO | `admin/oidc.go` | Implemented |
| Branding | `admin/server.go:handleBranding` | Implemented |

### 5.2 Partially Implemented

| Spec Requirement | Implementation | Gap |
|-----------------|----------------|-----|
| Custom error pages per route | Not found in codebase | All errors are JSON responses — no HTML error pages |
| Field-level authorization for GraphQL | `plugin/graphql_guard.go` exists but basic | No `@authorized` directive parsing in federation composer |
| Query batching | Federation executor supports batch but not client-facing batching endpoint | Spec says "Query batching support" — only internal batching in executor |
| Per-route rate limit in gateway | Rate limit plugin exists but not auto-applied | Must be explicitly added to route's plugin config |

---

## 6. Performance Considerations

### 6.1 Hot Path Analysis

The request hot path is: `ServeHTTP` → `router.Match()` → `pipeline.Execute()` → `balancer.Next()` → `proxy.Forward()`

**Bottlenecks identified**:

1. **`ServeHTTP` sequential logic** (lines 191-597, ~400 lines): Every request walks through all plugin phases even if no plugins are registered for that route. The `OptimizedPipeline` helps with caching but the gateway-level logic still iterates through everything.

2. **SQLite writes on critical path**: Billing deduction (`applyBillingPostProxy`) writes to SQLite on every request that has billing enabled. With SQLITE_BUSY retry (up to 5 attempts with backoff), this can add 50-200ms of latency under write contention.

3. **Audit logging channel**: Non-blocking drop when channel is full. Under high throughput (>10K req/s), audit entries will be silently dropped. The `Logger.Dropped()` counter exists but isn't exposed on any endpoint.

4. **Radix tree rebuild**: `Router.Rebuild()` allocates entirely new trees. During hot reload with many routes, this creates GC pressure. The swap itself is atomic (pointer assignment), but the allocation is not incremental.

5. **Analytics ring buffer**: Pushes to both ring buffer and time series store synchronously in the request path. The `OptimizedEngine` queues metrics for async processing, but the default `Engine` in `analytics/engine.go` is synchronous.

### 6.2 Memory Usage

| Component | Memory Pattern | Concern |
|-----------|---------------|---------|
| Analytics ring buffer (100K entries) | ~8MB at steady state | Bounded, predictable |
| Time series store | Grows with unique route/consumer combinations | Reservoir sampling caps at 10K latencies per bucket |
| Audit channel (10K buffer) | ~4MB | Bounded, but entries dropped on overflow |
| JWT replay cache | Unbounded map with TTL cleanup | Could grow under attack |
| GraphQL query cache | 1,000 entries with LRU | Bounded |
| Connection pool | sync.Pool, GC-managed | No leak risk |
| WebSocket connections | 1 goroutine pair per connection | ~2 goroutines per active WS |

### 6.3 Benchmark Results

The benchmark suite (`test/benchmark/`) is minimal — only basic throughput tests. There are no:
- Latency percentile benchmarks (p50, p95, p99)
- Memory allocation benchmarks
- Concurrent client benchmarks
- WebSocket throughput benchmarks
- gRPC vs HTTP transcoding overhead benchmarks

---

## 7. Developer Experience

### 7.1 Build & Development

| Aspect | Status |
|--------|--------|
| Single binary build | `make build` |
| Hot config reload | SIGHUP + fsnotify |
| Local dev setup | `make build` + config file |
| Test execution | `go test ./...` — all pass |
| Coverage report | `make coverage` → HTML report |
| Race detection | `make test-race` |
| Linting | `make lint` — `go vet` + golangci-lint |
| Docker build | `make docker` |
| K8s deployment | `make deploy-k8s` |
| CI pipeline | `make ci` |

### 7.2 Code Organization

**Strengths**:
- Clean package boundaries — each package has a clear responsibility
- `internal/pkg/` for shared utilities (jwt, yaml, json, uuid, netutil, template)
- Consistent naming conventions
- Good use of Go interfaces for testability (e.g., `Storage` interface in raft, `WebhookStore` interface in admin)

**Weaknesses**:
- 15+ files exceed 700 lines
- Duplicated type-coercion helpers across 5 files
- Some packages mix concerns (e.g., `store/audit_repo.go` handles search, export, retention, CSV, JSONL — could be split)
- No shared error types package — each package defines its own error structs

### 7.3 Documentation

| Document | Status |
|----------|--------|
| README.md | Comprehensive, but stats are outdated |
| CLAUDE.md | Detailed architecture + commands |
| AGENT_DIRECTIVES.md | Clear coding rules |
| SPECIFICATION.md | Detailed feature spec |
| API.md | Complete endpoint reference |
| CONTRIBUTING.md | Basic guidelines |
| CHANGELOG.md | Version history |
| RUNBOOK.md | Operational procedures |
| SECURITY.md | Security practices |
| docs/ARCHITECTURE_DECISIONS.md | ADRs |
| docs/TRACING.md | Tracing setup |
| docs/WASM_PLUGINS.md | WASM plugin guide |
| docs/KAFKA_AUDIT_STREAMING.md | Kafka streaming guide |
| docs/REDIS_RATE_LIMITING.md | Redis rate limiting guide |
| docs/ACME_RAFT_SYNC.md | ACME + Raft cert sync |
| docs/SECURITY_AUDIT.md | Security audit findings |
| docs/architecture/ | Architecture docs |
| docs/production/ | Production guides |
| security-report/ | Detailed security findings |
| .project/TASKS.md | All tasks marked complete — over-claimed |

**Concern**: `.project/TASKS.md` claims 100% completion of all 490 tasks across 17 versions. Many items marked `[x]` are not fully implemented (see Section 5 gap analysis).

---

## 8. Technical Debt

### 8.1 High Priority

| Issue | Location | Impact | Effort |
|-------|----------|--------|--------|
| `ServeHTTP` too large (400+ lines) | `gateway/server.go:191-597` | Maintainability, bug risk | Medium |
| Duplicated type coercion helpers | 5+ files | Code duplication, inconsistency | Low |
| Duplicated config cloning | 5 files | Code duplication, drift risk | Low |
| Audit entries silently dropped under load | `audit/logger.go:202` | Data loss | Medium |
| Rate limit plugin not auto-applied to routes | `plugin/rate_limit.go` | Security gap if not manually configured | Low |
| Custom error pages not implemented | N/A | Missing spec requirement | Medium |

### 8.2 Medium Priority

| Issue | Location | Impact | Effort |
|-------|----------|--------|--------|
| `mcp/server.go` — 39 tools in one file (1,284 lines) | `mcp/server.go` | Maintainability | Medium |
| `federation/executor.go` — too many responsibilities (892 lines) | `federation/executor.go` | Maintainability | Medium |
| No fuzzing for security-sensitive parsers | JWT, YAML, router regex | Security | Medium |
| No load testing with realistic concurrency | `test/benchmark/` | Unknown production capacity | Medium |
| JWT replay cache unbounded | `plugin/jti_replay.go` | Memory leak under attack | Low |
| `store/audit_repo.go` — 938 lines, too many responsibilities | `store/audit_repo.go` | Maintainability | Medium |

### 8.3 Low Priority

| Issue | Location | Impact | Effort |
|-------|----------|--------|--------|
| Go version in README says 1.25+ but go.mod says 1.26.2 | `README.md:8` | Confusion | Trivial |
| Coverage badge says 81.2% but actual is ~85% | `README.md:10` | Outdated | Trivial |
| Go source file count says 137 but actual is 233 | `README.md:42` | Outdated | Trivial |
| Test file count says 162 but actual is 399 | `README.md:43` | Outdated | Trivial |
| `internal/version/` has no testable statements | `internal/version/version.go` | Coverage noise | Trivial |
| `test/helpers` at 48% coverage | `test/helpers/` | Test quality | Low |

---

## 9. Metrics Summary

| Metric | Value |
|--------|-------|
| Go source files | 233 |
| Test files | 399 |
| Go lines of code | ~184,000 |
| Frontend lines of code | ~25,000 |
| Internal packages | 28 |
| Total test packages | 37 (all passing) |
| Overall coverage | ~85% |
| Admin API endpoints | 70+ |
| CLI commands | 40+ |
| Load balancing algorithms | 11 |
| Plugin types | 30+ |
| External dependencies | 19 direct, 27 indirect |
| Raft cluster nodes | 3+ supported |
| MCP tools | 39 |
