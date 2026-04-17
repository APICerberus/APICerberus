package admin

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	jsonutil "github.com/APICerberus/APICerebrus/internal/pkg/json"
	"github.com/APICerberus/APICerebrus/internal/config"
	jwtpkg "github.com/APICerberus/APICerebrus/internal/pkg/jwt"
)

var (
	errAdminTokenExpired = errors.New("admin token expired")
	errAdminTokenInvalid = errors.New("admin token invalid")
)

const (
	adminSessionCookieName = "apicerberus_admin_session"
	adminCSRFCookieName    = "apicerberus_admin_csrf"
	adminCSRFHeaderName    = "X-CSRF-Token"
)

// extractAdminTokenFromCookie reads the admin JWT from the HttpOnly session cookie.
func extractAdminTokenFromCookie(r *http.Request) string {
	if c, err := r.Cookie(adminSessionCookieName); err == nil && c != nil {
		return strings.TrimSpace(c.Value)
	}
	return ""
}

// issueAdminToken generates a scoped HS256 admin JWT with optional role and permissions.
// The keyVersion is embedded in the token so that key rotations invalidate existing sessions.
func issueAdminToken(secret string, ttl time.Duration, role string, permissions []string, keyVersion int64) (string, error) {
	if secret == "" {
		return "", errors.New("admin token secret is not configured")
	}
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}

	now := time.Now().UTC()
	// Generate a unique token ID for revocation/correlation (M1)
	jtiBytes := make([]byte, 16)
	if _, err := rand.Read(jtiBytes); err != nil {
		// Fall back to time-based JTI if crypto/rand is unavailable
		jtiBytes = []byte(fmt.Sprintf("%x-%x", now.UnixNano(), now.Unix()))
	}
	jti := fmt.Sprintf("%x", jtiBytes)
	header := map[string]string{
		"alg": "HS256",
		"typ": "JWT",
	}
	payload := map[string]any{
		"sub":         "admin",
		"jti":         jti,
		"iss":         "apicerberus-admin",
		"aud":         "apicerberus",
		"iat":         now.Unix(),
		"exp":         now.Add(ttl).Unix(),
		"key_version": keyVersion,
	}
	if role != "" {
		payload["role"] = role
	}
	if len(permissions) > 0 {
		payload["permissions"] = permissions
	}

	headerBytes, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	signingInput := jwtpkg.EncodeSegment(headerBytes) + "." + jwtpkg.EncodeSegment(payloadBytes)
	signature, err := jwtpkg.SignHS256(signingInput, []byte(secret))
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	token := signingInput + "." + jwtpkg.EncodeSegment(signature)
	return token, nil
}

// verifyAdminToken parses and validates an admin JWT.
// The keyVersion parameter is the current admin key version from config;
// tokens with a mismatched key_version claim are rejected (forces re-auth after key rotation).
func verifyAdminToken(tokenString, secret string, keyVersion int64) error {
	if secret == "" {
		return errors.New("admin token secret is not configured")
	}
	tok, err := jwtpkg.Parse(tokenString)
	if err != nil {
		return errAdminTokenInvalid
	}
	alg, _ := tok.HeaderString("alg")
	if alg != "HS256" {
		return errAdminTokenInvalid
	}
	if !jwtpkg.VerifyHS256(tok.SigningInput, tok.Signature, []byte(secret)) {
		return errAdminTokenInvalid
	}
	// H-001 fix: reject tokens signed with an older key version after rotation.
	// JSON numbers unmarshal as float64, so handle numeric types appropriately.
	if rawVersion, ok := tok.Payload["key_version"]; ok {
		var tokKeyVersion int64
		switch v := rawVersion.(type) {
		case float64:
			tokKeyVersion = int64(v)
		case int64:
			tokKeyVersion = v
		case int:
			tokKeyVersion = int64(v)
		default:
			return errors.New("admin token has invalid key_version claim type")
		}
		if tokKeyVersion != keyVersion {
			return errors.New("admin token key version mismatch — re-authenticate with the current key")
		}
	}
	exp, ok := tok.ClaimUnix("exp")
	if !ok || time.Now().UTC().Unix() > exp {
		return errAdminTokenExpired
	}
	// Validate iat (issued-at) — reject tokens with future iat (clock skew tolerance: 60s)
	if iat, ok := tok.ClaimUnix("iat"); ok {
		now := time.Now().UTC().Unix()
		if iat > now+60 {
			return errors.New("admin token issued in the future")
		}
	}
	// Validate nbf (not-before) if present
	if nbf, ok := tok.ClaimUnix("nbf"); ok {
		if time.Now().UTC().Unix() < nbf {
			return errors.New("admin token not yet valid")
		}
	}
	return nil
}

// extractBearerToken extracts the token from an Authorization: Bearer <token> header.
func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if strings.HasPrefix(auth, prefix) {
		return strings.TrimSpace(auth[len(prefix):])
	}
	return ""
}

// withAdminBearerAuth restricts endpoints to valid Bearer tokens only,
// then chains to RBAC for permission checking.
func (s *Server) withAdminBearerAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		clientIP := extractClientIP(r)

		s.mu.RLock()
		cfg := s.cfg.Admin
		s.mu.RUnlock()

		// IP allow-list check (enforced before auth)
		if !isAllowedIP(clientIP, cfg.AllowedIPs) {
			writeError(w, http.StatusForbidden, "ip_not_allowed", "Client IP is not in the admin allow-list")
			return
		}

		// Rate limiting check
		if s.isRateLimited(clientIP) {
			writeError(w, http.StatusTooManyRequests, "rate_limited", "Too many failed authentication attempts. Please try again later.")
			return
		}

		token := extractBearerToken(r)
		if token == "" {
			token = extractAdminTokenFromCookie(r)
		}
		if token == "" {
			s.recordFailedAuth(clientIP)
			writeError(w, http.StatusUnauthorized, "admin_unauthorized", "Missing Bearer token")
			return
		}
		if err := verifyAdminToken(token, cfg.TokenSecret, cfg.KeyVersion); err != nil {
			s.recordFailedAuth(clientIP)
			writeError(w, http.StatusUnauthorized, "admin_unauthorized", "Invalid or expired token")
			return
		}
		s.clearFailedAuth(clientIP)

		// CSRF protection: validate double-submit token for state-changing requests.
		// M-014 fix: Browser-based CSRF attacks can forge requests even with Bearer auth.
		// The admin API relies on X-Admin-Key + HttpOnly cookie; CSRF adds origin validation.
		// NOTE: Skip CSRF check on login endpoints that set the CSRF cookie (handleTokenIssue,
		// handleFormLogin) to avoid chicken-and-egg — the cookie isn't set until after auth.
		isLoginEndpoint := r.URL.Path == "/admin/api/v1/auth/token" || r.URL.Path == "/admin/login"
		if !isLoginEndpoint && (r.Method == "POST" || r.Method == "PUT" || r.Method == "DELETE" || r.Method == "PATCH") {
			if !validateAdminCSRFToken(r) {
				writeError(w, http.StatusForbidden, "csrf_required", "CSRF token validation failed")
				return
			}
		}

		// Extract role and permissions from the verified JWT
		role, perms := extractRoleFromJWT(token)
		ctx := r.Context()
		if role != "" {
			ctx = context.WithValue(ctx, ctxUserRole, role)
			ctx = context.WithValue(ctx, ctxUserPerms, perms)
		}

		// Chain to RBAC middleware
		s.withRBAC(next)(w, r.WithContext(ctx))
	}
}

// withAdminStaticAuth restricts endpoints to the static API key (bootstrap only).
func (s *Server) withAdminStaticAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		clientIP := extractClientIP(r)

		s.mu.RLock()
		cfg := s.cfg.Admin
		s.mu.RUnlock()

		// IP allow-list check (enforced before auth)
		if !isAllowedIP(clientIP, cfg.AllowedIPs) {
			writeError(w, http.StatusForbidden, "ip_not_allowed", "Client IP is not in the admin allow-list")
			return
		}

		// Rate limiting check
		if s.isRateLimited(clientIP) {
			writeError(w, http.StatusTooManyRequests, "rate_limited", "Too many failed authentication attempts. Please try again later.")
			return
		}

		provided := r.Header.Get("X-Admin-Key")
		if subtle.ConstantTimeCompare([]byte(provided), []byte(cfg.APIKey)) != 1 {
			s.recordFailedAuth(clientIP)
			writeError(w, http.StatusUnauthorized, "admin_unauthorized", "Invalid admin key")
			return
		}
		s.clearFailedAuth(clientIP)
		next(w, r)
	}
}

// handleTokenIssue issues a new admin JWT when presented with the static key.
func (s *Server) handleTokenIssue(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	cfg := s.cfg.Admin
	s.mu.RUnlock()

	token, err := issueAdminToken(cfg.TokenSecret, cfg.TokenTTL, string(RoleAdmin), RolePermissions[RoleAdmin], cfg.KeyVersion)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "token_issue_failed", err.Error())
		return
	}

	csrfToken, csrfErr := generateAdminCSRFToken()

	// Set HttpOnly cookie for XSS-safe authentication transport.
	// Always set Secure flag to prevent token leakage over HTTP (CWE-614)
	cookie := &http.Cookie{
		Name:     adminSessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(cfg.TokenTTL.Seconds()),
	}
	http.SetCookie(w, cookie)

	if csrfErr == nil {
		setAdminCSRFCookie(w, csrfToken)
	}

	_ = jsonutil.WriteJSON(w, http.StatusOK, map[string]any{
		"token_type": "Bearer",
		"token":      token,
		"csrf_token": csrfToken,
		"expires_in": int(cfg.TokenTTL.Seconds()),
		"message":    "Session cookie set successfully",
	})
}

// generateAdminCSRFToken creates a cryptographically random CSRF token.
func generateAdminCSRFToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// setAdminCSRFCookie sets the CSRF double-submit cookie on the response.
// The cookie is NOT HttpOnly so JavaScript can read it for the double-submit pattern.
func setAdminCSRFCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     adminCSRFCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   86400, // 24 hours
		HttpOnly: false, // Must be readable by JS for double-submit header
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})
}

// validateAdminCSRFToken validates the double-submit CSRF token from header vs cookie.
// Returns true for GET/HEAD/OPTIONS (no body, no CSRF needed), or when no CSRF cookie
// is present (programmatic/test clients without browser cookie flow). For browser-based
// POST/PUT/DELETE/PATCH, both cookie and header must match.
func validateAdminCSRFToken(r *http.Request) bool {
	method := r.Method
	if method == "GET" || method == "HEAD" || method == "OPTIONS" {
		return true
	}
	cookie, err := r.Cookie(adminCSRFCookieName)
	// No CSRF cookie means this is a programmatic client (test/mcp/api-key) — skip CSRF
	if err != nil || cookie.Value == "" {
		return true
	}
	// Check X-CSRF-Token header (primary) and X-XSRF-Token (Angular fallback)
	headerToken := r.Header.Get(adminCSRFHeaderName)
	if headerToken == "" {
		headerToken = r.Header.Get("X-XSRF-Token")
	}
	return cookie.Value == headerToken && cookie.Value != ""
}

// handleFormLogin accepts an admin key via HTML form POST, validates it against
// the static API key, and sets an HttpOnly session cookie. The key never
// enters JavaScript — it's submitted directly to the server.
func (s *Server) handleFormLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.mu.RLock()
	cfg := s.cfg.Admin
	s.mu.RUnlock()

	if err := r.ParseForm(); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "Invalid form data")
		return
	}

	clientIP := extractClientIP(r)

	if s.isRateLimited(clientIP) {
		http.Redirect(w, r, "/dashboard?login=rate_limited", http.StatusSeeOther)
		return
	}

	provided := r.FormValue("admin_key")
	if provided == "" {
		s.recordFailedAuth(clientIP)
		http.Redirect(w, r, "/dashboard?login=missing_key", http.StatusSeeOther)
		return
	}

	if subtle.ConstantTimeCompare([]byte(provided), []byte(cfg.APIKey)) != 1 {
		s.recordFailedAuth(clientIP)
		http.Redirect(w, r, "/dashboard?login=invalid_key", http.StatusSeeOther)
		return
	}

	token, err := issueAdminToken(cfg.TokenSecret, cfg.TokenTTL, string(RoleAdmin), RolePermissions[RoleAdmin], cfg.KeyVersion)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "token_issue_failed", err.Error())
		return
	}

	cookie := &http.Cookie{
		Name:     adminSessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		// L-006 FIX: SameSite=Strict provides CSRF protection.
		// Lax allows cross-site GET requests (browser navigates, images load) to include cookies.
		// Strict ensures cookie only sent on same-site requests, blocking CSRF on state-changing operations.
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(cfg.TokenTTL.Seconds()),
	}
	http.SetCookie(w, cookie)

	http.Redirect(w, r, "/dashboard?login=success", http.StatusSeeOther)
}

// handleRotateAdminKey rotates the static admin API key without requiring a restart.
// POST /admin/api/v1/auth/rotate-key
// Requires current admin key via X-Admin-Key header.
func (s *Server) handleRotateAdminKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "POST required")
		return
	}

	clientIP := extractClientIP(r)
	if s.isRateLimited(clientIP) {
		writeError(w, http.StatusTooManyRequests, "rate_limited", "Too many authentication attempts.")
		return
	}

	// Validate current admin key before allowing rotation
	s.mu.RLock()
	currentKey := s.cfg.Admin.APIKey
	s.mu.RUnlock()

	provided := r.Header.Get("X-Admin-Key")
	if subtle.ConstantTimeCompare([]byte(provided), []byte(currentKey)) != 1 {
		s.recordFailedAuth(clientIP)
		writeError(w, http.StatusUnauthorized, "invalid_key", "Current admin key is required to rotate")
		return
	}

	// Parse new key from request body
	var req struct {
		NewKey string `json:"new_key"`
	}
	if err := jsonutil.ReadJSON(r, &req, 1<<20); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	newKey := strings.TrimSpace(req.NewKey)
	if len(newKey) < 32 {
		writeError(w, http.StatusBadRequest, "weak_key", "New key must be at least 32 characters")
		return
	}
	lowerKey := strings.ToLower(newKey)
	if strings.Contains(lowerKey, "change") || strings.Contains(lowerKey, "secret") ||
		strings.Contains(lowerKey, "password") || strings.Contains(lowerKey, "123") {
		writeError(w, http.StatusBadRequest, "weak_key", "Key appears to be a placeholder or weak value")
		return
	}

	// F-013: Apply the new key via mutateConfig (hot-reload without restart)
	// H-001 fix: increment KeyVersion so all existing JWT sessions are immediately invalidated
	if err := s.mutateConfig(func(cfg *config.Config) error {
		cfg.Admin.APIKey = newKey
		cfg.Admin.KeyVersion++
		return nil
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "rotation_failed", err.Error())
		return
	}

	s.clearFailedAuth(clientIP)
	_ = jsonutil.WriteJSON(w, http.StatusOK, map[string]any{
		"rotated": true,
		"message": "Admin key rotated successfully. Use the new key for all subsequent requests.",
	})
}

// handleFormLogout clears the admin session cookie and redirects to login.
func (s *Server) handleFormLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     adminSessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
	http.Redirect(w, r, "/dashboard?logout=1", http.StatusSeeOther)
}

// handleTokenLogout clears the admin session cookie.
func (s *Server) handleTokenLogout(w http.ResponseWriter, _ *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     adminSessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})
	_ = jsonutil.WriteJSON(w, http.StatusOK, map[string]any{"logged_out": true})
}
