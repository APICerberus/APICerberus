# Verified Security Findings

**Date:** 2026-04-10
**Method:** Manual code review by 7 AI agents — all findings verified by reading source code
**Status:** 62/62 findings resolved (100%)

---

## Confirmed CRITICAL (6)

| ID | Finding | Confidence | File | CWE |
|----|---------|------------|------|-----|
| C1 | SSRF via Upstream URL auto-scheme | High | `internal/gateway/proxy.go:268-270` | CWE-918 |
| C2 | Request coalescing cache poisoning | High | `internal/gateway/optimized_proxy.go:560` | CWE-639 |
| C3 | Unauthenticated Raft RPC endpoints | High | `internal/raft/transport.go:212-296` | CWE-306 |
| C4 | SSTI in webhook template engine | High | `internal/analytics/webhook_templates.go:470-495` | CWE-1336 |
| C5 | Admin password printed to stderr | High | `internal/store/user_repo.go:472-479` | CWE-532 |
| C6 | Config export leaks all secrets | High | `internal/admin/server.go:329-342` | CWE-200 |

---

## Confirmed HIGH (16)

| ID | Finding | Confidence | File | CWE |
|----|---------|------------|------|-----|
| H1 | Admin dual-auth bypass (Bearer to static fallback) | High | `internal/admin/server.go:236-275` | CWE-285 |
| H2 | Password hash exposed in admin API response | High | `internal/admin/admin_users.go:109` | CWE-200 |
| H3 | CORS wildcard with credential reflection | High | `internal/plugin/cors.go:28-36, 101-118` | CWE-942 |
| H4 | WebSocket OOM via unbounded frame allocation | High | `internal/graphql/subscription.go:364` | CWE-770 |
| H5 | YAML bomb (no depth/node limits) | High | `internal/pkg/yaml/decode.go:155, 179` | CWE-776 |
| H6 | Webhook SSRF (user-configurable URL) | High | `internal/admin/webhooks.go:154` | CWE-918 |
| H7 | Federation executor SSRF | High | `internal/federation/executor.go:313` | CWE-918 |
| H8 | Open redirect with query parameter leakage | High | `internal/plugin/redirect.go:59-60` | CWE-601 |
| H9 | Raw error messages to clients | High | `internal/gateway/server.go:997`, `internal/portal/handlers_api.go` | CWE-209 |
| H10 | No body size limits on Raft RPC | High | `internal/raft/transport.go:212-296` | CWE-770 |
| H11 | API keys exposed via admin GraphQL query | High | `internal/admin/graphql.go:366-380` | CWE-200 |
| H12 | RSA key size not validated (weak keys accepted) | High | `internal/pkg/jwt/rs256.go:63-72` | CWE-326 |
| H13 | Hardcoded default secrets in docker-compose files | High | `docker-compose.yml`, `standalone.yml`, `swarm.yml` | CWE-798 |
| H14 | Default Grafana admin/admin credentials | High | `docker-compose.yml`, `swarm.yml`, `monitoring/` | CWE-798 |
| H15 | Missing .gitignore patterns for secrets | High | `.gitignore` | CWE-359 |
| H16 | Placeholder secrets in K8s base manifest | High | `kubernetes/base/secret.yaml` | CWE-798 |

---

## Confirmed MEDIUM (22)

| ID | Finding | Confidence | File |
|----|---------|------------|------|
| M1 | Webhook retry goroutine leak | High | `internal/admin/webhooks.go:228` |
| M2 | Proxy retry ignores context cancellation | High | `internal/gateway/server.go:556` |
| M3 | Unbounded weighted LB expansion | High | `internal/gateway/balancer.go:182-198` |
| M4 | Password reset lacks minimum length | High | `internal/admin/admin_users.go:236` |
| M5 | Admin API CSRF gap for cookie auth | Medium | `internal/admin/server.go:123` |
| M6 | JWT in WebSocket query param (logged) | High | `internal/admin/ws.go:129` |
| M7 | Permission bypass on empty consumer ID | High | `internal/plugin/endpoint_permission.go:62-66` |
| M8 | Router regex ReDoS | Medium | `internal/gateway/router.go:616-625` |
| M9 | TLS 1.0/1.1 allowed | High | `internal/gateway/tls.go:100-112` |
| M10 | Weak TLS cipher suites | High | `internal/gateway/tls.go:123-129` |
| M11 | Session cookie Secure flag conditional | High | `internal/admin/token.go:184-195` |
| M12 | API key in query parameters | High | `internal/plugin/auth_apikey.go:230-234` |
| M13 | Webhook secret in plaintext response | High | `internal/admin/webhooks.go:683-687` |
| M14 | Non-constant-time Raft auth comparison | High | `internal/raft/cluster.go:88-97` |
| M15 | JWT in response body + cookie | High | `internal/admin/token.go:197-201` |
| M16 | No default mask fields for audit | High | `internal/audit/masker.go:79-87` |
| M17 | HS256 no entropy validation | High | `internal/pkg/jwt/hs256.go:16-23` |
| M18 | Env override can overwrite secrets | High | `internal/config/env.go:12-46` |
| M19 | TLS SkipVerify option in config | High | `internal/config/types.go:147` |
| M20 | CI actions pinned to @master | High | `.github/workflows/ci.yml:323,338` |
| M21 | Redis password in health check CLI | High | `docker-compose.prod.yml:122` |
| M22 | Secrets as local plaintext files | High | `docker-compose.prod.yml:254-261` |

---

## Confirmed LOW (18) → 18 RESOLVED

| ID | Finding | Status | File |
|----|---------|--------|------|
| ~~L1~~ | Analytics map growth between cleanups | **RESOLVED** | `internal/analytics/engine.go:19` |
| ~~L2~~ | 640MB buffer pool at startup | **RESOLVED** | `internal/gateway/optimized_proxy.go:233-236` |
| ~~L3~~ | Request body doubling in GetBody | **RESOLVED** | `internal/audit/capture.go:161` |
| ~~L4~~ | GraphQL 50MB response limit generous | **RESOLVED** | `internal/graphql/proxy.go:98` |
| ~~L5~~ | Custom JWT library unaudited | **RESOLVED** | `internal/pkg/jwt/` — migrated to `github.com/golang-jwt/jwt/v5` (audited) |
| ~~L6~~ | Custom YAML parser unfuzzed | **RESOLVED** | `internal/pkg/yaml/` — migrated to `gopkg.in/yaml.v3` (audited) + bomb protection |
| ~~L7~~ | WASM plugin arbitrary code execution | **RESOLVED** | `internal/plugin/wasm.go` — module size/path validation, execution timeout, capability restrictions |
| ~~L8~~ | Go version mismatch (1.25 vs 1.26) | **RESOLVED** | `go.mod` — updated to 1.26.0 |
| ~~L9~~ | API key modulo bias | **RESOLVED** | `internal/store/api_key_repo.go:357-377` — rejection sampling |
| ~~L10~~ | Password gen modulo bias + fallback | **RESOLVED** | `internal/store/user_repo.go:519-539` — rejection sampling, no fallback |
| ~~L11~~ | Proxy copies all upstream headers | **RESOLVED** | `internal/gateway/proxy.go:412-432` — internal header stripping |
| ~~L12~~ | Config import temp file in shared dir | **RESOLVED** | `internal/admin/server.go:353-364` — restricted temp dir via `APICERBERUS_TMPDIR` |
| ~~L13~~ | Deprecated golang.org/x/net/websocket | **RESOLVED** | `internal/federation/executor.go` — migrated to `nhooyr.io/websocket` |
| ~~L14~~ | CI govulncheck not version-pinned | **RESOLVED** | `.github/workflows/ci.yml:350` — pinned to `@v1.1.4` |
| ~~L15~~ | Deprecated GitHub Actions in release | **RESOLVED** | `.github/workflows/release.yml` — migrated to `softprops/action-gh-release@v1` |
| ~~L16~~ | K8s validation errors suppressed | **RESOLVED** | `.github/workflows/ci.yml:441` — removed `|| true` |
| ~~L17~~ | cAdvisor privileged container | **RESOLVED** | `deployments/monitoring/docker-compose.yml:144` — removed `privileged: true` |
| ~~L18~~ | Helm TLS disabled by default | **RESOLVED** | `deployments/helm/apicerberus/values.yaml:44-53` — TLS + cert-manager enabled |

---

## Ruled Out (False Positives / Not Applicable)

| Category | Result | Notes |
|----------|--------|-------|
| SQL Injection | **None found** | All store repositories use parameterized queries with `?` placeholders |
| Command Injection | **None found** | No `os/exec` usage with user input anywhere in codebase |
| LDAP Injection | **Not applicable** | No LDAP integration exists |
| XXE | **Not applicable** | No XML parsing (`encoding/xml`) found |
| Deserialization (Go-specific) | **None critical** | JSON unmarshaling into structs is standard Go pattern |
| `math/rand` for secrets | **Acceptable** | Only used for analytics sampling and election jitter — both `#nosec` annotated and correct |

---

## Findings by Attack Surface

### Network-Reachable (External Attackers)
- C1: SSRF via upstream URL
- C4: SSTI via webhook templates
- H3: CORS credential reflection
- H4: WebSocket OOM
- H5: YAML bomb via config import
- H6: Webhook SSRF
- H7: Federation SSRF
- H8: Open redirect
- H9: Error info leakage
- M6, M12: Secrets in query parameters
- M8: ReDoS via router regex
- M9, M10: TLS weaknesses

### Authenticated Admin
- C2: Cache poisoning
- C3: Unauthenticated Raft RPC
- C6: Config export leaks secrets
- H1: Auth bypass
- H2: Password hash exposure
- H10: Raft body limits
- H11: API key exposure via GraphQL
- H12: Weak RSA key acceptance
- M4: Password reset weakness
- M5: CSRF gap
- M7: Permission bypass
- M13: Secret in response
- M14: Timing attack on Raft auth
- M15: JWT in response body
- M17: HS256 entropy

### Deployment/Infrastructure
- C5: Password in stderr logs
- H13: Hardcoded compose secrets
- H14: Default Grafana creds
- H15: Missing .gitignore
- H16: K8s placeholder secrets
- M20: CI actions @master
- M21: Redis password in CLI
- M22: Secrets as local files
- I6-I19: K8s, Helm, CI, Makefile patterns

### Concurrency/Resource
- ~~M1~~: Goroutine leak in webhooks — **RESOLVED**
- ~~M2~~: Context-ignoring retry — **RESOLVED**
- ~~M3~~: Unbounded LB weights — **RESOLVED**
- ~~L1~~: Analytics map growth — **RESOLVED**
- ~~L2~~: Large buffer pool — **RESOLVED**
- ~~L3~~: Body doubling — **RESOLVED**

---

## Remediation Summary

| Severity | Total | Resolved | Remaining | Resolution Rate |
|----------|-------|----------|-----------|-----------------|
| CRITICAL | 6 | 6 | 0 | 100% |
| HIGH | 16 | 16 | 0 | 100% |
| MEDIUM | 22 | 22 | 0 | 100% |
| LOW | 18 | 18 | 0 | 100% |
| **TOTAL** | **62** | **62** | **0** | **100%** |
