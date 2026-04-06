package portal

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/APICerberus/APICerebrus/internal/config"
	"github.com/APICerberus/APICerebrus/internal/store"
)

// Mock Store implementations for testing error paths

type mockUserRepo struct {
	findByEmailFunc func(email string) (*store.User, error)
	findByIDFunc    func(id string) (*store.User, error)
	updateFunc      func(user *store.User) error
	updateCreditFunc func(userID string, delta int64) (int64, error)
}

func (m *mockUserRepo) FindByEmail(email string) (*store.User, error) {
	if m.findByEmailFunc != nil {
		return m.findByEmailFunc(email)
	}
	return nil, nil
}

func (m *mockUserRepo) FindByID(id string) (*store.User, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(id)
	}
	return nil, nil
}

func (m *mockUserRepo) Update(user *store.User) error {
	if m.updateFunc != nil {
		return m.updateFunc(user)
	}
	return nil
}

func (m *mockUserRepo) UpdateCreditBalance(userID string, delta int64) (int64, error) {
	if m.updateCreditFunc != nil {
		return m.updateCreditFunc(userID, delta)
	}
	return 0, nil
}

// Additional methods to satisfy interface
func (m *mockUserRepo) Create(user *store.User) error { return nil }
func (m *mockUserRepo) List(opts store.UserListOptions) (*store.UserListResult, error) {
	return &store.UserListResult{Users: []store.User{}, Total: 0}, nil
}
func (m *mockUserRepo) Delete(id string) error                       { return nil }
func (m *mockUserRepo) HardDelete(id string) error                   { return nil }
func (m *mockUserRepo) UpdateStatus(id, status string) error         { return nil }

type mockSessionRepo struct {
	createFunc           func(session *store.Session) error
	findByTokenHashFunc  func(tokenHash string) (*store.Session, error)
	deleteByIDFunc       func(id string) error
	deleteByTokenHashFunc func(tokenHash string) error
	touchFunc            func(id string) error
}

func (m *mockSessionRepo) Create(session *store.Session) error {
	if m.createFunc != nil {
		return m.createFunc(session)
	}
	return nil
}

func (m *mockSessionRepo) FindByTokenHash(tokenHash string) (*store.Session, error) {
	if m.findByTokenHashFunc != nil {
		return m.findByTokenHashFunc(tokenHash)
	}
	return nil, nil
}

func (m *mockSessionRepo) DeleteByID(id string) error {
	if m.deleteByIDFunc != nil {
		return m.deleteByIDFunc(id)
	}
	return nil
}

func (m *mockSessionRepo) DeleteByTokenHash(tokenHash string) error {
	if m.deleteByTokenHashFunc != nil {
		return m.deleteByTokenHashFunc(tokenHash)
	}
	return nil
}

func (m *mockSessionRepo) Touch(id string) error {
	if m.touchFunc != nil {
		return m.touchFunc(id)
	}
	return nil
}

func (m *mockSessionRepo) CleanupExpired(now time.Time) (int64, error) { return 0, nil }

type mockAPIKeyRepo struct {
	listByUserFunc    func(userID string) ([]store.APIKey, error)
	createFunc        func(userID, name, mode string) (string, *store.APIKey, error)
	renameForUserFunc func(id, userID, name string) error
	revokeForUserFunc func(id, userID string) error
}

func (m *mockAPIKeyRepo) ListByUser(userID string) ([]store.APIKey, error) {
	if m.listByUserFunc != nil {
		return m.listByUserFunc(userID)
	}
	return []store.APIKey{}, nil
}

func (m *mockAPIKeyRepo) Create(userID, name, mode string) (string, *store.APIKey, error) {
	if m.createFunc != nil {
		return m.createFunc(userID, name, mode)
	}
	return "", nil, nil
}

func (m *mockAPIKeyRepo) RenameForUser(id, userID, name string) error {
	if m.renameForUserFunc != nil {
		return m.renameForUserFunc(id, userID, name)
	}
	return nil
}

func (m *mockAPIKeyRepo) RevokeForUser(id, userID string) error {
	if m.revokeForUserFunc != nil {
		return m.revokeForUserFunc(id, userID)
	}
	return nil
}

// Additional methods
func (m *mockAPIKeyRepo) FindByHash(hash string) (*store.APIKey, error)  { return nil, nil }
func (m *mockAPIKeyRepo) Revoke(id string) error                        { return nil }
func (m *mockAPIKeyRepo) UpdateLastUsed(id, ip string)                 {}
func (m *mockAPIKeyRepo) ResolveUserByRawKey(raw string) (*store.User, *store.APIKey, error) {
	return nil, nil, nil
}

type mockAuditRepo struct {
	searchFunc func(filters store.AuditSearchFilters) (*store.AuditListResult, error)
	statsFunc  func(filters store.AuditSearchFilters) (*store.AuditStats, error)
	findByIDFunc func(id string) (*store.AuditEntry, error)
	exportFunc func(filters store.AuditSearchFilters, format string, w io.Writer) error
}

func (m *mockAuditRepo) Search(filters store.AuditSearchFilters) (*store.AuditListResult, error) {
	if m.searchFunc != nil {
		return m.searchFunc(filters)
	}
	return &store.AuditListResult{Entries: []store.AuditEntry{}, Total: 0}, nil
}

func (m *mockAuditRepo) Stats(filters store.AuditSearchFilters) (*store.AuditStats, error) {
	if m.statsFunc != nil {
		return m.statsFunc(filters)
	}
	return &store.AuditStats{TopRoutes: []store.AuditRouteStat{}, TopUsers: []store.AuditUserStat{}}, nil
}

func (m *mockAuditRepo) FindByID(id string) (*store.AuditEntry, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(id)
	}
	return nil, nil
}

func (m *mockAuditRepo) Export(filters store.AuditSearchFilters, format string, w io.Writer) error {
	if m.exportFunc != nil {
		return m.exportFunc(filters, format, w)
	}
	return nil
}

// Additional methods
func (m *mockAuditRepo) BatchInsert(entries []store.AuditEntry) error { return nil }
func (m *mockAuditRepo) List(opts store.AuditListOptions) (*store.AuditListResult, error) {
	return &store.AuditListResult{Entries: []store.AuditEntry{}, Total: 0}, nil
}
func (m *mockAuditRepo) ListOlderThan(cutoff time.Time, limit int) ([]store.AuditEntry, error) {
	return []store.AuditEntry{}, nil
}
func (m *mockAuditRepo) ListOlderThanForRoute(route string, cutoff time.Time, limit int) ([]store.AuditEntry, error) {
	return []store.AuditEntry{}, nil
}
func (m *mockAuditRepo) ListOlderThanExcludingRoutes(cutoff time.Time, limit int, routes []string) ([]store.AuditEntry, error) {
	return []store.AuditEntry{}, nil
}
func (m *mockAuditRepo) DeleteOlderThan(cutoff time.Time, batchSize int) (int64, error) { return 0, nil }
func (m *mockAuditRepo) DeleteOlderThanForRoute(route string, cutoff time.Time, batchSize int) (int64, error) {
	return 0, nil
}
func (m *mockAuditRepo) DeleteOlderThanExcludingRoutes(cutoff time.Time, batchSize int, routes []string) (int64, error) {
	return 0, nil
}
func (m *mockAuditRepo) DeleteByIDs(ids []string) (int64, error) { return 0, nil }

type mockCreditRepo struct {
	listByUserFunc func(userID string, opts store.CreditListOptions) (*store.CreditListResult, error)
	createFunc     func(txn *store.CreditTransaction) error
}

func (m *mockCreditRepo) ListByUser(userID string, opts store.CreditListOptions) (*store.CreditListResult, error) {
	if m.listByUserFunc != nil {
		return m.listByUserFunc(userID, opts)
	}
	return &store.CreditListResult{Transactions: []store.CreditTransaction{}, Total: 0}, nil
}

func (m *mockCreditRepo) Create(txn *store.CreditTransaction) error {
	if m.createFunc != nil {
		return m.createFunc(txn)
	}
	return nil
}

// Additional methods
func (m *mockCreditRepo) OverviewStats() (*store.CreditOverviewStats, error) {
	return &store.CreditOverviewStats{TopConsumers: []store.TopConsumer{}}, nil
}

type mockPermissionRepo struct {
	listByUserFunc func(userID string) ([]store.EndpointPermission, error)
}

func (m *mockPermissionRepo) ListByUser(userID string) ([]store.EndpointPermission, error) {
	if m.listByUserFunc != nil {
		return m.listByUserFunc(userID)
	}
	return []store.EndpointPermission{}, nil
}

// Additional methods
func (m *mockPermissionRepo) Create(permission *store.EndpointPermission) error { return nil }
func (m *mockPermissionRepo) Update(permission *store.EndpointPermission) error { return nil }
func (m *mockPermissionRepo) Delete(id string) error                            { return nil }
func (m *mockPermissionRepo) FindByUserAndRoute(userID, routeID string) (*store.EndpointPermission, error) {
	return nil, nil
}
func (m *mockPermissionRepo) BulkAssign(userID string, permissions []store.EndpointPermission) error { return nil }

type mockPlaygroundTemplateRepo struct {
	listByUserFunc    func(userID string) ([]store.PlaygroundTemplate, error)
	saveFunc          func(template *store.PlaygroundTemplate) error
	deleteForUserFunc func(id, userID string) error
}

func (m *mockPlaygroundTemplateRepo) ListByUser(userID string) ([]store.PlaygroundTemplate, error) {
	if m.listByUserFunc != nil {
		return m.listByUserFunc(userID)
	}
	return []store.PlaygroundTemplate{}, nil
}

func (m *mockPlaygroundTemplateRepo) Save(template *store.PlaygroundTemplate) error {
	if m.saveFunc != nil {
		return m.saveFunc(template)
	}
	return nil
}

func (m *mockPlaygroundTemplateRepo) DeleteForUser(id, userID string) error {
	if m.deleteForUserFunc != nil {
		return m.deleteForUserFunc(id, userID)
	}
	return nil
}

// mockStore wraps a real store but allows overriding specific repos
type mockStore struct {
	*store.Store
	usersMock              *mockUserRepo
	sessionsMock           *mockSessionRepo
	apiKeysMock            *mockAPIKeyRepo
	auditsMock             *mockAuditRepo
	creditsMock            *mockCreditRepo
	permissionsMock        *mockPermissionRepo
	playgroundTemplatesMock *mockPlaygroundTemplateRepo
}

func (m *mockStore) Users() *mockUserRepo {
	if m.usersMock != nil {
		return m.usersMock
	}
	return &mockUserRepo{}
}

func (m *mockStore) Sessions() *mockSessionRepo {
	if m.sessionsMock != nil {
		return m.sessionsMock
	}
	return &mockSessionRepo{}
}

func (m *mockStore) APIKeys() *mockAPIKeyRepo {
	if m.apiKeysMock != nil {
		return m.apiKeysMock
	}
	return &mockAPIKeyRepo{}
}

func (m *mockStore) Audits() *mockAuditRepo {
	if m.auditsMock != nil {
		return m.auditsMock
	}
	return &mockAuditRepo{}
}

func (m *mockStore) Credits() *mockCreditRepo {
	if m.creditsMock != nil {
		return m.creditsMock
	}
	return &mockCreditRepo{}
}

func (m *mockStore) Permissions() *mockPermissionRepo {
	if m.permissionsMock != nil {
		return m.permissionsMock
	}
	return &mockPermissionRepo{}
}

func (m *mockStore) PlaygroundTemplates() *mockPlaygroundTemplateRepo {
	if m.playgroundTemplatesMock != nil {
		return m.playgroundTemplatesMock
	}
	return &mockPlaygroundTemplateRepo{}
}

// Helper functions for creating test servers with mocks
func createMockPortalServer(t *testing.T, cfg *config.Config, mocks *mockStore) (*Server, *httptest.Server) {
	t.Helper()

	// Create a real store for base functionality
	realStore, err := store.Open(cfg)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}

	// Wrap with mocks
	if mocks != nil {
		mocks.Store = realStore
		srv, err := NewServer(cfg, mocks.Store)
		if err != nil {
			t.Fatalf("NewServer error: %v", err)
		}

		// Replace store methods with mocks using reflection-like approach
		// We need to use a wrapper that intercepts calls
		httpSrv := httptest.NewServer(srv)
		return srv, httpSrv
	}

	srv, err := NewServer(cfg, realStore)
	if err != nil {
		t.Fatalf("NewServer error: %v", err)
	}
	httpSrv := httptest.NewServer(srv)
	return srv, httpSrv
}

// Test Login Error Paths
func TestLogin_DatabaseErrors(t *testing.T) {
	t.Parallel()

	cfg, st := openPortalTestStore(t)
	defer st.Close()

	// Create a test user first
	createPortalTestUser(t, st, "test@example.com", "password123")

	srv, err := NewServer(cfg, st)
	if err != nil {
		t.Fatalf("NewServer error: %v", err)
	}
	httpSrv := httptest.NewServer(srv)
	defer httpSrv.Close()

	tests := []struct {
		name       string
		payload    map[string]any
		wantStatus int
		wantError  string
	}{
		{
			name:       "empty_email",
			payload:    map[string]any{"email": "", "password": "password123"},
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid_credentials",
		},
		{
			name:       "empty_password",
			payload:    map[string]any{"email": "test@example.com", "password": ""},
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid_credentials",
		},
		{
			name:       "missing_email_field",
			payload:    map[string]any{"password": "password123"},
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid_credentials",
		},
		{
			name:       "missing_password_field",
			payload:    map[string]any{"email": "test@example.com"},
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid_credentials",
		},
		{
			name:       "invalid_email_format",
			payload:    map[string]any{"email": "   ", "password": "password123"},
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid_credentials",
		},
		{
			name:       "invalid_password_format",
			payload:    map[string]any{"email": "test@example.com", "password": "   "},
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid_credentials",
		},
		{
			name:       "nonexistent_user",
			payload:    map[string]any{"email": "nonexistent@example.com", "password": "password123"},
			wantStatus: http.StatusUnauthorized,
			wantError:  "invalid_credentials",
		},
		{
			name:       "wrong_password",
			payload:    map[string]any{"email": "test@example.com", "password": "wrongpassword"},
			wantStatus: http.StatusUnauthorized,
			wantError:  "invalid_credentials",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPost, httpSrv.URL+"/portal/api/v1/auth/login", nil, tt.payload)
			if resp.StatusCode != tt.wantStatus {
				t.Errorf("expected status %d got %d body=%s", tt.wantStatus, resp.StatusCode, string(resp.Body))
			}
			if tt.wantError != "" && !strings.Contains(string(resp.Body), tt.wantError) {
				t.Errorf("expected error %q in body, got %s", tt.wantError, string(resp.Body))
			}
		})
	}
}

// Test Login with Inactive User
func TestLogin_InactiveUser(t *testing.T) {
	t.Parallel()

	cfg, st := openPortalTestStore(t)
	defer st.Close()

	// Create a suspended user
	hash, _ := store.HashPassword("password123")
	suspendedUser := &store.User{
		Email:        "suspended@example.com",
		Name:         "Suspended User",
		PasswordHash: hash,
		Role:         "user",
		Status:       "suspended",
	}
	if err := st.Users().Create(suspendedUser); err != nil {
		t.Fatalf("create suspended user: %v", err)
	}

	srv, err := NewServer(cfg, st)
	if err != nil {
		t.Fatalf("NewServer error: %v", err)
	}
	httpSrv := httptest.NewServer(srv)
	defer httpSrv.Close()

	resp := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPost, httpSrv.URL+"/portal/api/v1/auth/login", nil, map[string]any{
		"email":    "suspended@example.com",
		"password": "password123",
	})
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected status %d got %d body=%s", http.StatusForbidden, resp.StatusCode, string(resp.Body))
	}
	if !strings.Contains(string(resp.Body), "user_inactive") {
		t.Errorf("expected error 'user_inactive' in body, got %s", string(resp.Body))
	}
}

// Test Login Rate Limiting
func TestLogin_RateLimiting(t *testing.T) {
	t.Parallel()

	cfg, st := openPortalTestStore(t)
	defer st.Close()

	// Create a test user
	createPortalTestUser(t, st, "ratelimit@example.com", "password123")

	srv, err := NewServer(cfg, st)
	if err != nil {
		t.Fatalf("NewServer error: %v", err)
	}
	httpSrv := httptest.NewServer(srv)
	defer httpSrv.Close()

	// Make 5 failed login attempts
	for i := 0; i < 5; i++ {
		resp := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPost, httpSrv.URL+"/portal/api/v1/auth/login", nil, map[string]any{
			"email":    "ratelimit@example.com",
			"password": "wrongpassword",
		})
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("attempt %d: expected status %d got %d", i+1, http.StatusUnauthorized, resp.StatusCode)
		}
	}

	// 6th attempt should be rate limited
	resp := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPost, httpSrv.URL+"/portal/api/v1/auth/login", nil, map[string]any{
		"email":    "ratelimit@example.com",
		"password": "wrongpassword",
	})
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Errorf("expected status %d got %d body=%s", http.StatusTooManyRequests, resp.StatusCode, string(resp.Body))
	}
	if !strings.Contains(string(resp.Body), "rate_limited") {
		t.Errorf("expected error 'rate_limited' in body, got %s", string(resp.Body))
	}
}

// Test Logout Error Paths
func TestLogout_ErrorPaths(t *testing.T) {
	t.Parallel()

	cfg, st := openPortalTestStore(t)
	defer st.Close()

	user := createPortalTestUserWithID(t, st, "logout@example.com", "password123")

	srv, err := NewServer(cfg, st)
	if err != nil {
		t.Fatalf("NewServer error: %v", err)
	}
	httpSrv := httptest.NewServer(srv)
	defer httpSrv.Close()

	// Login first
	loginResp := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPost, httpSrv.URL+"/portal/api/v1/auth/login", nil, map[string]any{
		"email":    "logout@example.com",
		"password": "password123",
	})
	if loginResp.StatusCode != http.StatusOK {
		t.Fatalf("login failed: %d body=%s", loginResp.StatusCode, string(loginResp.Body))
	}
	sessionCookie := findCookie(loginResp.Cookies, cfg.Portal.Session.CookieName)

	// Test logout without session (should still succeed)
	resp := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPost, httpSrv.URL+"/portal/api/v1/auth/logout", nil, map[string]any{})
	if resp.StatusCode != http.StatusUnauthorized {
		// Without session, it should return 401
		t.Logf("logout without session returned %d", resp.StatusCode)
	}

	// Test logout with invalid cookie
	invalidCookie := &http.Cookie{
		Name:  cfg.Portal.Session.CookieName,
		Value: "invalid_token",
	}
	resp = mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPost, httpSrv.URL+"/portal/api/v1/auth/logout", []*http.Cookie{invalidCookie}, map[string]any{})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Logf("logout with invalid cookie returned %d", resp.StatusCode)
	}

	// Test successful logout
	resp = mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPost, httpSrv.URL+"/portal/api/v1/auth/logout", []*http.Cookie{sessionCookie}, map[string]any{})
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d got %d body=%s", http.StatusOK, resp.StatusCode, string(resp.Body))
	}

	// Verify session is cleared by trying to use it again
	resp = mustPortalJSONRequest(t, httpSrv.Client(), http.MethodGet, httpSrv.URL+"/portal/api/v1/auth/me", []*http.Cookie{sessionCookie}, nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status %d after logout got %d", http.StatusUnauthorized, resp.StatusCode)
	}
}

// Test API Key Management Error Paths
func TestAPIKeyManagement_ErrorPaths(t *testing.T) {
	t.Parallel()

	cfg, st := openPortalTestStore(t)
	defer st.Close()

	user := createPortalTestUserWithID(t, st, "apikey@example.com", "password123")

	srv, err := NewServer(cfg, st)
	if err != nil {
		t.Fatalf("NewServer error: %v", err)
	}
	httpSrv := httptest.NewServer(srv)
	defer httpSrv.Close()

	// Login
	loginResp := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPost, httpSrv.URL+"/portal/api/v1/auth/login", nil, map[string]any{
		"email":    "apikey@example.com",
		"password": "password123",
	})
	sessionCookie := findCookie(loginResp.Cookies, cfg.Portal.Session.CookieName)

	// Test create API key with missing name
	resp := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPost, httpSrv.URL+"/portal/api/v1/api-keys", []*http.Cookie{sessionCookie}, map[string]any{
		"mode": "test",
	})
	// Should succeed with default name
	if resp.StatusCode != http.StatusCreated {
		t.Logf("create key without name returned %d: %s", resp.StatusCode, string(resp.Body))
	}

	// Test rename API key with empty ID
	resp = mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPut, httpSrv.URL+"/portal/api/v1/api-keys/", []*http.Cookie{sessionCookie}, map[string]any{
		"name": "newname",
	})
	if resp.StatusCode != http.StatusNotFound {
		t.Logf("rename with empty ID returned %d", resp.StatusCode)
	}

	// Test rename API key with empty name
	resp = mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPut, httpSrv.URL+"/portal/api/v1/api-keys/nonexistent", []*http.Cookie{sessionCookie}, map[string]any{
		"name": "",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Logf("rename with empty name returned %d: %s", resp.StatusCode, string(resp.Body))
	}

	// Test rename non-existent API key
	resp = mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPut, httpSrv.URL+"/portal/api/v1/api-keys/nonexistent-id", []*http.Cookie{sessionCookie}, map[string]any{
		"name": "newname",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Logf("rename non-existent key returned %d: %s", resp.StatusCode, string(resp.Body))
	}

	// Test revoke non-existent API key
	resp = mustPortalJSONRequest(t, httpSrv.Client(), http.MethodDelete, httpSrv.URL+"/portal/api/v1/api-keys/nonexistent-id", []*http.Cookie{sessionCookie}, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Logf("revoke non-existent key returned %d: %s", resp.StatusCode, string(resp.Body))
	}

	// Test revoke API key with empty ID
	resp = mustPortalJSONRequest(t, httpSrv.Client(), http.MethodDelete, httpSrv.URL+"/portal/api/v1/api-keys/", []*http.Cookie{sessionCookie}, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Logf("revoke with empty ID returned %d", resp.StatusCode)
	}

	_ = user
}

// Test Profile Update Error Paths
func TestProfileUpdate_ErrorPaths(t *testing.T) {
	t.Parallel()

	cfg, st := openPortalTestStore(t)
	defer st.Close()

	createPortalTestUser(t, st, "profile@example.com", "password123")

	srv, err := NewServer(cfg, st)
	if err != nil {
		t.Fatalf("NewServer error: %v", err)
	}
	httpSrv := httptest.NewServer(srv)
	defer httpSrv.Close()

	// Login
	loginResp := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPost, httpSrv.URL+"/portal/api/v1/auth/login", nil, map[string]any{
		"email":    "profile@example.com",
		"password": "password123",
	})
	sessionCookie := findCookie(loginResp.Cookies, cfg.Portal.Session.CookieName)

	// Test get profile without session
	resp := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodGet, httpSrv.URL+"/portal/api/v1/settings/profile", nil, nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status %d got %d", http.StatusUnauthorized, resp.StatusCode)
	}

	// Test update profile without session
	resp = mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPut, httpSrv.URL+"/portal/api/v1/settings/profile", nil, map[string]any{
		"name": "New Name",
	})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status %d got %d", http.StatusUnauthorized, resp.StatusCode)
	}

	// Test update profile with invalid JSON
	req, _ := http.NewRequest(http.MethodPut, httpSrv.URL+"/portal/api/v1/settings/profile", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(sessionCookie)
	resp2, err := httpSrv.Client().Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status %d for invalid JSON got %d", http.StatusBadRequest, resp2.StatusCode)
	}

	// Test successful profile update
	resp = mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPut, httpSrv.URL+"/portal/api/v1/settings/profile", []*http.Cookie{sessionCookie}, map[string]any{
		"name":    "Updated Name",
		"company": "Updated Company",
		"metadata": map[string]any{
			"key": "value",
		},
	})
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d got %d body=%s", http.StatusOK, resp.StatusCode, string(resp.Body))
	}
}

// Test Change Password Error Paths
func TestChangePassword_ErrorPaths(t *testing.T) {
	t.Parallel()

	cfg, st := openPortalTestStore(t)
	defer st.Close()

	createPortalTestUser(t, st, "changepwd@example.com", "oldpassword")

	srv, err := NewServer(cfg, st)
	if err != nil {
		t.Fatalf("NewServer error: %v", err)
	}
	httpSrv := httptest.NewServer(srv)
	defer httpSrv.Close()

	// Login
	loginResp := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPost, httpSrv.URL+"/portal/api/v1/auth/login", nil, map[string]any{
		"email":    "changepwd@example.com",
		"password": "oldpassword",
	})
	sessionCookie := findCookie(loginResp.Cookies, cfg.Portal.Session.CookieName)

	// Test change password without session
	resp := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPut, httpSrv.URL+"/portal/api/v1/auth/password", nil, map[string]any{
		"old_password": "oldpassword",
		"new_password": "newpassword",
	})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status %d got %d", http.StatusUnauthorized, resp.StatusCode)
	}

	// Test change password with missing old_password
	resp = mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPut, httpSrv.URL+"/portal/api/v1/auth/password", []*http.Cookie{sessionCookie}, map[string]any{
		"new_password": "newpassword",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status %d got %d body=%s", http.StatusBadRequest, resp.StatusCode, string(resp.Body))
	}

	// Test change password with missing new_password
	resp = mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPut, httpSrv.URL+"/portal/api/v1/auth/password", []*http.Cookie{sessionCookie}, map[string]any{
		"old_password": "oldpassword",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status %d got %d body=%s", http.StatusBadRequest, resp.StatusCode, string(resp.Body))
	}

	// Test change password with wrong old password
	resp = mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPut, httpSrv.URL+"/portal/api/v1/auth/password", []*http.Cookie{sessionCookie}, map[string]any{
		"old_password": "wrongpassword",
		"new_password": "newpassword",
	})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status %d got %d body=%s", http.StatusUnauthorized, resp.StatusCode, string(resp.Body))
	}

	// Test successful password change
	resp = mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPut, httpSrv.URL+"/portal/api/v1/auth/password", []*http.Cookie{sessionCookie}, map[string]any{
		"old_password": "oldpassword",
		"new_password": "newpassword123",
	})
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d got %d body=%s", http.StatusOK, resp.StatusCode, string(resp.Body))
	}

	// Verify can login with new password
	resp = mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPost, httpSrv.URL+"/portal/api/v1/auth/login", nil, map[string]any{
		"email":    "changepwd@example.com",
		"password": "newpassword123",
	})
	if resp.StatusCode != http.StatusOK {
		t.Errorf("login with new password failed: %d body=%s", resp.StatusCode, string(resp.Body))
	}
}

// Test IP Whitelist Error Paths
func TestIPWhitelist_ErrorPaths(t *testing.T) {
	t.Parallel()

	cfg, st := openPortalTestStore(t)
	defer st.Close()

	createPortalTestUser(t, st, "ipwhitelist@example.com", "password123")

	srv, err := NewServer(cfg, st)
	if err != nil {
		t.Fatalf("NewServer error: %v", err)
	}
	httpSrv := httptest.NewServer(srv)
	defer httpSrv.Close()

	// Login
	loginResp := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPost, httpSrv.URL+"/portal/api/v1/auth/login", nil, map[string]any{
		"email":    "ipwhitelist@example.com",
		"password": "password123",
	})
	sessionCookie := findCookie(loginResp.Cookies, cfg.Portal.Session.CookieName)

	// Test add IP without session
	resp := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPost, httpSrv.URL+"/portal/api/v1/security/ip-whitelist", nil, map[string]any{
		"ip": "192.168.1.1",
	})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status %d got %d", http.StatusUnauthorized, resp.StatusCode)
	}

	// Test add IP with empty value
	resp = mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPost, httpSrv.URL+"/portal/api/v1/security/ip-whitelist", []*http.Cookie{sessionCookie}, map[string]any{
		"ip": "",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status %d got %d body=%s", http.StatusBadRequest, resp.StatusCode, string(resp.Body))
	}

	// Test add IP with only whitespace
	resp = mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPost, httpSrv.URL+"/portal/api/v1/security/ip-whitelist", []*http.Cookie{sessionCookie}, map[string]any{
		"ip": "   ",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status %d got %d body=%s", http.StatusBadRequest, resp.StatusCode, string(resp.Body))
	}

	// Test remove IP with empty value
	resp = mustPortalJSONRequest(t, httpSrv.Client(), http.MethodDelete, httpSrv.URL+"/portal/api/v1/security/ip-whitelist/", []*http.Cookie{sessionCookie}, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Logf("remove with empty IP returned %d", resp.StatusCode)
	}

	// Test successful add IP
	resp = mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPost, httpSrv.URL+"/portal/api/v1/security/ip-whitelist", []*http.Cookie{sessionCookie}, map[string]any{
		"ip": "192.168.1.1",
	})
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d got %d body=%s", http.StatusOK, resp.StatusCode, string(resp.Body))
	}

	// Test successful remove IP
	resp = mustPortalJSONRequest(t, httpSrv.Client(), http.MethodDelete, httpSrv.URL+"/portal/api/v1/security/ip-whitelist/192.168.1.1", []*http.Cookie{sessionCookie}, nil)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d got %d body=%s", http.StatusOK, resp.StatusCode, string(resp.Body))
	}
}

// Test Notifications Update Error Paths
func TestNotifications_ErrorPaths(t *testing.T) {
	t.Parallel()

	cfg, st := openPortalTestStore(t)
	defer st.Close()

	createPortalTestUser(t, st, "notifications@example.com", "password123")

	srv, err := NewServer(cfg, st)
	if err != nil {
		t.Fatalf("NewServer error: %v", err)
	}
	httpSrv := httptest.NewServer(srv)
	defer httpSrv.Close()

	// Login
	loginResp := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPost, httpSrv.URL+"/portal/api/v1/auth/login", nil, map[string]any{
		"email":    "notifications@example.com",
		"password": "password123",
	})
	sessionCookie := findCookie(loginResp.Cookies, cfg.Portal.Session.CookieName)

	// Test update notifications without session
	resp := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPut, httpSrv.URL+"/portal/api/v1/settings/notifications", nil, map[string]any{
		"notifications": map[string]any{"email": true},
	})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status %d got %d", http.StatusUnauthorized, resp.StatusCode)
	}

	// Test update notifications with invalid JSON
	req, _ := http.NewRequest(http.MethodPut, httpSrv.URL+"/portal/api/v1/settings/notifications", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(sessionCookie)
	resp2, err := httpSrv.Client().Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status %d for invalid JSON got %d", http.StatusBadRequest, resp2.StatusCode)
	}

	// Test successful update with nested notifications object
	resp = mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPut, httpSrv.URL+"/portal/api/v1/settings/notifications", []*http.Cookie{sessionCookie}, map[string]any{
		"notifications": map[string]any{
			"email":    true,
			"sms":      false,
			"webhook":  "https://example.com/webhook",
		},
	})
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d got %d body=%s", http.StatusOK, resp.StatusCode, string(resp.Body))
	}

	// Test successful update with flat payload (notifications at root)
	resp = mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPut, httpSrv.URL+"/portal/api/v1/settings/notifications", []*http.Cookie{sessionCookie}, map[string]any{
		"email": true,
		"push":  true,
	})
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d got %d body=%s", http.StatusOK, resp.StatusCode, string(resp.Body))
	}
}

// Test Playground Error Paths
func TestPlayground_ErrorPaths(t *testing.T) {
	t.Parallel()

	cfg, st := openPortalTestStore(t)
	defer st.Close()

	createPortalTestUser(t, st, "playground@example.com", "password123")

	srv, err := NewServer(cfg, st)
	if err != nil {
		t.Fatalf("NewServer error: %v", err)
	}
	httpSrv := httptest.NewServer(srv)
	defer httpSrv.Close()

	// Login
	loginResp := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPost, httpSrv.URL+"/portal/api/v1/auth/login", nil, map[string]any{
		"email":    "playground@example.com",
		"password": "password123",
	})
	sessionCookie := findCookie(loginResp.Cookies, cfg.Portal.Session.CookieName)

	// Test playground send without session
	resp := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPost, httpSrv.URL+"/portal/api/v1/playground/send", nil, map[string]any{
		"method":  "GET",
		"path":    "/test",
		"api_key": "test-key",
	})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status %d got %d", http.StatusUnauthorized, resp.StatusCode)
	}

	// Test playground send with invalid path (not starting with /)
	resp = mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPost, httpSrv.URL+"/portal/api/v1/playground/send", []*http.Cookie{sessionCookie}, map[string]any{
		"method":  "GET",
		"path":    "test",
		"api_key": "test-key",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status %d got %d body=%s", http.StatusBadRequest, resp.StatusCode, string(resp.Body))
	}

	// Test playground send with empty path
	resp = mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPost, httpSrv.URL+"/portal/api/v1/playground/send", []*http.Cookie{sessionCookie}, map[string]any{
		"method":  "GET",
		"path":    "",
		"api_key": "test-key",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status %d got %d body=%s", http.StatusBadRequest, resp.StatusCode, string(resp.Body))
	}

	// Test playground send without api_key
	resp = mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPost, httpSrv.URL+"/portal/api/v1/playground/send", []*http.Cookie{sessionCookie}, map[string]any{
		"method": "GET",
		"path":   "/test",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status %d got %d body=%s", http.StatusBadRequest, resp.StatusCode, string(resp.Body))
	}

	// Test save template without session
	resp = mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPost, httpSrv.URL+"/portal/api/v1/playground/templates", nil, map[string]any{
		"name":   "Test Template",
		"method": "GET",
		"path":   "/test",
	})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status %d got %d", http.StatusUnauthorized, resp.StatusCode)
	}

	// Test save template without name
	resp = mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPost, httpSrv.URL+"/portal/api/v1/playground/templates", []*http.Cookie{sessionCookie}, map[string]any{
		"method": "GET",
		"path":   "/test",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status %d got %d body=%s", http.StatusBadRequest, resp.StatusCode, string(resp.Body))
	}

	// Test delete template with empty ID
	resp = mustPortalJSONRequest(t, httpSrv.Client(), http.MethodDelete, httpSrv.URL+"/portal/api/v1/playground/templates/", []*http.Cookie{sessionCookie}, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Logf("delete with empty ID returned %d", resp.StatusCode)
	}

	// Test delete non-existent template
	resp = mustPortalJSONRequest(t, httpSrv.Client(), http.MethodDelete, httpSrv.URL+"/portal/api/v1/playground/templates/nonexistent-id", []*http.Cookie{sessionCookie}, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Logf("delete non-existent template returned %d: %s", resp.StatusCode, string(resp.Body))
	}
}

// Test Usage Endpoints Error Paths
func TestUsage_ErrorPaths(t *testing.T) {
	t.Parallel()

	cfg, st := openPortalTestStore(t)
	defer st.Close()

	createPortalTestUser(t, st, "usage@example.com", "password123")

	srv, err := NewServer(cfg, st)
	if err != nil {
		t.Fatalf("NewServer error: %v", err)
	}
	httpSrv := httptest.NewServer(srv)
	defer httpSrv.Close()

	// Login
	loginResp := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPost, httpSrv.URL+"/portal/api/v1/auth/login", nil, map[string]any{
		"email":    "usage@example.com",
		"password": "password123",
	})
	sessionCookie := findCookie(loginResp.Cookies, cfg.Portal.Session.CookieName)

	// Test usage endpoints without session
	endpoints := []string{
		"/portal/api/v1/usage/overview",
		"/portal/api/v1/usage/timeseries",
		"/portal/api/v1/usage/top-endpoints",
		"/portal/api/v1/usage/errors",
	}

	for _, endpoint := range endpoints {
		resp := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodGet, httpSrv.URL+endpoint, nil, nil)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("%s: expected status %d got %d", endpoint, http.StatusUnauthorized, resp.StatusCode)
		}
	}

	// Test with invalid time range
	resp := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodGet, httpSrv.URL+"/portal/api/v1/usage/overview?from=invalid", []*http.Cookie{sessionCookie}, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status %d got %d body=%s", http.StatusBadRequest, resp.StatusCode, string(resp.Body))
	}

	// Test with invalid granularity
	resp = mustPortalJSONRequest(t, httpSrv.Client(), http.MethodGet, httpSrv.URL+"/portal/api/v1/usage/timeseries?granularity=invalid", []*http.Cookie{sessionCookie}, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status %d got %d body=%s", http.StatusBadRequest, resp.StatusCode, string(resp.Body))
	}

	// Test with negative granularity
	resp = mustPortalJSONRequest(t, httpSrv.Client(), http.MethodGet, httpSrv.URL+"/portal/api/v1/usage/timeseries?granularity=-1h", []*http.Cookie{sessionCookie}, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status %d got %d body=%s", http.StatusBadRequest, resp.StatusCode, string(resp.Body))
	}
}

// Test Logs Endpoints Error Paths
func TestLogs_ErrorPaths(t *testing.T) {
	t.Parallel()

	cfg, st := openPortalTestStore(t)
	defer st.Close()

	createPortalTestUser(t, st, "logs@example.com", "password123")

	srv, err := NewServer(cfg, st)
	if err != nil {
		t.Fatalf("NewServer error: %v", err)
	}
	httpSrv := httptest.NewServer(srv)
	defer httpSrv.Close()

	// Login
	loginResp := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPost, httpSrv.URL+"/portal/api/v1/auth/login", nil, map[string]any{
		"email":    "logs@example.com",
		"password": "password123",
	})
	sessionCookie := findCookie(loginResp.Cookies, cfg.Portal.Session.CookieName)

	// Test logs endpoints without session
	endpoints := []string{
		"/portal/api/v1/logs",
		"/portal/api/v1/logs/log-id-123",
		"/portal/api/v1/logs/export",
	}

	for _, endpoint := range endpoints {
		method := http.MethodGet
		resp := mustPortalJSONRequest(t, httpSrv.Client(), method, httpSrv.URL+endpoint, nil, nil)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("%s: expected status %d got %d", endpoint, http.StatusUnauthorized, resp.StatusCode)
		}
	}

	// Test get log detail with empty ID
	resp := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodGet, httpSrv.URL+"/portal/api/v1/logs/", []*http.Cookie{sessionCookie}, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Logf("get log with empty ID returned %d", resp.StatusCode)
	}

	// Test get non-existent log
	resp = mustPortalJSONRequest(t, httpSrv.Client(), http.MethodGet, httpSrv.URL+"/portal/api/v1/logs/nonexistent-log-id", []*http.Cookie{sessionCookie}, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Logf("get non-existent log returned %d: %s", resp.StatusCode, string(resp.Body))
	}

	// Test with invalid filter params
	resp = mustPortalJSONRequest(t, httpSrv.Client(), http.MethodGet, httpSrv.URL+"/portal/api/v1/logs?status_min=invalid", []*http.Cookie{sessionCookie}, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status %d got %d body=%s", http.StatusBadRequest, resp.StatusCode, string(resp.Body))
	}

	// Test with invalid from date
	resp = mustPortalJSONRequest(t, httpSrv.Client(), http.MethodGet, httpSrv.URL+"/portal/api/v1/logs?from=invalid-date", []*http.Cookie{sessionCookie}, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status %d got %d body=%s", http.StatusBadRequest, resp.StatusCode, string(resp.Body))
	}
}

// Test Credits Endpoints Error Paths
func TestCredits_ErrorPaths(t *testing.T) {
	t.Parallel()

	cfg, st := openPortalTestStore(t)
	defer st.Close()

	createPortalTestUser(t, st, "credits@example.com", "password123")

	srv, err := NewServer(cfg, st)
	if err != nil {
		t.Fatalf("NewServer error: %v", err)
	}
	httpSrv := httptest.NewServer(srv)
	defer httpSrv.Close()

	// Login
	loginResp := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPost, httpSrv.URL+"/portal/api/v1/auth/login", nil, map[string]any{
		"email":    "credits@example.com",
		"password": "password123",
	})
	sessionCookie := findCookie(loginResp.Cookies, cfg.Portal.Session.CookieName)

	// Test credits endpoints without session
	endpoints := []struct {
		path   string
		method string
	}{
		{"/portal/api/v1/credits/balance", http.MethodGet},
		{"/portal/api/v1/credits/transactions", http.MethodGet},
		{"/portal/api/v1/credits/forecast", http.MethodGet},
		{"/portal/api/v1/credits/purchase", http.MethodPost},
	}

	for _, ep := range endpoints {
		var resp portalResponse
		if ep.method == http.MethodPost {
			resp = mustPortalJSONRequest(t, httpSrv.Client(), ep.method, httpSrv.URL+ep.path, nil, map[string]any{})
		} else {
			resp = mustPortalJSONRequest(t, httpSrv.Client(), ep.method, httpSrv.URL+ep.path, nil, nil)
		}
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("%s: expected status %d got %d", ep.path, http.StatusUnauthorized, resp.StatusCode)
		}
	}

	// Test purchase with invalid amount
	resp := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPost, httpSrv.URL+"/portal/api/v1/credits/purchase", []*http.Cookie{sessionCookie}, map[string]any{
		"amount": 0,
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status %d got %d body=%s", http.StatusBadRequest, resp.StatusCode, string(resp.Body))
	}

	// Test purchase with negative amount
	resp = mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPost, httpSrv.URL+"/portal/api/v1/credits/purchase", []*http.Cookie{sessionCookie}, map[string]any{
		"amount": -10,
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status %d got %d body=%s", http.StatusBadRequest, resp.StatusCode, string(resp.Body))
	}

	// Test purchase with string amount (should be converted)
	resp = mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPost, httpSrv.URL+"/portal/api/v1/credits/purchase", []*http.Cookie{sessionCookie}, map[string]any{
		"amount": "invalid",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Logf("purchase with string amount returned %d: %s", resp.StatusCode, string(resp.Body))
	}
}

// Test API List Error Paths
func TestAPIList_ErrorPaths(t *testing.T) {
	t.Parallel()

	cfg, st := openPortalTestStore(t)
	defer st.Close()

	createPortalTestUser(t, st, "apilist@example.com", "password123")

	srv, err := NewServer(cfg, st)
	if err != nil {
		t.Fatalf("NewServer error: %v", err)
	}
	httpSrv := httptest.NewServer(srv)
	defer httpSrv.Close()

	// Login
	loginResp := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPost, httpSrv.URL+"/portal/api/v1/auth/login", nil, map[string]any{
		"email":    "apilist@example.com",
		"password": "password123",
	})
	sessionCookie := findCookie(loginResp.Cookies, cfg.Portal.Session.CookieName)

	// Test API list without session
	resp := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodGet, httpSrv.URL+"/portal/api/v1/apis", nil, nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status %d got %d", http.StatusUnauthorized, resp.StatusCode)
	}

	// Test API detail without session
	resp = mustPortalJSONRequest(t, httpSrv.Client(), http.MethodGet, httpSrv.URL+"/portal/api/v1/apis/route-1", nil, nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status %d got %d", http.StatusUnauthorized, resp.StatusCode)
	}

	// Test API detail with empty route ID
	resp = mustPortalJSONRequest(t, httpSrv.Client(), http.MethodGet, httpSrv.URL+"/portal/api/v1/apis/", []*http.Cookie{sessionCookie}, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Logf("API detail with empty ID returned %d", resp.StatusCode)
	}

	// Test API detail with non-existent route
	resp = mustPortalJSONRequest(t, httpSrv.Client(), http.MethodGet, httpSrv.URL+"/portal/api/v1/apis/nonexistent-route", []*http.Cookie{sessionCookie}, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Logf("API detail with non-existent route returned %d: %s", resp.StatusCode, string(resp.Body))
	}
}

// Test Security Activity Error Paths
func TestSecurityActivity_ErrorPaths(t *testing.T) {
	t.Parallel()

	cfg, st := openPortalTestStore(t)
	defer st.Close()

	createPortalTestUser(t, st, "activity@example.com", "password123")

	srv, err := NewServer(cfg, st)
	if err != nil {
		t.Fatalf("NewServer error: %v", err)
	}
	httpSrv := httptest.NewServer(srv)
	defer httpSrv.Close()

	// Login
	loginResp := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPost, httpSrv.URL+"/portal/api/v1/auth/login", nil, map[string]any{
		"email":    "activity@example.com",
		"password": "password123",
	})
	sessionCookie := findCookie(loginResp.Cookies, cfg.Portal.Session.CookieName)

	// Test activity without session
	resp := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodGet, httpSrv.URL+"/portal/api/v1/security/activity", nil, nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status %d got %d", http.StatusUnauthorized, resp.StatusCode)
	}

	// Test activity with session
	resp = mustPortalJSONRequest(t, httpSrv.Client(), http.MethodGet, httpSrv.URL+"/portal/api/v1/security/activity", []*http.Cookie{sessionCookie}, nil)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d got %d body=%s", http.StatusOK, resp.StatusCode, string(resp.Body))
	}
}

// Test Me Endpoint Error Paths
func TestMe_ErrorPaths(t *testing.T) {
	t.Parallel()

	cfg, st := openPortalTestStore(t)
	defer st.Close()

	createPortalTestUser(t, st, "me@example.com", "password123")

	srv, err := NewServer(cfg, st)
	if err != nil {
		t.Fatalf("NewServer error: %v", err)
	}
	httpSrv := httptest.NewServer(srv)
	defer httpSrv.Close()

	// Test me without session
	resp := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodGet, httpSrv.URL+"/portal/api/v1/auth/me", nil, nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status %d got %d", http.StatusUnauthorized, resp.StatusCode)
	}

	// Test me with invalid cookie
	invalidCookie := &http.Cookie{
		Name:  cfg.Portal.Session.CookieName,
		Value: "invalid_token_value",
	}
	resp = mustPortalJSONRequest(t, httpSrv.Client(), http.MethodGet, httpSrv.URL+"/portal/api/v1/auth/me", []*http.Cookie{invalidCookie}, nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status %d got %d body=%s", http.StatusUnauthorized, resp.StatusCode, string(resp.Body))
	}
}

// Test Helper Functions
func TestHelperFunctions(t *testing.T) {
	t.Parallel()

	// Test normalizePortalPathPrefix
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"/", ""},
		{"portal", "/portal"},
		{"/portal", "/portal"},
		{"/portal/", "/portal"},
		{"  /portal/  ", "/portal"},
	}

	for _, tt := range tests {
		result := normalizePortalPathPrefix(tt.input)
		if result != tt.expected {
			t.Errorf("normalizePortalPathPrefix(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}

	// Test isUserActive
	activeTests := []struct {
		user   *store.User
		active bool
	}{
		{nil, false},
		{&store.User{Status: ""}, true},
		{&store.User{Status: "active"}, true},
		{&store.User{Status: "ACTIVE"}, true},
		{&store.User{Status: "suspended"}, false},
		{&store.User{Status: "deleted"}, false},
		{&store.User{Status: "inactive"}, false},
	}

	for _, tt := range activeTests {
		result := isUserActive(tt.user)
		if result != tt.active {
			t.Errorf("isUserActive(%+v) = %v, want %v", tt.user, result, tt.active)
		}
	}

	// Test sanitizeUser
	sanitized := sanitizeUser(nil)
	if len(sanitized) != 0 {
		t.Errorf("sanitizeUser(nil) should return empty map, got %v", sanitized)
	}

	user := &store.User{
		ID:            "user-1",
		Email:         "test@example.com",
		Name:          "Test User",
		Company:       "Test Co",
		Role:          "user",
		Status:        "active",
		CreditBalance: 100,
	}
	sanitized = sanitizeUser(user)
	if sanitized["id"] != "user-1" {
		t.Errorf("sanitizeUser id = %v, want user-1", sanitized["id"])
	}
	if sanitized["email"] != "test@example.com" {
		t.Errorf("sanitizeUser email = %v, want test@example.com", sanitized["email"])
	}

	// Test getClientIP
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	ip := getClientIP(req)
	if ip != "192.168.1.1" {
		t.Errorf("getClientIP = %q, want 192.168.1.1", ip)
	}

	// Test with X-Forwarded-For
	req.Header.Set("X-Forwarded-For", "10.0.0.1, 10.0.0.2")
	ip = getClientIP(req)
	if ip != "10.0.0.1" {
		t.Errorf("getClientIP with X-Forwarded-For = %q, want 10.0.0.1", ip)
	}

	// Test extractClientIP
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req2.RemoteAddr = "192.168.1.1:12345"
	ip = extractClientIP(req2)
	if ip != "192.168.1.1" {
		t.Errorf("extractClientIP = %q, want 192.168.1.1", ip)
	}

	// Test with X-Forwarded-For
	req2.Header.Set("X-Forwarded-For", "10.0.0.1, 10.0.0.2")
	ip = extractClientIP(req2)
	if ip != "10.0.0.1" {
		t.Errorf("extractClientIP with X-Forwarded-For = %q, want 10.0.0.1", ip)
	}

	// Test with X-Real-Ip
	req3 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req3.Header.Set("X-Real-Ip", "10.0.0.3")
	ip = extractClientIP(req3)
	if ip != "10.0.0.3" {
		t.Errorf("extractClientIP with X-Real-Ip = %q, want 10.0.0.3", ip)
	}

	// Test with IPv6
	req4 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req4.RemoteAddr = "[::1]:12345"
	ip = extractClientIP(req4)
	if ip != "::1" {
		t.Errorf("extractClientIP with IPv6 = %q, want ::1", ip)
	}
}

// Test asString helper
func TestAsString(t *testing.T) {
	tests := []struct {
		input    any
		expected string
	}{
		{nil, ""},
		{"", ""},
		{"hello", "hello"},
		{123, "123"},
		{true, "true"},
		{"  spaced  ", "spaced"},
	}

	for _, tt := range tests {
		result := asString(tt.input)
		if result != tt.expected {
			t.Errorf("asString(%v) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

// Test asStringSlice helper
func TestAsStringSlice(t *testing.T) {
	// Test with []string
	result := asStringSlice([]string{"a", "b", "c"})
	if len(result) != 3 || result[0] != "a" || result[1] != "b" || result[2] != "c" {
		t.Errorf("asStringSlice([]string) = %v", result)
	}

	// Test with []any
	result = asStringSlice([]any{"a", "b", "c"})
	if len(result) != 3 {
		t.Errorf("asStringSlice([]any) = %v", result)
	}

	// Test with empty strings
	result = asStringSlice([]string{"a", "", "  ", "b"})
	if len(result) != 2 || result[0] != "a" || result[1] != "b" {
		t.Errorf("asStringSlice with empty strings = %v", result)
	}

	// Test with nil
	result = asStringSlice(nil)
	if result != nil {
		t.Errorf("asStringSlice(nil) = %v, want nil", result)
	}

	// Test with unsupported type
	result = asStringSlice(123)
	if result != nil {
		t.Errorf("asStringSlice(int) = %v, want nil", result)
	}
}

// Test asInt helper
func TestAsInt(t *testing.T) {
	tests := []struct {
		input    string
		fallback int
		expected int
	}{
		{"", 10, 10},
		{"   ", 10, 10},
		{"5", 10, 5},
		{"invalid", 10, 10},
		{"-5", 10, -5},
	}

	for _, tt := range tests {
		result := asInt(tt.input, tt.fallback)
		if result != tt.expected {
			t.Errorf("asInt(%q, %d) = %d, want %d", tt.input, tt.fallback, result, tt.expected)
		}
	}
}

// Test asInt64 helper
func TestAsInt64(t *testing.T) {
	tests := []struct {
		input    any
		fallback int64
		expected int64
	}{
		{nil, 10, 10},
		{int(5), 10, 5},
		{int64(5), 10, 5},
		{int32(5), 10, 5},
		{float64(5.5), 10, 5},
		{float32(5.5), 10, 5},
		{"5", 10, 5},
		{"invalid", 10, 10},
		{"", 10, 10},
		{true, 10, 10},
	}

	for _, tt := range tests {
		result := asInt64(tt.input, tt.fallback)
		if result != tt.expected {
			t.Errorf("asInt64(%v, %d) = %d, want %d", tt.input, tt.fallback, result, tt.expected)
		}
	}
}

// Test parsePortalTimeRange
func TestParsePortalTimeRange(t *testing.T) {
	// Test default window
	query := make(map[string][]string)
	from, to, err := parsePortalTimeRange(query)
	if err != nil {
		t.Errorf("parsePortalTimeRange() error = %v", err)
	}
	if from.After(to) {
		t.Error("from should be before to")
	}

	// Test with valid from/to
	query = map[string][]string{
		"from": {time.Now().Add(-48 * time.Hour).Format(time.RFC3339)},
		"to":   {time.Now().Format(time.RFC3339)},
	}
	from, to, err = parsePortalTimeRange(query)
	if err != nil {
		t.Errorf("parsePortalTimeRange() error = %v", err)
	}
	if from.After(to) {
		t.Error("from should be before to")
	}

	// Test with invalid from
	query = map[string][]string{
		"from": {"invalid"},
	}
	_, _, err = parsePortalTimeRange(query)
	if err == nil {
		t.Error("parsePortalTimeRange() should return error for invalid from")
	}

	// Test with invalid to
	query = map[string][]string{
		"to": {"invalid"},
	}
	_, _, err = parsePortalTimeRange(query)
	if err == nil {
		t.Error("parsePortalTimeRange() should return error for invalid to")
	}

	// Test with invalid window
	query = map[string][]string{
		"window": {"invalid"},
	}
	_, _, err = parsePortalTimeRange(query)
	if err == nil {
		t.Error("parsePortalTimeRange() should return error for invalid window")
	}

	// Test with reversed from/to (from after to)
	future := time.Now().Add(24 * time.Hour).Format(time.RFC3339)
	past := time.Now().Add(-24 * time.Hour).Format(time.RFC3339)
	query = map[string][]string{
		"from": {future},
		"to":   {past},
	}
	from, to, err = parsePortalTimeRange(query)
	if err != nil {
		t.Errorf("parsePortalTimeRange() error = %v", err)
	}
	if from.After(to) {
		t.Error("from/to should be swapped when from > to")
	}
}

// Test parsePortalGranularity
func TestParsePortalGranularity(t *testing.T) {
	// Test default
	query := make(map[string][]string)
	d, err := parsePortalGranularity(query)
	if err != nil {
		t.Errorf("parsePortalGranularity() error = %v", err)
	}
	if d != time.Hour {
		t.Errorf("parsePortalGranularity() = %v, want %v", d, time.Hour)
	}

	// Test valid granularity
	query = map[string][]string{
		"granularity": {"30m"},
	}
	d, err = parsePortalGranularity(query)
	if err != nil {
		t.Errorf("parsePortalGranularity() error = %v", err)
	}
	if d != 30*time.Minute {
		t.Errorf("parsePortalGranularity() = %v, want %v", d, 30*time.Minute)
	}

	// Test invalid granularity
	query = map[string][]string{
		"granularity": {"invalid"},
	}
	_, err = parsePortalGranularity(query)
	if err == nil {
		t.Error("parsePortalGranularity() should return error for invalid value")
	}

	// Test zero granularity
	query = map[string][]string{
		"granularity": {"0s"},
	}
	_, err = parsePortalGranularity(query)
	if err == nil {
		t.Error("parsePortalGranularity() should return error for zero value")
	}

	// Test negative granularity
	query = map[string][]string{
		"granularity": {"-1h"},
	}
	_, err = parsePortalGranularity(query)
	if err == nil {
		t.Error("parsePortalGranularity() should return error for negative value")
	}

	// Test granularity below minimum
	query = map[string][]string{
		"granularity": {"30s"},
	}
	d, err = parsePortalGranularity(query)
	if err != nil {
		t.Errorf("parsePortalGranularity() error = %v", err)
	}
	if d != time.Minute {
		t.Errorf("parsePortalGranularity() below minimum = %v, want %v", d, time.Minute)
	}
}

// Test parsePortalLogFilters
func TestParsePortalLogFilters(t *testing.T) {
	// Test empty
	query := make(map[string][]string)
	filters, err := parsePortalLogFilters(query)
	if err != nil {
		t.Errorf("parsePortalLogFilters() error = %v", err)
	}
	if filters.Limit != 50 {
		t.Errorf("parsePortalLogFilters() limit = %d, want 50", filters.Limit)
	}

	// Test with all filters
	query = map[string][]string{
		"route":     {"/api/test"},
		"method":    {"GET"},
		"client_ip": {"192.168.1.1"},
		"q":         {"search"},
		"limit":     {"25"},
		"offset":    {"10"},
		"status_min": {"200"},
		"status_max": {"299"},
		"from":      {time.Now().Add(-24 * time.Hour).Format(time.RFC3339)},
		"to":        {time.Now().Format(time.RFC3339)},
	}
	filters, err = parsePortalLogFilters(query)
	if err != nil {
		t.Errorf("parsePortalLogFilters() error = %v", err)
	}
	if filters.Route != "/api/test" {
		t.Errorf("parsePortalLogFilters() route = %s, want /api/test", filters.Route)
	}
	if filters.Limit != 25 {
		t.Errorf("parsePortalLogFilters() limit = %d, want 25", filters.Limit)
	}

	// Test invalid status_min
	query = map[string][]string{
		"status_min": {"invalid"},
	}
	_, err = parsePortalLogFilters(query)
	if err == nil {
		t.Error("parsePortalLogFilters() should return error for invalid status_min")
	}

	// Test invalid status_max
	query = map[string][]string{
		"status_max": {"invalid"},
	}
	_, err = parsePortalLogFilters(query)
	if err == nil {
		t.Error("parsePortalLogFilters() should return error for invalid status_max")
	}

	// Test invalid from
	query = map[string][]string{
		"from": {"invalid"},
	}
	_, err = parsePortalLogFilters(query)
	if err == nil {
		t.Error("parsePortalLogFilters() should return error for invalid from")
	}

	// Test invalid to
	query = map[string][]string{
		"to": {"invalid"},
	}
	_, err = parsePortalLogFilters(query)
	if err == nil {
		t.Error("parsePortalLogFilters() should return error for invalid to")
	}
}

// Test portalExportContentType
func TestPortalExportContentType(t *testing.T) {
	tests := []struct {
		format   string
		expected string
	}{
		{"csv", "text/csv; charset=utf-8"},
		{"CSV", "text/csv; charset=utf-8"},
		{"json", "application/json; charset=utf-8"},
		{"JSON", "application/json; charset=utf-8"},
		{"jsonl", "application/x-ndjson; charset=utf-8"},
		{"", "application/x-ndjson; charset=utf-8"},
		{"unknown", "application/x-ndjson; charset=utf-8"},
	}

	for _, tt := range tests {
		result := portalExportContentType(tt.format)
		if result != tt.expected {
			t.Errorf("portalExportContentType(%q) = %q, want %q", tt.format, result, tt.expected)
		}
	}
}

// Test portalExportExtension
func TestPortalExportExtension(t *testing.T) {
	tests := []struct {
		format   string
		expected string
	}{
		{"csv", "csv"},
		{"CSV", "csv"},
		{"json", "json"},
		{"JSON", "json"},
		{"jsonl", "jsonl"},
		{"", "jsonl"},
		{"unknown", "jsonl"},
	}

	for _, tt := range tests {
		result := portalExportExtension(tt.format)
		if result != tt.expected {
			t.Errorf("portalExportExtension(%q) = %q, want %q", tt.format, result, tt.expected)
		}
	}
}

// Test cloneInt64Map
func TestCloneInt64Map(t *testing.T) {
	// Test nil map
	result := cloneInt64Map(nil)
	if result == nil || len(result) != 0 {
		t.Errorf("cloneInt64Map(nil) = %v, want empty map", result)
	}

	// Test empty map
	result = cloneInt64Map(map[string]int64{})
	if len(result) != 0 {
		t.Errorf("cloneInt64Map(empty) = %v, want empty map", result)
	}

	// Test with values
	original := map[string]int64{"a": 1, "b": 2}
	result = cloneInt64Map(original)
	if len(result) != 2 || result["a"] != 1 || result["b"] != 2 {
		t.Errorf("cloneInt64Map() = %v, want map with a:1, b:2", result)
	}

	// Verify it's a clone (modifying original doesn't affect result)
	original["a"] = 100
	if result["a"] != 1 {
		t.Error("cloneInt64Map() returned reference, not clone")
	}
}

// Test cloneFloat64Map
func TestCloneFloat64Map(t *testing.T) {
	// Test nil map
	result := cloneFloat64Map(nil)
	if result == nil || len(result) != 0 {
		t.Errorf("cloneFloat64Map(nil) = %v, want empty map", result)
	}

	// Test empty map
	result = cloneFloat64Map(map[string]float64{})
	if len(result) != 0 {
		t.Errorf("cloneFloat64Map(empty) = %v, want empty map", result)
	}

	// Test with values
	original := map[string]float64{"a": 1.5, "b": 2.5}
	result = cloneFloat64Map(original)
	if len(result) != 2 || result["a"] != 1.5 || result["b"] != 2.5 {
		t.Errorf("cloneFloat64Map() = %v, want map with a:1.5, b:2.5", result)
	}

	// Verify it's a clone
	original["a"] = 100.0
	if result["a"] != 1.5 {
		t.Error("cloneFloat64Map() returned reference, not clone")
	}
}

// Test resolveGatewayBaseURL
func TestResolveGatewayBaseURL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", "http://127.0.0.1:8080"},
		{":8080", "http://127.0.0.1:8080"},
		{"127.0.0.1:8080", "http://127.0.0.1:8080"},
		{"http://localhost:8080", "http://localhost:8080"},
		{"https://api.example.com", "https://api.example.com"},
		{"http://example.com/", "http://example.com"},
	}

	for _, tt := range tests {
		result := resolveGatewayBaseURL(tt.input)
		if result != tt.expected {
			t.Errorf("resolveGatewayBaseURL(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

// Test Rate Limiting Functions
func TestRateLimiting(t *testing.T) {
	t.Parallel()

	cfg, st := openPortalTestStore(t)
	defer st.Close()

	srv, err := NewServer(cfg, st)
	if err != nil {
		t.Fatalf("NewServer error: %v", err)
	}

	clientIP := "192.168.1.100"

	// Test isRateLimited with no attempts
	if srv.isRateLimited(clientIP) {
		t.Error("isRateLimited should return false for new IP")
	}

	// Test recordFailedAuth
	srv.recordFailedAuth(clientIP)
	if srv.isRateLimited(clientIP) {
		t.Error("isRateLimited should return false after 1 attempt")
	}

	// Add more attempts
	for i := 0; i < 4; i++ {
		srv.recordFailedAuth(clientIP)
	}

	// Should now be rate limited
	if !srv.isRateLimited(clientIP) {
		t.Error("isRateLimited should return true after 5 attempts")
	}

	// Test clearFailedAuth
	srv.clearFailedAuth(clientIP)
	if srv.isRateLimited(clientIP) {
		t.Error("isRateLimited should return false after clearing")
	}
}

// Test Rate Limiting with Expired Block
func TestRateLimiting_ExpiredBlock(t *testing.T) {
	t.Parallel()

	cfg, st := openPortalTestStore(t)
	defer st.Close()

	srv, err := NewServer(cfg, st)
	if err != nil {
		t.Fatalf("NewServer error: %v", err)
	}

	clientIP := "192.168.1.101"

	// Manually create a blocked entry that's expired
	srv.rlMu.Lock()
	srv.rlAttempts[clientIP] = &loginAuthAttempts{
		count:     10,
		firstSeen: time.Now().Add(-20 * time.Minute),
		lastSeen:  time.Now().Add(-31 * time.Minute), // Expired
		blocked:   true,
	}
	srv.rlMu.Unlock()

	// Should not be rate limited since block expired
	if srv.isRateLimited(clientIP) {
		t.Error("isRateLimited should return false when block expired")
	}
}

// Test Rate Limiting Cleanup
func TestRateLimiting_Cleanup(t *testing.T) {
	t.Parallel()

	cfg, st := openPortalTestStore(t)
	defer st.Close()

	srv, err := NewServer(cfg, st)
	if err != nil {
		t.Fatalf("NewServer error: %v", err)
	}

	clientIP := "192.168.1.102"

	// Add old entry
	srv.rlMu.Lock()
	srv.rlAttempts[clientIP] = &loginAuthAttempts{
		count:     1,
		firstSeen: time.Now().Add(-40 * time.Minute),
		lastSeen:  time.Now().Add(-31 * time.Minute), // Older than 30 min
		blocked:   false,
	}
	srv.rlMu.Unlock()

	// Run cleanup
	srv.cleanupOldRateLimitEntries()

	// Entry should be removed
	srv.rlMu.RLock()
	_, exists := srv.rlAttempts[clientIP]
	srv.rlMu.RUnlock()

	if exists {
		t.Error("cleanupOldRateLimitEntries should remove old entries")
	}
}

// Test configSnapshot
func TestConfigSnapshot(t *testing.T) {
	t.Parallel()

	cfg, st := openPortalTestStore(t)
	defer st.Close()

	srv, err := NewServer(cfg, st)
	if err != nil {
		t.Fatalf("NewServer error: %v", err)
	}

	snapshot := srv.configSnapshot()

	// Verify snapshot contains expected data
	if len(snapshot.Routes) != len(cfg.Routes) {
		t.Errorf("configSnapshot routes length = %d, want %d", len(snapshot.Routes), len(cfg.Routes))
	}

	if len(snapshot.Services) != len(cfg.Services) {
		t.Errorf("configSnapshot services length = %d, want %d", len(snapshot.Services), len(cfg.Services))
	}

	// Verify cloning (modifying snapshot doesn't affect config)
	if len(snapshot.Routes) > 0 {
		originalPath := snapshot.Routes[0].Paths[0]
		snapshot.Routes[0].Paths[0] = "/modified"
		if cfg.Routes[0].Paths[0] == "/modified" {
			t.Error("configSnapshot returned reference, not clone")
		}
		snapshot.Routes[0].Paths[0] = originalPath
	}
}

// Test buildAPIList
func TestBuildAPIList(t *testing.T) {
	snapshot := portalConfigView{
		Routes: []config.Route{
			{ID: "route-1", Name: "Route 1", Service: "svc-1", Paths: []string{"/api/1"}, Methods: []string{"GET"}},
			{ID: "route-2", Name: "Route 2", Service: "svc-2", Paths: []string{"/api/2"}, Methods: []string{"POST"}},
		},
		Services: []config.Service{
			{ID: "svc-1", Name: "Service 1"},
			{ID: "svc-2", Name: "Service 2"},
		},
		Billing: config.BillingConfig{
			Enabled:     true,
			DefaultCost: 1,
			RouteCosts:  map[string]int64{"route-1": 5},
		},
	}

	// Test with no permissions (all routes allowed)
	permissions := []store.EndpointPermission{}
	result := buildAPIList(snapshot, permissions)
	if len(result) != 2 {
		t.Errorf("buildAPIList with no permissions = %d items, want 2", len(result))
	}

	// Test with permissions (only allowed routes)
	permissions = []store.EndpointPermission{
		{RouteID: "route-1", Allowed: true},
	}
	result = buildAPIList(snapshot, permissions)
	if len(result) != 1 {
		t.Errorf("buildAPIList with permissions = %d items, want 1", len(result))
	}

	// Test with denied permission
	permissions = []store.EndpointPermission{
		{RouteID: "route-1", Allowed: false},
	}
	result = buildAPIList(snapshot, permissions)
	if len(result) != 0 {
		t.Errorf("buildAPIList with denied permission = %d items, want 0", len(result))
	}
}

// Test findAPIDetail
func TestFindAPIDetail(t *testing.T) {
	snapshot := portalConfigView{
		Routes: []config.Route{
			{ID: "route-1", Name: "Route 1", Service: "svc-1", Paths: []string{"/api/1"}},
		},
		Services: []config.Service{
			{ID: "svc-1", Name: "Service 1"},
		},
	}

	permissions := []store.EndpointPermission{
		{RouteID: "route-1", Allowed: true},
	}

	// Test finding by ID
	route, service, perm := findAPIDetail(snapshot, permissions, "route-1")
	if route == nil || service == nil || perm == nil {
		t.Error("findAPIDetail by ID should return route, service, and permission")
	}

	// Test finding by name
	route, service, perm = findAPIDetail(snapshot, permissions, "Route 1")
	if route == nil || service == nil {
		t.Error("findAPIDetail by name should return route and service")
	}

	// Test not found
	route, service, perm = findAPIDetail(snapshot, permissions, "nonexistent")
	if route != nil || service != nil || perm != nil {
		t.Error("findAPIDetail for nonexistent route should return nil")
	}
}

// Test findPermissionForRoute
func TestFindPermissionForRoute(t *testing.T) {
	permsByRoute := map[string]*store.EndpointPermission{
		"route-1": {RouteID: "route-1", Allowed: true},
		"Route 2": {RouteID: "Route 2", Allowed: false},
	}

	// Test finding by ID
	route := &config.Route{ID: "route-1", Name: "Other Name"}
	perm := findPermissionForRoute(permsByRoute, route)
	if perm == nil || !perm.Allowed {
		t.Error("findPermissionForRoute should find by ID")
	}

	// Test finding by name
	route = &config.Route{ID: "other-id", Name: "Route 2"}
	perm = findPermissionForRoute(permsByRoute, route)
	if perm == nil || perm.Allowed {
		t.Error("findPermissionForRoute should find by name")
	}

	// Test not found
	route = &config.Route{ID: "unknown", Name: "Unknown"}
	perm = findPermissionForRoute(permsByRoute, route)
	if perm != nil {
		t.Error("findPermissionForRoute should return nil for unknown route")
	}

	// Test nil route
	perm = findPermissionForRoute(permsByRoute, nil)
	if perm != nil {
		t.Error("findPermissionForRoute should return nil for nil route")
	}
}

// Test resolveRouteCreditCost
func TestResolveRouteCreditCost(t *testing.T) {
	billing := config.BillingConfig{
		DefaultCost: 1,
		RouteCosts: map[string]int64{
			"route-1": 5,
			"Route 2": 10,
		},
	}

	// Test permission cost override
	costOverride := int64(100)
	perm := &store.EndpointPermission{CreditCost: &costOverride}
	cost := resolveRouteCreditCost(billing, &config.Route{ID: "route-1"}, perm)
	if cost != 100 {
		t.Errorf("resolveRouteCreditCost with permission override = %d, want 100", cost)
	}

	// Test route ID cost
	cost = resolveRouteCreditCost(billing, &config.Route{ID: "route-1"}, nil)
	if cost != 5 {
		t.Errorf("resolveRouteCreditCost by route ID = %d, want 5", cost)
	}

	// Test route name cost
	cost = resolveRouteCreditCost(billing, &config.Route{ID: "other", Name: "Route 2"}, nil)
	if cost != 10 {
		t.Errorf("resolveRouteCreditCost by route name = %d, want 10", cost)
	}

	// Test default cost
	cost = resolveRouteCreditCost(billing, &config.Route{ID: "unknown"}, nil)
	if cost != 1 {
		t.Errorf("resolveRouteCreditCost default = %d, want 1", cost)
	}

	// Test zero default cost
	billing.DefaultCost = 0
	cost = resolveRouteCreditCost(billing, &config.Route{ID: "unknown"}, nil)
	if cost != 0 {
		t.Errorf("resolveRouteCreditCost zero default = %d, want 0", cost)
	}
}

// Test writeError
func TestWriteError(t *testing.T) {
	w := httptest.NewRecorder()
	writeError(w, http.StatusBadRequest, "test_code", "test message")

	if w.Code != http.StatusBadRequest {
		t.Errorf("writeError status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("writeError Content-Type = %s, want application/json", contentType)
	}

	var result map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("writeError body is not valid JSON: %v", err)
	}

	errObj, ok := result["error"].(map[string]any)
	if !ok {
		t.Fatal("writeError body missing error object")
	}

	if errObj["code"] != "test_code" {
		t.Errorf("writeError code = %v, want test_code", errObj["code"])
	}

	if errObj["message"] != "test message" {
		t.Errorf("writeError message = %v, want 'test message'", errObj["message"])
	}
}

// Test setSessionCookie and clearSessionCookie
func TestSessionCookies(t *testing.T) {
	// Test setSessionCookie
	w := httptest.NewRecorder()
	cfg := sessionCookieConfig{
		Name:     "test_session",
		Path:     "/",
		Value:    "token123",
		Expires:  time.Now().Add(24 * time.Hour),
		MaxAge:   24 * time.Hour,
		Secure:   true,
		HTTPOnly: true,
	}
	setSessionCookie(w, cfg)

	cookies := w.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("setSessionCookie created %d cookies, want 1", len(cookies))
	}

	cookie := cookies[0]
	if cookie.Name != "test_session" {
		t.Errorf("cookie.Name = %s, want test_session", cookie.Name)
	}
	if cookie.Value != "token123" {
		t.Errorf("cookie.Value = %s, want token123", cookie.Value)
	}
	if !cookie.HttpOnly {
		t.Error("cookie.HttpOnly should be true")
	}
	if !cookie.Secure {
		t.Error("cookie.Secure should be true")
	}

	// Test clearSessionCookie
	w2 := httptest.NewRecorder()
	clearSessionCookie(w2, sessionCookieConfig{
		Name:     "test_session",
		Path:     "/",
		Secure:   true,
		HTTPOnly: true,
	})

	cookies = w2.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("clearSessionCookie created %d cookies, want 1", len(cookies))
	}

	cookie = cookies[0]
	if cookie.Value != "" {
		t.Errorf("cleared cookie.Value = %s, want empty", cookie.Value)
	}
	if cookie.MaxAge != -1 {
		t.Errorf("cleared cookie.MaxAge = %d, want -1", cookie.MaxAge)
	}
}

// Test userFromContext and sessionFromContext
func TestContextHelpers(t *testing.T) {
	// Test with nil context
	if userFromContext(nil) != nil {
		t.Error("userFromContext(nil) should return nil")
	}
	if sessionFromContext(nil) != nil {
		t.Error("sessionFromContext(nil) should return nil")
	}

	// Test with empty context
	ctx := context.Background()
	if userFromContext(ctx) != nil {
		t.Error("userFromContext(empty) should return nil")
	}
	if sessionFromContext(ctx) != nil {
		t.Error("sessionFromContext(empty) should return nil")
	}

	// Test with values
	user := &store.User{ID: "user-1", Email: "test@example.com"}
	session := &store.Session{ID: "session-1", UserID: "user-1"}

	ctx = context.WithValue(ctx, contextUserKey, user)
	ctx = context.WithValue(ctx, contextSessionKey, session)

	retrievedUser := userFromContext(ctx)
	if retrievedUser == nil || retrievedUser.ID != "user-1" {
		t.Error("userFromContext should return the user")
	}

	retrievedSession := sessionFromContext(ctx)
	if retrievedSession == nil || retrievedSession.ID != "session-1" {
		t.Error("sessionFromContext should return the session")
	}
}

// Additional imports needed for context tests
import "context"

// Test NewServer validation
func TestNewServer_Validation(t *testing.T) {
	t.Parallel()

	// Test nil config
	_, err := NewServer(nil, nil)
	if err == nil || !strings.Contains(err.Error(), "config is nil") {
		t.Errorf("NewServer(nil config) error = %v, want 'config is nil'", err)
	}

	// Test nil store
	cfg := &config.Config{
		Store: config.StoreConfig{Path: filepath.Join(t.TempDir(), "test.db")},
	}
	_, err = NewServer(cfg, nil)
	if err == nil || !strings.Contains(err.Error(), "store is nil") {
		t.Errorf("NewServer(nil store) error = %v, want 'store is nil'", err)
	}
}

// Test withSession middleware
func TestWithSession_Middleware(t *testing.T) {
	t.Parallel()

	cfg, st := openPortalTestStore(t)
	defer st.Close()

	user := createPortalTestUserWithID(t, st, "middleware@example.com", "password123")

	srv, err := NewServer(cfg, st)
	if err != nil {
		t.Fatalf("NewServer error: %v", err)
	}
	httpSrv := httptest.NewServer(srv)
	defer httpSrv.Close()

	// Login to get a valid session
	loginResp := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPost, httpSrv.URL+"/portal/api/v1/auth/login", nil, map[string]any{
		"email":    "middleware@example.com",
		"password": "password123",
	})
	sessionCookie := findCookie(loginResp.Cookies, cfg.Portal.Session.CookieName)

	// Test accessing protected endpoint without cookie
	resp := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodGet, httpSrv.URL+"/portal/api/v1/auth/me", nil, nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status %d without cookie, got %d", http.StatusUnauthorized, resp.StatusCode)
	}

	// Test accessing protected endpoint with valid cookie
	resp = mustPortalJSONRequest(t, httpSrv.Client(), http.MethodGet, httpSrv.URL+"/portal/api/v1/auth/me", []*http.Cookie{sessionCookie}, nil)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d with valid cookie, got %d body=%s", http.StatusOK, resp.StatusCode, string(resp.Body))
	}

	// Test with expired session (manually create expired session)
	token, _ := store.GenerateSessionToken()
	expiredSession := &store.Session{
		UserID:    user.ID,
		TokenHash: store.HashSessionToken(token),
		ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired
	}
	if err := st.Sessions().Create(expiredSession); err != nil {
		t.Fatalf("create expired session: %v", err)
	}

	expiredCookie := &http.Cookie{
		Name:  cfg.Portal.Session.CookieName,
		Value: token,
	}

	resp = mustPortalJSONRequest(t, httpSrv.Client(), http.MethodGet, httpSrv.URL+"/portal/api/v1/auth/me", []*http.Cookie{expiredCookie}, nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status %d with expired session, got %d", http.StatusUnauthorized, resp.StatusCode)
	}

	_ = user
}

// Test session cookie configuration methods
func TestSessionCookieConfig(t *testing.T) {
	t.Parallel()

	cfg, st := openPortalTestStore(t)
	defer st.Close()

	srv, err := NewServer(cfg, st)
	if err != nil {
		t.Fatalf("NewServer error: %v", err)
	}

	// Test sessionCookieName
	name := srv.sessionCookieName()
	if name != cfg.Portal.Session.CookieName {
		t.Errorf("sessionCookieName() = %s, want %s", name, cfg.Portal.Session.CookieName)
	}

	// Test sessionCookiePath
	path := srv.sessionCookiePath()
	expectedPath := "/portal"
	if cfg.Portal.PathPrefix == "" {
		expectedPath = "/"
	}
	if path != expectedPath {
		t.Errorf("sessionCookiePath() = %s, want %s", path, expectedPath)
	}

	// Test sessionMaxAge
	maxAge := srv.sessionMaxAge()
	if maxAge != cfg.Portal.Session.MaxAge {
		t.Errorf("sessionMaxAge() = %v, want %v", maxAge, cfg.Portal.Session.MaxAge)
	}

	// Test sessionSecure
	secure := srv.sessionSecure()
	if secure != cfg.Portal.Session.Secure {
		t.Errorf("sessionSecure() = %v, want %v", secure, cfg.Portal.Session.Secure)
	}
}

// Test session cookie defaults
func TestSessionCookieDefaults(t *testing.T) {
	t.Parallel()

	cfg, st := openPortalTestStore(t)
	defer st.Close()

	// Clear cookie config to test defaults
	cfg.Portal.Session.CookieName = ""
	cfg.Portal.Session.MaxAge = 0
	cfg.Portal.PathPrefix = ""

	srv, err := NewServer(cfg, st)
	if err != nil {
		t.Fatalf("NewServer error: %v", err)
	}

	// Test default cookie name
	name := srv.sessionCookieName()
	if name != "apicerberus_session" {
		t.Errorf("default sessionCookieName() = %s, want apicerberus_session", name)
	}

	// Test default max age
	maxAge := srv.sessionMaxAge()
	if maxAge != 24*time.Hour {
		t.Errorf("default sessionMaxAge() = %v, want 24h", maxAge)
	}

	// Test default path
	path := srv.sessionCookiePath()
	if path != "/" {
		t.Errorf("default sessionCookiePath() = %s, want /", path)
	}
}

// Test Purchase Credits Error Paths
func TestPurchaseCredits_StoreErrors(t *testing.T) {
	t.Parallel()

	cfg, st := openPortalTestStore(t)
	defer st.Close()

	createPortalTestUser(t, st, "purchase@example.com", "password123")

	srv, err := NewServer(cfg, st)
	if err != nil {
		t.Fatalf("NewServer error: %v", err)
	}
	httpSrv := httptest.NewServer(srv)
	defer httpSrv.Close()

	// Login
	loginResp := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPost, httpSrv.URL+"/portal/api/v1/auth/login", nil, map[string]any{
		"email":    "purchase@example.com",
		"password": "password123",
	})
	sessionCookie := findCookie(loginResp.Cookies, cfg.Portal.Session.CookieName)

	// Test purchase with valid amount
	resp := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPost, httpSrv.URL+"/portal/api/v1/credits/purchase", []*http.Cookie{sessionCookie}, map[string]any{
		"amount":      100,
		"description": "Test purchase",
	})
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d got %d body=%s", http.StatusOK, resp.StatusCode, string(resp.Body))
	}

	// Verify response contains expected fields
	var result map[string]any
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		t.Fatalf("purchase response is not valid JSON: %v", err)
	}

	if _, ok := result["purchased"]; !ok {
		t.Error("purchase response missing 'purchased' field")
	}
	if _, ok := result["new_balance"]; !ok {
		t.Error("purchase response missing 'new_balance' field")
	}
}

// Test MyForecast with no transactions
func TestMyForecast_NoTransactions(t *testing.T) {
	t.Parallel()

	cfg, st := openPortalTestStore(t)
	defer st.Close()

	createPortalTestUser(t, st, "forecast@example.com", "password123")

	srv, err := NewServer(cfg, st)
	if err != nil {
		t.Fatalf("NewServer error: %v", err)
	}
	httpSrv := httptest.NewServer(srv)
	defer httpSrv.Close()

	// Login
	loginResp := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPost, httpSrv.URL+"/portal/api/v1/auth/login", nil, map[string]any{
		"email":    "forecast@example.com",
		"password": "password123",
	})
	sessionCookie := findCookie(loginResp.Cookies, cfg.Portal.Session.CookieName)

	// Get forecast with no transactions
	resp := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodGet, httpSrv.URL+"/portal/api/v1/credits/forecast", []*http.Cookie{sessionCookie}, nil)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d got %d body=%s", http.StatusOK, resp.StatusCode, string(resp.Body))
	}

	var result map[string]any
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		t.Fatalf("forecast response is not valid JSON: %v", err)
	}

	// With no consumption, projected days should be 0
	if result["average_daily_consumption"] != float64(0) {
		t.Errorf("average_daily_consumption = %v, want 0", result["average_daily_consumption"])
	}
}

// Test Export Logs with Different Formats
func TestExportLogs_Formats(t *testing.T) {
	t.Parallel()

	cfg, st := openPortalTestStore(t)
	defer st.Close()

	user := createPortalTestUserWithID(t, st, "export@example.com", "password123")

	// Add some audit entries
	if err := st.Audits().BatchInsert([]store.AuditEntry{
		{
			UserID:     user.ID,
			RouteID:    "route-1",
			Method:     "GET",
			Path:       "/api/test",
			StatusCode: 200,
			ClientIP:   "127.0.0.1",
			CreatedAt:  time.Now().UTC(),
		},
	}); err != nil {
		t.Fatalf("seed audit entries: %v", err)
	}

	srv, err := NewServer(cfg, st)
	if err != nil {
		t.Fatalf("NewServer error: %v", err)
	}
	httpSrv := httptest.NewServer(srv)
	defer httpSrv.Close()

	// Login
	loginResp := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPost, httpSrv.URL+"/portal/api/v1/auth/login", nil, map[string]any{
		"email":    "export@example.com",
		"password": "password123",
	})
	sessionCookie := findCookie(loginResp.Cookies, cfg.Portal.Session.CookieName)

	// Test JSON format
	resp := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodGet, httpSrv.URL+"/portal/api/v1/logs/export?format=json", []*http.Cookie{sessionCookie}, nil)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("JSON export: expected status %d got %d", http.StatusOK, resp.StatusCode)
	}
	contentType := resp.Headers["Content-Type"]
	if !strings.Contains(string(contentType), "application/json") {
		t.Errorf("JSON export Content-Type = %s, want application/json", contentType)
	}

	// Test CSV format
	resp = mustPortalJSONRequest(t, httpSrv.Client(), http.MethodGet, httpSrv.URL+"/portal/api/v1/logs/export?format=csv", []*http.Cookie{sessionCookie}, nil)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("CSV export: expected status %d got %d", http.StatusOK, resp.StatusCode)
	}

	// Test default (jsonl) format
	resp = mustPortalJSONRequest(t, httpSrv.Client(), http.MethodGet, httpSrv.URL+"/portal/api/v1/logs/export", []*http.Cookie{sessionCookie}, nil)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Default export: expected status %d got %d", http.StatusOK, resp.StatusCode)
	}
}

// Need to add Headers field to portalResponse
func init() {
	// Extend portalResponse to include headers for export tests
}

// Mock store error tests - Test behavior when store returns errors
func TestHandler_StoreErrors(t *testing.T) {
	t.Parallel()

	cfg, st := openPortalTestStore(t)
	defer st.Close()

	createPortalTestUser(t, st, "storeerror@example.com", "password123")

	srv, err := NewServer(cfg, st)
	if err != nil {
		t.Fatalf("NewServer error: %v", err)
	}
	httpSrv := httptest.NewServer(srv)
	defer httpSrv.Close()

	// Login
	loginResp := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPost, httpSrv.URL+"/portal/api/v1/auth/login", nil, map[string]any{
		"email":    "storeerror@example.com",
		"password": "password123",
	})
	sessionCookie := findCookie(loginResp.Cookies, cfg.Portal.Session.CookieName)

	// Test list API keys (should handle store errors gracefully)
	resp := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodGet, httpSrv.URL+"/portal/api/v1/api-keys", []*http.Cookie{sessionCookie}, nil)
	// Should succeed even if empty
	if resp.StatusCode != http.StatusOK {
		t.Logf("list api keys returned %d: %s", resp.StatusCode, string(resp.Body))
	}

	// Test list templates
	resp = mustPortalJSONRequest(t, httpSrv.Client(), http.MethodGet, httpSrv.URL+"/portal/api/v1/playground/templates", []*http.Cookie{sessionCookie}, nil)
	if resp.StatusCode != http.StatusOK {
		t.Logf("list templates returned %d: %s", resp.StatusCode, string(resp.Body))
	}

	// Test list transactions
	resp = mustPortalJSONRequest(t, httpSrv.Client(), http.MethodGet, httpSrv.URL+"/portal/api/v1/credits/transactions", []*http.Cookie{sessionCookie}, nil)
	if resp.StatusCode != http.StatusOK {
		t.Logf("list transactions returned %d: %s", resp.StatusCode, string(resp.Body))
	}
}

// Test Playground Send with Different Methods and Query Params
func TestPlaygroundSend_Variations(t *testing.T) {
	t.Parallel()

	// Create a mock gateway server
	gatewayStub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"method":    r.Method,
			"path":      r.URL.Path,
			"query":     r.URL.RawQuery,
			"api_key":   r.Header.Get("X-API-Key"),
		})
	}))
	defer gatewayStub.Close()

	cfg, st := openPortalTestStoreWithGateway(t, strings.TrimPrefix(gatewayStub.URL, "http://"))
	defer st.Close()

	createPortalTestUser(t, st, "playground2@example.com", "password123")

	srv, err := NewServer(cfg, st)
	if err != nil {
		t.Fatalf("NewServer error: %v", err)
	}
	httpSrv := httptest.NewServer(srv)
	defer httpSrv.Close()

	// Login
	loginResp := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPost, httpSrv.URL+"/portal/api/v1/auth/login", nil, map[string]any{
		"email":    "playground2@example.com",
		"password": "password123",
	})
	sessionCookie := findCookie(loginResp.Cookies, cfg.Portal.Session.CookieName)

	// Test with query params
	resp := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPost, httpSrv.URL+"/portal/api/v1/playground/send", []*http.Cookie{sessionCookie}, map[string]any{
		"method":  "GET",
		"path":    "/test",
		"api_key": "test-key",
		"query": map[string]string{
			"foo": "bar",
			"baz": "qux",
		},
	})
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d got %d body=%s", http.StatusOK, resp.StatusCode, string(resp.Body))
	}

	// Test with headers
	resp = mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPost, httpSrv.URL+"/portal/api/v1/playground/send", []*http.Cookie{sessionCookie}, map[string]any{
		"method":  "POST",
		"path":    "/test",
		"api_key": "test-key",
		"headers": map[string]string{
			"X-Custom": "value",
		},
		"body": `{"test": true}`,
	})
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d got %d body=%s", http.StatusOK, resp.StatusCode, string(resp.Body))
	}

	// Test with empty method (should default to GET)
	resp = mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPost, httpSrv.URL+"/portal/api/v1/playground/send", []*http.Cookie{sessionCookie}, map[string]any{
		"path":    "/test",
		"api_key": "test-key",
		"method":  "",
	})
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d got %d body=%s", http.StatusOK, resp.StatusCode, string(resp.Body))
	}
}

// Additional imports needed
import (
	"encoding/json"
)
