package plugin

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/APICerberus/APICerebrus/internal/config"
)

func TestAuthAPIKeyValid(t *testing.T) {
	t.Parallel()

	auth := NewAuthAPIKey([]config.Consumer{
		{
			Name: "mobile-app",
			APIKeys: []config.ConsumerAPIKey{
				{Key: "ck_live_valid"},
			},
		},
	}, AuthAPIKeyOptions{})

	req := httptest.NewRequest(http.MethodGet, "http://example.com/x", nil)
	req.Header.Set("X-API-Key", "ck_live_valid")

	consumer, err := auth.Authenticate(req)
	if err != nil {
		t.Fatalf("Authenticate error: %v", err)
	}
	if consumer == nil || consumer.Name != "mobile-app" {
		t.Fatalf("unexpected consumer: %#v", consumer)
	}
}

func TestAuthAPIKeyInvalid(t *testing.T) {
	t.Parallel()

	auth := NewAuthAPIKey([]config.Consumer{
		{
			Name: "mobile-app",
			APIKeys: []config.ConsumerAPIKey{
				{Key: "ck_live_valid"},
			},
		},
	}, AuthAPIKeyOptions{})

	req := httptest.NewRequest(http.MethodGet, "http://example.com/x", nil)
	req.Header.Set("X-API-Key", "wrong")

	consumer, err := auth.Authenticate(req)
	if consumer != nil {
		t.Fatalf("expected nil consumer, got %#v", consumer)
	}
	if err != ErrInvalidAPIKey {
		t.Fatalf("expected ErrInvalidAPIKey got %v", err)
	}
}

func TestAuthAPIKeyExpired(t *testing.T) {
	t.Parallel()

	auth := NewAuthAPIKey([]config.Consumer{
		{
			Name: "mobile-app",
			APIKeys: []config.ConsumerAPIKey{
				{
					Key:       "ck_live_expired",
					ExpiresAt: time.Now().Add(-time.Hour).UTC().Format(time.RFC3339),
				},
			},
		},
	}, AuthAPIKeyOptions{})

	req := httptest.NewRequest(http.MethodGet, "http://example.com/x", nil)
	req.Header.Set("X-API-Key", "ck_live_expired")

	consumer, err := auth.Authenticate(req)
	if consumer != nil {
		t.Fatalf("expected nil consumer, got %#v", consumer)
	}
	if err != ErrExpiredAPIKey {
		t.Fatalf("expected ErrExpiredAPIKey got %v", err)
	}
}

func TestAuthAPIKeyMultipleSources(t *testing.T) {
	t.Parallel()

	auth := NewAuthAPIKey([]config.Consumer{
		{
			Name: "mobile-app",
			APIKeys: []config.ConsumerAPIKey{
				{Key: "header-key"},
				{Key: "query-key"},
				{Key: "cookie-key"},
				{Key: "bearer-key"},
			},
		},
	}, AuthAPIKeyOptions{
		KeyNames:    []string{"X-Custom-Key", "Authorization"},
		QueryNames:  []string{"k"},
		CookieNames: []string{"k"},
	})

	reqHeader := httptest.NewRequest(http.MethodGet, "http://example.com/x", nil)
	reqHeader.Header.Set("X-Custom-Key", "header-key")
	assertAuthConsumer(t, auth, reqHeader, "mobile-app")

	reqQuery := httptest.NewRequest(http.MethodGet, "http://example.com/x?k=query-key", nil)
	assertAuthConsumer(t, auth, reqQuery, "mobile-app")

	reqCookie := httptest.NewRequest(http.MethodGet, "http://example.com/x", nil)
	reqCookie.AddCookie(&http.Cookie{Name: "k", Value: "cookie-key"})
	assertAuthConsumer(t, auth, reqCookie, "mobile-app")

	reqBearer := httptest.NewRequest(http.MethodGet, "http://example.com/x", nil)
	reqBearer.Header.Set("Authorization", "Bearer bearer-key")
	assertAuthConsumer(t, auth, reqBearer, "mobile-app")
}

func TestAuthAPIKeyMissing(t *testing.T) {
	t.Parallel()

	auth := NewAuthAPIKey(nil, AuthAPIKeyOptions{})
	req := httptest.NewRequest(http.MethodGet, "http://example.com/x", nil)
	consumer, err := auth.Authenticate(req)
	if consumer != nil {
		t.Fatalf("expected nil consumer")
	}
	if err != ErrMissingAPIKey {
		t.Fatalf("expected ErrMissingAPIKey got %v", err)
	}
}

func TestAuthAPIKeyExternalLookup(t *testing.T) {
	t.Parallel()

	auth := NewAuthAPIKey([]config.Consumer{
		{
			Name: "local-consumer",
			APIKeys: []config.ConsumerAPIKey{
				{Key: "ck_live_local"},
			},
		},
	}, AuthAPIKeyOptions{
		Lookup: func(rawKey string, req *http.Request) (*config.Consumer, error) {
			if req == nil {
				t.Fatalf("expected non-nil request in lookup callback")
			}
			if rawKey == "ck_live_external" {
				return &config.Consumer{
					ID:   "ext-1",
					Name: "external-consumer",
				}, nil
			}
			return nil, ErrInvalidAPIKey
		},
	})

	reqExternal := httptest.NewRequest(http.MethodGet, "http://example.com/x", nil)
	reqExternal.Header.Set("X-API-Key", "ck_live_external")
	consumer, err := auth.Authenticate(reqExternal)
	if err != nil {
		t.Fatalf("Authenticate external error: %v", err)
	}
	if consumer == nil || consumer.Name != "external-consumer" {
		t.Fatalf("unexpected consumer from external lookup: %#v", consumer)
	}

	reqLocal := httptest.NewRequest(http.MethodGet, "http://example.com/x", nil)
	reqLocal.Header.Set("X-API-Key", "ck_live_local")
	consumer, err = auth.Authenticate(reqLocal)
	if consumer != nil {
		t.Fatalf("expected nil consumer for local key when external lookup is enabled")
	}
	if err != ErrInvalidAPIKey {
		t.Fatalf("expected ErrInvalidAPIKey got %v", err)
	}
}

func assertAuthConsumer(t *testing.T, auth *AuthAPIKey, req *http.Request, expectedName string) {
	t.Helper()
	consumer, err := auth.Authenticate(req)
	if err != nil {
		t.Fatalf("Authenticate error: %v", err)
	}
	if consumer == nil || consumer.Name != expectedName {
		t.Fatalf("unexpected consumer: %#v", consumer)
	}
}
