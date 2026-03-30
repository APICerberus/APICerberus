package portal

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/APICerberus/APICerebrus/internal/config"
	"github.com/APICerberus/APICerebrus/internal/store"
)

func TestPortalAuthSessionFlow(t *testing.T) {
	t.Parallel()

	cfg, st := openPortalTestStore(t)
	defer st.Close()
	createPortalTestUser(t, st, "portal-user@example.com", "portal-pass")

	srv, err := NewServer(cfg, st)
	if err != nil {
		t.Fatalf("NewServer error: %v", err)
	}
	httpSrv := httptest.NewServer(srv)
	defer httpSrv.Close()

	loginResp := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPost, httpSrv.URL+"/portal/api/v1/auth/login", nil, map[string]any{
		"email":    "portal-user@example.com",
		"password": "portal-pass",
	})
	if loginResp.StatusCode != http.StatusOK {
		t.Fatalf("expected login 200 got %d body=%s", loginResp.StatusCode, string(loginResp.Body))
	}
	sessionCookie := findCookie(loginResp.Cookies, cfg.Portal.Session.CookieName)
	if sessionCookie == nil || sessionCookie.Value == "" {
		t.Fatalf("expected session cookie %q to be set", cfg.Portal.Session.CookieName)
	}

	meResp := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodGet, httpSrv.URL+"/portal/api/v1/auth/me", []*http.Cookie{sessionCookie}, nil)
	if meResp.StatusCode != http.StatusOK {
		t.Fatalf("expected me 200 got %d body=%s", meResp.StatusCode, string(meResp.Body))
	}

	logoutResp := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPost, httpSrv.URL+"/portal/api/v1/auth/logout", []*http.Cookie{sessionCookie}, map[string]any{})
	if logoutResp.StatusCode != http.StatusOK {
		t.Fatalf("expected logout 200 got %d body=%s", logoutResp.StatusCode, string(logoutResp.Body))
	}

	postLogoutMe := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodGet, httpSrv.URL+"/portal/api/v1/auth/me", []*http.Cookie{sessionCookie}, nil)
	if postLogoutMe.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected me after logout to return 401 got %d body=%s", postLogoutMe.StatusCode, string(postLogoutMe.Body))
	}
}

func TestPortalLoginRejectsInvalidCredentials(t *testing.T) {
	t.Parallel()

	cfg, st := openPortalTestStore(t)
	defer st.Close()
	createPortalTestUser(t, st, "portal-invalid@example.com", "portal-pass")

	srv, err := NewServer(cfg, st)
	if err != nil {
		t.Fatalf("NewServer error: %v", err)
	}
	httpSrv := httptest.NewServer(srv)
	defer httpSrv.Close()

	resp := mustPortalJSONRequest(t, httpSrv.Client(), http.MethodPost, httpSrv.URL+"/portal/api/v1/auth/login", nil, map[string]any{
		"email":    "portal-invalid@example.com",
		"password": "wrong",
	})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected login 401 got %d body=%s", resp.StatusCode, string(resp.Body))
	}
}

type portalResponse struct {
	StatusCode int
	Body       []byte
	Cookies    []*http.Cookie
}

func mustPortalJSONRequest(t *testing.T, client *http.Client, method, rawURL string, cookies []*http.Cookie, payload any) portalResponse {
	t.Helper()

	var bodyReader *bytes.Reader
	if payload == nil {
		bodyReader = bytes.NewReader(nil)
	} else {
		data, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("json marshal request: %v", err)
		}
		bodyReader = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, rawURL, bodyReader)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	return portalResponse{
		StatusCode: resp.StatusCode,
		Body:       body,
		Cookies:    resp.Cookies(),
	}
}

func findCookie(cookies []*http.Cookie, name string) *http.Cookie {
	for _, cookie := range cookies {
		if cookie == nil {
			continue
		}
		if cookie.Name == name {
			return cookie
		}
	}
	return nil
}

func openPortalTestStore(t *testing.T) (*config.Config, *store.Store) {
	t.Helper()

	cfg := &config.Config{
		Store: config.StoreConfig{
			Path:        filepath.Join(t.TempDir(), "portal-auth.db"),
			BusyTimeout: time.Second,
			JournalMode: "WAL",
			ForeignKeys: true,
		},
		Portal: config.PortalConfig{
			Enabled:    true,
			Addr:       "127.0.0.1:0",
			PathPrefix: "/portal",
			Session: config.PortalSessionConfig{
				CookieName: "portal_test_session",
				MaxAge:     2 * time.Hour,
				Secure:     false,
			},
		},
	}
	st, err := store.Open(cfg)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	return cfg, st
}

func createPortalTestUser(t *testing.T, st *store.Store, email, password string) {
	t.Helper()
	hash, err := store.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword error: %v", err)
	}
	user := &store.User{
		Email:        email,
		Name:         "Portal User",
		PasswordHash: hash,
		Role:         "user",
		Status:       "active",
	}
	if err := st.Users().Create(user); err != nil {
		t.Fatalf("create user: %v", err)
	}
}
