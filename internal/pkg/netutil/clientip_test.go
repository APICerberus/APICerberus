package netutil

import (
	"net/http"
	"testing"
)

func TestRemoteAddrIP(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"192.168.1.1:8080", "192.168.1.1"},
		{"10.0.0.1:12345", "10.0.0.1"},
		{"[::1]:8080", "::1"},
		{"[2001:db8::1]:443", "2001:db8::1"},
		{"127.0.0.1", "127.0.0.1"},
		{"", ""},
	}

	for _, tt := range tests {
		result := RemoteAddrIP(tt.input)
		if result != tt.expected {
			t.Errorf("RemoteAddrIP(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestExtractClientIP_NilRequest(t *testing.T) {
	if got := ExtractClientIP(nil); got != "" {
		t.Errorf("ExtractClientIP(nil) = %q, want %q", got, "")
	}
}

// --- Secure-by-default: no trusted proxies ===

func TestExtractClientIP_NoTrustedProxies_IgnoresXFF(t *testing.T) {
	SetTrustedProxies(nil)
	defer SetTrustedProxies(nil)

	req := &http.Request{
		RemoteAddr: "10.0.0.1:8080",
		Header:     http.Header{"X-Forwarded-For": []string{"203.0.113.5, 10.0.0.2"}},
	}
	// No trusted proxies configured → XFF ignored, RemoteAddr used
	if got := ExtractClientIP(req); got != "10.0.0.1" {
		t.Errorf("ExtractClientIP() = %q, want %q", got, "10.0.0.1")
	}
}

func TestExtractClientIP_NoTrustedProxies_IgnoresXRealIP(t *testing.T) {
	SetTrustedProxies(nil)
	defer SetTrustedProxies(nil)

	req := &http.Request{
		RemoteAddr: "10.0.0.1:8080",
		Header:     http.Header{"X-Real-Ip": []string{"198.51.100.10"}},
	}
	if got := ExtractClientIP(req); got != "10.0.0.1" {
		t.Errorf("ExtractClientIP() = %q, want %q", got, "10.0.0.1")
	}
}

// --- Untrusted source (not in trusted proxy list) ---

func TestExtractClientIP_UntrustedSource_IgnoresXFF(t *testing.T) {
	SetTrustedProxies([]string{"10.0.0.1"})
	defer SetTrustedProxies(nil)

	req := &http.Request{
		RemoteAddr: "192.168.1.1:8080",
		Header:     http.Header{"X-Forwarded-For": []string{"203.0.113.5"}},
	}
	// Source is untrusted → XFF ignored
	if got := ExtractClientIP(req); got != "192.168.1.1" {
		t.Errorf("ExtractClientIP() = %q, want %q", got, "192.168.1.1")
	}
}

// --- Trusted source (RemoteAddr is a trusted proxy) ---

func TestExtractClientIP_TrustedSource_SingleXFF(t *testing.T) {
	SetTrustedProxies([]string{"10.0.0.1"})
	defer SetTrustedProxies(nil)

	req := &http.Request{
		RemoteAddr: "10.0.0.1:8080",
		Header:     http.Header{"X-Forwarded-For": []string{"203.0.113.5"}},
	}
	if got := ExtractClientIP(req); got != "203.0.113.5" {
		t.Errorf("ExtractClientIP() = %q, want %q", got, "203.0.113.5")
	}
}

// --- Right-to-left parsing (multi-hop proxy chain) ---

func TestExtractClientIP_RightToLeftParsing(t *testing.T) {
	SetTrustedProxies([]string{"10.0.0.0/8", "172.16.0.0/12"})
	defer SetTrustedProxies(nil)

	// XFF: "client, proxy1, proxy2, proxy3"
	// Right-to-left: proxy3 (trusted) → proxy2 (trusted) → proxy1 (trusted) → client (untrusted)
	req := &http.Request{
		RemoteAddr: "10.0.0.1:8080",
		Header:     http.Header{"X-Forwarded-For": []string{"203.0.113.5, 172.16.0.1, 10.0.0.2, 10.0.0.3"}},
	}
	if got := ExtractClientIP(req); got != "203.0.113.5" {
		t.Errorf("ExtractClientIP() = %q, want %q (right-to-left should find real client)", got, "203.0.113.5")
	}
}

func TestExtractClientIP_RightToLeft_SpoofedLeftEntries(t *testing.T) {
	SetTrustedProxies([]string{"10.0.0.0/8"})
	defer SetTrustedProxies(nil)

	// Attacker sends: X-Forwarded-For: "1.2.3.4, 5.6.7.8, 10.0.0.5"
	// Right-to-left: 10.0.0.5 (trusted proxy) → 5.6.7.8 (untrusted → this is the real client)
	// The left-side "1.2.3.4" is ignored because it's to the left of the real client
	req := &http.Request{
		RemoteAddr: "10.0.0.1:8080",
		Header:     http.Header{"X-Forwarded-For": []string{"1.2.3.4, 5.6.7.8, 10.0.0.5"}},
	}
	if got := ExtractClientIP(req); got != "5.6.7.8" {
		t.Errorf("ExtractClientIP() = %q, want %q (spoofed left entries should be skipped)", got, "5.6.7.8")
	}
}

func TestExtractClientIP_RightToLeft_AllTrusted_ReturnsRemoteAddr(t *testing.T) {
	SetTrustedProxies([]string{"10.0.0.0/8", "203.0.113.0/24"})
	defer SetTrustedProxies(nil)

	// All entries in XFF are trusted proxies → fall back to RemoteAddr
	req := &http.Request{
		RemoteAddr: "10.0.0.1:8080",
		Header:     http.Header{"X-Forwarded-For": []string{"10.0.0.2, 10.0.0.3, 203.0.113.1"}},
	}
	if got := ExtractClientIP(req); got != "10.0.0.1" {
		t.Errorf("ExtractClientIP() = %q, want %q", got, "10.0.0.1")
	}
}

// --- CIDR support for trusted proxies ---

func TestSetTrustedProxies_CIDR(t *testing.T) {
	SetTrustedProxies([]string{"10.0.0.0/8", "192.168.1.100"})
	defer SetTrustedProxies(nil)

	// Any IP in 10.0.0.0/8 should be trusted
	req1 := &http.Request{
		RemoteAddr: "10.1.2.3:8080",
		Header:     http.Header{"X-Forwarded-For": []string{"203.0.113.5"}},
	}
	if got := ExtractClientIP(req1); got != "203.0.113.5" {
		t.Errorf("ExtractClientIP() = %q, want %q (CIDR should match)", got, "203.0.113.5")
	}

	// Specific IP should be trusted
	req2 := &http.Request{
		RemoteAddr: "192.168.1.100:443",
		Header:     http.Header{"X-Forwarded-For": []string{"203.0.113.6"}},
	}
	if got := ExtractClientIP(req2); got != "203.0.113.6" {
		t.Errorf("ExtractClientIP() = %q, want %q (specific IP should match)", got, "203.0.113.6")
	}

	// IP outside CIDR should not be trusted
	req3 := &http.Request{
		RemoteAddr: "172.16.0.1:8080",
		Header:     http.Header{"X-Forwarded-For": []string{"203.0.113.7"}},
	}
	if got := ExtractClientIP(req3); got != "172.16.0.1" {
		t.Errorf("ExtractClientIP() = %q, want %q (untrusted IP should use RemoteAddr)", got, "172.16.0.1")
	}
}

func TestSetTrustedProxies_InvalidCIDRSkipped(t *testing.T) {
	SetTrustedProxies([]string{"invalid-cidr", "10.0.0.1"})
	defer SetTrustedProxies(nil)

	req := &http.Request{
		RemoteAddr: "10.0.0.1:8080",
		Header:     http.Header{"X-Forwarded-For": []string{"203.0.113.5"}},
	}
	// Invalid CIDR skipped, valid IP still works
	if got := ExtractClientIP(req); got != "203.0.113.5" {
		t.Errorf("ExtractClientIP() = %q, want %q", got, "203.0.113.5")
	}
}

// --- Whitespace handling ---

func TestSetTrustedProxies_WithWhitespace(t *testing.T) {
	SetTrustedProxies([]string{" 10.0.0.1 ", "", " 192.168.1.1 "})
	defer SetTrustedProxies(nil)

	req1 := &http.Request{
		RemoteAddr: "10.0.0.1:8080",
		Header:     http.Header{"X-Forwarded-For": []string{"203.0.113.5"}},
	}
	if got := ExtractClientIP(req1); got != "203.0.113.5" {
		t.Errorf("ExtractClientIP() = %q, want %q", got, "203.0.113.5")
	}

	req2 := &http.Request{
		RemoteAddr: "192.168.1.1:8080",
		Header:     http.Header{"X-Forwarded-For": []string{"203.0.113.6"}},
	}
	if got := ExtractClientIP(req2); got != "203.0.113.6" {
		t.Errorf("ExtractClientIP() = %q, want %q", got, "203.0.113.6")
	}
}

// --- X-Real-IP fallback ---

func TestExtractClientIP_XRealIPFallback(t *testing.T) {
	SetTrustedProxies([]string{"10.0.0.1"})
	defer SetTrustedProxies(nil)

	req := &http.Request{
		RemoteAddr: "10.0.0.1:8080",
		Header:     http.Header{"X-Real-Ip": []string{"198.51.100.10"}},
	}
	if got := ExtractClientIP(req); got != "198.51.100.10" {
		t.Errorf("ExtractClientIP() = %q, want %q", got, "198.51.100.10")
	}
}

func TestExtractClientIP_XFFPreferredOverXRealIP(t *testing.T) {
	SetTrustedProxies([]string{"10.0.0.1"})
	defer SetTrustedProxies(nil)

	req := &http.Request{
		RemoteAddr: "10.0.0.1:8080",
		Header: http.Header{
			"X-Forwarded-For": []string{"203.0.113.5"},
			"X-Real-Ip":       []string{"198.51.100.10"},
		},
	}
	if got := ExtractClientIP(req); got != "203.0.113.5" {
		t.Errorf("ExtractClientIP() = %q, want %q", got, "203.0.113.5")
	}
}

// --- Empty/missing headers ---

func TestExtractClientIP_EmptyXFFFallsBackToRemoteAddr(t *testing.T) {
	SetTrustedProxies([]string{"10.0.0.1"})
	defer SetTrustedProxies(nil)

	req := &http.Request{
		RemoteAddr: "10.0.0.1:8080",
		Header:     http.Header{"X-Forwarded-For": []string{""}},
	}
	if got := ExtractClientIP(req); got != "10.0.0.1" {
		t.Errorf("ExtractClientIP() = %q, want %q", got, "10.0.0.1")
	}
}

func TestExtractClientIP_NoHeaders(t *testing.T) {
	SetTrustedProxies(nil)
	defer SetTrustedProxies(nil)

	req := &http.Request{RemoteAddr: "192.168.1.1:8080"}
	if got := ExtractClientIP(req); got != "192.168.1.1" {
		t.Errorf("ExtractClientIP() = %q, want %q", got, "192.168.1.1")
	}
}

// --- IPv6 support ---

func TestExtractClientIP_IPv6CIDR(t *testing.T) {
	SetTrustedProxies([]string{"::1/128", "fe80::/10"})
	defer SetTrustedProxies(nil)

	req := &http.Request{
		RemoteAddr: "[::1]:8080",
		Header:     http.Header{"X-Forwarded-For": []string{"2001:db8::1"}},
	}
	if got := ExtractClientIP(req); got != "2001:db8::1" {
		t.Errorf("ExtractClientIP() = %q, want %q", got, "2001:db8::1")
	}
}

// --- IsAllowedIP tests ---

func TestIsAllowedIP_EmptyList(t *testing.T) {
	if !IsAllowedIP("1.2.3.4", nil) {
		t.Error("expected true for nil slice")
	}
	if !IsAllowedIP("1.2.3.4", []string{}) {
		t.Error("expected true for empty slice")
	}
}

func TestIsAllowedIP_ExactMatch(t *testing.T) {
	if !IsAllowedIP("10.0.0.1", []string{"10.0.0.1"}) {
		t.Error("expected true for exact match")
	}
	if IsAllowedIP("10.0.0.2", []string{"10.0.0.1"}) {
		t.Error("expected false for non-match")
	}
}

func TestIsAllowedIP_CIDRMatch(t *testing.T) {
	if !IsAllowedIP("10.1.2.3", []string{"10.0.0.0/8"}) {
		t.Error("expected true for CIDR match")
	}
	if IsAllowedIP("192.168.1.1", []string{"10.0.0.0/8"}) {
		t.Error("expected false for CIDR non-match")
	}
}

func TestIsAllowedIP_InvalidIP(t *testing.T) {
	if IsAllowedIP("not-an-ip", []string{"10.0.0.0/8"}) {
		t.Error("expected false for invalid IP")
	}
	if IsAllowedIP("", []string{"10.0.0.1"}) {
		t.Error("expected false for empty IP")
	}
}

func TestIsAllowedIP_InvalidCIDR(t *testing.T) {
	// Invalid CIDR should be skipped; valid IP should still match
	if !IsAllowedIP("10.0.0.1", []string{"invalid", "10.0.0.1"}) {
		t.Error("expected true for valid IP even with invalid CIDR in list")
	}
}

func TestIsAllowedIP_MultipleRules(t *testing.T) {
	rules := []string{"10.0.0.0/8", "192.168.1.0/24", "172.16.0.1"}
	if !IsAllowedIP("10.5.5.5", rules) {
		t.Error("expected true for 10.x.x.x")
	}
	if !IsAllowedIP("192.168.1.100", rules) {
		t.Error("expected true for 192.168.1.x")
	}
	if !IsAllowedIP("172.16.0.1", rules) {
		t.Error("expected true for exact match 172.16.0.1")
	}
	if IsAllowedIP("8.8.8.8", rules) {
		t.Error("expected false for non-matching IP")
	}
}

func TestIsAllowedIP_EmptyRuleInList(t *testing.T) {
	// Empty rules should be skipped
	if !IsAllowedIP("10.0.0.1", []string{"", "10.0.0.1"}) {
		t.Error("expected true with empty rule in list")
	}
}

func TestIsAllowedIP_Whitespace(t *testing.T) {
	if !IsAllowedIP("10.0.0.1", []string{"  10.0.0.1  "}) {
		t.Error("expected true with whitespace in rule")
	}
}
