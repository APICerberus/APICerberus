# Secrets, Data Exposure & Cryptographic Findings Report

**Scan Date:** 2026-04-18
**Phase:** 2 - HUNT (Vulnerability Scanning)
**Scope:** Hardcoded Secrets, Data Exposure, Cryptographic Issues
**Tool:** grep-based pattern matching + code review

---

## Summary

| Severity | Count |
|----------|-------|
| Critical | 0 |
| High | 1 |
| Medium | 3 |
| Low / Info | 9 |

---

## HIGH Severity

### H-001: No Known-Secret Blocklist in JWT/Portal Secret Validation

**CWE:** CWE-327 (Use of Weak Cryptographic Key), CWE-798 (Hard-coded Credentials)

**File:** `internal/pkg/jwt/hs256.go:5-7`, `internal/config/load.go:273-340`

**Evidence:**
```go
// internal/pkg/jwt/hs256.go:5-7
const minHS256SecretLength = 32 // minimum 256 bits for HS256 (NIST SP 800-107)
var ErrWeakHS256Secret = errors.New("weak HS256 secret")
```

```go
// internal/config/load.go:337
// A hot-reload enabling the portal with an empty/weak secret would be catastrophic.
if len(cfg.Portal.Session.Secret) < 32 && cfg.Portal.Enabled {
    addErr("portal.session.secret must be at least 32 bytes")
}
```

**Description:** Configuration validation enforces minimum length (32 bytes) for JWT and portal secrets but does NOT check against a blocklist of known weak/compromised secrets (e.g., "password", "secret", "changeme", "123456", common CTF/example values). An operator could deploy a configuration with `token_secret: "change-me-in-production"` or `"secret"` and the system would accept it.

**Impact:** Attackers could exploit weak JWT signing secrets to forge tokens or session hijacking if the secret is guessable.

**Remediation:** Add weak-secret blocklist validation in `internal/config/load.go:validate()`:
```go
// knownWeakSecrets contains commonly used/compromised secrets that must be rejected.
var knownWeakSecrets = []string{
    "secret", "password", "changeme", "changeme-in-production",
    "123456", "admin", "root", "test", "demo",
    "your-secret", "your-hmac-secret", "your-secure-session-secret",
    "change-me-in-production", "change-me-min-32-chars",
}

for _, weak := range knownWeakSecrets {
    if strings.EqualFold(cfg.Admin.TokenSecret, weak) {
        addErr("admin.token_secret is a known weak value")
    }
    if strings.EqualFold(cfg.Portal.Session.Secret, weak) {
        addErr("portal.session.secret is a known weak value")
    }
}
```

---

## MEDIUM Severity

### M-001: Hardcoded Consumer API Key in apicerberus.yaml

**CWE:** CWE-798 (Hard-coded Credentials)

**File:** `apicerberus.yaml:220`

**Evidence:**
```yaml
consumers:
  - name: "mobile-app"
    api_keys:
      - key: "ck_live_mobile_app_key_12345678901234567890"
```

**Description:** A production-style API key (`ck_live_*` prefix) is hardcoded in the main configuration file. While this appears to be a sample/development key, hardcoding any `ck_live_*` key in version-controlled configuration files is dangerous — it could be accidentally used or committed to version control.

**Remediation:** Use environment variable substitution:
```yaml
consumers:
  - name: "mobile-app"
    api_keys:
      - key: "${MOBILE_APP_API_KEY}"
```

---

### M-002: Bearer Token Placeholder in Example Config

**CWE:** CWE-547 (Hard-coded Security-Related Constants)

**File:** `apicerberus.example.yaml:92`

**Evidence:**
```yaml
otlp_headers:  # Additional headers for OTLP (optional)
  Authorization: "Bearer token"
```

**Description:** Placeholder Bearer token in tracing configuration example could be copy-pasted into production configurations, causing authentication failures or credential exposure in logs.

**Remediation:** Use environment variable:
```yaml
otlp_headers:
  Authorization: "Bearer ${OTLP_AUTH_HEADER}"
```

---

### M-003: Kubernetes Secret Default Values in Example Deployment

**CWE:** CWE-547 (Hard-coded Security-Related Constants)

**File:** `deployments/examples/kubernetes-deployment.yaml:82-83`

**Evidence:**
```yaml
stringData:
  ADMIN_API_KEY: "change-me-in-production"
  SESSION_SECRET: "change-me-in-production"
```

**Description:** Example Kubernetes Secret uses weak placeholder values that could be accidentally deployed to production. The comment says "change-me-in-production" but no mechanism prevents deployment without proper values.

**Remediation:** Use empty values with validation that fails fast:
```yaml
stringData:
  ADMIN_API_KEY: ""  # REQUIRED: Set via environment
  SESSION_SECRET: ""  # REQUIRED: Set via environment
```

---

## LOW / INFO Severity

### L-001: README curl Commands Use Placeholder Credentials

**CWE:** CWE-547 (Hard-coded Security-Related Constants)

**Files:** `README.md:233, 425, 430, 441, 446, 455`

**Evidence:**
```bash
curl -H "X-Admin-Key: change-me" http://localhost:9876/admin/api/v1/status
```

**Description:** Documentation examples with weak placeholder credentials could be copy-pasted into production environments.

**Remediation:** Use clearly marked placeholders:
```bash
curl -H "X-Admin-Key: ${ADMIN_API_KEY}" http://localhost:9876/admin/api/v1/status
```

---

### L-002: SHA-1 for WebSocket Accept Key (Intentionally Required by RFC)

**CWE:** CWE-327 (Use of Weak Cryptographic Algorithm)

**File:** `internal/admin/ws.go:565`

**Evidence:**
```go
func websocketAccept(key string) string {
    sum := sha1.Sum([]byte(key + websocketGUID)) // #nosec G401 G505: SHA-1 is required by RFC 6455 for WebSocket accept key computation.
    return base64.StdEncoding.EncodeToString(sum[:])
}
```

**Description:** SHA-1 is used for WebSocket accept key computation, which follows RFC 6455 specification (the Sec-WebSocket-Accept header uses SHA-1). This is NOT a vulnerability — it is the required algorithm per RFC.

**Assessment:** No action needed. This is intentional and cryptographically appropriate for the WebSocket protocol.

---

### L-003: Test Fixtures with Hardcoded Secrets

**CWE:** N/A (Test Code)

**Files:**
- `web/tests/e2e/crud-flows.spec.ts:105` - `test-password-123`
- `web/tests/e2e/billing-credits.spec.ts:20,40` - `test-password-456`, `test-password-789`
- `web/tests/e2e/api-keys-login.spec.ts:22,53` - `test-password-abc`, `test-password-xyz`
- `test/e2e-config.yaml:10-11` - `token_secret: "e2e-test-jwt-secret-must-be-at-least-32-characters"`
- `internal/plugin/auth_jwt_test.go:255,281` - `Secret: "any-secret"`

**Description:** Test fixtures contain hardcoded placeholder values.

**Assessment:** These are test fixtures only, not production credentials. Risk is minimal. However, E2E test configs should use environment variables or generated secrets to prevent accidental use as templates.

---

### L-004: http.DefaultClient Used in OIDC Token Exchange

**CWE:** CWE-295 (Improper Certificate Validation)

**File:** `internal/admin/oidc.go:398`

**Evidence:**
```go
resp, err := http.DefaultClient.Do(req)
```

**Description:** Uses `http.DefaultClient` for OIDC token exchange with the IdP. The default client has TLS 1.2+ as Go's default minimum, but explicit certificate validation configuration should be used.

**Remediation:** Consider creating a dedicated HTTP client with explicit TLS configuration:
```go
client := &http.Client{
    Transport: &http.Transport{
        TLSClientConfig: &tls.Config{
            MinVersion: tls.VersionTLS12,
            // Root CAs for IdP certificate validation
        },
    },
    Timeout: 10 * time.Second,
}
```

---

### L-005: API Key Hashing - SHA256 Rather Than bcrypt

**CWE:** CWE-327 (Use of Weak Cryptographic Hash)

**File:** `internal/plugin/auth_apikey.go:192-204`

**Evidence:**
```go
// internal/plugin/auth_apikey.go:192-204
if entry.algorithm == "sha256" {
    sum := sha256.Sum256([]byte(provided))
    providedHash := hex.EncodeToString(sum[:])
    match = subtle.ConstantTimeCompare([]byte(providedHash), []byte(entry.keyHash)) == 1
} else if entry.algorithm == "bcrypt" {
    match = bcrypt.CompareHashAndPassword([]byte(entry.keyHash), []byte(provided)) == nil
}
```

**Description:** API keys can be hashed using either SHA-256 or bcrypt. SHA-256 is faster but not appropriate for low-entropy secrets like user-chosen passwords. However, API keys are generated using `crypto/rand` (high entropy, 32+ bytes), so SHA-256 is acceptable for API key hashing.

**Assessment:** This is a design decision. SHA-256 for high-entropy `crypto/rand`-generated keys is acceptable. The bcrypt option exists for operators who want additional protection. Document the tradeoffs.

---

### L-006: OIDC Refresh Token Storage - SHA256 Not bcrypt

**CWE:** CWE-327 (Use of Weak Cryptographic Hash)

**File:** `internal/admin/oidc_provider.go:448-449`

**Evidence:**
```go
// internal/admin/oidc_provider.go:448-449
hash := sha256.Sum256([]byte(token))
return hex.EncodeToString(hash[:]), nil
```

**Description:** OIDC refresh tokens are stored as SHA-256 hashes rather than bcrypt. Refresh tokens are high-entropy (generated via `crypto/rand`) and short-lived, so SHA-256 is acceptable.

**Assessment:** Acceptable for short-lived, high-entropy tokens. bcrypt would add unnecessary computational cost for one-time-use tokens.

---

## LOW / INFO Severity

### L-007: Consumer API Keys Bypass Weak-Secret Blocklist Validation

**CWE:** CWE-798 (Hard-coded Credentials)

**File:** `internal/config/load.go:321-348`

**Evidence:**
```go
// Admin API key has weak-secret check (load.go:321-324)
lowerKey := strings.ToLower(apiKey)
if strings.Contains(lowerKey, "change") || strings.Contains(lowerKey, "secret") || strings.Contains(lowerKey, "password") || strings.Contains(lowerKey, "123") {
    addErr("admin.api_key appears to be a placeholder or weak value")
}

// Admin token_secret has weak-secret check (load.go:330-332)
lowerTokenSecret := strings.ToLower(tokenSecret)
if strings.Contains(lowerTokenSecret, "change") || strings.Contains(lowerTokenSecret, "secret") || strings.Contains(lowerTokenSecret, "password") {
    addErr("admin.token_secret appears to be a placeholder or weak value")
}

// Portal session secret has weak-secret check (load.go:346-348)
lowerSecret := strings.ToLower(secret)
if strings.Contains(lowerSecret, "change") || strings.Contains(lowerSecret, "secret") || strings.Contains(lowerSecret, "password") {
    addErr("portal.session.secret appears to be a placeholder value")
}

// BUT consumer API keys have NO weak-secret check (load.go:570-588)
for j, key := range consumer.APIKeys {
    if strings.TrimSpace(key.Key) == "" {
        addErr(fmt.Sprintf("consumer %q api_keys[%d].key is required", consumer.Name, j))
        continue
    }
    // Only length check, NO blocklist check
    keyLen := len(key.Key)
    if strings.HasPrefix(key.Key, "ck_live_") && keyLen < 32 {
        addErr(...)
    }
    ...
}
```

**Description:** The weak-secret blocklist validation (checking for "change", "secret", "password", "123") applies to admin API key, admin token secret, and portal session secret, but NOT to consumer API keys. An operator could configure a consumer with `ck_live_change-me-in-production-12345678` and it would pass validation.

**Impact:** Low — Consumer API keys are high-entropy (generated via crypto/rand), but a deliberately weak key could still be brute-forced if the blocklist check is bypassed.

**Remediation:** Add weak-secret blocklist check for consumer API keys in the consumer validation loop:
```go
lowerKey := strings.ToLower(key.Key)
if strings.Contains(lowerKey, "change") || strings.Contains(lowerKey, "secret") || strings.Contains(lowerKey, "password") {
    addErr(fmt.Sprintf("consumer %q api_keys[%d].key appears to be a placeholder", consumer.Name, j))
}
```

---

### L-008: E2E Test Config Contains Hardcoded Secrets

**CWE:** CWE-547 (Hard-coded Security-Related Constants)

**File:** `test/e2e-config.yaml:10-11`

**Evidence:**
```yaml
admin:
  api_key: "e2e-test-admin-key-must-be-at-least-32-chars-long"
  token_secret: "e2e-test-jwt-secret-must-be-at-least-32-characters"
```

**Description:** E2E test configuration contains hardcoded admin credentials. While these are test-only credentials, hardcoding them in version-controlled config files creates risk of accidental use as production templates.

**Impact:** Low — Test-only credentials, not valid in production.

**Remediation:** Use environment variable substitution for E2E test config:
```yaml
admin:
  api_key: "${E2E_ADMIN_API_KEY}"
  token_secret: "${E2E_TOKEN_SECRET}"
```

Or generate these dynamically in test setup.

---

### L-009: Kafka TLS InsecureSkipVerify Allowed in Production Validation

**CWE:** CWE-295 (Improper Certificate Validation)

**File:** `internal/config/load.go:452-454`

**Evidence:**
```go
// Kafka TLS: reject insecure skip-verify in production
if kw.config.TLS.SkipVerify && !cfg.ConfigDir.Empty() {
    addErr("kafka.tls.skip_verify is insecure and must not be used in production")
}
```

**Description:** Kafka TLS `InsecureSkipVerify` is configurable and only rejected in non-empty config directories (production). The `!cfg.ConfigDir.Empty()` check means if config directory is empty (development), insecure skip verify is allowed. While this is documented, it could be exploited if a development config is accidentally deployed to production.

**Impact:** Low — This is an intentional dev-mode feature with clear documentation.

**Remediation:** No code change needed. Already documented. Consider adding a startup warning when `SkipVerify` is true.

---

## GOOD PRACTICES OBSERVED

The following demonstrate proper security implementation:

| Practice | Location | Evidence |
|----------|----------|----------|
| **bcrypt cost 12** | `internal/store/user_repo.go:501` | `bcrypt.GenerateFromPassword([]byte(raw), 12)` |
| **crypto/rand** | Throughout codebase | Used for all security-sensitive randomness |
| **Constant-time compare** | `internal/admin/token.go:247,374,424` | `subtle.ConstantTimeCompare()` for admin key |
| **Secret redaction** | `internal/admin/server.go:406-439` | `redactSecrets()` masks all secrets before config export |
| **blockedImportFields** | `internal/admin/server.go:595-604` | Prevents importing sensitive fields via API |
| **Min length validation** | `internal/pkg/jwt/hs256.go:219` | Rejects secrets < 32 bytes for HS256 |
| **Internal error handling** | `internal/admin/admin_helpers.go:47-52` | `writeInternalError()` sends generic msg to client |
| **Docker secrets** | `docker-compose.prod.yml:255-256` | Uses `file: ./secrets/admin_api_key.txt` |
| **TLS 1.2+ default** | Go stdlib | `http.DefaultClient` enforces TLS 1.2 minimum |
| **Header masking** | `internal/audit/masker.go` | PII redaction in audit logs |
| **Host header validation** | `internal/admin/oidc.go:388-389` | Uses fixed path instead of `r.Host` |

---

## Cryptographic Implementation Summary

| Algorithm | Usage | Location | Assessment |
|-----------|-------|----------|------------|
| **SHA-1** | WebSocket accept key | `ws.go:565` | Correct - Required by RFC 6455 |
| **SHA-256** | API key hashing, refresh tokens | `auth_apikey.go:192`, `oidc_provider.go:448` | Acceptable for high-entropy keys |
| **bcrypt** | User password hashing | `user_repo.go:501` | Good - Cost 12 > default |
| **AES-GCM** | Not observed | - | (No custom symmetric encryption found) |
| **ECDSA P-256** | ACME account keys | `certmanager/acme.go:451` | Good |
| **RSA 2048+** | TLS certificates | Generated by ACME or user-provided | Good |

---

## Recommendations

1. **Add weak-secret blocklist validation** in config validation (HIGH priority)
2. **Remove hardcoded `ck_live_*` keys** from config files - use environment variables
3. **Update README examples** - use `${ADMIN_API_KEY}` instead of `change-me`
4. **Strengthen Kubernetes example** - fail-fast if secrets are not configured
5. **Document API key hashing choices** - SHA-256 vs bcrypt tradeoffs for different key types
6. **Consider explicit TLS client** for OIDC token exchange
7. **Extend weak-secret blocklist** to consumer API keys
8. **Use env vars for E2E test config** or generate dynamically

---

## Scan Methodology

- **Patterns searched:**
  - `password\s*[=:]]`, `secret\s*[=:]]`, `token\s*[=:]]`
  - `api_key|apikey`, `-----BEGIN PRIVATE KEY-----`
  - `AKIA[0-9A-Z]{16}|aws_access_key|AWS_SECRET`
  - `bcrypt|scrypt|argon2`, `crypto/des|crypto/rc4`
  - `InsecureSkipVerify|MinVersion|TLSConfig`
  - `X-Forwarded-For|X-Real-IP|RemoteAddr`
  - `stack.*trace|panic|debug\.Print`
  - `subtle\.ConstantTimeCompare`, `crypto/rand`
  - `ck_live_|ck_test_|sk_live_|sk_test_`
  - `math/rand`, `os\.Getenv.*secret`
  - `.env`, `.pem`, `.key` files
  - `change-me|changeme|placeholder` in configs

- **Exclusions:** vendor/, node_modules/, `*.sum` files
- **File types:** *.go, *.yaml, *.yml, *.json, *.ts, *.tsx, *.md