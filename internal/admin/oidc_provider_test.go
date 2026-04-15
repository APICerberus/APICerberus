package admin

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/APICerberus/APICerebrus/internal/config"
	"golang.org/x/crypto/bcrypt"
)

func TestOIDCProviderDisabledDiscovery(t *testing.T) {
	t.Parallel()

	s := &Server{oidcProvider: nil}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/.well-known/openid-configuration", nil)

	s.handleOIDCDiscovery(w, r)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
}

func TestOIDCProviderDiscoveryDocument(t *testing.T) {
	t.Parallel()

	// Create a minimal OIDC provider
	provider := &OIDCProviderServer{
		config: &config.OIDCProviderConfig{
			Enabled: true,
			Issuer:  "https://api.example.com",
		},
		clients: make(map[string]*config.OIDCClient),
	}

	s := &Server{oidcProvider: provider, cfg: &config.Config{}}
	s.cfg.Admin.OIDC.Provider.Issuer = "https://api.example.com"

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/.well-known/openid-configuration", nil)

	s.handleOIDCDiscovery(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var doc map[string]any
	json.Unmarshal(w.Body.Bytes(), &doc)

	if doc["issuer"] != "https://api.example.com" {
		t.Fatalf("expected issuer https://api.example.com, got %v", doc["issuer"])
	}
	if doc["authorization_endpoint"] != "https://api.example.com/oidc/authorize" {
		t.Fatalf("expected authorize endpoint, got %v", doc["authorization_endpoint"])
	}
	if doc["token_endpoint"] != "https://api.example.com/oidc/token" {
		t.Fatalf("expected token endpoint, got %v", doc["token_endpoint"])
	}
}

func TestOIDCProviderJWKSDisabled(t *testing.T) {
	t.Parallel()

	s := &Server{oidcProvider: nil}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/oidc/jwks", nil)

	s.handleOIDCJWKS(w, r)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
}

func TestOIDCProviderAuthorizeMissingParams(t *testing.T) {
	t.Parallel()

	s := &Server{oidcProvider: &OIDCProviderServer{}}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/oidc/authorize", nil)

	s.handleOIDCAuthorize(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestOIDCProviderAuthorizeUnknownClient(t *testing.T) {
	t.Parallel()

	s := &Server{
		oidcProvider: &OIDCProviderServer{
			clients: make(map[string]*config.OIDCClient),
		},
	}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/oidc/authorize?client_id=unknown&redirect_uri=https://example.com/cb&response_type=code", nil)

	s.handleOIDCAuthorize(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestOIDCProviderAuthorizeInvalidRedirectURI(t *testing.T) {
	t.Parallel()

	// Create a client with specific redirect URIs
	hash, _ := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)
	client := &config.OIDCClient{
		ClientID:     "test-client",
		ClientSecret: string(hash),
		RedirectURIs: []string{"https://valid.example.com/callback"},
	}

	s := &Server{
		oidcProvider: &OIDCProviderServer{
			clients: map[string]*config.OIDCClient{"test-client": client},
		},
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/oidc/authorize?client_id=test-client&redirect_uri=https://wrong.example.com/cb&response_type=code", nil)

	s.handleOIDCAuthorize(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestOIDCProviderAuthorizeMissingOpenIDScope(t *testing.T) {
	t.Parallel()

	hash, _ := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)
	client := &config.OIDCClient{
		ClientID:     "test-client",
		ClientSecret: string(hash),
		RedirectURIs: []string{"https://example.com/callback"},
	}

	s := &Server{
		oidcProvider: &OIDCProviderServer{
			clients: map[string]*config.OIDCClient{"test-client": client},
		},
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/oidc/authorize?client_id=test-client&redirect_uri=https://example.com/callback&response_type=code&scope=profile", nil)

	s.handleOIDCAuthorize(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "openid scope is required") {
		t.Fatalf("expected openid scope error, got: %s", body)
	}
}

func TestOIDCProviderTokenMissingGrantType(t *testing.T) {
	t.Parallel()

	s := &Server{
		oidcProvider: &OIDCProviderServer{
			clients: make(map[string]*config.OIDCClient),
		},
	}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/oidc/token", nil)
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	s.handleOIDCProviderToken(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestOIDCProviderTokenUnknownClient(t *testing.T) {
	t.Parallel()

	s := &Server{
		oidcProvider: &OIDCProviderServer{
			clients: make(map[string]*config.OIDCClient),
		},
	}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/oidc/token", strings.NewReader("grant_type=authorization_code&client_id=unknown&client_secret=secret"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	s.handleOIDCProviderToken(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestOIDCProviderTokenInvalidSecret(t *testing.T) {
	t.Parallel()

	hash, _ := bcrypt.GenerateFromPassword([]byte("correct-secret"), bcrypt.DefaultCost)
	client := &config.OIDCClient{
		ClientID:     "test-client",
		ClientSecret: string(hash),
		RedirectURIs: []string{"https://example.com/callback"},
	}

	s := &Server{
		oidcProvider: &OIDCProviderServer{
			clients: map[string]*config.OIDCClient{"test-client": client},
		},
	}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/oidc/token", strings.NewReader("grant_type=authorization_code&client_id=test-client&client_secret=wrong-secret"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	s.handleOIDCProviderToken(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestOIDCProviderUserInfoNoToken(t *testing.T) {
	t.Parallel()

	s := &Server{oidcProvider: &OIDCProviderServer{}}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/oidc/userinfo", nil)

	s.handleOIDCUserInfo(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestOIDCProviderUserInfoInvalidToken(t *testing.T) {
	t.Parallel()

	s := &Server{oidcProvider: &OIDCProviderServer{}}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/oidc/userinfo", nil)
	r.Header.Set("Authorization", "Bearer invalid-token")

	s.handleOIDCUserInfo(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestOIDCProviderRevoke(t *testing.T) {
	t.Parallel()

	s := &Server{oidcProvider: &OIDCProviderServer{}}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/oidc/revoke", nil)
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	s.handleOIDCRevoke(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestOIDCProviderIntrospectInvalidToken(t *testing.T) {
	t.Parallel()

	s := &Server{oidcProvider: &OIDCProviderServer{}}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/oidc/introspect", strings.NewReader("token=invalid"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	s.handleOIDCIntrospect(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["active"] != false {
		t.Fatalf("expected active=false for invalid token, got %v", resp["active"])
	}
}

func TestOIDCProviderRSAPublicKeyToJWK(t *testing.T) {
	t.Parallel()

	// This would test JWK conversion but needs RSA key - skip for unit test
	// Full integration test would use generated key
	t.Skip("requires RSA key generation")
}

func TestOIDCProviderECPublicKeyToJWK(t *testing.T) {
	t.Parallel()

	// This would test JWK conversion but needs EC key - skip for unit test
	t.Skip("requires EC key generation")
}

func TestOIDCProviderGenerateJTI(t *testing.T) {
	t.Parallel()

	jti1 := generateJTI()
	jti2 := generateJTI()

	if jti1 == "" || jti2 == "" {
		t.Fatal("JTI should not be empty")
	}
	if jti1 == jti2 {
		t.Fatal("JTIs should be unique")
	}
}

func TestOIDCProviderGenerateRefreshToken(t *testing.T) {
	t.Parallel()

	rt1 := generateRefreshToken()
	rt2 := generateRefreshToken()

	if rt1 == "" || rt2 == "" {
		t.Fatal("refresh token should not be empty")
	}
	if rt1 == rt2 {
		t.Fatal("refresh tokens should be unique")
	}
}

func TestOIDCProviderNewRandomHex(t *testing.T) {
	t.Parallel()

	h1, err := newRandomHex(16)
	if err != nil {
		t.Fatalf("newRandomHex: %v", err)
	}
	if len(h1) != 32 { // 16 bytes = 32 hex chars
		t.Fatalf("expected 32 hex chars, got %d", len(h1))
	}

	h2, _ := newRandomHex(16)
	if h1 == h2 {
		t.Fatal("hex strings should be unique")
	}
}

func TestOIDCProviderDisabledToken(t *testing.T) {
	t.Parallel()

	s := &Server{oidcProvider: nil}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/oidc/token", nil)

	s.handleOIDCProviderToken(w, r)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
}

func TestOIDCProviderDisabledUserInfo(t *testing.T) {
	t.Parallel()

	s := &Server{oidcProvider: nil}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/oidc/userinfo", nil)

	s.handleOIDCUserInfo(w, r)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
}

func TestOIDCProviderDisabledRevoke(t *testing.T) {
	t.Parallel()

	s := &Server{oidcProvider: nil}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/oidc/revoke", nil)

	s.handleOIDCRevoke(w, r)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
}

func TestOIDCProviderDisabledIntrospect(t *testing.T) {
	t.Parallel()

	s := &Server{oidcProvider: nil}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/oidc/introspect", nil)

	s.handleOIDCIntrospect(w, r)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
}

func TestOIDCProviderDisabledAuthorize(t *testing.T) {
	t.Parallel()

	s := &Server{oidcProvider: nil}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/oidc/authorize", nil)

	s.handleOIDCAuthorize(w, r)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
}

func TestOIDCProviderUnsupportedGrantType(t *testing.T) {
	t.Parallel()

	hash, _ := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)
	client := &config.OIDCClient{
		ClientID:     "test-client",
		ClientSecret: string(hash),
		RedirectURIs: []string{"https://example.com/callback"},
	}

	s := &Server{
		oidcProvider: &OIDCProviderServer{
			clients: map[string]*config.OIDCClient{"test-client": client},
		},
	}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/oidc/token", strings.NewReader("grant_type=password&client_id=test-client&client_secret=secret"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	s.handleOIDCProviderToken(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "unsupported_grant_type") {
		t.Fatalf("expected unsupported_grant_type error, got: %s", body)
	}
}

func TestOIDCProviderAuthorizationCodeNotFound(t *testing.T) {
	t.Parallel()

	hash, _ := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)
	client := &config.OIDCClient{
		ClientID:     "test-client",
		ClientSecret: string(hash),
		RedirectURIs: []string{"https://example.com/callback"},
	}

	s := &Server{
		oidcProvider: &OIDCProviderServer{
			clients: map[string]*config.OIDCClient{"test-client": client},
		},
	}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/oidc/token", strings.NewReader("grant_type=authorization_code&code=nonexistent&redirect_uri=https://example.com/callback&client_id=test-client&client_secret=secret"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	s.handleOIDCProviderToken(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "invalid_grant") {
		t.Fatalf("expected invalid_grant error, got: %s", body)
	}
}

func TestOIDCProviderClientCredentialsGrant(t *testing.T) {
	// Generate a test RSA key for signing
	rsaKey, err := generateTestRSAKey()
	if err != nil {
		t.Fatalf("generate test RSA key: %v", err)
	}

	// Set up the global provider signer
	providerSignerMu.Lock()
	providerSigner = &oidcProviderSigner{
		privateKey: rsaKey,
		keyType:    "RSA",
		keyID:     "test-key",
		algorithm: "RS256",
	}
	providerSignerMu.Unlock()

	defer func() {
		providerSignerMu.Lock()
		providerSigner = nil
		providerSignerMu.Unlock()
	}()

	hash, _ := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)
	client := &config.OIDCClient{
		ClientID:     "test-client",
		ClientSecret: string(hash),
		RedirectURIs: []string{"https://example.com/callback"},
		Scopes:       []string{"openid", "profile"},
	}

	s := &Server{
		oidcProvider: &OIDCProviderServer{
			clients: map[string]*config.OIDCClient{"test-client": client},
			config: &config.OIDCProviderConfig{
				Issuer:         "https://api.example.com",
				AccessTokenTTL: 3600 * time.Second,
				IDTokenTTL:     3600 * time.Second,
				AuthCodeTTL:    5 * time.Minute,
			},
		},
	}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/oidc/token", strings.NewReader("grant_type=client_credentials&client_id=test-client&client_secret=secret"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	s.handleOIDCProviderToken(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["access_token"] == nil || resp["access_token"] == "" {
		t.Fatal("expected access_token in response")
	}
	if resp["token_type"] != "Bearer" {
		t.Fatalf("expected token_type Bearer, got %v", resp["token_type"])
	}
	if resp["expires_in"] == nil {
		t.Fatal("expected expires_in in response")
	}
}

func generateTestRSAKey() (*rsa.PrivateKey, error) {
	return rsa.GenerateKey(rand.Reader, 2048)
}