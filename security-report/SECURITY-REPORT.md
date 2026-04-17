# APICerebrus Security Report

**Date:** 2026-04-17
**Project:** APICerebrus API Gateway
**Scope:** Full codebase (Go backend + React frontend + Infrastructure)
**Phase:** Complete — Recon -> Hunt -> Verify -> Report
**Analysis:** 10 parallel vulnerability scanning agents across 48 security skills

## Executive Summary

APICerebrus demonstrates a **strong security posture** overall. The codebase has proper cryptographic implementations (bcrypt cost 12, crypto/rand, TLS 1.2+ enforcement, constant-time comparisons, HS256 minimum 32-byte secret). Recent commits show active security remediation (5 security commits in recent history addressing WASM panic recovery, GraphQL auth, subscription origin validation, and config import).

**Critical Vulnerabilities: 0**
**High Vulnerabilities: 5**
**Medium Vulnerabilities: 14**
**Low/Info Findings: 23**

**Overall Risk Level: MEDIUM**

---

## High — 5 Findings

| ID | Category | CWE | Title | Location |
|----|----------|-----|-------|----------|
| H-001 | Auth | CWE-287 | Admin key rotation does not revoke existing sessions | internal/admin/token.go:311-373 |
| H-002 | AuthZ | CWE-862 | Config import allows replacing admin credentials | internal/admin/server.go:427-482 |
| H-003 | Business Logic | CWE-362 | TOCTOU race condition in credit PreCheck vs Deduct | internal/billing/engine.go:92-192 |
| H-004 | Business Logic | CWE-284 | Test key bypass if test_mode_enabled accidentally set in production | internal/billing/engine.go:107 |
| H-005 | Data | CWE-311 | SQLite database not encrypted at rest | internal/store/store.go |

---

## Medium — 14 Findings

| ID | Category | CWE | Title | Location |
|----|----------|-----|-------|----------|
| M-001 | Secrets | CWE-798 | Admin API key has no minimum length validation | internal/config/load.go:314-321 |
| M-002 | Auth | CWE-613 | Logout does not invalidate JWT tokens | internal/admin/token.go:375-400 |
| M-003 | Auth | CWE-942 | gRPC-Web uses wildcard origin with credentials pass-through | internal/grpc/proxy.go:100,218 |
| M-004 | AuthZ | CWE-639 | EndpointPermission lacks IDOR validation | internal/plugin/endpoint_permission.go:55-167 |
| M-005 | Concurrency | CWE-362 | Sliding window rate limiter has race window | internal/ratelimit/sliding_window.go:57-63 |
| M-006 | Config | CWE-915 | Config import mass assignment | internal/admin/server.go:466-468 |
| M-007 | API | CWE-307 | Missing rate limiting on admin credit endpoints | internal/admin/server.go |
| M-008 | Rate Limit | CWE-346 | X-Forwarded-For spoofing when trusted_proxies misconfigured | internal/plugin/rate_limit.go:609-617 |
| M-009 | SSRF | CWE-918 | DNS resolution failure allows unresolved hostnames through | internal/gateway/proxy.go:333-337 |
| M-010 | Infra | CWE-1104 | Security scans skipped on pull requests from forks | .github/workflows/ci.yml:406 |
| M-011 | Infra | CWE-532 | Secrets passed as Helm --set arguments in CI | .github/workflows/ci.yml:491-492,568-569 |
| M-012 | Infra | CWE-285 | Production deployment requires manual approval gate | .github/workflows/ci.yml:528-536 |
| M-013 | Config | CWE-200 | Health endpoint exposes internal details by default | apicerberus.example.yaml:43-49 |
| M-014 | Frontend | CWE-79 | Auth state stored in sessionStorage (XSS exfiltration risk) | web/src/lib/api.ts:38-54 |

---

## Low / Info — 23 Findings

| ID | Category | CWE | Title | Location |
|----|----------|-----|-------|----------|
| L-001 | Secrets | CWE-532 | Generated admin password written to stderr | internal/store/user_repo.go:538-541 |
| L-002 | Crypto | CWE-327 | API key hash uses SHA-256, not password KDF | internal/store/api_key_repo.go:353-355 |
| L-003 | Crypto | CWE-330 | Raft CA certificate uses predictable serial numbers | internal/raft/tls.go:40,80 |
| L-004 | Crypto | CWE-326 | TLS 1.3 has no explicit cipher configuration | internal/gateway/tls.go:70-98 |
| L-005 | Data | CWE-201 | PII masking missing fields (ssn, bank_account, dob) | internal/audit/masker.go:17-25 |
| L-006 | Data | CWE-201 | user.metadata JSON field not masked | internal/audit/masker.go:22 |
| L-007 | Auth | CWE-1275 | OIDC cookies use SameSite=Lax instead of Strict | internal/admin/oidc.go:138,150 |
| L-008 | Auth | CWE-1275 | Session cookie SameSite inconsistency (OIDC vs static) | internal/admin/oidc.go:343 |
| L-009 | Session | CWE-770 | Rate limit cleanup never unblocks IPs permanently | internal/admin/server.go:87-88 |
| L-010 | AuthZ | CWE-362 | Config import has no atomic transaction boundary | internal/admin/server.go:466-472 |
| L-011 | WASM | CWE-739 | WASM module size hard cap 100MB | internal/plugin/wasm.go:23 |
| L-012 | WASM | CWE-78 | WASI instantiated only when AllowFilesystem=true | internal/plugin/wasm.go:108-113 |
| L-013 | WASM | CWE-111 | EnvVars field exists but not wired | internal/plugin/wasm.go:60-64 |
| L-014 | Error | CWE-391 | Multiple w.Write errors discarded | internal/admin/server.go:299,311,329... |
| L-015 | Concurrency | CWE-362 | LoadOrStore pattern in token_bucket and leaky_bucket | internal/ratelimit/token_bucket.go:56 |
| L-016 | SSRF | CWE-918 | Webhook URL validation missing private IP check | internal/admin/webhooks.go:711-741 |
| L-017 | CORS | CWE-346 | Gateway WebSocket proxy has no CORS headers | internal/gateway/proxy.go:161-265 |
| L-018 | Clickjack | CWE-693 | GraphQL endpoint missing clickjacking protection | internal/admin/graphql.go:876-880 |
| L-019 | Infra | CWE-1204 | Prometheus/Grafana images use :latest tag | docker-compose.yml:81,101,130... |
| L-020 | Infra | CWE-1204 | Kubernetes deployment uses :latest tag | deployments/kubernetes/base/deployment.yaml:39 |
| L-021 | Infra | CWE-284 | Network policy disabled by default in Helm | deployments/helm/apicerberus/values.yaml:215 |
| L-022 | Infra | CWE-311 | Portal session secure cookie disabled in Helm | deployments/helm/apicerberus/values.yaml:122 |
| L-023 | Frontend | CWE-79 | CSS custom property injection from server | web/src/components/layout/BrandingProvider.tsx:52 |

---

## Positive Security Findings

| Category | Finding |
|----------|---------|
| Password Hashing | bcrypt cost 12 |
| Admin JWT Secret | Minimum 32 characters enforced |
| crypto/rand | All random generation uses crypto/rand.Reader correctly |
| Constant-Time Compare | Admin key uses subtle.ConstantTimeCompare() |
| TLS Enforcement | TLS 1.0/1.1 rejected, TLS 1.2 minimum |
| Raft mTLS | TLS 1.3 minimum, client certs required |
| HttpOnly Cookies | Admin cookies set HttpOnly, Secure, SameSite=StrictMode |
| SQL Injection | All queries use parameterized placeholders |
| NoSQL Injection | Redis Lua scripts use KEYS/ARGV safely |
| XSS | No dangerouslySetInnerHTML, no innerHTML assignments |
| WASM Panic Recovery | SEC-WASM-003: defer recover() implemented |
| WASM Phase Validation | SEC-WASM-001/002: PhaseAuth and PhasePostProxy forbidden |
| Non-root Containers | All Docker images run as non-root |

---

## Dependency Audit

| Category | Status |
|----------|--------|
| Direct Dependencies | 23 |
| Indirect Dependencies | 27 |
| Known CVEs | 0 unpatched |
| License Compliance | CLEAN |
| Unofficial Modules | NONE |

---

## Remediated Since Last Audit

| ID | Description | Commit |
|----|-------------|--------|
| WASM-003 | Panic recovery in WASM Execute/Run/AfterProxy | 8787ce2 |
| GQL-011 | X-Admin-Key required on GET /sse | b9f221a |
| GQL-010 | Drop path arg from system.config.import | c9add9d |
| GQL-007 | Origin allow-list for subscription WS+SSE | 96d32aa |
| GQL-006 | @authorized enforced at execution time | 1ea67fa |

---

## Remediation Roadmap

### Immediate (High)
1. H-001: Implement JWT token revocation on admin key rotation
2. H-002: Add field allowlisting to config import
3. H-003: Use SELECT FOR UPDATE for atomic billing
4. H-004: Reject test_mode_enabled in production
5. H-005: Document SQLite access controls

### Short-term (Medium)
6. M-001: Add admin key minimum length validation
7. M-002: Implement JWT blacklisting on logout
8. M-003: gRPC-Web — configurable allowed origins
9. M-005: Fix sliding window race condition
10. M-007: Add rate limiting to credit endpoints
11. M-009: Reject unresolved hostnames
12. M-010: Run security scans on forked PRs
13. M-013: Set allowed_health_ips default to localhost

---
Report generated: 2026-04-17
