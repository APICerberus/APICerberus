# Web Dashboard Security Findings

**Scan Date:** 2026-04-18
**Scanner:** Web Dashboard Security Scanner (Phase 2)
**Scope:** `web/src/` (React/TypeScript), `internal/admin/server.go` (CORS/admin API)
**Previous Report:** `security-report/verified-findings.md`

---

## Executive Summary

The APICerebrus React web dashboard implements several security controls:
- Admin API uses HttpOnly session cookies with CSRF double-submit protection
- Portal API uses CSRF tokens stored in cookies (non-HttpOnly for JS access)
- Dashboard has security headers (CSP, X-Frame-Options, X-Content-Type-Options)
- WebSocket origin validation with allow-list support
- No `dangerouslySetInnerHTML` usage found
- npm audit: 0 vulnerabilities

**New Findings:**
- CSP allows `'unsafe-inline'` which weakens XSS mitigation
- Portal CSRF cookie is non-HttpOnly (accessible to XSS)
- WebSocket URL resolution lacks origin validation on client side
- SessionStorage auth state is accessible to XSS

---

## Detailed Findings

### Finding 1: CSP Allows Unsafe-Inline Weakening XSS Mitigation

| Field | Value |
|-------|-------|
| **CWE** | CWE-1035 (Security Configuration - CSP Weakness) |
| **CVSS 3.1** | 5.3 (Medium) |
| **File:Line** | `internal/admin/ui.go:54` |
| **Severity** | Medium |
| **Confidence** | High |

**Code:**
```go
w.Header().Set("Content-Security-Policy",
    "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; ...")
```

**Evidence:**
The CSP header permits `unsafe-inline` in script-src and style-src. This allows inline script tags and style elements to execute, significantly reducing XSS mitigation.

**Impact:**
- Stored XSS payloads in user data could execute
- CSP bypass via script injection

**Remediation:**
1. Remove `unsafe-inline` from script-src and style-src
2. Use nonce-based CSP for inline scripts
3. Replace `unsafe-eval` with strict-dynamic if possible

---

### Finding 2: Portal CSRF Cookie Is Non-HttpOnly (XSS Token Theft Risk)

| Field | Value |
|-------|-------|
| **CWE** | CWE-1004 (Sensitive Cookie Without HttpOnly Flag) |
| **CVSS 3.1** | 6.5 (Medium) |
| **File:Line** | `web/src/lib/portal-api.ts:8-9` |
| **Severity** | Medium |
| **Confidence** | High |

**Code:**
```typescript
const CSRF_COOKIE_NAME = "csrf_token";

export function getPortalCSRFToken(): string | null {
  const match = document.cookie.match(new RegExp("(^| )" + CSRF_COOKIE_NAME + "=([^;]+)"));
  return match ? match[2] : null;
}
```

**Evidence:**
The portal CSRF token is stored in a non-HttpOnly cookie (readable by JavaScript). This is intentional for the double-submit pattern but creates an XSS vector.

**Attack Scenario:**
1. Attacker finds XSS in portal
2. Injected script reads document.cookie matching csrf_token
3. Sends state-changing request with stolen CSRF token

**Remediation:**
1. Use SameSite=Strict cookies (already implemented) to prevent CSRF entirely
2. For XSS resistance: consider sessionStorage instead of cookie
3. The existing SameSite=Strict provides good CSRF protection

---

### Finding 3: Admin Auth State in sessionStorage Is XSS Accessible

| Field | Value |
|-------|-------|
| **CWE** | CWE-315 (Cleartext Storage in Session Storage) |
| **CVSS 3.1** | 4.5 (Medium-Low) |
| **File:Line** | `web/src/lib/api.ts:38-43`, `web/src/lib/constants.ts:35` |
| **Severity** | Low-Medium |
| **Confidence** | High |

**Code:**
```typescript
// constants.ts
adminAuthStateKey: "apicerberus_admin_authenticated",

// api.ts
export function isAdminAuthenticated(): boolean {
  return window.sessionStorage.getItem(API_CONFIG.adminAuthStateKey) === "true";
}
```

**Evidence:**
Admin authentication state is stored in sessionStorage as a boolean flag. The actual JWT is in an HttpOnly cookie (good), but the sessionStorage flag is readable by injected JavaScript.

**Note:** The actual admin token is HttpOnly and cannot be stolen via XSS. The sessionStorage flag only reveals login status.

**Remediation:**
1. Check HttpOnly session cookie presence for auth state
2. Remove sessionStorage auth flag

---

### Finding 4: WebSocket Token in Query Parameter

| Field | Value |
|-------|-------|
| **CWE** | CWE-598 (Query String in GET Request) |
| **CVSS 3.1** | 4.5 (Medium-Low) |
| **File:Line** | `internal/admin/ws.go:131-138` |
| **Severity** | Low-Medium |
| **Confidence** | High |

**Code:**
```go
if token := strings.TrimSpace(r.URL.Query().Get("token")); token != "" {
    // Clear token from URL to prevent logging (CWE-532)
    q := r.URL.Query()
    q.Del("token")
    r.URL.RawQuery = q.Encode()
    return true
}
```

**Evidence:**
WebSocket clients can authenticate via ?token= query parameter. The server clears it after verification, but token appears in initial URL.

**Remediation:**
1. Prefer cookie-based auth for WebSocket (already implemented)
2. If query param required, ensure HTTPS only and short token TTL

---

### Finding 5: localStorage Used for Tour State (No Sensitive Data)

| Field | Value |
|-------|-------|
| **CWE** | CWE-922 (Insecure Storage) |
| **CVSS 3.1** | 1.5 (Low) |
| **File:Line** | `web/src/App.tsx:139` |
| **Severity** | Info |
| **Confidence** | High |

**Evidence:**
Tour state (apicerberus.welcome_shown, apicerberus.tour_completed) stored in localStorage. Not sensitive - only UI preferences.

**Assessment:** No security risk.

---

## Verified Secure Implementations

| Component | Implementation | Location |
|-----------|---------------|----------|
| Admin Session Cookie | HttpOnly + Secure + SameSite=Strict | token.go:273-281 |
| Admin JWT Storage | HttpOnly cookie only | token.go:33-37 |
| Admin Login Form | HTML form POST; key goes server-side only | Login.tsx:54 |
| Portal Auth | CSRF double-submit + session cookie | portal-api.ts:124-131 |
| Dashboard Security Headers | CSP + X-Frame-Options + X-Content-Type-Options | ui.go:49-55 |
| WebSocket Origin Validation | Server-side allow-list with wildcard support | ws.go:171-238 |
| WebSocket Auth | Cookie + Bearer token | ws.go:117-169 |
| dangerouslySetInnerHTML | Zero usage across web/src | Grep result: 0 matches |

---

## Summary

| Severity | Count | Status |
|----------|-------|--------|
| Critical | 0 | None found |
| High | 0 | None found |
| Medium | 2 | CSP weak (Finding 1), CSRF cookie non-HttpOnly (Finding 2) |
| Low | 2 | WebSocket URL token (Finding 4), sessionStorage auth state (Finding 3) |
| Info | 1 | localStorage tour state (Finding 5 - no risk) |

**Overall Assessment:** The web dashboard has solid security foundations. Primary concern is the weak CSP (unsafe-inline), which significantly reduces XSS protection. The CSRF token in a non-HttpOnly cookie is acceptable when paired with SameSite=Strict.

---

## Recommendations

### Priority 1 (Should Fix)
1. **CSP Hardening**: Remove unsafe-inline from script-src and style-src. Use nonce-based approach.

### Priority 2 (Recommended)
2. **CSRF Token Storage**: Consider whether CSRF token needs to be JS-readable. SameSite=Strict already prevents CSRF.

### Priority 3 (Nice to Have)
3. **Auth State Check**: Refactor isAdminAuthenticated() to check HttpOnly session cookie instead of sessionStorage.

---

*Report generated: 2026-04-18*
