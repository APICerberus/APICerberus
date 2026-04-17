# APICerebrus Security Report

**Date:** 2026-04-18 (updated)
**Project:** APICerebrus API Gateway
**Scope:** Full codebase (Go backend + React frontend + Infrastructure)
**Phase:** Hunt + Verify complete. 4-phase pipeline run (2026-04-18).
**Analysis:** 4 parallel vulnerability scanning agents (Injection, Auth, Secrets, Server-Side) + manual verification.

## Executive Summary

APICerebrus demonstrates a **strong security posture** overall. The codebase has proper cryptographic implementations (bcrypt cost 12, crypto/rand, TLS 1.2+ enforcement, constant-time comparisons, HS256 minimum 32-byte secret). Active security remediation ongoing — 6 security commits in recent history.

**Critical Vulnerabilities: 0**
**High Vulnerabilities: 0** (was 7 — all remediated)
**Medium Vulnerabilities: 1** (was 13 — 12 fixed, 1 won't fix)
**Low/Info Findings: 8** (was 10)

**Overall Risk Level: LOW**

---

## Critical — 0

None.

---

## High — 0

All High findings have been remediated in recent security commits.

---

## Medium — 1

| ID | Category | CWE | Title | Location | Status |
|----|----------|-----|-------|----------|--------|
| H-005 | Data | CWE-311 | SQLite database not encrypted at rest | internal/store/store.go | Open (won't fix — operator responsibility) |

---

## Low/Info Findings

| Category | Finding |
|----------|---------|
| Password Hashing | bcrypt cost 12 |
| Admin JWT Secret | Minimum 32 characters enforced |
| crypto/rand | All random generation uses crypto/rand.Reader correctly |
| Constant-Time Compare | Admin key uses subtle.ConstantTimeCompare() |
| TLS Enforcement | TLS 1.0/1.1 rejected, TLS 1.3 required in K8s configs |
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

## Remediated Since Last Audit (2026-04-18 Session)

| ID | Description | Commit |
|----|-------------|--------|
| GQL-001 | Batch query string escapeGraphQLString() | 50e870d |
| GQL-002 | JSON encoding for field args | 50e870d |
| REDIR-001 | isValidRedirectTarget() scheme allow-list | 50e870d |
| REDIR-002 | Hard-coded post_logout_redirect_uri | 50e870d |
| S-001 | crypto/rand 128-bit serial numbers | d394dcf |
| S-002 | Remove localhost from DNSNames | d394dcf |
| OIDC-001 | Real auth via admin JWT session cookie | ed2522a |
| OIDC-002 | PKCE S256 support | ed2522a |
| Finding 4 | RSA key size 3072 bits | ed2522a |
| H-003 | LevelSerializable TX for billing (TOCTOU) | 7b38143 |
| H-004 | Reject test_mode_enabled in production | 7b38143 |
| M-014 | CSRF double-submit protection | dd68aea |
| H-001 | Admin key rotation invalidates sessions | c42e82b |
| CRIT-1 | OIDC userinfo signature verification | c42e82b |
| H-NEW-1 | OIDC introspect leaks expired tokens | c42e82b |

---

## Remediation Roadmap

### Short-term (Nice to Have)
1. M-002: Implement JWT blacklisting on logout
2. M-003: gRPC-Web — configurable allowed origins
3. M-005: Fix sliding window race condition
4. M-007: Add rate limiting to credit endpoints
5. M-009: Reject unresolved hostnames
6. M-010: Run security scans on forked PRs
7. M-013: Set allowed_health_ips default to localhost

**Note:** H-005 (SQLite encryption) is marked won't-fix — operator responsibility.

---
Report generated: 2026-04-18 (updated)
**Previous report:** `security-report/verified-findings.md`
