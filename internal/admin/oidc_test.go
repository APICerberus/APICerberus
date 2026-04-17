package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/APICerberus/APICerebrus/internal/config"
)

func TestIssueAdminTokenWithPayload(t *testing.T) {
	t.Parallel()

	secret := "test-secret-at-least-32-chars-long!!"
	payload := map[string]any{
		"sub":   "oidc:abc123",
		"email": "user@example.com",
		"role":  "manager",
		"iat":   time.Now().UTC().Unix(),
		"exp":   time.Now().UTC().Add(15 * time.Minute).Unix(),
	}

	token, err := issueAdminTokenWithPayload(secret, 15*time.Minute, payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("expected 3 parts, got %d", len(parts))
	}

	err = verifyAdminToken(token, secret, 1)
	if err != nil {
		t.Errorf("token verification failed: %v", err)
	}

	role, perms := extractRoleFromJWT(token)
	if role != "manager" {
		t.Errorf("role = %q, want %q", role, "manager")
	}
	if perms != nil {
		t.Errorf("expected no perms in this token, got %v", perms)
	}
}

func TestIssueAdminTokenWithPayload_EmptySecret(t *testing.T) {
	t.Parallel()

	_, err := issueAdminTokenWithPayload("", 0, nil)
	if err == nil {
		t.Error("expected error for empty secret")
	}
}

func TestExtractClaimName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		claims   map[string]any
		expected string
	}{
		{"name claim", map[string]any{"name": "John Doe", "email": "john@example.com"}, "John Doe"},
		{"given + family", map[string]any{"given_name": "John", "family_name": "Doe", "email": "john@example.com"}, "John Doe"},
		{"given only", map[string]any{"given_name": "John", "email": "john@example.com"}, "John"},
		{"fallback email", map[string]any{"email": "john@example.com"}, "john@example.com"},
		{"empty", map[string]any{}, "SSO User"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := extractClaimName(tt.claims)
			if got != tt.expected {
				t.Errorf("extractClaimName(%v) = %q, want %q", tt.claims, got, tt.expected)
			}
		})
	}
}

func TestMapOIDCRole_Groups(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		claims   map[string]any
		expected string
	}{
		{"admin group", map[string]any{"groups": []any{"admin", "developers"}}, "admin"},
		{"admins group", map[string]any{"groups": []any{"admins"}}, "admin"},
		{"apicerberus-admin", map[string]any{"groups": []any{"apicerberus-admin"}}, "admin"},
		{"manager group", map[string]any{"groups": []any{"managers"}}, "manager"},
		{"apicerberus-manager", map[string]any{"groups": []any{"apicerberus-manager"}}, "manager"},
		{"no matching", map[string]any{"groups": []any{"developers"}}, ""},
		{"no groups", map[string]any{"email": "user@example.com"}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := mapOIDCRole(tt.claims, config.OIDCConfig{})
			if got != tt.expected {
				t.Errorf("mapOIDCRole(%v) = %q, want %q", tt.claims, got, tt.expected)
			}
		})
	}
}

func TestMapOIDCRole_ClaimMapping(t *testing.T) {
	t.Parallel()

	cfg := config.OIDCConfig{ClaimMapping: map[string]string{"role": "custom_role"}}

	tests := []struct {
		name     string
		claims   map[string]any
		expected string
	}{
		{"valid role", map[string]any{"custom_role": "viewer"}, "viewer"},
		{"invalid role", map[string]any{"custom_role": "superadmin"}, ""},
		{"missing claim", map[string]any{"email": "user@example.com"}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := mapOIDCRole(tt.claims, cfg)
			if got != tt.expected {
				t.Errorf("mapOIDCRole(%v) = %q, want %q", tt.claims, got, tt.expected)
			}
		})
	}
}

func TestConstantTimeEqual(t *testing.T) {
	t.Parallel()

	if !constantTimeEqual("abc", "abc") {
		t.Error("expected equal strings to match")
	}
	if constantTimeEqual("abc", "def") {
		t.Error("expected different strings to not match")
	}
	if constantTimeEqual("abc", "ab") {
		t.Error("expected different length strings to not match")
	}
}

func TestGenerateRandomHex(t *testing.T) {
	t.Parallel()

	s1, err := generateRandomHex(32)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s2, err := generateRandomHex(32)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s1 == s2 {
		t.Error("expected different random values")
	}
	if len(s1) != 64 {
		t.Errorf("expected 64 hex chars, got %d", len(s1))
	}
}

func TestHandleOIDCStatus_Disabled(t *testing.T) {
	t.Parallel()

	srv := &Server{cfg: &config.Config{}}
	req := httptest.NewRequest("GET", "/admin/api/v1/auth/sso/status", nil)
	rec := httptest.NewRecorder()
	srv.handleOIDCStatus(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if body["enabled"] != false {
		t.Error("expected enabled=false")
	}
}

func TestHandleOIDCStatus_Enabled(t *testing.T) {
	t.Parallel()

	srv := &Server{cfg: &config.Config{
		Admin: config.AdminConfig{
			OIDC: config.OIDCConfig{
				Enabled:       true,
				IssuerURL:     "https://accounts.google.com",
				ClientID:      "test-client-id.apps.googleusercontent.com",
				ClientSecret:  "secret",
				RedirectURL:   "http://localhost:9876/callback",
				Scopes:        []string{"openid", "email"},
				AutoProvision: true,
				DefaultRole:   "user",
			},
		},
	}}

	req := httptest.NewRequest("GET", "/admin/api/v1/auth/sso/status", nil)
	req = req.WithContext(contextWithRole(req.Context(), "admin", RolePermissions[RoleAdmin]))
	rec := httptest.NewRecorder()
	srv.handleOIDCStatus(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if body["enabled"] != true {
		t.Error("expected enabled=true")
	}
	if body["issuer_url"] != "https://accounts.google.com" {
		t.Errorf("issuer_url = %v", body["issuer_url"])
	}
	if _, exists := body["client_secret"]; exists {
		t.Error("client_secret should not be exposed")
	}
}

func TestHandleOIDCLogin_NotConfigured(t *testing.T) {
	t.Parallel()

	srv := &Server{cfg: &config.Config{}}
	req := httptest.NewRequest("GET", "/admin/api/v1/auth/sso/login", nil)
	rec := httptest.NewRecorder()
	srv.handleOIDCLogin(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	errObj, _ := body["error"].(map[string]any)
	if errObj == nil || errObj["code"] != "oidc_not_configured" {
		t.Errorf("expected oidc_not_configured error, got %v", body)
	}
}

func TestHandleOIDCLogout_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	srv := &Server{cfg: &config.Config{
		Admin: config.AdminConfig{
			OIDC: config.OIDCConfig{Enabled: true},
		},
	}}

	req := httptest.NewRequest("GET", "/admin/api/v1/auth/sso/logout", nil)
	rec := httptest.NewRecorder()
	srv.handleOIDCLogout(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleOIDCLogout_NotConfigured(t *testing.T) {
	t.Parallel()

	srv := &Server{cfg: &config.Config{}}
	req := httptest.NewRequest("POST", "/admin/api/v1/auth/sso/logout", nil)
	rec := httptest.NewRecorder()
	srv.handleOIDCLogout(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}
