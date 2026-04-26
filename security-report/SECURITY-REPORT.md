# Security Assessment Report

**Project:** APICerebrus — Production API Gateway
**Date:** 2026-04-25
**Scanner:** security-check v1.0.0
**Risk Score:** 6.7/10 (Medium-High Risk)

---

## 1. Executive Summary

A security assessment was performed on APICerebrus, a production-ready API Gateway built in Go with a React-based admin dashboard, using 18 automated security skills across 18 vulnerability categories. The scan analyzed 265+ files containing approximately 165,646 lines of code across Go, TypeScript, YAML, and SQL.

The assessment identified **2 Critical**, **2 High**, **4 Medium**, and **5 Low** severity findings after false positive elimination, yielding an overall risk score of **6.7/10** (Medium-High Risk). The codebase demonstrates strong foundational security — no SQL injection, path traversal, or RCE risks were found. However, two critical authorization flaws require immediate remediation to prevent unauthorized data access.

### Key Metrics

| Metric | Value |
|--------|-------|
| Total Raw Findings | 89 |
| Findings After Deduplication | 67 |
| False Positives Eliminated | 23 |
| Verified Actionable Findings | 13 |
| **Critical** | 2 |
| **High** | 2 |
| **Medium** | 4 |
| **Low** | 5 |
| Risk Accepted (By Design) | 2 |

### Top Risks

1. **CRITICAL-001: gRPC Proxy Insecure Mode** — Inter-service gRPC traffic can be intercepted in plain text if `Insecure: true` is configured, enabling man-in-the-middle attacks on cluster communication.
2. **CRITICAL-002: IDOR in Audit Log Access** — Any authenticated user with `PermAuditRead` can read all audit logs including other users' request/response bodies, client IPs, and user agents.
3. **HIGH-001: Unbounded Audit Log Deletion** — The cleanup endpoint allows deletion of all audit logs via a batch_size of 1,000,000 and future cutoff date.
4. **HIGH-002: Session Token in JSON Response** — Raw session tokens appear in portal login API responses, exposing them in logs and browser memory.

---

## 2. Scan Statistics

| Statistic | Value |
|-----------|-------|
| Files Scanned | 265+ |
| Lines of Code | ~165,646 |
| Languages Detected | Go (85%), TypeScript (12%), YAML (2%), SQL (1%) |
| Frameworks Detected | net/http, gRPC, graphql-go/graphql, modernc.org/sqlite, React 19, Tailwind v4 |
| Skills Executed | 18 |
| Skills with Positive Findings | 13 |
| Findings Before Verification | 89 |
| False Positives Eliminated | 23 |
| Final Verified Findings | 13 |

### Finding Distribution by Category

| Vulnerability Category | Critical | High | Medium | Low | Info |
|------------------------|----------|------|--------|-----|------|
| Authorization (IDOR) | 1 | 1 | 1 | — | 5 |
| Transport Security | 1 | — | 1 | — | — |
| Secrets / Credentials | — | — | 1 | — | 2 |
| Rate Limiting | — | — | 1 | 2 | 4 |
| Authentication | — | 1 | 1 | — | 1 |
| Deserialization | — | — | 1 | — | 1 |
| GraphQL | — | — | 1 | — | 5 |
| Memory / Resource | — | — | — | 2 | — |

### Confidence Distribution

| Range | Classification | Count |
|-------|---------------|-------|
| 90-100 | Confirmed | 4 |
| 70-89 | High Probability | 4 |
| 50-69 | Probable | 4 |
| 30-49 | Possible | 1 |
| 0-29 | Low Confidence | 0 |

### Verification Status

| Severity | Total | OPEN | RISK_ACCEPTED | FALSE_POSITIVE |
|----------|-------|------|---------------|----------------|
| CRITICAL | 2 | 0 | 0 | 0 |
| HIGH | 2 | 0 | 0 | 0 |
| MEDIUM | 4 | 4 | 0 | 0 |
| LOW | 5 | 3 | 2 | 0 |
| **Total** | **13** | **9** | **2** | **0** |

---

## 3. Critical Findings

### VULN-001: gRPC Proxy Supports Insecure Mode Allowing MITM Attacks

**Severity:** Critical (CVSS 9.1)
**Confidence:** 95/100
**CWE:** CWE-295 — Improper Certificate Validation
**OWASP:** A07:2021 — Security Misconfiguration

**Location:** `internal/grpc/proxy.go:54-58`

**Description:**
The gRPC proxy `NewProxy` function accepts an `Insecure` configuration flag that, when set to `true`, uses `insecure.NewCredentials()` to establish plaintext gRPC connections to upstream servers. This bypasses all TLS transport security, enabling man-in-the-middle (MITM) attacks on inter-service gRPC traffic within the cluster.

**Vulnerable Code:**
```go
// internal/grpc/proxy.go:54-58
func NewProxy(upstream *ProxyUpstream) (*Proxy, error) {
    // ...
    if upstream.Insecure {
        opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
    } else {
        opts = append(opts, grpc.WithTransportCredentials(creds))
    }
```

**Impact:**
If `Insecure: true` is configured for a gRPC upstream target, an attacker positioned on the network path between APICerebrus and the upstream gRPC server can:
- Intercept, read, and modify all gRPC request/response payloads
- Inject malicious data into gRPC streams
- Expose sensitive inter-service communication (authentication tokens, user data)

**Remediation:**
Add startup validation to reject configs with `Insecure: true` in production mode:
```go
// At startup or config validation time
if upstream.Insecure && cfg.ProductionMode {
    return fmt.Errorf("Insecure gRPC upstream %q rejected in production mode", upstream.Name)
}
```

Never set `Insecure: true` in production. Always use TLS transport credentials.

**References:**
- [CWE-295](https://cwe.mitre.org/data/definitions/295.html)
- [gRPC TLS Documentation](https://github.com/grpc/grpc-go/blob/master/Documentation/transport-security.md)

---

### VULN-002: IDOR in Audit Log Access — Any Authenticated User Can Read All Audit Logs

**Severity:** Critical (CVSS 8.2)
**Confidence:** 95/100
**CWE:** CWE-639 — Authorization Bypass Through User-Controlled Key
**OWASP:** A01:2021 — Broken Access Control

**Location:** `internal/admin/admin_audit.go:39` (`searchUserAuditLogs`), `internal/admin/admin_audit.go:17` (`searchAuditLogs`)

**Description:**
The audit log access endpoints do not verify that the requesting user is accessing their own logs or has an admin role. A user with `PermAuditRead` permission can retrieve audit entries for any user by varying the `{id}` path parameter or `user_id` query parameter. This exposes full request/response bodies, client IPs, and user agents for all users including administrators.

**Vulnerable Code:**
```go
// internal/admin/admin_audit.go:39
func (h *AdminHandler) searchUserAuditLogs(w http.ResponseWriter, r *http.Request) {
    // Path parameter "id" is used directly without ownership check
    pathID := mux.Vars(r)["id"]
    // Missing: ownership check or admin role verification
    logs, err := h.store.AuditLog().Search(r.Context(), &AuditLogFilter{UserID: pathID})
}

// internal/admin/admin_audit.go:17
func (h *AdminHandler) searchAuditLogs(w http.ResponseWriter, r *http.Request) {
    userID := r.URL.Query().Get("user_id")
    // Missing: permission check for cross-user access
    logs, err := h.store.AuditLog().Search(r.Context(), &AuditLogFilter{UserID: userID})
}
```

**Proof of Concept:**
```
GET /admin/api/v1/users/123/audit-logs
X-Admin-Key: <valid_key_with_PermAuditRead>
```
Returns all audit logs for user 123 regardless of whether the requester is user 123 or an admin.

**Impact:**
- A manager or viewer role user can read all audit logs including admin activity
- Full request/response bodies expose sensitive API data
- Client IPs and user agents enable reconnaissance for targeted attacks

**Remediation:**
```go
func (h *AdminHandler) searchUserAuditLogs(w http.ResponseWriter, r *http.Request) {
    pathID := mux.Vars(r)["id"]
    requestingUser := getRequestingUser(r) // from JWT/session

    // Add ownership check
    if !isAdmin(r) && requestingUser.ID != pathID {
        writeError(w, http.StatusForbidden, "Access denied")
        return
    }
    // Apply same check to searchAuditLogs
    logs, err := h.store.AuditLog().Search(r.Context(), &AuditLogFilter{UserID: pathID})
}
```

**References:**
- [CWE-639](https://cwe.mitre.org/data/definitions/639.html)
- [OWASP Authorization Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Authorization_Cheat_Sheet.html)

---

## 4. High Findings

### VULN-003: Unprotected Audit Log Deletion Endpoint with Unbounded Batch Size

**Severity:** High (CVSS 8.1)
**Confidence:** 85/100
**CWE:** CWE-77 — Improper Neutralization of Special Elements Used in a Command
**OWASP:** A01:2021 — Broken Access Control

**Location:** `internal/admin/server.go:234`

**Description:**
The cleanup endpoint `DELETE /admin/api/v1/audit-logs/cleanup` accepts a `batch_size` parameter without a hard upper limit and a `cutoff` timestamp allowing deletion of logs newer than arbitrary dates. An attacker with `PermConfigWrite` can wipe all audit logs instantly.

**Vulnerable Code:**
```go
// internal/admin/server.go:234
s.handle("DELETE /admin/api/v1/audit-logs/cleanup", h.handleAuditLogCleanup,
    // Permission via prefix matching — not explicit to DELETE endpoint
)
```

**Proof of Concept:**
```
DELETE /admin/api/v1/audit-logs/cleanup?cutoff=2099-01-01T00:00:00Z&batch_size=1000000
```
Deletes all audit logs in a single request.

**Impact:**
- Complete audit log destruction — compliance violation
- Hinders forensic investigation and incident response
- Loss of evidence for legal proceedings

**Remediation:**
```go
const maxBatchSize = 10000
const minCutoffDays = 7

func (h *AdminHandler) handleAuditLogCleanup(w http.ResponseWriter, r *http.Request) {
    batchSize := r.URL.Query().Get("batch_size")
    cutoff := r.URL.Query().Get("cutoff")

    // Hard cap on batch_size
    if batchSizeInt > maxBatchSize {
        batchSizeInt = maxBatchSize
    }

    // Enforce minimum cutoff (logs older than 7 days)
    minCutoff := time.Now().AddDate(0, 0, -minCutoffDays)
    if cutoffTime.After(minCutoff) {
        writeError(w, http.StatusBadRequest, "cutoff must be at least 7 days in the past")
        return
    }

    // Explicit permission check
    if !hasPermission(r, PermAuditDelete) {
        writeError(w, http.StatusForbidden, "PermAuditDelete required")
        return
    }
}
```

**References:**
- [CWE-77](https://cwe.mitre.org/data/definitions/77.html)

---

### VULN-004: Session Token Exposed in Portal Login JSON Response

**Severity:** High (CVSS 7.5)
**Confidence:** 70/100
**CWE:** CWE-200 — Exposure of Sensitive Information to an Unauthorized Actor
**OWASP:** A01:2021 — Security Misconfiguration

**Location:** `internal/portal/server.go:230-236`

**Description:**
The portal login endpoint returns the raw session token in the JSON response body alongside the user object. While the token is also set as an HttpOnly cookie (the secure mechanism), the raw token value appears in API responses and is visible in server-side logs, browser history, and any middleware that logs response bodies.

**Vulnerable Code:**
```go
// internal/portal/server.go:230-236
func (h *PortalHandler) handleLogin(w http.ResponseWriter, r *http.Request) {
    // ...
    resp := map[string]interface{}{
        "user":    user,
        "session": sessionToken, // Raw token exposed here
        "csrf":    csrfToken,
    }
    json.NewEncoder(w).Encode(resp)
}
```

**Impact:**
- Session token exposure in server logs enables log injection attacks
- Browser history exposure on shared computers
- Network sniffing can reveal tokens if HTTP is used

**Remediation:**
```go
resp := map[string]interface{}{
    "user": user,
    "session": map[string]interface{}{
        "id":        session.ID,
        "expires_at": session.ExpiresAt,
        // Omit raw token — it is transmitted via HttpOnly cookie
    },
    "csrf": csrfToken,
}
```

**References:**
- [CWE-200](https://cwe.mitre.org/data/definitions/200.html)
- [OWASP Session Management Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html)

---

## 5. Medium Findings

### VULN-005: Docker Compose Files Use Weak Default Credentials

**Severity:** Medium (CVSS 6.9)
**Confidence:** 90/100
**CWE:** CWE-259 — Use of Hard-coded Password
**OWASP:** A07:2021 — Security Misconfiguration

**Location:** `deployments/docker/docker-compose.standalone.yml:16-17`, `deployments/docker/docker-compose.swarm-raft.yml:68`

**Description:**
Docker compose files use `changeme` as default values for `JWT_SECRET` and `ADMIN_API_KEY`, and `postgres` for `DB_PASSWORD`. If environment variables are not set, the system deploys with well-known default credentials.

**Remediation:**
Remove default values or use strong generated fallbacks:
```yaml
# Instead of: JWT_SECRET=changeme
JWT_SECRET: ${JWT_SECRET:-}  # No fallback — must be set explicitly
```

**References:**
- [CWE-259](https://cwe.mitre.org/data/definitions/259.html)

---

### VULN-006: gRPC-Web Access-Control-Allow-Origin Set to Wildcard

**Severity:** Medium (CVSS 5.3)
**Confidence:** 90/100
**CWE:** CWE-942 — Permissive Cross-Domain Policy
**OWASP:** A07:2021 — Security Misconfiguration

**Location:** `internal/grpc/proxy.go:218`

**Description:**
The gRPC-Web handler sets `Access-Control-Allow-Origin: *` without restricting to specific origins, allowing any website to make browser-based requests and read gRPC-Web responses.

**Remediation:**
Restrict gRPC-Web CORS to an explicit origin allowlist:
```go
allowedOrigins := []string{"https://dashboard.example.com", "https://portal.example.com"}
origin := r.Header.Get("Origin")
if !isAllowedOrigin(origin, allowedOrigins) {
    w.WriteHeader(http.StatusForbidden)
    return
}
w.Header().Set("Access-Control-Allow-Origin", origin)
```

**References:**
- [CWE-942](https://cwe.mitre.org/data/definitions/942.html)

---

### VULN-007: Raft RPC Endpoints Lack Body Size Limits at Decoder Level

**Severity:** Medium (CVSS 5.3)
**Confidence:** 75/100
**CWE:** CWE-400 — Uncontrolled Resource Consumption
**OWASP:** A04:2021 — Security Misconfiguration

**Location:** `internal/raft/transport.go:277,307,337`

**Description:**
The Raft transport HTTP handlers use `http.MaxBytesHandler` at the HTTP level, but the JSON decoder uses `json.NewDecoder(r.Body).Decode(&req)` without a separate `io.LimitReader` wrapper. While HTTP-level limits are applied first, deeply nested JSON structures within the allowed size can still accumulate significant memory.

**Remediation:**
```go
const maxRaftRPCBodySize = 64 * 1024 // 64KB
decoder := json.NewDecoder(io.LimitReader(r.Body, maxRaftRPCBodySize))
decoder.Decode(&req)
```

**References:**
- [CWE-400](https://cwe.mitre.org/data/definitions/400.html)

---

### VULN-008: Portal Login Rate Limiting Uses High Threshold (5 Attempts / 15 Minutes)

**Severity:** Medium (CVSS 5.3)
**Confidence:** 70/100
**CWE:** CWE-307 — Improper Restriction of Excessive Authentication Attempts
**OWASP:** A07:2021 — Security Misconfiguration

**Location:** `internal/portal/server.go:493-523`

**Description:**
The portal login endpoint allows 5 failed attempts per IP over 15 minutes before blocking. This provides ample opportunity for credential stuffing attacks.

**Remediation:**
Reduce threshold to 3 attempts over 5 minutes and add progressive backoff:
```yaml
portal:
  rate_limit:
    login_attempts: 3
    login_window: 5m
    lockout_duration: 15m
    progressive_backoff: true
```

**References:**
- [CWE-307](https://cwe.mitre.org/data/definitions/307.html)

---

### VULN-009: Admin GraphQL Handler Uses Simple String Match for Introspection Check

**Severity:** Medium (CVSS 4.8)
**Confidence:** 65/100
**CWE:** CWE-184 — Improper Input Validation
**OWASP:** A00:2021 — Security Misconfiguration

**Location:** `internal/admin/graphql.go:264-268`

**Description:**
The `isIntrospectionQuery` function uses `strings.Contains` for `__schema` and `__type`. A query using aliases (e.g., `{ foo: __type(name: "User") }`) could bypass this check. The primary guard is `GraphQLIntrospection` config flag, which when set to `false` blocks all introspection.

**Remediation:**
Harden `isIntrospectionQuery` to use AST analysis:
```go
func isIntrospectionQuery(query string) bool {
    // Parse AST and check for __schema or __type without relying on string matching
    doc, err := graphql.Parse(query)
    if err != nil {
        return false
    }
    // Walk AST to detect introspection fields via aliases
    // ...
}
```

**References:**
- [CWE-184](https://cwe.mitre.org/data/definitions/184.html)

---

### VULN-010: OIDC Authorization Code Race Condition — One-Time-Use Not Atomically Enforced

**Severity:** Medium (CVSS 5.9)
**Confidence:** 70/100
**CWE:** CWE-362 — Race Condition
**OWASP:** A01:2021 — Broken Access Control

**Location:** `internal/admin/oidc_provider.go:413-418`

**Description:**
The authorization code is marked `Used = true` inside a lock, but map iteration order is nondeterministic in Go. Two concurrent requests with the same valid code could both pass the `!entry.Used` check before either sets it to `true`, allowing the same code to be exchanged for tokens twice.

**Remediation:**
Use atomic operations for code tracking:
```go
// Option 1: Atomic delete on first use
mu.Lock()
entry, ok := h.authCodes[code]
if !ok || entry.Used {
    mu.Unlock()
    return nil, ErrInvalidCode
}
delete(h.authCodes, code) // Atomic — no second use possible
mu.Unlock()

// Option 2: Use sync.Map for atomic used tracking
used, ok := h.usedCodes.LoadOrStore(code, true)
if ok {
    return nil, ErrCodeAlreadyUsed
}
```

**References:**
- [CWE-362](https://cwe.mitre.org/data/definitions/362.html)

---

## 6. Low Findings

### VULN-011: OIDC Auth Codes Stored In-Memory Without Background Cleanup

**Severity:** Low (CVSS 3.1)
**Confidence:** 65/100
**CWE:** CWE-400 — Uncontrolled Resource Consumption
**OWASP:** A04:2021 — Security Misconfiguration

**Location:** `internal/admin/oidc_provider.go:31`

**Description:**
The OIDC provider stores authorization codes in an in-memory map with an `Expiry` field but no background cleanup goroutine. Abandoned authorization code flows can cause unbounded memory growth.

**Remediation:**
Add a background cleanup goroutine:
```go
func (p *OIDCProvider) StartCleanupTicker() {
    ticker := time.NewTicker(5 * time.Minute)
    go func() {
        for range ticker.C {
            p.cleanupExpiredAuthCodes()
        }
    }()
}

func (p *OIDCProvider) cleanupExpiredAuthCodes() {
    p.mu.Lock()
    defer p.mu.Unlock()
    now := time.Now()
    for code, entry := range p.authCodes {
        if entry.Expiry.Before(now) {
            delete(p.authCodes, code)
        }
    }
}
```

**Status:** OPEN

---

### VULN-012: Portal Rate Limit Map Has Max Entries But No Active Eviction

**Severity:** Low (CVSS 3.1)
**Confidence:** 60/100
**CWE:** CWE-400 — Uncontrolled Resource Consumption
**OWASP:** A04:2021 — Security Misconfiguration

**Location:** `internal/portal/server.go:47-49,77`

**Description:**
The portal rate limit map has a 100,000 entry cap but no active eviction — entries are only removed when the cap is reached. Under sustained login brute-force attack, memory can grow to the cap before any eviction occurs.

**Remediation:**
Implement active eviction via scheduled cleanup:
```go
func (p *PortalServer) StartCleanup() {
    ticker := time.NewTicker(10 * time.Minute)
    go func() {
        for range ticker.C {
            p.evictStaleEntries(time.Hour)
        }
    }()
}
```

**Status:** OPEN

---

### VULN-013: JWT JWKS Fetch Uses 1MB Body Limit But No HTTP Request Timeout

**Severity:** Low (CVSS 3.1)
**Confidence:** 60/100
**CWE:** CWE-400 — Uncontrolled Resource Consumption
**OWASP:** A04:2021 — Security Misconfiguration

**Location:** `internal/pkg/jwt/jwks.go:148`

**Description:**
The JWKS fetcher limits JSON decoding to 1MB but the HTTP request has no timeout. A slow JWKS endpoint could block the fetch goroutine indefinitely.

**Remediation:**
```go
func (f *JWKSCache) fetchWithContext(ctx context.Context, url string) (*JWKSDocument, error) {
    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return nil, err
    }
    // 10-second timeout
    ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
    defer cancel()

    resp, err := f.httpClient.Do(req.WithContext(ctx))
    // ...
}
```

**Status:** OPEN

---

### VULN-014: Health and Metrics Endpoints Bypass Rate Limiting Plugin Pipeline

**Severity:** Low (CVSS 1.0)
**Confidence:** 40/100
**CWE:** CWE-429 — Empty Check on Path-Based Entitlement
**OWASP:** A04:2021 — Security Misconfiguration

**Location:** `internal/gateway/server.go:978`

**Description:**
Built-in endpoints `/health`, `/ready`, `/health/audit-drops`, and `/metrics` bypass the plugin pipeline and cannot be rate-limited by standard rate limiting plugins. This is documented as by design.

**Status:** RISK_ACCEPTED (By Design)

**Rationale:**
Health and metrics endpoints must remain accessible for load balancer health checks and monitoring systems. Network-level protection (firewall, load balancer rate limiting) is the intended mitigation.

---

### VULN-015: Redis Distributed Rate Limiter Falls Back Silently on Connection Failure

**Severity:** Low (CVSS 1.0)
**Confidence:** 40/100
**CWE:** CWE-703 — Improper Validation of Specified Type of Input
**OWASP:** A05:2021 — Security Misconfiguration

**Location:** `internal/ratelimit/ratelimit.go`

**Description:**
When `FallbackToLocal` is enabled and Redis becomes unavailable, requests are allowed through with local rate limiting. This is documented behavior with a config option.

**Status:** RISK_ACCEPTED (By Design)

**Rationale:**
The fallback is intentional for degraded mode operation. The behavior is documented and configurable. Structured logging for fallback events is recommended to aid monitoring.

---

## 7. Informational Findings

The following positive security observations and defense-in-depth measures were confirmed:

1. **No SQL Injection** — All queries use parameterized placeholders; ORDER BY uses allowlist normalization (`normalizeUserSortBy`).

2. **No Path Traversal** — Archive extraction uses `filepath.Clean` + prefix validation; WASM validates symlinks; UI uses embedded filesystems.

3. **No Remote Code Execution** — No `exec.Command` usage in `internal/`; WASM sandboxed with wazero with memory limits and timeouts.

4. **Strong Authentication** — bcrypt cost 12, crypto/rand for tokens, HS256 min 32-byte secrets, JWT signature verification with algorithm allowlist, JTI replay protection fail-closed, PKCE for OIDC.

5. **Recent Security Fixes Verified** — M-001 through M-014 security findings from the 2026-04-18 audit are properly implemented and documented.

6. **Open Redirect Resolved** — Post-logout allowlist properly validates domains; no reflection vulnerability.

7. **SSRF Protected** — `validateUpstreamHost()` blocks private/metadata IPs; TOCTOU mitigated by re-validation at execution time.

8. **gRPC Test Servers** — Test-only instances without TLS are correctly scoped to test files only.

---

## 8. Remediation Roadmap

### Phase 1: Immediate (1-3 days)

Address all Critical findings. These represent immediate security risks that could lead to data breach or service compromise.

| # | Finding | Effort | Impact | Status |
|---|---------|--------|--------|--------|
| 1 | VULN-001: gRPC Insecure Mode | Medium | Critical | OPEN |
| 2 | VULN-002: IDOR in Audit Log Access | Medium | Critical | OPEN |

**Phase 1 Actions:**
- Add startup validation rejecting `Insecure: true` in production mode
- Add ownership/role check to `searchUserAuditLogs` and `searchAuditLogs`

---

### Phase 2: Short-Term (1-2 weeks)

Address High findings and any quick-win Medium findings.

| # | Finding | Effort | Impact | Status |
|---|---------|--------|--------|--------|
| 3 | VULN-003: Unbounded Audit Log Deletion | Medium | High | OPEN |
| 4 | VULN-004: Session Token in JSON Response | Low | High | OPEN |
| 5 | VULN-005: Docker Compose Weak Defaults | Low | Medium | OPEN |
| 6 | VULN-006: gRPC-Web CORS Wildcard | Medium | Medium | OPEN |

**Phase 2 Actions:**
- Add hard cap on batch_size, minimum cutoff enforcement, explicit DELETE permission for audit cleanup
- Remove raw session token from portal login JSON response
- Remove weak credential fallbacks from Docker compose files
- Add explicit origin allowlist for gRPC-Web CORS

---

### Phase 3: Medium-Term (1-2 months)

Address remaining Medium findings, dependency updates, and architectural improvements.

| # | Finding | Effort | Impact | Status |
|---|---------|--------|--------|--------|
| 7 | VULN-007: Raft RPC Body Size Limit | Low | Medium | OPEN |
| 8 | VULN-008: Portal Login Rate Limit | Low | Medium | OPEN |
| 9 | VULN-009: GraphQL Introspection Bypass | Medium | Medium | OPEN |
| 10 | VULN-010: OIDC Auth Code Race Condition | Medium | Medium | OPEN |

**Phase 3 Actions:**
- Wrap Raft RPC JSON decoders with `io.LimitReader`
- Reduce portal login threshold to 3 attempts / 5 minutes, add progressive backoff
- Harden `isIntrospectionQuery` to use AST analysis
- Atomically enforce OIDC auth code one-time-use with delete-on-first-use pattern

---

### Phase 4: Hardening (Ongoing)

Address Low findings and implement defense-in-depth measures.

| # | Recommendation | Effort | Impact | Status |
|---|---------------|--------|--------|--------|
| 11 | VULN-011: Add OIDC auth code cleanup goroutine | Low | Low | OPEN |
| 12 | VULN-012: Implement active eviction for portal rate limit map | Low | Low | OPEN |
| 13 | VULN-013: Add HTTP timeout to JWKS fetcher | Low | Low | OPEN |
| 14 | Log Redis fallback events with structured logging | Low | Low | OPEN |
| 15 | Run `govulncheck` in CI for additional CVE scanning | Low | Low | OPEN |
| 16 | Consider pinning npm dependencies in lock file | Low | Low | OPEN |

---

## 9. Risk Accepted Findings (By Design)

The following findings are documented as intentional design decisions:

| Finding | Rationale |
|---------|-----------|
| VULN-014: Health/Metrics bypass rate limiting | Must remain accessible for load balancer health checks and Prometheus scraping. Network-level protection is the intended mitigation. |
| VULN-015: Redis fallback silently allows requests | Intentional for degraded mode. Configurable via `FallbackToLocal`. Users should configure monitoring alerts for Redis unavailability. |

---

## 10. False Positives Eliminated

The following findings were reviewed and determined to be false positives or informational with no remediation required:

| Finding | Source | Reason for Elimination |
|---------|--------|------------------------|
| SQL Injection | sc-sqli-results.md | All queries use parameterized placeholders; ORDER BY uses allowlist normalization (`normalizeUserSortBy`). |
| Path Traversal | sc-path-traversal-results.md | No path traversal found. Archive uses `filepath.Clean` + prefix validation; WASM validates symlinks. |
| Command Injection | sc-rce-results.md | No `exec.Command` in `internal/`. All exec calls are in test files. WASM uses wazero with limits. |
| Open Redirect (resolved) | sc-open-redirect-results.md | Post-logout allowlist properly validates domains. Already fixed in recent commits. |
| SSRF (8 Info findings) | sc-ssrf-results.md | All protected by design. `validateUpstreamHost()` blocks private/metadata IPs. TOCTOU mitigated. |
| GraphQL Batch Limit | sc-graphql-results.md | `maxBatchSize=100` exists but dead code for admin API — only applies to internal federation. |
| GraphQL Depth Limit | sc-graphql-results.md | Parser `maxDepth=50` vs analyzer `maxDepth=15` — more restrictive limit wins. |
| GraphQL Integer Overflow | sc-graphql-results.md | Theoretical only; requires `len(arguments) > math.MaxInt`. Not realistic. |
| GraphQL String Escaping | sc-graphql-results.md | Federation executor internal use only. |
| TLS 1.0/1.1 Rejected | sc-crypto-results.md | Actively rejected with warning log; TLS 1.2 enforced. |
| SHA1 for WebSocket | sc-crypto-results.md | Required by RFC 6455 for Sec-WebSocket-Accept header. |
| math/rand for Raft Jitter | sc-crypto-results.md | Go 1.22+ auto-seeds from crypto/rand; Raft jitter is non-security context. |
| OIDC Discovery Decode | sc-lang-go-results.md | Remote IdP is trusted; 1MB limit provides mitigation. |
| Admin Credit Rate Limit Map | sc-lang-go-results.md | Has cleanup goroutine (confirmed in sc-rate-limiting-results.md). |
| WASM Unsafe.Pointer | sc-lang-go-results.md | Documentation examples only. |
| gRPC Test Servers | sc-lang-go-results.md | Test code only (`internal/grpc/*_test.go`). |
| Bulk Import Test | sc-deserialization-results.md | Test code only. |

**Total false positives eliminated:** 23

---

## 11. Methodology

This assessment was performed using **security-check**, an AI-powered static analysis tool that uses large language model reasoning to detect security vulnerabilities.

### Pipeline Phases

1. **Reconnaissance (Phase 1)** — Automated codebase architecture mapping and technology detection via 18 specialized skills
2. **Vulnerability Hunting (Phase 2)** — 18 specialized skills scanned for vulnerabilities across OWASP Top 10 and language-specific patterns
3. **Verification (Phase 3)** — False positive elimination with confidence scoring (0-100) and CWE/OWASP mapping
4. **Reporting (Phase 4)** — CVSS-aligned severity classification and prioritized remediation roadmap

### Skills Executed

| Skill | Category | Findings |
|-------|----------|----------|
| sc-sqli | SQL Injection | 0 verified |
| sc-path-traversal | Path Traversal | 0 verified |
| sc-rce | Remote Code Execution | 0 verified |
| sc-open-redirect | Open Redirect | 0 verified (resolved) |
| sc-ssrf | SSRF | 0 verified |
| sc-auth | Authentication | 1 HIGH |
| sc-authz | Authorization | 3 (1 CRITICAL, 1 HIGH, 1 MEDIUM) |
| sc-cors | CORS | 1 MEDIUM |
| sc-crypto | Cryptography | 0 verified |
| sc-jwt | JWT | 0 verified |
| sc-graphql | GraphQL | 1 MEDIUM |
| sc-deserialization | Deserialization | 1 MEDIUM |
| sc-lang-go | Go Language | 1 LOW |
| sc-lang-typescript | TypeScript | 0 verified |
| sc-secrets | Secrets/Credentials | 1 MEDIUM |
| sc-rate-limiting | Rate Limiting | 2 LOW |
| sc-lang-go, sc-crypto | Resource Consumption | 1 LOW |

### Limitations

- Static analysis only — no runtime testing or dynamic analysis performed
- AI-based reasoning may miss vulnerabilities requiring deep domain knowledge
- Confidence scores are estimates, not guarantees
- Custom business logic flaws may require manual review
- Scan coverage is based on detected languages and frameworks; additional languages may exist undetected

---

## 12. Disclaimer

This security assessment was performed using automated AI-powered static analysis. It does not constitute a comprehensive penetration test or security audit. The findings represent potential vulnerabilities identified through code pattern analysis and LLM reasoning. False positives and false negatives are possible.

This report should be used as a starting point for security remediation, not as a definitive statement of the application's security posture. A professional security audit by qualified security engineers is recommended for production applications handling sensitive data.

**Generated by security-check** — github.com/ersinkoc/security-check

---

*Report generated: 2026-04-25*
*Scanner version: security-check v1.0.0*
*APICerebrus version: main branch (commit 7c52571)*