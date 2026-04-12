# APICerebrus Security Audit Report

**Date:** 2026-04-09
**Scope:** Full codebase audit — 4-phase pipeline (Recon → Hunt → Verify → Report)
**Scanners:** 7 parallel agents across injection, auth/access control, Go deep scan, secrets/data exposure, dependency audit, IaC/Docker
**Total Findings:** 62 across all categories

---

## Executive Summary

APICerebrus demonstrates strong foundational security practices: parameterized SQL queries throughout, `crypto/rand` for secrets, bcrypt password hashing, trusted proxy discipline, and security headers. However, the audit identified **6 CRITICAL**, **16 HIGH**, **22 MEDIUM**, and **18 LOW** severity issues that require remediation before production deployment.

### Severity Distribution

| Severity | Count | Key Themes |
|----------|-------|------------|
| CRITICAL | 6 | SSRF, cache poisoning, unauthenticated Raft RPC, SSTI, admin password in stderr |
| HIGH | 16 | Auth bypass, secrets exposure (config export, password hash, API keys), WebSocket OOM, CORS credential reflection, YAML bomb, SSRF vectors, hardcoded compose secrets, error info leaks |
| MEDIUM | 22 | TLS weaknesses, JWT in query strings/body, context-ignoring goroutines, ReDoS, CSRF gaps, audit masking gaps, CI/CD pinning |
| LOW/INFO | 18 | Memory design choices, modulo bias in token generation, CI action pinning, K8s/Helm patterns |

---

## CRITICAL Findings

### C1: SSRF via Upstream URL Construction
- **File:** `internal/gateway/proxy.go:268-270`
- **CWE:** CWE-918
- **Issue:** `buildUpstreamURL` auto-prepends `http://` to upstream addresses lacking a scheme. Admin-configurable upstreams can target `169.254.169.254/latest/meta-data/` or internal services.
- **Impact:** Access to cloud metadata, internal APIs, and services behind the gateway.

### C2: Request Coalescing Cache Poisoning Across Users
- **File:** `internal/gateway/optimized_proxy.go:560`
- **CWE:** CWE-639
- **Issue:** `coalesceKey` does not include authentication headers in its cache key. Two users requesting the same path receive coalesced responses, potentially leaking authenticated data between users.
- **Impact:** Cross-user data leakage in proxied responses.

### C3: Unauthenticated Raft RPC Endpoints
- **File:** `internal/raft/transport.go:212-296`
- **CWE:** CWE-306
- **Issue:** All Raft HTTP RPC handlers (`handleRequestVote`, `handleAppendEntries`, `handleInstallSnapshot`) accept requests from any network-accessible client with zero authentication.
- **Impact:** An attacker reaching the Raft port can forge votes, inject log entries, or overwrite cluster state.

### C4: Server-Side Template Injection (SSTI) in Webhook Templates
- **File:** `internal/analytics/webhook_templates.go:470-495, 663-689`
- **CWE:** CWE-1336
- **Issue:** `CreateCustomTemplate` accepts user-provided Go `text/template` bodies without sanitization. Arbitrary Go template expressions including method calls and data exfiltration are possible.
- **Impact:** Arbitrary data exfiltration through webhook notification bodies sent to external endpoints.

### C5: Admin Password Printed to stderr in Plaintext
- **File:** `internal/store/user_repo.go:472-479`
- **CWE:** CWE-532
- **Issue:** When `APICERBERUS_ADMIN_PASSWORD` is not set, the auto-generated admin password is printed to stderr in plaintext alongside the admin email. In containerized environments, stderr is captured by log aggregation systems (CloudWatch, ELK, Splunk) accessible to multiple team members.
- **Impact:** Full admin credentials exposed in log aggregation systems.

### C6: Config Export Leaks All Secrets in Plaintext
- **File:** `internal/admin/server.go:329-342`
- **CWE:** CWE-200
- **Issue:** `handleConfigExport` serializes the entire `*config.Config` via YAML including `admin.api_key`, `admin.token_secret`, `portal.session.secret`, `redis.password`, Kafka SASL credentials. No redaction is performed.
- **Impact:** Single API call yields every secret in the system to any authenticated admin.

---

## HIGH Findings

### H1: Admin Dual-Auth Fallback Bypass
- **File:** `internal/admin/server.go:236-275`
- **CWE:** CWE-285, CWE-306
- **Issue:** Failed Bearer token validation silently falls through to static key check without recording a failed auth attempt, enabling unlimited Bearer token probing.

### H2: Admin getUser Returns Raw Password Hash
- **File:** `internal/admin/admin_users.go:109`
- **CWE:** CWE-200
- **Issue:** `getUser` returns the raw `store.User` struct including `PasswordHash`. `sanitizeUser()` exists in portal code but is never used in admin endpoints.

### H3: CORS Wildcard with Credential Reflection
- **File:** `internal/plugin/cors.go:28-36, 101-118`
- **CWE:** CWE-942
- **Issue:** `allowed_origins: ["*"]` with `credentials: true` reflects any origin back with `Access-Control-Allow-Credentials: true`, enabling cross-origin credentialed requests from any website.

### H4: WebSocket Frame Unbounded Payload (OOM)
- **File:** `internal/graphql/subscription.go:364`
- **CWE:** CWE-770
- **Issue:** `readWSFrame` allocates `payload = make([]byte, length)` directly from the 8-byte wire-declared frame size without maximum validation.

### H5: YAML Billion Laughs / Entity Expansion
- **File:** `internal/pkg/yaml/decode.go:155, 179`
- **CWE:** CWE-776, CWE-770
- **Issue:** Custom YAML parser has no maximum depth or node count limit. Malicious YAML with recursive anchors or millions of keys causes exponential memory/CPU consumption.

### H6: Webhook Delivery SSRF
- **File:** `internal/admin/webhooks.go:154`
- **CWE:** CWE-918
- **Issue:** HTTP POST to user-configured `webhook.URL` without SSRF validation. Code is marked `#nosec G704`, acknowledging the risk.

### H7: Federation Executor SSRF
- **File:** `internal/federation/executor.go:313`
- **CWE:** CWE-918
- **Issue:** Subgraph URL validation missing. Requests sent to arbitrary URLs from user-configurable subgraph definitions.

### H8: Open Redirect with Parameter Leakage
- **File:** `internal/plugin/redirect.go:59-60`
- **CWE:** CWE-601
- **Issue:** Redirect target is user-configured without validation. Original request query string (potentially containing tokens) is appended to redirect URL.

### H9: Raw Error Messages Leaked to Clients
- **File:** `internal/gateway/server.go:997`, `internal/portal/handlers_api.go:19, 35, 85, 124, 133, 155`
- **CWE:** CWE-209
- **Issue:** `err.Error()` written directly to HTTP responses across gateway and portal. Internal details (file paths, SQL queries) exposed.

### H10: Raft Transport — No Request Body Size Limits
- **File:** `internal/raft/transport.go:212-296`
- **CWE:** CWE-770
- **Issue:** Raft RPC HTTP handlers have no `http.MaxBytesReader` or equivalent body size limiting.

### H11: GraphQL Admin — API Keys Exposed via Query
- **File:** `internal/admin/graphql.go:366-380`
- **CWE:** CWE-200
- **Issue:** `consumerType` GraphQL field exposes raw `"key"` field of API keys. Any user with admin GraphQL access can query all consumer API keys in plaintext.

### H12: RSA Key — No Minimum Size Validation
- **File:** `internal/pkg/jwt/rs256.go:63-72`
- **CWE:** CWE-326
- **Issue:** `ParseRSAPublicKeyFromJWK` accepts weak RSA keys (512-bit, 1024-bit) without enforcing a 2048-bit minimum.

### H13: Hardcoded Default Secrets in Docker Compose Files
- **Files:** `docker-compose.yml:42-43`, `docker-compose.standalone.yml:16-17`, `docker-compose.swarm.yml:148`
- **CWE:** CWE-798
- **Issue:** JWT secret defaults to `dev-jwt-secret-change-in-production`, admin key to `dev-admin-key-change-in-production`, standalone to `changeme`, PostgreSQL to `postgres`. If `.env` variables are not set, weak defaults are used.

### H14: Default Grafana admin/admin Credentials
- **Files:** `docker-compose.yml:113-114`, `docker-compose.swarm.yml:255`, `monitoring/docker-compose.yml:51-52`
- **CWE:** CWE-798
- **Issue:** Default Grafana credentials are `admin`/`admin`. Anyone running the compose stack exposes monitoring dashboards to unauthorized access.

### H15: Missing .gitignore Patterns for Secrets
- **File:** `.gitignore`
- **CWE:** CWE-359
- **Issue:** Root `.gitignore` does not exclude `.env`, `*.pem`, `*.key`, `*.crt`, `secrets/`, `credentials*`, `kubeconfig`, `terraform.tfstate`, `*.jks`. The `.dockerignore` already has many of these — they were simply not copied to `.gitignore`.

### H16: Placeholder Secrets in Kubernetes Base Manifest
- **File:** `deployments/kubernetes/base/secret.yaml:16-17`
- **CWE:** CWE-798
- **Issue:** Secret manifest contains literal `CHANGE_ME_IN_PRODUCTION` values. If applied directly without overlay, production services use these weak placeholders.

---

## MEDIUM Findings

| ID | Finding | File | CWE |
|----|---------|------|-----|
| M1 | Webhook retry goroutine leak — no context cancellation | `internal/admin/webhooks.go:228` | CWE-404 |
| M2 | Proxy retry backoff ignores context cancellation | `internal/gateway/server.go:556` | CWE-404 |
| M3 | Load balancer unbounded weighted expansion | `internal/gateway/balancer.go:182-198` | CWE-770 |
| M4 | Admin password reset lacks minimum length | `internal/admin/admin_users.go:236` | CWE-521 |
| M5 | Admin API lacks CSRF protection for cookie auth | `internal/admin/server.go:123, 139-168` | CWE-352 |
| M6 | JWT token in WebSocket query parameter (logged) | `internal/admin/ws.go:129` | CWE-598 |
| M7 | Endpoint permission bypass on empty consumer ID | `internal/plugin/endpoint_permission.go:62-66` | CWE-862 |
| M8 | Router regex complexity — ReDoS | `internal/gateway/router.go:616-625` | CWE-1333 |
| M9 | TLS allows deprecated versions (1.0, 1.1) | `internal/gateway/tls.go:100-112` | CWE-327 |
| M10 | TLS weak cipher suites (no forward secrecy) | `internal/gateway/tls.go:123-129` | CWE-327 |
| M11 | Admin session cookie Secure flag conditional | `internal/admin/token.go:184-195` | CWE-614 |
| M12 | API key accepted via URL query parameters | `internal/plugin/auth_apikey.go:230-234` | CWE-598 |
| M13 | Webhook secret returned in plaintext response | `internal/admin/webhooks.go:683-687` | CWE-532 |
| M14 | Raft cluster auth uses non-constant-time comparison | `internal/raft/cluster.go:88-97` | CWE-208 |
| M15 | JWT returned in response body + HttpOnly cookie | `internal/admin/token.go:197-201` | CWE-598 |
| M16 | No default mask fields for audit logging | `internal/audit/masker.go:79-87` | CWE-359 |
| M17 | HS256 with no entropy validation on secret | `internal/pkg/jwt/hs256.go:16-23` | CWE-326 |
| M18 | Env override can overwrite any config secret | `internal/config/env.go:12-46` | CWE-15 |
| M19 | TLS SkipVerify option present in config | `internal/config/types.go:147` | CWE-295 |
| M20 | CI/CD actions pinned to `@master` branch | `.github/workflows/ci.yml:323,338` | CWE-829 |
| M21 | Redis password exposed in health check CLI args | `docker-compose.prod.yml:122` | CWE-214 |
| M22 | Secrets stored as local plaintext files in compose | `docker-compose.prod.yml:254-261` | CWE-359 |

---

## LOW / Informational Findings

| ID | Finding | File |
|----|---------|------|
| L1 | Analytics TimeSeries unbounded map growth between cleanups | `internal/analytics/engine.go:241` |
| L2 | 640MB pre-allocated buffer pool at startup | `internal/gateway/optimized_proxy.go:233-236` |
| L3 | Request body doubling in Capture GetBody closure | `internal/audit/capture.go:161` |
| L4 | GraphQL proxy 50MB response read limit generous | `internal/graphql/proxy.go:98` |
| L5 | Custom homegrown JWT library (`internal/pkg/jwt/`) | `internal/pkg/jwt/` |
| L6 | Custom YAML parser not fuzz-tested | `internal/pkg/yaml/` |
| L7 | WASM plugin system allows arbitrary code execution in-process | `internal/plugin/wasm.go` |
| L8 | Go version in go.mod (1.25.0) lags installed (1.26.1) | `go.mod` |
| L9 | API key token generation has modulo bias | `internal/store/api_key_repo.go:368` |
| L10 | Password generation modulo bias + deterministic fallback | `internal/store/user_repo.go:512-526` |
| L11 | Proxy copies all upstream response headers to client | `internal/gateway/proxy.go:131` |
| L12 | Config import uses world-readable temp directory | `internal/admin/server.go:356` |
| L13 | Deprecated `golang.org/x/net/websocket` package | `internal/federation/executor.go` |
| L14 | CI `govulncheck` not version-pinned | `.github/workflows/ci.yml:350` |
| L15 | Deprecated GitHub Actions (create-release, upload-release-asset) | `.github/workflows/release.yml` |
| L16 | K8s validation errors suppressed with `\|\| true` | `.github/workflows/ci.yml:441` |
| L17 | cAdvisor runs as privileged container | `monitoring/docker-compose.yml:144` |
| L18 | Helm TLS disabled by default in ingress | `helm/apicerberus/values.yaml:44-53` |

---

## Positive Security Findings

The audit confirmed these security controls are correctly implemented:

- **Parameterized SQL queries** throughout all store repositories
- **crypto/rand** for UUID, session tokens, and API key generation
- **bcrypt cost 10** for password hashing
- **SHA-256 hashing** for stored API keys (raw keys never stored in DB)
- **subtle.ConstantTimeCompare** for auth credential comparisons
- **Trusted proxy discipline** — forwarding headers ignored by default
- **Security headers** on all responses (HSTS, CSP, X-Frame-Options, etc.)
- **Raft TLS** enforces TLS 1.3 minimum with 4096-bit RSA certs
- **Config validation** rejects weak admin key patterns
- **IP allowlisting** on admin API before authentication
- **Audit log masking** of Authorization headers and sensitive body fields
- **Kubernetes deployment** runs as non-root with dropped capabilities, read-only filesystem
- **Network policies** restrict ingress/egress traffic
- **Distroless runtime** image with no shell or package manager

---

## Dependency Audit Summary

| Dependency | Version | Risk | Notes |
|------------|---------|------|-------|
| `golang.org/x/net/websocket` | — | MEDIUM | Deprecated package, migrate to `nhooyr.io/websocket` |
| `github.com/graphql-go/graphql` | v0.8.1 | MEDIUM | Stale (2022), no query depth limiting configured |
| `modernc.org/sqlite` | v1.48.0 | LOW | Complex transitive tree (15+ modernc.org packages), no CVEs |
| `github.com/gorilla/websocket` | — | LOW | **Not used** — admin WS uses raw hijack, federation uses deprecated x/net |
| `go.opentelemetry.io/otel` | v1.42.0 | LOW | Current release, no CVEs |
| `golang.org/x/crypto` | v0.49.0 | LOW | Recent, no CVEs |
| `google.golang.org/grpc` | v1.79.2 | LOW | Recent, no CVEs |
| Custom JWT (`internal/pkg/jwt/`) | — | MEDIUM | Homegrown, not independently audited |
| Custom YAML (`internal/pkg/yaml/`) | — | HIGH | Custom parser, lacks depth limits, not fuzzed |
| `github.com/redis/go-redis/v9` | v9.7.0 | LOW | Actively maintained, no CVEs |

**Key recommendations:**
1. Add GraphQL query depth/complexity limits
2. Migrate off `golang.org/x/net/websocket` to `nhooyr.io/websocket`
3. Validate JWKS URL scheme (enforce `https://`)
4. Pin `govulncheck` version in CI

---

## Infrastructure & Deployment Findings

| ID | Finding | Severity | File |
|----|---------|----------|------|
| I1 | Missing `.env`, `*.pem`, `secrets/` in `.gitignore` | HIGH | `.gitignore` |
| I2 | Placeholder secrets in K8s base manifests | HIGH | `kubernetes/base/secret.yaml` |
| I3 | Hardcoded JWT/admin keys in docker-compose files | HIGH | `docker-compose.yml`, `standalone.yml`, `swarm.yml` |
| I4 | Default Grafana admin/admin in compose files | HIGH | `docker-compose.yml`, `swarm.yml`, `monitoring/` |
| I5 | PostgreSQL default password "postgres" | HIGH | `docker-compose.swarm.yml`, `swarm-raft.yml` |
| I6 | `:latest` image tag in K8s deployment | MEDIUM | `kubernetes/base/deployment.yaml` |
| I7 | Helm empty secret defaults | MEDIUM | `helm/apicerberus/values.yaml` |
| I8 | Prometheus endpoint exposed without auth | MEDIUM | `docker-compose.swarm.yml` |
| I9 | Redis password in health check CLI args | MEDIUM | `docker-compose.prod.yml:122` |
| I10 | Secrets as local plaintext files | MEDIUM | `docker-compose.prod.yml:254-261` |
| I11 | cAdvisor runs as privileged container | MEDIUM | `monitoring/docker-compose.yml:144` |
| I12 | CI actions pinned to `@master` branch | MEDIUM | `.github/workflows/ci.yml:323,338` |
| I13 | Release workflow no PR approval gate | MEDIUM | `.github/workflows/release.yml` |
| I14 | TLS disabled by default in Helm ingress | LOW | `helm/apicerberus/values.yaml:44-53` |
| I15 | CI `govulncheck` not version-pinned | LOW | `.github/workflows/ci.yml:350` |
| I16 | Deprecated GitHub Actions in release workflow | LOW | `.github/workflows/release.yml` |
| I17 | K8s validation errors suppressed | LOW | `.github/workflows/ci.yml:441` |
| I18 | Monitoring images use `:latest` tags | LOW | Multiple compose files |
| I19 | Unquoted variable in Makefile restore target | MEDIUM | `Makefile:148` |
| I20 | Duplicate Makefile target definitions | LOW | `Makefile:89-102, 174-187` |

---

## Remediation Priority

### Immediate (Ship-Blockers)
1. **C1:** Validate upstream URL schemes; block private/metadata IP ranges
2. **C2:** Include auth identity in request coalesce key
3. **C3:** Add authentication to Raft RPC HTTP endpoints (shared secret or mTLS)
4. **C4:** Replace `text/template` with a sandboxed template engine or whitelist-based parser
5. **C5:** Remove password from stderr output; use secure secret delivery mechanism
6. **C6:** Redact all secrets from config export endpoint

### High Priority (Before Production)
7. **H1:** Record failed Bearer attempts in rate limiter; remove silent fallback
8. **H2:** Use `sanitizeUser()` for all admin user serialization
9. **H3:** Reject `credentials: true` with `allowed_origins: ["*"]`
10. **H4:** Enforce maximum WebSocket frame size (e.g., 1MB)
11. **H5:** Add depth and node count limits to YAML parser
12. **H6:** Validate webhook URLs against private IP ranges
13. **H9:** Sanitize error messages before sending to clients
14. **H13:** Remove all hardcoded secrets from docker-compose files
15. **H14:** Require unique Grafana credentials via env vars with no defaults
16. **H15:** Add secret file patterns to `.gitignore`
17. **H16:** Remove `stringData` from K8s base secret manifest

### Medium Priority (Next Sprint)
18. **M9-M10:** Enforce TLS 1.2+ minimum, remove weak cipher suites
19. **M6, M12, M15:** Stop accepting secrets in URL query parameters and response bodies
20. **M4:** Enforce password minimum length on reset
21. **M7:** Require valid consumer ID for permission evaluation
22. **M16:** Add default mask fields (Authorization, Cookie, X-API-Key, password)
23. **M18:** Restrict env overrides from overwriting secret fields
24. **M20:** Pin CI/CD actions to version tags or commit SHAs

### Low Priority (Backlog)
25. Replace deprecated `golang.org/x/net/websocket` package
26. Add `govulncheck` to CI with pinned version
27. Update Go version in go.mod to match installed version
28. Add GraphQL query complexity limits
29. Enforce minimum RSA key size (2048-bit)
30. Add vendor directory for reproducible builds

---

## Methodology

This audit used a 4-phase pipeline executed by 7 parallel scanning agents:

1. **RECON:** Architecture mapping, tech stack detection, endpoint enumeration, configuration analysis
2. **HUNT:** 6 specialized vulnerability scanners running in parallel:
   - Injection flaws (SQL, command, GraphQL, LDAP, XXE, SSTI)
   - Authentication & access control bypasses
   - Go deep scan (concurrency, memory, input validation, crypto, serialization)
   - Secrets & data exposure (hardcoded secrets, logging, crypto, masking)
   - Dependency audit (CVEs, staleness, supply chain)
   - IaC/Docker/CI/CD security
3. **VERIFY:** False positive elimination and confidence scoring via source code reading
4. **REPORT:** This consolidated report with CVSS-aligned severity and remediation roadmap

All findings were verified by reading the actual source code. No static analysis tools were used — this is a manual code review performed by AI agents.
