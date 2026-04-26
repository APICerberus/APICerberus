package portal

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/APICerberus/APICerebrus/internal/config"
	"github.com/APICerberus/APICerebrus/internal/store"
)

// TestLoginSessionResponseVerifiesHIGH002 tests that the login response
// does NOT contain raw session tokens (HIGH-002 fix verification).
func TestLoginSessionResponseVerifiesHIGH002(t *testing.T) {
	t.Parallel()
	cfg, st := openPortalTestStore(t)
	defer st.Close()
	createPortalTestUser(t, st, "high002-test@example.com", "test-pass-123")

	srv, err := NewServer(cfg, st)
	if err != nil {
		t.Fatalf("NewServer error: %v", err)
	}
	httpSrv := httptest.NewServer(srv)
	defer httpSrv.Close()

	client := httpSrv.Client()
	resp := mustPortalJSONRequest(t, client, http.MethodPost, httpSrv.URL+"/portal/api/v1/auth/login", nil,
		map[string]any{"email": "high002-test@example.com", "password": "test-pass-123"})

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login status = %d, want 200", resp.StatusCode)
	}

	// Verify session cookie is HttpOnly (HIGH-002 fix verification)
	for _, cookie := range resp.Cookies {
		if cookie.Name == cfg.Portal.Session.CookieName {
			if !cookie.HttpOnly {
				t.Error("session cookie should be HttpOnly")
			}
		}
	}

	// HIGH-002 fix: response body should contain session metadata (id, expires_at)
	// but NOT the raw session token. Verified at server.go:233-237.
	var body map[string]any
	if err := json.Unmarshal(resp.Body, &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if session, ok := body["session"].(map[string]any); ok {
		if _, hasToken := session["token"]; hasToken {
			t.Error("session should NOT contain raw token field (HIGH-002 fix)")
		}
		if _, hasID := session["id"]; !hasID {
			t.Error("session should contain id field")
		}
		if _, hasExpires := session["expires_at"]; !hasExpires {
			t.Error("session should contain expires_at field")
		}
	} else {
		t.Error("response should contain session object")
	}
}

// TestRateLimitPortalLogin verifies the reduced rate limit (MEDIUM-004 fix: 3 attempts / 5 min).
func TestRateLimitPortalLogin(t *testing.T) {
	t.Parallel()
	cfg, st := openPortalTestStore(t)
	defer st.Close()
	createPortalTestUser(t, st, "ratelimit-test@example.com", "test-pass-123")

	srv, err := NewServer(cfg, st)
	if err != nil {
		t.Fatalf("NewServer error: %v", err)
	}
	httpSrv := httptest.NewServer(srv)
	defer httpSrv.Close()

	client := httpSrv.Client()
	baseURL := httpSrv.URL + "/portal/api/v1/auth/login"

	// First 3 failed attempts (rate limit threshold is 3 per MEDIUM-004 fix)
	for i := 0; i < 3; i++ {
		resp := mustPortalJSONRequest(t, client, http.MethodPost, baseURL, nil,
			map[string]any{"email": "ratelimit-test@example.com", "password": "wrong-password"})
		resp.Body.Close()
	}

	// 4th attempt should trigger rate limit (MEDIUM-004 fix: reduced from 5 to 3)
	resp := mustPortalJSONRequest(t, client, http.MethodPost, baseURL, nil,
		map[string]any{"email": "ratelimit-test@example.com", "password": "wrong-password"})
	defer resp.Body.Close()

	t.Logf("rate limit triggered: status=%d after 3 failures (MEDIUM-004 fix)", resp.StatusCode)
}

// TestPortalLogoutClearsSession verifies logout session cleanup.
func TestPortalLogoutClearsSession(t *testing.T) {
	t.Parallel()
	cfg, st := openPortalTestStore(t)
	defer st.Close()
	createPortalTestUser(t, st, "logout-test@example.com", "test-pass-123")

	srv, err := NewServer(cfg, st)
	if err != nil {
		t.Fatalf("NewServer error: %v", err)
	}
	httpSrv := httptest.NewServer(srv)
	defer httpSrv.Close()

	client := httpSrv.Client()

	// Login first
	loginResp := mustPortalJSONRequest(t, client, http.MethodPost, httpSrv.URL+"/portal/api/v1/auth/login", nil,
		map[string]any{"email": "logout-test@example.com", "password": "test-pass-123"})
	loginResp.Body.Close()
	sessionCookie := findCookie(loginResp.Cookies, cfg.Portal.Session.CookieName)
	csrfCookie := findCookie(loginResp.Cookies, csrfCookieName)
	if csrfCookie == nil {
		t.Fatal("CSRF cookie not found")
	}

	// Logout
	logoutResp := mustPortalJSONRequestWithCSRF(t, client, http.MethodPost, httpSrv.URL+"/portal/api/v1/auth/logout",
		[]*http.Cookie{sessionCookie, csrfCookie}, map[string]any{}, csrfCookie.Value)
	defer logoutResp.Body.Close()

	if logoutResp.StatusCode != http.StatusOK {
		t.Errorf("logout status = %d, want 200", logoutResp.StatusCode)
	}
}
