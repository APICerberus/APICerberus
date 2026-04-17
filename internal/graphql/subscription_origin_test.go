package graphql

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIsSubscriptionOriginAllowed_CompatMode(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		origin string
	}{
		{"no_origin", ""},
		{"browser_origin", "https://app.example.com"},
		{"attacker_origin", "https://evil.com"},
		{"null_origin", "null"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := httptest.NewRequest(http.MethodGet, "http://gw/subscriptions", nil)
			if tc.origin != "" {
				r.Header.Set("Origin", tc.origin)
			}
			if !isSubscriptionOriginAllowed(r, nil) {
				t.Fatalf("compat mode (empty allow-list) must accept origin %q", tc.origin)
			}
		})
	}
}

func TestIsSubscriptionOriginAllowed_StrictMode_AllowsMatching(t *testing.T) {
	t.Parallel()

	allow := []string{
		"https://app.example.com",
		"admin.example.com",
		"*.internal.example.com",
	}

	cases := []struct {
		name   string
		origin string
	}{
		{"exact_url", "https://app.example.com"},
		{"exact_url_with_default_port", "https://app.example.com:443"},
		{"bare_host", "https://admin.example.com"},
		{"bare_host_http", "http://admin.example.com"},
		{"wildcard_single_label", "https://tenant-a.internal.example.com"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := httptest.NewRequest(http.MethodGet, "http://gw/subscriptions", nil)
			r.Header.Set("Origin", tc.origin)
			if !isSubscriptionOriginAllowed(r, allow) {
				t.Fatalf("origin %q should be allowed by list %v", tc.origin, allow)
			}
		})
	}
}

func TestIsSubscriptionOriginAllowed_StrictMode_BlocksEverythingElse(t *testing.T) {
	t.Parallel()

	allow := []string{"https://app.example.com", "*.internal.example.com"}

	cases := []struct {
		name   string
		origin string
	}{
		{"missing_origin_is_rejected_in_strict_mode", ""},
		{"null_origin_rejected", "null"},
		{"attacker_http_scheme", "http://app.example.com"},
		{"attacker_different_host", "https://evil.com"},
		{"attacker_substring_trick", "https://app.example.com.evil.com"},
		{"attacker_ftp_scheme", "ftp://app.example.com"},
		{"attacker_wildcard_apex_mismatch", "https://internal.example.com"},
		{"attacker_wildcard_multi_label", "https://a.b.internal.example.com"},
		{"attacker_non_default_port", "https://app.example.com:8443"},
		{"malformed_origin", "not-a-url"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := httptest.NewRequest(http.MethodGet, "http://gw/subscriptions", nil)
			if tc.origin != "" {
				r.Header.Set("Origin", tc.origin)
			}
			if isSubscriptionOriginAllowed(r, allow) {
				t.Fatalf("origin %q MUST be rejected by allow-list %v", tc.origin, allow)
			}
		})
	}
}

// TestHandleSubscription_RejectsDisallowedOrigin asserts the gate is wired
// into HandleSubscription itself, not just the helper — a simulated cross-site
// upgrade must 403 before the WS hijack.
func TestHandleSubscription_RejectsDisallowedOrigin(t *testing.T) {
	t.Parallel()

	sp := NewSubscriptionProxy("ws://upstream/graphql")
	sp.SetAllowedOrigins([]string{"https://app.example.com"})

	r := httptest.NewRequest(http.MethodGet, "/subscriptions", nil)
	r.Header.Set("Connection", "Upgrade")
	r.Header.Set("Upgrade", "websocket")
	r.Header.Set("Sec-WebSocket-Protocol", "graphql-transport-ws")
	r.Header.Set("Origin", "https://evil.com")
	w := httptest.NewRecorder()

	sp.HandleSubscription(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for disallowed origin, got %d", w.Code)
	}
}

func TestHandleSSE_RejectsDisallowedOrigin(t *testing.T) {
	t.Parallel()

	p := NewSSESubscriptionProxy("ws://upstream/graphql")
	p.SetAllowedOrigins([]string{"https://app.example.com"})

	r := httptest.NewRequest(http.MethodGet, "/subscriptions?transport=sse", nil)
	r.Header.Set("Origin", "https://evil.com")
	w := httptest.NewRecorder()

	p.HandleSSE(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for disallowed origin on SSE, got %d", w.Code)
	}
}
