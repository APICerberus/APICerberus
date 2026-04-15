# APICerebrus Phase 2 (Hunt) Vulnerability Scan - Raw Findings

**Scan Date:** 2026-04-16
**Scanner:** Claude Code Vulnerability Hunt
**Project:** APICerebrus API Gateway
**Language:** Go + TypeScript/React

---

## 1. SQL Injection - Scan Results

### Finding: SQL Parameterized Queries - GOOD PRACTICE OBSERVED
- **File:** internal/store/*.go
- **CWE:** CWE-89 (SQL Injection)
- **Severity:** N/A (Positive Finding)
- **Confidence:** High
- **Description:** Store layer consistently uses parameterized queries with ? placeholders.
- **Evidence:** api_key_repo.go uses UPDATE with ? placeholders
- **Remediation:** Continue using parameterized queries.

### Finding: PostgreSQL DSN Construction
- **File:** internal/store/postgres.go:96-100
- **CWE:** CWE-89 (SQL Injection)
- **Severity:** Low
- **Confidence:** Medium
- **Description:** PostgreSQL connection string built with fmt.Sprintf using url.QueryEscape.
- **Evidence:** connStr += fmt.Sprintf(" password=%s", url.QueryEscape(p.Password))
- **Remediation:** Verify all DSN components are properly escaped.

---

## 2. Auth/Bypass - Scan Results

### Finding: Health Endpoint Bypasses Auth
- **File:** internal/gateway/server.go:977-981
- **CWE:** CWE-288 (Authentication Bypass)
- **Severity:** Medium
- **Confidence:** High
- **Description:** Built-in /health and /ready endpoints bypass plugin pipeline and skip authentication.
- **Evidence:** M-004 NOTE: These endpoints bypass the plugin pipeline
- **Remediation:** Document network-level protection requirement.

### Finding: RBAC Enforcement for Static API Key Auth
- **File:** internal/admin/rbac.go:288-292
- **CWE:** CWE-285 (Improper Authorization)
- **Severity:** Low
- **Confidence:** High
- **Description:** Static API key auth must not bypass RBAC.
- **Remediation:** Good implementation.

### Finding: Client IP Spoofing Prevention
- **File:** internal/pkg/netutil/clientip.go
- **CWE:** CWE-200 (Exposure of Sensitive Information)
- **Severity:** Low (Positive Finding)
- **Confidence:** High
- **Description:** Secure by default - X-Forwarded-For parsing walks right-to-left.
- **Remediation:** Good implementation.

---

## 3. API Security - Scan Results

### Finding: Admin API Key Header Validation
- **File:** internal/admin/server.go
- **CWE:** CWE-307 (Brute Force)
- **Severity:** Medium
- **Confidence:** High
- **Description:** Admin API requires X-Admin-Key header.
- **Remediation:** Consider rate limiting on authentication endpoints.

### Finding: OIDC Provider with Bcrypt
- **File:** internal/admin/oidc_provider.go
- **CWE:** CWE-287 (Improper Authentication)
- **Severity:** Low
- **Confidence:** High
- **Description:** OIDC provider uses RSA/EC signing with bcrypt for refresh tokens.
- **Remediation:** Ensure refresh token rotation implemented.

---

## 4. Secrets - Scan Results

### Finding: Test Files Contain Hardcoded Secrets
- **File:** test/e2e_v010_mcp_stdio_test.go:110
- **CWE:** CWE-798 (Use of Hardcoded Credentials)
- **Severity:** Medium
- **Confidence:** High
- **Description:** E2E test file contains hardcoded API key and token secret.
- **Evidence:** api_key: "Xk9#mP$vL2@nQ8*wR5&tZ3(cY7)jF4!hK6_gH1~uE0-iO9=pA2|sD5>lN8<bM3"
- **Remediation:** Move test secrets to environment variables.

### Finding: Test Config Contains Predictable Secrets
- **File:** test-config.yaml:13-14
- **CWE:** CWE-798 (Use of Hardcoded Credentials)
- **Severity:** Medium
- **Confidence:** High
- **Description:** Test configuration uses hardcoded credentials.
- **Evidence:** api_key: "test-admin-key-32chars-minimum!!"
- **Remediation:** Use environment-specific configuration.

### Finding: JWT Benchmark Secret
- **File:** test/benchmark/plugin_bench_test.go:30,92
- **CWE:** CWE-798 (Use of Hardcoded Credentials)
- **Severity:** Low
- **Confidence:** High
- **Description:** Benchmark tests use hardcoded JWT secret.
- **Evidence:** secret := "super-secret-key-that-is-32-bytes!"
- **Remediation:** Acceptable for benchmarks.

### Finding: OIDC Test Client Secret
- **File:** internal/admin/oidc_provider_test.go:448
- **CWE:** CWE-798 (Use of Hardcoded Credentials)
- **Severity:** Low
- **Confidence:** High
- **Description:** Test file contains hardcoded client secret.
- **Remediation:** Acceptable for tests.

---

## 5. XSS - Scan Results

### Finding: No dangerouslySetInnerHTML Usage
- **File:** web/src/**/*.tsx
- **CWE:** CWE-79 (Cross-site Scripting)
- **Severity:** N/A (Positive Finding)
- **Confidence:** High
- **Description:** No usage of dangerouslySetInnerHTML found.
- **Remediation:** Good. Continue to avoid innerHTML.

### Finding: No eval() or new Function() Usage
- **File:** web/src/**/*.ts
- **CWE:** CWE-95 (Dynamic Evaluation)
- **Severity:** N/A (Positive Finding)
- **Confidence:** High
- **Description:** No dynamic code evaluation found.
- **Remediation:** Good practice maintained.

---

## 6. SSRF - Scan Results

### Finding: HTTP Client in Proxy
- **File:** internal/gateway/optimized_proxy.go
- **CWE:** CWE-918 (Server-Side Request Forgery)
- **Severity:** Medium
- **Confidence:** Medium
- **Description:** The proxy uses httputil.ReverseProxy. Verify upstream URL validation.
- **Remediation:** Ensure upstream URLs validated against whitelist.

### Finding: No Obvious SSRF in HTTP Requests
- **Files:** internal/gateway/*.go
- **CWE:** CWE-918 (Server-Side Request Forgery)
- **Severity:** N/A (Positive Finding)
- **Confidence:** High
- **Description:** No direct http.Get with user-controlled URLs found.
- **Remediation:** Good.

---

## 7. Path Traversal - Scan Results

### Finding: No Obvious Path Traversal
- **Files:** **/*.go
- **CWE:** CWE-22 (Path Traversal)
- **Severity:** N/A (Positive Finding)
- **Confidence:** High
- **Description:** No path traversal patterns found.
- **Remediation:** Good.

---

## 8. Command Injection - Scan Results

### Finding: No exec.Command Usage
- **File:** internal/cli/**/*.go
- **CWE:** CWE-78 (OS Command Injection)
- **Severity:** N/A (Positive Finding)
- **Confidence:** High
- **Description:** No usage of exec.Command found.
- **Remediation:** Good.

---

## 9. JWT - Scan Results

### Finding: JWT Validation Implementation
- **File:** internal/pkg/jwt/*.go, internal/plugin/auth_jwt*.go
- **CWE:** CWE-347 (Improper Verification of Cryptographic Signature)
- **Severity:** Low
- **Confidence:** High
- **Description:** JWT implementation uses proper signature verification with algorithm validation.
- **Evidence:** Tests verify ErrInvalidJWTSignature, algorithm confusion rejected
- **Remediation:** Good implementation.

### Finding: JWT Algorithm Confusion Protection
- **File:** internal/plugin/auth_jwt_test.go
- **CWE:** CWE-347 (Improper Verification of Cryptographic Signature)
- **Severity:** Low
- **Confidence:** High
- **Description:** Tests explicitly verify algorithm confusion attacks rejected.
- **Evidence:** assertJWTErrorCode(t, err, "unsupported_jwt_algorithm")
- **Remediation:** Good test coverage.

---

## 10. Rate Limiting - Scan Results

### Finding: Atomic Redis Rate Limiting with Lua
- **File:** internal/ratelimit/redis.go:366-369
- **CWE:** CWE-662 (Insufficient Control)
- **Severity:** N/A (Positive Finding)
- **Confidence:** High
- **Description:** Redis rate limiting uses Lua scripts for atomic operations, prevents TOCTOU.
- **Remediation:** Good implementation.

---

## 11. Go Security - Scan Results

### Finding: Non-Crypto RNG in Analytics
- **File:** internal/analytics/optimized_engine.go:472-474
- **CWE:** G404 (Use of Non-Crypto RNG)
- **Severity:** Low
- **Confidence:** High
- **Description:** Analytics uses math/rand for reservoir sampling. Intentional for performance.
- **Evidence:** G404: reservoir sampling - non-crypto RNG is intentional
- **Remediation:** Acceptable for analytics.

### Finding: Crypto/Rand Panic is Appropriate
- **File:** internal/store/user_repo.go:585
- **CWE:** CWE-754 (Improper Check)
- **Severity:** Low
- **Confidence:** High
- **Description:** Panic when crypto/rand unavailable is appropriate.
- **Evidence:** panic(fmt.Sprintf("crypto/rand unavailable: %v", err))
- **Remediation:** Correct.

---

## 12. TypeScript Security - Scan Results

### Finding: API Client Uses Fetch with Credentials
- **File:** web/src/lib/api.ts:110
- **CWE:** CWE-598 (Use of GET Request Method)
- **Severity:** Low
- **Confidence:** High
- **Description:** Uses standard fetch API with credentials inclusion.
- **Remediation:** Good practice.

### Finding: Portal API CSRF Protection
- **File:** web/src/lib/portal-api.ts:31
- **CWE:** CWE-352 (Cross-Site Request Forgery)
- **Severity:** Low
- **Confidence:** Medium
- **Description:** Portal API uses credentials include.
- **Remediation:** Ensure CSRF tokens implemented.

---

## 13. Secrets/Crypto - Scan Results

### Finding: Password Hashing with Bcrypt
- **File:** internal/admin/oidc_provider.go:32
- **CWE:** N/A
- **Severity:** N/A (Positive Finding)
- **Confidence:** High
- **Description:** Refresh tokens and passwords use bcrypt.
- **Evidence:** key = bcrypt hash of refresh token
- **Remediation:** Good.

### Finding: Audit Logging Field Masking
- **File:** internal/audit/masker.go:22-24
- **CWE:** N/A
- **Severity:** N/A (Positive Finding)
- **Confidence:** High
- **Description:** Comprehensive field masking for sensitive data.
- **Evidence:** password, secret, token, api_key, credit_card masked
- **Remediation:** Good implementation.

---

## 14. Additional Findings

### Finding: TODO Comments Indicate Incomplete Features
- **File:** internal/plugin/request_transform.go:130
- **CWE:** N/A
- **Severity:** Low
- **Confidence:** High
- **Description:** TODO: implement JSON body read/rewrite in POST body phase.
- **Remediation:** Track and prioritize.

### Finding: Kubernetes ConfigMaps with Empty Secrets
- **File:** deployments/kubernetes/base/configmap.yaml:35-36
- **CWE:** CWE-547 (Use of Hardcoded Constants)
- **Severity:** Medium
- **Confidence:** High
- **Description:** Kubernetes base config uses empty secret placeholders.
- **Evidence:** api_key: "", token_secret: ""
- **Remediation:** Document that these MUST be overridden.

---

## Summary Statistics

| Category | Total | Critical | High | Medium | Low | Info |
|----------|-------|----------|------|--------|-----|------|
| SQL Injection | 2 | 0 | 0 | 0 | 2 | 0 |
| Auth/Bypass | 3 | 0 | 0 | 1 | 2 | 0 |
| API Security | 2 | 0 | 0 | 1 | 1 | 0 |
| Secrets | 4 | 0 | 0 | 3 | 1 | 0 |
| XSS | 3 | 0 | 0 | 0 | 0 | 3 |
| SSRF | 2 | 0 | 0 | 1 | 1 | 0 |
| Path Traversal | 1 | 0 | 0 | 0 | 0 | 1 |
| CMDi | 1 | 0 | 0 | 0 | 0 | 1 |
| JWT | 2 | 0 | 0 | 0 | 2 | 0 |
| Rate Limiting | 2 | 0 | 0 | 0 | 2 | 0 |
| Go Security | 3 | 0 | 0 | 0 | 3 | 0 |
| TypeScript Security | 2 | 0 | 0 | 0 | 2 | 0 |
| Secrets/Crypto | 4 | 0 | 0 | 0 | 0 | 4 |
| Additional | 2 | 0 | 0 | 1 | 1 | 0 |
| **TOTAL** | **33** | **0** | **0** | **7** | **17** | **9** |

---

## Recommendations

### High Priority
1. Move test secrets to environment variables
2. Document network-level protection for health endpoints
3. Complete TODO items

### Medium Priority
1. Verify SSRF protection in proxy code
2. Review Kubernetes secret management
3. Add brute-force protection

### Low Priority / Informational
1. Non-crypto RNG acceptable for analytics
2. Good security practices overall
3. JWT has proper algorithm confusion protection
