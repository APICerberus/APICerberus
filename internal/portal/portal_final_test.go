package portal

import (
	"context"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/APICerberus/APICerebrus/internal/store"
)

// MockFS implements fs.FS for testing UI handler errors
type MockFS struct {
	OpenFunc func(name string) (fs.File, error)
}

func (m *MockFS) Open(name string) (fs.File, error) {
	if m.OpenFunc != nil {
		return m.OpenFunc(name)
	}
	return nil, fs.ErrNotExist
}

// MockFile implements fs.File for testing
type MockFile struct {
	name    string
	content []byte
	pos     int
	statErr error
}

func (m *MockFile) Stat() (fs.FileInfo, error) {
	if m.statErr != nil {
		return nil, m.statErr
	}
	return &MockFileInfo{name: m.name, size: int64(len(m.content))}, nil
}

func (m *MockFile) Read(p []byte) (int, error) {
	if m.pos >= len(m.content) {
		return 0, io.EOF
	}
	n := copy(p, m.content[m.pos:])
	m.pos += n
	return n, nil
}

func (m *MockFile) Close() error { return nil }

// MockFileInfo implements fs.FileInfo
type MockFileInfo struct {
	name string
	size int64
}

func (m *MockFileInfo) Name() string       { return m.name }
func (m *MockFileInfo) Size() int64        { return m.size }
func (m *MockFileInfo) Mode() fs.FileMode  { return 0644 }
func (m *MockFileInfo) ModTime() time.Time { return time.Now() }
func (m *MockFileInfo) IsDir() bool        { return false }
func (m *MockFileInfo) Sys() any           { return nil }


// Test newPortalUIHandler with non-GET method
func TestNewPortalUIHandler_NonGetMethod(t *testing.T) {
	t.Parallel()

	cfg, st := openPortalTestStore(t)
	defer st.Close()

	srv, err := NewServer(cfg, st)
	if err != nil {
		t.Fatalf("NewServer error: %v", err)
	}

	// Test POST to UI path returns 405
	req := httptest.NewRequest(http.MethodPost, "/portal/", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 for POST, got %d", w.Code)
	}
}

// Test newPortalUIHandler with non-existent asset
func TestNewPortalUIHandler_NonExistentAsset(t *testing.T) {
	t.Parallel()

	cfg, st := openPortalTestStore(t)
	defer st.Close()

	srv, err := NewServer(cfg, st)
	if err != nil {
		t.Fatalf("NewServer error: %v", err)
	}

	// Test request for non-existent asset falls back to index.html
	req := httptest.NewRequest(http.MethodGet, "/portal/nonexistent-asset.js", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	// Should serve index.html for SPA routing
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for SPA fallback, got %d", w.Code)
	}
}

// Test portalAssetExists with nil filesystem
func TestPortalAssetExists_NilFS(t *testing.T) {
	t.Parallel()

	exists := portalAssetExists(nil, "test.txt")
	if exists {
		t.Error("expected false for nil filesystem")
	}
}

// Test startRateLimitCleanup and cleanupOldRateLimitEntries
func TestStartRateLimitCleanup_Manual(t *testing.T) {
	t.Parallel()

	cfg, st := openPortalTestStore(t)
	defer st.Close()

	srv, err := NewServer(cfg, st)
	if err != nil {
		t.Fatalf("NewServer error: %v", err)
	}

	// Add some rate limit entries
	srv.rlMu.Lock()
	srv.rlAttempts["192.168.1.1"] = &loginAuthAttempts{
		count:     3,
		firstSeen: time.Now().Add(-31 * time.Minute), // Old entry
		lastSeen:  time.Now().Add(-31 * time.Minute),
		blocked:   false,
	}
	srv.rlAttempts["192.168.1.2"] = &loginAuthAttempts{
		count:     5,
		firstSeen: time.Now(), // Recent entry
		lastSeen:  time.Now(),
		blocked:   true,
	}
	srv.rlMu.Unlock()

	// Manually trigger cleanup
	srv.cleanupOldRateLimitEntries()

	// Check that old entry was removed
	srv.rlMu.RLock()
	_, exists := srv.rlAttempts["192.168.1.1"]
	srv.rlMu.RUnlock()
	if exists {
		t.Error("expected old rate limit entry to be cleaned up")
	}

	// Check that recent entry still exists
	srv.rlMu.RLock()
	_, exists = srv.rlAttempts["192.168.1.2"]
	srv.rlMu.RUnlock()
	if !exists {
		t.Error("expected recent rate limit entry to still exist")
	}
}

// Test rate limiting with expired block
func TestRateLimiting_ExpiredBlock(t *testing.T) {
	t.Parallel()

	cfg, st := openPortalTestStore(t)
	defer st.Close()

	srv, err := NewServer(cfg, st)
	if err != nil {
		t.Fatalf("NewServer error: %v", err)
	}

	// Add an old blocked entry (more than 30 minutes ago)
	srv.rlMu.Lock()
	srv.rlAttempts["192.168.1.100"] = &loginAuthAttempts{
		count:     5,
		firstSeen: time.Now().Add(-40 * time.Minute),
		lastSeen:  time.Now().Add(-31 * time.Minute), // Block expired
		blocked:   true,
	}
	srv.rlMu.Unlock()

	// Should not be rate limited anymore
	if srv.isRateLimited("192.168.1.100") {
		t.Error("expected rate limit to be expired")
	}
}

// Test logout with session from context
func TestLogout_WithSessionContext(t *testing.T) {
	t.Parallel()

	cfg, st := openPortalTestStore(t)
	defer st.Close()
	user := createPortalTestUserWithID(t, st, "logout-ctx@example.com", "portal-pass")

	srv, err := NewServer(cfg, st)
	if err != nil {
		t.Fatalf("NewServer error: %v", err)
	}

	// Create a session
	token, err := store.GenerateSessionToken()
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	session := &store.Session{
		UserID:    user.ID,
		TokenHash: store.HashSessionToken(token),
		ExpiresAt: time.Now().Add(2 * time.Hour),
	}
	if err := st.Sessions().Create(session); err != nil {
		t.Fatalf("create session: %v", err)
	}

	// Create request with session context
	ctx := context.WithValue(context.Background(), contextUserKey, user)
	ctx = context.WithValue(ctx, contextSessionKey, session)
	req := httptest.NewRequest(http.MethodPost, "/portal/api/v1/auth/logout", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	srv.logout(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// Test logout with cookie but no session context
func TestLogout_WithCookieNoContext(t *testing.T) {
	t.Parallel()

	cfg, st := openPortalTestStore(t)
	defer st.Close()
	user := createPortalTestUserWithID(t, st, "logout-cookie@example.com", "portal-pass")

	srv, err := NewServer(cfg, st)
	if err != nil {
		t.Fatalf("NewServer error: %v", err)
	}

	// Create a session
	token, err := store.GenerateSessionToken()
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	session := &store.Session{
		UserID:    user.ID,
		TokenHash: store.HashSessionToken(token),
		ExpiresAt: time.Now().Add(2 * time.Hour),
	}
	if err := st.Sessions().Create(session); err != nil {
		t.Fatalf("create session: %v", err)
	}

	// Create request with cookie but no session context
	req := httptest.NewRequest(http.MethodPost, "/portal/api/v1/auth/logout", nil)
	req.AddCookie(&http.Cookie{
		Name:  cfg.Portal.Session.CookieName,
		Value: token,
	})
	w := httptest.NewRecorder()

	srv.logout(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// Test logout without session or valid cookie
func TestLogout_NoSessionOrCookie(t *testing.T) {
	t.Parallel()

	cfg, st := openPortalTestStore(t)
	defer st.Close()

	srv, err := NewServer(cfg, st)
	if err != nil {
		t.Fatalf("NewServer error: %v", err)
	}

	// Create request without session context and without valid cookie
	req := httptest.NewRequest(http.MethodPost, "/portal/api/v1/auth/logout", nil)
	w := httptest.NewRecorder()

	srv.logout(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// Test me endpoint with nil user
func TestMe_NilUser(t *testing.T) {
	t.Parallel()

	cfg, st := openPortalTestStore(t)
	defer st.Close()

	srv, err := NewServer(cfg, st)
	if err != nil {
		t.Fatalf("NewServer error: %v", err)
	}

	// Create request with nil user in context
	ctx := context.WithValue(context.Background(), contextUserKey, (*store.User)(nil))
	req := httptest.NewRequest(http.MethodGet, "/portal/api/v1/auth/me", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	srv.me(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

// Test withSession middleware with expired session
func TestWithSession_ExpiredSession(t *testing.T) {
	t.Parallel()

	cfg, st := openPortalTestStore(t)
	defer st.Close()
	user := createPortalTestUserWithID(t, st, "expired-session@example.com", "portal-pass")

	srv, err := NewServer(cfg, st)
	if err != nil {
		t.Fatalf("NewServer error: %v", err)
	}

	// Create an expired session
	token, err := store.GenerateSessionToken()
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	session := &store.Session{
		UserID:    user.ID,
		TokenHash: store.HashSessionToken(token),
		ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired
	}
	if err := st.Sessions().Create(session); err != nil {
		t.Fatalf("create session: %v", err)
	}

	// Create request with expired session cookie
	req := httptest.NewRequest(http.MethodGet, "/portal/api/v1/auth/me", nil)
	req.AddCookie(&http.Cookie{
		Name:  cfg.Portal.Session.CookieName,
		Value: token,
	})
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for expired session, got %d", w.Code)
	}
}

// Test withSession middleware with inactive user
func TestWithSession_InactiveUser(t *testing.T) {
	t.Parallel()

	cfg, st := openPortalTestStore(t)
	defer st.Close()

	// Create inactive user
	hash, _ := store.HashPassword("portal-pass")
	user := &store.User{
		Email:        "inactive@example.com",
		Name:         "Inactive User",
		PasswordHash: hash,
		Role:         "user",
		Status:       "inactive",
	}
	if err := st.Users().Create(user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	srv, err := NewServer(cfg, st)
	if err != nil {
		t.Fatalf("NewServer error: %v", err)
	}

	// Create session for inactive user
	token, err := store.GenerateSessionToken()
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	session := &store.Session{
		UserID:    user.ID,
		TokenHash: store.HashSessionToken(token),
		ExpiresAt: time.Now().Add(2 * time.Hour),
	}
	if err := st.Sessions().Create(session); err != nil {
		t.Fatalf("create session: %v", err)
	}

	// Create request with session cookie
	req := httptest.NewRequest(http.MethodGet, "/portal/api/v1/auth/me", nil)
	req.AddCookie(&http.Cookie{
		Name:  cfg.Portal.Session.CookieName,
		Value: token,
	})
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for inactive user, got %d", w.Code)
	}
}

// Test withSession middleware with deleted user
func TestWithSession_DeletedUser(t *testing.T) {
	t.Parallel()

	cfg, st := openPortalTestStore(t)
	defer st.Close()
	user := createPortalTestUserWithID(t, st, "deleted@example.com", "portal-pass")

	srv, err := NewServer(cfg, st)
	if err != nil {
		t.Fatalf("NewServer error: %v", err)
	}

	// Create session
	token, err := store.GenerateSessionToken()
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	session := &store.Session{
		UserID:    user.ID,
		TokenHash: store.HashSessionToken(token),
		ExpiresAt: time.Now().Add(2 * time.Hour),
	}
	if err := st.Sessions().Create(session); err != nil {
		t.Fatalf("create session: %v", err)
	}

	// Delete the user
	if err := st.Users().Delete(user.ID); err != nil {
		t.Fatalf("delete user: %v", err)
	}

	// Create request with session cookie for deleted user
	req := httptest.NewRequest(http.MethodGet, "/portal/api/v1/auth/me", nil)
	req.AddCookie(&http.Cookie{
		Name:  cfg.Portal.Session.CookieName,
		Value: token,
	})
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for deleted user, got %d", w.Code)
	}
}

// Test userFromContext with nil context
func TestUserFromContext_NilContext(t *testing.T) {
	t.Parallel()

	user := userFromContext(nil)
	if user != nil {
		t.Error("expected nil user for nil context")
	}
}

// Test sessionFromContext with nil context
func TestSessionFromContext_NilContext(t *testing.T) {
	t.Parallel()

	session := sessionFromContext(nil)
	if session != nil {
		t.Error("expected nil session for nil context")
	}
}

// Test isUserActive with various statuses
func TestIsUserActive_VariousStatuses(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status   string
		expected bool
	}{
		{"", true},           // Empty status defaults to active
		{"active", true},     // Explicit active
		{"ACTIVE", true},     // Case insensitive
		{"inactive", false},  // Inactive
		{"INACTIVE", false},  // Case insensitive
		{"suspended", false}, // Suspended
		{"banned", false},    // Banned
		{"deleted", false},   // Deleted
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			user := &store.User{Status: tt.status}
			result := isUserActive(user)
			if result != tt.expected {
				t.Errorf("status %q: expected %v, got %v", tt.status, tt.expected, result)
			}
		})
	}
}

// Test isUserActive with nil user
func TestIsUserActive_NilUser(t *testing.T) {
	t.Parallel()

	result := isUserActive(nil)
	if result {
		t.Error("expected false for nil user")
	}
}

// Test sanitizeUser with nil user
func TestSanitizeUser_NilUser(t *testing.T) {
	t.Parallel()

	result := sanitizeUser(nil)
	if result == nil {
		t.Error("expected non-nil map for nil user")
	}
	if len(result) != 0 {
		t.Errorf("expected empty map, got %v", result)
	}
}

// Test getClientIP with various scenarios
func TestGetClientIP_VariousScenarios(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		xff        string
		remoteAddr string
		expected   string
	}{
		{
			name:       "X-Forwarded-For with single IP",
			xff:        "192.168.1.1",
			remoteAddr: "10.0.0.1:12345",
			expected:   "192.168.1.1",
		},
		{
			name:       "X-Forwarded-For with multiple IPs",
			xff:        "192.168.1.1, 10.0.0.2, 10.0.0.3",
			remoteAddr: "10.0.0.1:12345",
			expected:   "192.168.1.1",
		},
		{
			name:       "No X-Forwarded-For",
			xff:        "",
			remoteAddr: "10.0.0.1:12345",
			expected:   "10.0.0.1",
		},
		{
			name:       "RemoteAddr without port",
			xff:        "",
			remoteAddr: "10.0.0.1",
			expected:   "10.0.0.1",
		},
		{
			name:       "Empty X-Forwarded-For with spaces",
			xff:        "   ",
			remoteAddr: "10.0.0.1:12345",
			expected:   "10.0.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.xff != "" {
				req.Header.Set("X-Forwarded-For", tt.xff)
			}
			req.RemoteAddr = tt.remoteAddr

			result := getClientIP(req)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// Test getClientIP with nil request
func TestGetClientIP_NilRequest(t *testing.T) {
	t.Parallel()

	result := getClientIP(nil)
	if result != "" {
		t.Errorf("expected empty string for nil request, got %q", result)
	}
}

// Test extractClientIP with IPv6
func TestExtractClientIP_IPv6(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "[::1]:12345"

	result := extractClientIP(req)
	if result != "::1" {
		t.Errorf("expected ::1, got %q", result)
	}
}

// Test sessionMaxAge with zero/negative values
func TestSessionMaxAge_Default(t *testing.T) {
	t.Parallel()

	cfg, st := openPortalTestStore(t)
	defer st.Close()

	// Set MaxAge to 0 (should default to 24 hours)
	cfg.Portal.Session.MaxAge = 0

	srv, err := NewServer(cfg, st)
	if err != nil {
		t.Fatalf("NewServer error: %v", err)
	}

	maxAge := srv.sessionMaxAge()
	expected := 24 * time.Hour
	if maxAge != expected {
		t.Errorf("expected %v, got %v", expected, maxAge)
	}
}

// Test sessionMaxAge with negative value
func TestSessionMaxAge_Negative(t *testing.T) {
	t.Parallel()

	cfg, st := openPortalTestStore(t)
	defer st.Close()

	// Set MaxAge to negative (should default to 24 hours)
	cfg.Portal.Session.MaxAge = -1 * time.Hour

	srv, err := NewServer(cfg, st)
	if err != nil {
		t.Fatalf("NewServer error: %v", err)
	}

	maxAge := srv.sessionMaxAge()
	expected := 24 * time.Hour
	if maxAge != expected {
		t.Errorf("expected %v, got %v", expected, maxAge)
	}
}

// Test sessionCookieName with empty config
func TestSessionCookieName_Empty(t *testing.T) {
	t.Parallel()

	cfg, st := openPortalTestStore(t)
	defer st.Close()

	// Set CookieName to empty (should default)
	cfg.Portal.Session.CookieName = ""

	srv, err := NewServer(cfg, st)
	if err != nil {
		t.Fatalf("NewServer error: %v", err)
	}

	name := srv.sessionCookieName()
	expected := "apicerberus_session"
	if name != expected {
		t.Errorf("expected %q, got %q", expected, name)
	}
}

// Test sessionCookiePath with empty prefix
func TestSessionCookiePath_EmptyPrefix(t *testing.T) {
	t.Parallel()

	cfg, st := openPortalTestStore(t)
	defer st.Close()

	// Set PathPrefix to empty
	cfg.Portal.PathPrefix = ""

	srv, err := NewServer(cfg, st)
	if err != nil {
		t.Fatalf("NewServer error: %v", err)
	}

	path := srv.sessionCookiePath()
	expected := "/"
	if path != expected {
		t.Errorf("expected %q, got %q", expected, path)
	}
}

// Test normalizePortalPathPrefix with various inputs
func TestNormalizePortalPathPrefix_VariousInputs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"/", ""},
		{"portal", "/portal"},
		{"/portal", "/portal"},
		{"/portal/", "/portal"},
		{"portal/", "/portal"},
		{"  /portal/  ", "/portal"},
		{"   ", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizePortalPathPrefix(tt.input)
			if result != tt.expected {
				t.Errorf("input %q: expected %q, got %q", tt.input, tt.expected, result)
			}
		})
	}
}

// Test playgroundSend with missing API key
func TestPlaygroundSend_MissingAPIKey(t *testing.T) {
	t.Parallel()

	cfg, st := openPortalTestStore(t)
	defer st.Close()
	createPortalTestUser(t, st, "playground-nokey@example.com", "portal-pass")

	srv, err := NewServer(cfg, st)
	if err != nil {
		t.Fatalf("NewServer error: %v", err)
	}

	httpSrv := httptest.NewServer(srv)
	defer httpSrv.Close()

	// Login first
	loginResp := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPost, httpSrv.URL+"/portal/api/v1/auth/login", nil, map[string]any{
		"email":    "playground-nokey@example.com",
		"password": "portal-pass",
	})
	sessionCookie := findCookie(loginResp.Cookies, cfg.Portal.Session.CookieName)

	// Send request without API key
	playgroundResp := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPost, httpSrv.URL+"/portal/api/v1/playground/send", []*http.Cookie{sessionCookie}, map[string]any{
		"method": "GET",
		"path":   "/api/test",
	})

	if playgroundResp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for missing API key, got %d", playgroundResp.StatusCode)
	}
}

// Test playgroundSend with invalid path
func TestPlaygroundSend_InvalidPath(t *testing.T) {
	t.Parallel()

	cfg, st := openPortalTestStore(t)
	defer st.Close()
	createPortalTestUser(t, st, "playground-path@example.com", "portal-pass")

	srv, err := NewServer(cfg, st)
	if err != nil {
		t.Fatalf("NewServer error: %v", err)
	}

	httpSrv := httptest.NewServer(srv)
	defer httpSrv.Close()

	// Login first
	loginResp := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPost, httpSrv.URL+"/portal/api/v1/auth/login", nil, map[string]any{
		"email":    "playground-path@example.com",
		"password": "portal-pass",
	})
	sessionCookie := findCookie(loginResp.Cookies, cfg.Portal.Session.CookieName)

	// Send request with invalid path (not starting with /)
	playgroundResp := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPost, httpSrv.URL+"/portal/api/v1/playground/send", []*http.Cookie{sessionCookie}, map[string]any{
		"method":  "GET",
		"path":    "api/test",
		"api_key": "test-key",
	})

	if playgroundResp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid path, got %d", playgroundResp.StatusCode)
	}
}

// Test changePassword with invalid JSON
func TestChangePassword_InvalidJSON(t *testing.T) {
	t.Parallel()

	cfg, st := openPortalTestStore(t)
	defer st.Close()
	createPortalTestUser(t, st, "change-pwd-json@example.com", "portal-pass")

	srv, err := NewServer(cfg, st)
	if err != nil {
		t.Fatalf("NewServer error: %v", err)
	}

	httpSrv := httptest.NewServer(srv)
	defer httpSrv.Close()

	// Login first
	loginResp := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPost, httpSrv.URL+"/portal/api/v1/auth/login", nil, map[string]any{
		"email":    "change-pwd-json@example.com",
		"password": "portal-pass",
	})
	sessionCookie := findCookie(loginResp.Cookies, cfg.Portal.Session.CookieName)

	// Send invalid JSON
	client := httpSrv.Client()
	req, _ := http.NewRequest(http.MethodPut, httpSrv.URL+"/portal/api/v1/auth/password", strings.NewReader(`{invalid json`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(sessionCookie)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

// Test changePassword with missing passwords
func TestChangePassword_MissingPasswords(t *testing.T) {
	t.Parallel()

	cfg, st := openPortalTestStore(t)
	defer st.Close()
	createPortalTestUser(t, st, "change-pwd-missing@example.com", "portal-pass")

	srv, err := NewServer(cfg, st)
	if err != nil {
		t.Fatalf("NewServer error: %v", err)
	}

	httpSrv := httptest.NewServer(srv)
	defer httpSrv.Close()

	// Login first
	loginResp := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPost, httpSrv.URL+"/portal/api/v1/auth/login", nil, map[string]any{
		"email":    "change-pwd-missing@example.com",
		"password": "portal-pass",
	})
	sessionCookie := findCookie(loginResp.Cookies, cfg.Portal.Session.CookieName)

	// Send empty passwords
	pwdResp := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPut, httpSrv.URL+"/portal/api/v1/auth/password", []*http.Cookie{sessionCookie}, map[string]any{
		"old_password": "",
		"new_password": "",
	})

	if pwdResp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", pwdResp.StatusCode)
	}
}

// Test changePassword with incorrect old password
func TestChangePassword_IncorrectOldPassword(t *testing.T) {
	t.Parallel()

	cfg, st := openPortalTestStore(t)
	defer st.Close()
	createPortalTestUser(t, st, "change-pwd-wrong@example.com", "portal-pass")

	srv, err := NewServer(cfg, st)
	if err != nil {
		t.Fatalf("NewServer error: %v", err)
	}

	httpSrv := httptest.NewServer(srv)
	defer httpSrv.Close()

	// Login first
	loginResp := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPost, httpSrv.URL+"/portal/api/v1/auth/login", nil, map[string]any{
		"email":    "change-pwd-wrong@example.com",
		"password": "portal-pass",
	})
	sessionCookie := findCookie(loginResp.Cookies, cfg.Portal.Session.CookieName)

	// Send wrong old password
	pwdResp := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPut, httpSrv.URL+"/portal/api/v1/auth/password", []*http.Cookie{sessionCookie}, map[string]any{
		"old_password": "wrong-password",
		"new_password": "new-password",
	})

	if pwdResp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", pwdResp.StatusCode)
	}
}

// Test renameMyAPIKey with empty name
func TestRenameMyAPIKey_EmptyName(t *testing.T) {
	t.Parallel()

	cfg, st := openPortalTestStore(t)
	defer st.Close()
	createPortalTestUser(t, st, "rename-key-empty@example.com", "portal-pass")

	srv, err := NewServer(cfg, st)
	if err != nil {
		t.Fatalf("NewServer error: %v", err)
	}

	httpSrv := httptest.NewServer(srv)
	defer httpSrv.Close()

	// Login first
	loginResp := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPost, httpSrv.URL+"/portal/api/v1/auth/login", nil, map[string]any{
		"email":    "rename-key-empty@example.com",
		"password": "portal-pass",
	})
	sessionCookie := findCookie(loginResp.Cookies, cfg.Portal.Session.CookieName)

	// Send empty name
	renameResp := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPut, httpSrv.URL+"/portal/api/v1/api-keys/some-id", []*http.Cookie{sessionCookie}, map[string]any{
		"name": "",
	})

	if renameResp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", renameResp.StatusCode)
	}
}

// Test addMyIP with invalid JSON
func TestAddMyIP_InvalidJSON(t *testing.T) {
	t.Parallel()

	cfg, st := openPortalTestStore(t)
	defer st.Close()
	createPortalTestUser(t, st, "add-ip-json@example.com", "portal-pass")

	srv, err := NewServer(cfg, st)
	if err != nil {
		t.Fatalf("NewServer error: %v", err)
	}

	httpSrv := httptest.NewServer(srv)
	defer httpSrv.Close()

	// Login first
	loginResp := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPost, httpSrv.URL+"/portal/api/v1/auth/login", nil, map[string]any{
		"email":    "add-ip-json@example.com",
		"password": "portal-pass",
	})
	sessionCookie := findCookie(loginResp.Cookies, cfg.Portal.Session.CookieName)

	// Send invalid JSON
	client := httpSrv.Client()
	req, _ := http.NewRequest(http.MethodPost, httpSrv.URL+"/portal/api/v1/security/ip-whitelist", strings.NewReader(`{invalid json`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(sessionCookie)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

// Test addMyIP with no IPs provided
func TestAddMyIP_NoIPs(t *testing.T) {
	t.Parallel()

	cfg, st := openPortalTestStore(t)
	defer st.Close()
	createPortalTestUser(t, st, "add-ip-none@example.com", "portal-pass")

	srv, err := NewServer(cfg, st)
	if err != nil {
		t.Fatalf("NewServer error: %v", err)
	}

	httpSrv := httptest.NewServer(srv)
	defer httpSrv.Close()

	// Login first
	loginResp := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPost, httpSrv.URL+"/portal/api/v1/auth/login", nil, map[string]any{
		"email":    "add-ip-none@example.com",
		"password": "portal-pass",
	})
	sessionCookie := findCookie(loginResp.Cookies, cfg.Portal.Session.CookieName)

	// Send empty request
	ipResp := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPost, httpSrv.URL+"/portal/api/v1/security/ip-whitelist", []*http.Cookie{sessionCookie}, map[string]any{})

	if ipResp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", ipResp.StatusCode)
	}
}

// Test updateProfile with invalid JSON
func TestUpdateProfile_InvalidJSON(t *testing.T) {
	t.Parallel()

	cfg, st := openPortalTestStore(t)
	defer st.Close()
	createPortalTestUser(t, st, "profile-json@example.com", "portal-pass")

	srv, err := NewServer(cfg, st)
	if err != nil {
		t.Fatalf("NewServer error: %v", err)
	}

	httpSrv := httptest.NewServer(srv)
	defer httpSrv.Close()

	// Login first
	loginResp := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPost, httpSrv.URL+"/portal/api/v1/auth/login", nil, map[string]any{
		"email":    "profile-json@example.com",
		"password": "portal-pass",
	})
	sessionCookie := findCookie(loginResp.Cookies, cfg.Portal.Session.CookieName)

	// Send invalid JSON
	client := httpSrv.Client()
	req, _ := http.NewRequest(http.MethodPut, httpSrv.URL+"/portal/api/v1/settings/profile", strings.NewReader(`{invalid json`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(sessionCookie)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

// Test updateNotifications with invalid JSON
func TestUpdateNotifications_InvalidJSON(t *testing.T) {
	t.Parallel()

	cfg, st := openPortalTestStore(t)
	defer st.Close()
	createPortalTestUser(t, st, "notif-json@example.com", "portal-pass")

	srv, err := NewServer(cfg, st)
	if err != nil {
		t.Fatalf("NewServer error: %v", err)
	}

	httpSrv := httptest.NewServer(srv)
	defer httpSrv.Close()

	// Login first
	loginResp := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPost, httpSrv.URL+"/portal/api/v1/auth/login", nil, map[string]any{
		"email":    "notif-json@example.com",
		"password": "portal-pass",
	})
	sessionCookie := findCookie(loginResp.Cookies, cfg.Portal.Session.CookieName)

	// Send invalid JSON
	client := httpSrv.Client()
	req, _ := http.NewRequest(http.MethodPut, httpSrv.URL+"/portal/api/v1/settings/notifications", strings.NewReader(`{invalid json`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(sessionCookie)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

// Test saveTemplate with invalid JSON
func TestSaveTemplate_InvalidJSON(t *testing.T) {
	t.Parallel()

	cfg, st := openPortalTestStore(t)
	defer st.Close()
	createPortalTestUser(t, st, "template-json@example.com", "portal-pass")

	srv, err := NewServer(cfg, st)
	if err != nil {
		t.Fatalf("NewServer error: %v", err)
	}

	httpSrv := httptest.NewServer(srv)
	defer httpSrv.Close()

	// Login first
	loginResp := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPost, httpSrv.URL+"/portal/api/v1/auth/login", nil, map[string]any{
		"email":    "template-json@example.com",
		"password": "portal-pass",
	})
	sessionCookie := findCookie(loginResp.Cookies, cfg.Portal.Session.CookieName)

	// Send invalid JSON
	client := httpSrv.Client()
	req, _ := http.NewRequest(http.MethodPost, httpSrv.URL+"/portal/api/v1/playground/templates", strings.NewReader(`{invalid json`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(sessionCookie)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

// Test purchaseCredits with invalid JSON
func TestPurchaseCredits_InvalidJSON(t *testing.T) {
	t.Parallel()

	cfg, st := openPortalTestStore(t)
	defer st.Close()
	createPortalTestUser(t, st, "purchase-json@example.com", "portal-pass")

	srv, err := NewServer(cfg, st)
	if err != nil {
		t.Fatalf("NewServer error: %v", err)
	}

	httpSrv := httptest.NewServer(srv)
	defer httpSrv.Close()

	// Login first
	loginResp := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPost, httpSrv.URL+"/portal/api/v1/auth/login", nil, map[string]any{
		"email":    "purchase-json@example.com",
		"password": "portal-pass",
	})
	sessionCookie := findCookie(loginResp.Cookies, cfg.Portal.Session.CookieName)

	// Send invalid JSON
	client := httpSrv.Client()
	req, _ := http.NewRequest(http.MethodPost, httpSrv.URL+"/portal/api/v1/credits/purchase", strings.NewReader(`{invalid json`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(sessionCookie)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

// Test purchaseCredits with zero amount
func TestPurchaseCredits_ZeroAmount(t *testing.T) {
	t.Parallel()

	cfg, st := openPortalTestStore(t)
	defer st.Close()
	createPortalTestUser(t, st, "purchase-zero@example.com", "portal-pass")

	srv, err := NewServer(cfg, st)
	if err != nil {
		t.Fatalf("NewServer error: %v", err)
	}

	httpSrv := httptest.NewServer(srv)
	defer httpSrv.Close()

	// Login first
	loginResp := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPost, httpSrv.URL+"/portal/api/v1/auth/login", nil, map[string]any{
		"email":    "purchase-zero@example.com",
		"password": "portal-pass",
	})
	sessionCookie := findCookie(loginResp.Cookies, cfg.Portal.Session.CookieName)

	// Send zero amount
	purchaseResp := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPost, httpSrv.URL+"/portal/api/v1/credits/purchase", []*http.Cookie{sessionCookie}, map[string]any{
		"amount": 0,
	})

	if purchaseResp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", purchaseResp.StatusCode)
	}
}

// Test purchaseCredits with negative amount
func TestPurchaseCredits_NegativeAmount(t *testing.T) {
	t.Parallel()

	cfg, st := openPortalTestStore(t)
	defer st.Close()
	createPortalTestUser(t, st, "purchase-neg@example.com", "portal-pass")

	srv, err := NewServer(cfg, st)
	if err != nil {
		t.Fatalf("NewServer error: %v", err)
	}

	httpSrv := httptest.NewServer(srv)
	defer httpSrv.Close()

	// Login first
	loginResp := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPost, httpSrv.URL+"/portal/api/v1/auth/login", nil, map[string]any{
		"email":    "purchase-neg@example.com",
		"password": "portal-pass",
	})
	sessionCookie := findCookie(loginResp.Cookies, cfg.Portal.Session.CookieName)

	// Send negative amount
	purchaseResp := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPost, httpSrv.URL+"/portal/api/v1/credits/purchase", []*http.Cookie{sessionCookie}, map[string]any{
		"amount": -100,
	})

	if purchaseResp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", purchaseResp.StatusCode)
	}
}

// Test playgroundSend with invalid JSON
func TestPlaygroundSend_InvalidJSON(t *testing.T) {
	t.Parallel()

	cfg, st := openPortalTestStore(t)
	defer st.Close()
	createPortalTestUser(t, st, "playground-json@example.com", "portal-pass")

	srv, err := NewServer(cfg, st)
	if err != nil {
		t.Fatalf("NewServer error: %v", err)
	}

	httpSrv := httptest.NewServer(srv)
	defer httpSrv.Close()

	// Login first
	loginResp := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPost, httpSrv.URL+"/portal/api/v1/auth/login", nil, map[string]any{
		"email":    "playground-json@example.com",
		"password": "portal-pass",
	})
	sessionCookie := findCookie(loginResp.Cookies, cfg.Portal.Session.CookieName)

	// Send invalid JSON
	client := httpSrv.Client()
	req, _ := http.NewRequest(http.MethodPost, httpSrv.URL+"/portal/api/v1/playground/send", strings.NewReader(`{invalid json`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(sessionCookie)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

// Test resolvePortalAssetPath with various scenarios
func TestResolvePortalAssetPath_VariousScenarios(t *testing.T) {
	t.Parallel()

	srv := &Server{pathPrefix: "/portal"}

	tests := []struct {
		cleanPath     string
		expectedReq   string
		expectedServe bool
	}{
		// When pathPrefix is "/portal", paths outside "/portal" don't serve UI
		{"", "", false},
		{".", "", false},
		{"/", "", false},
		{"/assets/main.js", "assets/main.js", true}, // Assets are always served
		{"/favicon.ico", "favicon.ico", true},       // Favicon is always served
		{"/portal", "", true},
		{"/portal/", "", true},
		{"/portal/page", "page", true},
		{"/other", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.cleanPath, func(t *testing.T) {
			req, serve := srv.resolvePortalAssetPath(tt.cleanPath)
			if req != tt.expectedReq {
				t.Errorf("expected request %q, got %q", tt.expectedReq, req)
			}
			if serve != tt.expectedServe {
				t.Errorf("expected serve %v, got %v", tt.expectedServe, serve)
			}
		})
	}
}

// Test resolvePortalAssetPath with empty prefix
func TestResolvePortalAssetPath_EmptyPrefix(t *testing.T) {
	t.Parallel()

	srv := &Server{pathPrefix: ""}

	tests := []struct {
		cleanPath     string
		expectedReq   string
		expectedServe bool
	}{
		{"", "", true},
		{"/", "", true},
		{"/assets/main.js", "assets/main.js", true},
		{"/page", "page", true},
	}

	for _, tt := range tests {
		t.Run(tt.cleanPath, func(t *testing.T) {
			req, serve := srv.resolvePortalAssetPath(tt.cleanPath)
			if req != tt.expectedReq {
				t.Errorf("expected request %q, got %q", tt.expectedReq, req)
			}
			if serve != tt.expectedServe {
				t.Errorf("expected serve %v, got %v", tt.expectedServe, serve)
			}
		})
	}
}

// Test portalAssetExists with directory
func TestPortalAssetExists_Directory_Test(t *testing.T) {
	t.Parallel()

	// Create a mock filesystem that returns a directory
	mockFS := &MockFS{
		OpenFunc: func(name string) (fs.File, error) {
			return &MockFile{name: name, content: []byte{}, statErr: nil}, nil
		},
	}

	// We can't easily test this without a real directory entry
	// but we can test that it returns false for errors
	exists := portalAssetExists(mockFS, "test.txt")
	if !exists {
		t.Error("expected true for existing file")
	}
}

// Test myBalance endpoint
func TestMyBalance_Final(t *testing.T) {
	t.Parallel()

	cfg, st := openPortalTestStore(t)
	defer st.Close()
	createPortalTestUser(t, st, "balance-final@example.com", "portal-pass")

	srv, err := NewServer(cfg, st)
	if err != nil {
		t.Fatalf("NewServer error: %v", err)
	}

	httpSrv := httptest.NewServer(srv)
	defer httpSrv.Close()

	// Login first
	loginResp := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPost, httpSrv.URL+"/portal/api/v1/auth/login", nil, map[string]any{
		"email":    "balance-final@example.com",
		"password": "portal-pass",
	})
	sessionCookie := findCookie(loginResp.Cookies, cfg.Portal.Session.CookieName)

	// Get balance
	balanceResp := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodGet, httpSrv.URL+"/portal/api/v1/credits/balance", []*http.Cookie{sessionCookie}, nil)

	if balanceResp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", balanceResp.StatusCode)
	}
}

// Test getProfile endpoint
func TestGetProfile_Final(t *testing.T) {
	t.Parallel()

	cfg, st := openPortalTestStore(t)
	defer st.Close()
	createPortalTestUser(t, st, "profile-final@example.com", "portal-pass")

	srv, err := NewServer(cfg, st)
	if err != nil {
		t.Fatalf("NewServer error: %v", err)
	}

	httpSrv := httptest.NewServer(srv)
	defer httpSrv.Close()

	// Login first
	loginResp := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPost, httpSrv.URL+"/portal/api/v1/auth/login", nil, map[string]any{
		"email":    "profile-final@example.com",
		"password": "portal-pass",
	})
	sessionCookie := findCookie(loginResp.Cookies, cfg.Portal.Session.CookieName)

	// Get profile
	profileResp := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodGet, httpSrv.URL+"/portal/api/v1/settings/profile", []*http.Cookie{sessionCookie}, nil)

	if profileResp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", profileResp.StatusCode)
	}
}
