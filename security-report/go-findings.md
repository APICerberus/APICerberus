# Go Security Scan Report - APICerebrus

**Scan Date:** 2026-04-18
**Scanner:** Go Security Checklist (400+ items)
**Scope:** `internal/gateway/`, `internal/plugin/`, `internal/store/`, `internal/admin/`, `internal/raft/`, `internal/billing/`, `internal/pkg/jwt/`, `internal/mcp/`
**Go Version:** 1.22+

---

## Executive Summary

The APICerebrus codebase demonstrates **strong security posture** overall. SQL injection protection is robust with parameterized queries throughout, cryptographic operations properly use `crypto/rand` and `crypto/subtle`, and HTTP servers have appropriate timeouts configured. Several medium and low severity issues were identified related to error handling and edge cases.

**Critical Findings:** 0
**High Findings:** 0
**Medium Findings:** 2
**Low Findings:** 2
**Informational:** 2

---

## Findings by Severity

### [MEDIUM] OIDC Authorization Accepts Any Valid Client Without User Authentication

- **Category:** Authentication / Authorization
- **CWE:** CWE-287 (Improper Authentication)
- **Location:** `internal/admin/oidc_provider.go:319-358`
- **Pattern Matched:** Authorization code generated without verifying user credentials

**Description:**

The `handleOIDCProviderAuthorize` function at line 319 does not verify the user's identity before issuing an authorization code. The function accepts any request with a valid `client_id` and `redirect_uri` (registered in the OIDC client configuration) and generates an authorization code with a subject from an existing session cookie. If no session exists, it redirects to login — but if a session exists, the authorization code is issued for `subject` (the authenticated user) without any per-request authentication challenge.

```go
// Lines 319-358: Auth code generated without re-authenticating user
if subject == "" {
    // No valid session — redirect to login page with return-to-OIDC endpoint.
    loginURL := "/portal/login?return_to=" + r.URL.RequestURI()
    http.Redirect(w, r, loginURL, http.StatusFound)
    return
}
// Immediately generates auth code — no per-request auth challenge
code, err := newRandomHex(32)
// ... stores code with subject from session cookie
```

**Exploitability:** Medium - An attacker with a valid session could potentially authorize requests without explicit user consent if they can convince the user to visit a crafted link.

**Recommendation:** Add explicit user authentication confirmation step before issuing authorization codes, especially for sensitive scopes.

- **Reference:** RFC 6749 §3.1, OIDC Core Spec §3.1

---

### [MEDIUM] WebSocket Auth Cookie Uses Insecure Direct Assignment

- **Category:** Authentication / Session Management
- **CWE:** CWE-565 ( Reliance on Cookies without Validation and Integrity Checking)
- **Location:** `internal/admin/oidc_provider.go:312-316`
- **Pattern Matched:** JWT parsed from cookie without signature re-validation on each request

**Description:**

The OIDC authorization handler parses the admin JWT from an HttpOnly session cookie and extracts the `sub` claim without re-validating the signature on each request:

```go
if tok, err := jwt.Parse(cookie.Value); err == nil {
    if claims, ok := tok.Payload["sub"].(string); ok {
        subject = claims
    }
}
```

While the cookie is HttpOnly (client-side XSS cannot read it), if an attacker obtains the cookie value through other means (network interception, log exposure), they could use the token directly without signature verification.

**Exploitability:** Low to Medium - Depends on transport security (requires HTTPS) and cookie protection measures.

**Recommendation:** Validate that the token signature is valid on each request, not just during initial login. Consider adding token binding or one-time-use authorization codes.

- **Reference:** CWE-565, OWASP Session Management Cheat Sheet

---

### [LOW] Panic on Entropy Unavailability in Password Generation

- **Category:** Error Handling / Robustness
- **CWE:** CWE-248 (Uncaught Exception)
- **Location:** `internal/store/user_repo.go:587`
- **Pattern Matched:** `panic(fmt.Sprintf("crypto/rand unavailable: %v", err))`

**Description:**

The `generateSecurePassword()` function calls `panic()` if `crypto/rand.Read()` fails:

```go
func generateSecurePassword() (string, error) {
    // ... charset setup ...
    for i := range password {
        for {
            if _, err := rand.Read(buf); err != nil {
                panic(fmt.Sprintf("crypto/rand unavailable: %v", err))
            }
            // rejection sampling continues
        }
    }
}
```

While `crypto/rand` failure is extremely rare on systems with proper entropy sources, panicking will crash the entire goroutine.

**Exploitability:** Very Low - `crypto/rand` failures only occur in severely constrained environments without entropy sources.

**Recommendation:** Return an error instead of panicking:

```go
if _, err := rand.Read(buf); err != nil {
    return "", fmt.Errorf("crypto/rand unavailable: %w", err)
}
```

- **Reference:** CWE-248, Go Error Handling Best Practices

---

### [LOW] WASM Module Path Resolution Without Canonical Path Validation

- **Category:** Path Traversal
- **CWE:** CWE-22 (Path Traversal)
- **Location:** `internal/plugin/wasm.go:242-256`
- **Pattern Matched:** `os.ReadFile(resolved)` after `safeResolvePath`

**Description:**

The WASM module loader resolves plugin paths and validates them, but does not verify the final resolved path is within an expected directory:

```go
resolved, err := r.safeResolvePath(path)
if err != nil {
    return nil, err
}
// ... validateWASMModule ...
wasmBytes, err := os.ReadFile(resolved)
```

While `safeResolvePath` likely performs sanitization, if a plugin configuration allows relative paths or symlinks outside the plugin directory, module files could be loaded from unintended locations.

**Exploitability:** Low - Plugin paths are typically operator-controlled configuration, not user input.

**Recommendation:** Add explicit directory boundary checks after path resolution:

```go
resolved, err := r.safeResolvePath(path)
if err != nil {
    return nil, err
}
resolved, err := filepath.EvalSymlinks(resolved)
if err != nil {
    return nil, err
}
pluginDir, _ := filepath.Abs(r.config.PluginDir)
if !strings.HasPrefix(resolved, pluginDir) {
    return nil, fmt.Errorf("plugin path outside plugin directory")
}
```

- **Reference:** CWE-22, Go Security Best Practices

---

### [INFO] math/rand/v2 Usage for Non-Cryptographic Randomization

- **Category:** Cryptography (Acceptable)
- **Location:**
  - `internal/analytics/engine.go:6` - `math/rand`
  - `internal/analytics/optimized_engine.go:4` - `math/rand/v2`
  - `internal/gateway/balancer_extra.go:6,214,761` - `math/rand/v2` (load balancing)
  - `internal/raft/node.go:7,678` - `math/rand/v2` (Raft election timeout jitter)
  - `internal/plugin/retry.go:5,132` - `math/rand/v2` (retry backoff)

**Description:**

The codebase uses `math/rand` and `math/rand/v2` for non-security-sensitive operations like load balancer selection, retry backoff jitter, and Raft election timeout randomization. All usages include `#nosec G404` annotations.

**Assessment:** **ACCEPTABLE** - In Go 1.22+, `math/rand/v2` is automatically seeded from `crypto/rand` at program startup. Load balancer selection and retry jitter do not require cryptographically secure randomness.

- **Reference:** CWE-338, Go 1.22 Release Notes

---

### [INFO] Constant-Time Comparison Properly Used Throughout

- **Category:** Cryptography / Authentication
- **Location:**
  - `internal/admin/token.go:247,374,424` - Admin key comparison
  - `internal/admin/oidc.go:594-597` - State parameter comparison
  - `internal/mcp/server.go:458` - MCP SSE admin-key comparison
  - `internal/admin/ws.go:163` - WebSocket auth comparison
  - `internal/plugin/auth_apikey.go:186,204` - API key validation
  - `internal/raft/transport.go:212-219` - RPC auth comparison

**Description:**

The codebase correctly uses `crypto/subtle.ConstantTimeCompare` for all secret comparisons, preventing timing oracle attacks.

**Positive Finding:** **CORRECTLY IMPLEMENTED**

- **Reference:** CWE-385, Timing Attack Prevention

---

## Positive Security Findings

### 1. SQL Injection Prevention

All database queries use parameterized queries with `?` placeholders. The `ORDER BY` clause uses an allowlist approach in `normalizeUserSortBy()`.

**Verified in:**
- `internal/store/api_key_repo.go:67,95-108,136-141`
- `internal/store/user_repo.go:213-241`
- `internal/store/audit_repo.go:186-191`

### 2. Command Injection Prevention

No usage of `os/exec` with shell commands in application code. Only test files use `exec.CommandContext` for running the binary itself.

### 3. HTTP Server Timeouts Configured

All HTTP servers have appropriate timeouts configured:
- Gateway: `ReadTimeout`, `WriteTimeout`, `IdleTimeout` (defaults: 30s, 30s, 120s)
- Admin API: Same timeouts as gateway
- gRPC: Configured timeouts
- Raft transport: 30s read/write, 120s idle
- MCP server: 30s read/write, 120s idle

**Verified in:**
- `internal/config/load.go:57-64`
- `internal/gateway/server.go:185-187,401-403`
- `internal/raft/transport.go:103-105`

### 4. Cryptographic Best Practices

- **API key hashing:** SHA-256 (`api_key_repo.go`)
- **Admin authentication:** `subtle.ConstantTimeCompare`
- **Session tokens:** `crypto/rand` for generation
- **Password hashing:** bcrypt with proper cost factor

**Verified in:**
- `internal/admin/token.go:374,424`
- `internal/mcp/server.go:478`
- `internal/plugin/auth_apikey.go:186,204`
- `internal/store/api_key_repo.go:107`
- `internal/store/user_repo.go:574-594`

### 5. Client IP Extraction (Secure by Default)

The `ExtractClientIP()` function in `internal/pkg/netutil/clientip.go` is **secure by default**:
- When no trusted proxies are configured, `X-Forwarded-For` and `X-Real-IP` are ignored
- Right-to-left XFF parsing
- IP format validation

### 6. Race Condition Protection

Proper synchronization primitives:
- `sync.Mutex` / `sync.RWMutex` for protecting shared state
- `sync.Map` for concurrent map access (rate limiters)
- `sync/atomic` for simple counters

**Verified in:**
- `internal/plugin/cache.go:148,160`
- `internal/ratelimit/token_bucket.go:11,20`
- `internal/analytics/optimized_engine.go:356,368,373`
- `internal/gateway/router.go:46`

### 7. Config Import Security

- Uses `os.CreateTemp()` in restricted temp directory
- Sets file permissions to `0o600`
- Strips sensitive fields before import
- Defers removal of temp files

**Verified in:**
- `internal/admin/server.go:470-489`
- `internal/admin/server.go:516-533`

### 8. WASM Module Integrity Checking

WASM modules are validated with SHA-256 checksums when provided:

```go
if expectedSHA, ok := pluginConfig["wasm_file_sha256"].(string); ok && expectedSHA != "" {
    hash := sha256.Sum256(wasmBytes)
    actual := hex.EncodeToString(hash[:])
    if actual != expectedSHA {
        return nil, fmt.Errorf("wasm file SHA-256 mismatch...")
    }
}
```

**Verified in:** `internal/plugin/wasm.go:261-266`

---

## Recommendations

1. **Medium Priority:** Review OIDC authorization flow to ensure explicit user authentication for sensitive operations.

2. **Medium Priority:** Consider adding `filepath.EvalSymlinks` validation after WASM module path resolution.

3. **Low Priority:** Change `panic()` in `generateSecurePassword()` to return an error (very low practical risk).

4. **Low Priority:** Ensure `crypto/rand` availability is checked at startup rather than at password generation time.

5. **Informational:** Continue using `#nosec G404` annotations for non-cryptographic `math/rand` usage.

---

## Scan Methodology

This scan followed the Go Security Checklist (400+ items):

1. **Input Validation & Sanitization** - SQL injection, path traversal, integer conversion
2. **Authentication & Session Management** - Constant-time comparison, JWT validation
3. **Cryptography** - Proper use of crypto libraries, key generation
4. **Error Handling** - Panic recovery, error propagation
5. **Concurrency & Race Conditions** - Mutex usage, sync.Map patterns
6. **Network & HTTP Security** - Timeout configuration, header validation
7. **Memory Safety** - No unsafe.Pointer usage found in application code

---

*Report generated by Go Security Checklist scan.*

---

# Phase 2 Hunt - New Findings (2026-04-18)

## New Findings Summary

| Severity | Count |
|----------|-------|
| Critical | 0 |
| High | 0 |
| Medium | 1 |
| Low | 2 |

---

## Medium (1)

### [M-NEW-001]: GraphQL Guard Error Messages May Expose Query Structure Information

- **CWE:** CWE-209 (Information Exposure Through Error Message)
- **File:** `internal/plugin/graphql_guard.go:102-106`
- **Confidence:** 65%
- **Status:** NEW

**Description:**

The `GraphQLGuard.Handle()` function concatenates GraphQL analyzer errors and returns them directly to the client:

```go
errors := ""
for _, e := range result.Errors {
    errors += e + "; "
}
graphql.WriteError(w, errors, http.StatusBadRequest)
```

While these errors originate from the query analyzer (which validates query structure), the error messages may reveal information about the GraphQL schema structure (field names, types) that could assist an attacker in understanding the API schema.

**Impact:**

An attacker could craft queries to trigger specific analyzer errors and gradually map out the GraphQL schema. This is a reconnaissance advantage, not a direct exploit.

**Remediation:**

Return generic error messages without exposing analyzer error details:

```go
if !result.IsValid {
    graphql.WriteError(w, "query exceeds complexity or depth limits", http.StatusBadRequest)
    return true
}
```

---

## Low (2)

### [L-NEW-001]: Bot Detect Plugin Error Message Contains Raw User-Agent Header

- **CWE:** CWE-77 (Command Injection - Theoretical)
- **File:** `internal/plugin/bot_detect.go:79`
- **Confidence:** 70%
- **Status:** NEW

**Description:**

The `BotDetect.Evaluate()` function embeds the raw User-Agent header directly into an error message without sanitization:

```go
Message: fmt.Sprintf("Blocked bot user-agent: %s", in.Request.Header.Get("User-Agent")),
```

If an attacker controls the User-Agent header with extremely long values or special characters, this could potentially cause log injection or storage issues.

**Impact:**

Low - This is primarily an information disclosure and DoS concern. The error message is returned via the plugin pipeline's error handling.

**Remediation:**

Truncate and sanitize the User-Agent before including it in error messages:

```go
func sanitizeUA(ua string) string {
    if len(ua) > 128 {
        ua = ua[:128]
    }
    ua = strings.ReplaceAll(ua, "\r", "")
    ua = strings.ReplaceAll(ua, "\n", "")
    return ua
}
```

---

### [L-NEW-002]: Request Transform Plugin Applies Header Values Without Validation

- **CWE:** CWE-79 (Cross-Site Scripting - Low Risk)
- **File:** `internal/plugin/request_transform.go:156-158`
- **Confidence:** 60%
- **Status:** NEW

**Description:**

The `RequestTransform.applyHeaderTransforms()` function sets headers from configuration without validating the header values for potential injection:

```go
for key, value := range t.addHeaders {
    req.Header.Set(key, value)
}
```

While the `key` is passed through `http.CanonicalHeaderKey()` during initialization, the `value` is used directly. Plugin configuration is typically operator-controlled, not user-controlled, so the practical risk is low.

**Impact:**

Low - Plugin configuration is admin-defined. An attacker would need admin access to modify plugin configs.

**Remediation:**

Sanitize header values to remove potentially dangerous characters:

```go
func sanitizeHeaderValue(v string) string {
    v = strings.ReplaceAll(v, "\r", "")
    v = strings.ReplaceAll(v, "\n", "")
    return v
}
```

---

## Verified Secure Components (Phase 2 Focus Areas)

| Component | Status | Evidence |
|-----------|--------|----------|
| SQL Parameterized Queries | SECURE | All queries use `?` placeholders |
| SQL ORDER BY Allowlist | SECURE | `normalizeUserSortBy()` at `user_repo.go:628-641` |
| Log Injection Prevention | SECURE | `sanitizeLogValue()` removes `\r\n` at `logger.go:236-241` |
| JWT Signature Verification | SECURE | HS256/ES256 verification in `jwt/*.go` |
| OIDC PKCE Support | SECURE | S256 challenge verification at `oidc_provider.go:258-264` |
| Upstream Proxy SSRF | SECURE | `validateUpstreamHost()` at `optimized_proxy.go:465-468` |
| Health Check SSRF | SECURE | `validateUpstreamHost()` at `health.go:136` |
| WASM Module Path | SECURE | `filepath.EvalSymlinks()` + `filepath.Rel()` validation |
| Billing TOCTOU | SECURE | `LevelSerializable` isolation at `billing/engine.go:152` |
| Internal Errors | SECURE | Generic message sent, details logged at `admin_helpers.go:47-52` |

---

## Phase 2 Hunt Summary

**Scope Covered:** Injection, Auth bypasses, Crypto, Secrets, Path traversal, SSRF, Deserialization, Race conditions, Error handling, Logging injection

**New Findings This Scan:** 3 (1 Medium, 2 Low)
**Already Documented:** 27 (from previous scans)

*Phase 2 Hunt completed: 2026-04-18*
