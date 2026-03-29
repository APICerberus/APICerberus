package plugin

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIPRestrictWhitelistHitAndMiss(t *testing.T) {
	t.Parallel()

	plugin, err := NewIPRestrict(IPRestrictConfig{
		Whitelist: []string{"10.0.0.1", "192.168.1.0/24"},
	})
	if err != nil {
		t.Fatalf("NewIPRestrict error: %v", err)
	}

	reqAllowed := httptest.NewRequest(http.MethodGet, "http://gateway.local/users", nil)
	reqAllowed.RemoteAddr = "10.0.0.1:1234"
	if err := plugin.Allow(reqAllowed); err != nil {
		t.Fatalf("whitelist hit should be allowed: %v", err)
	}

	reqAllowedCIDR := httptest.NewRequest(http.MethodGet, "http://gateway.local/users", nil)
	reqAllowedCIDR.RemoteAddr = "192.168.1.42:9999"
	if err := plugin.Allow(reqAllowedCIDR); err != nil {
		t.Fatalf("whitelist cidr hit should be allowed: %v", err)
	}

	reqDenied := httptest.NewRequest(http.MethodGet, "http://gateway.local/users", nil)
	reqDenied.RemoteAddr = "10.0.0.2:1234"
	err = plugin.Allow(reqDenied)
	if err == nil {
		t.Fatalf("whitelist miss should be denied")
	}
	restrictErr, ok := err.(*IPRestrictError)
	if !ok || restrictErr.Code != "ip_not_allowed" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIPRestrictBlacklistHitAndMiss(t *testing.T) {
	t.Parallel()

	plugin, err := NewIPRestrict(IPRestrictConfig{
		Blacklist: []string{"10.0.0.1", "172.16.0.0/16"},
	})
	if err != nil {
		t.Fatalf("NewIPRestrict error: %v", err)
	}

	reqBlocked := httptest.NewRequest(http.MethodGet, "http://gateway.local/users", nil)
	reqBlocked.RemoteAddr = "10.0.0.1:1234"
	err = plugin.Allow(reqBlocked)
	if err == nil {
		t.Fatalf("blacklist hit should be blocked")
	}
	restrictErr, ok := err.(*IPRestrictError)
	if !ok || restrictErr.Code != "ip_blocked" {
		t.Fatalf("unexpected error: %v", err)
	}

	reqBlockedCIDR := httptest.NewRequest(http.MethodGet, "http://gateway.local/users", nil)
	reqBlockedCIDR.RemoteAddr = "172.16.8.8:1234"
	if err := plugin.Allow(reqBlockedCIDR); err == nil {
		t.Fatalf("blacklist cidr hit should be blocked")
	}

	reqAllowed := httptest.NewRequest(http.MethodGet, "http://gateway.local/users", nil)
	reqAllowed.RemoteAddr = "192.168.1.1:1234"
	if err := plugin.Allow(reqAllowed); err != nil {
		t.Fatalf("blacklist miss should be allowed: %v", err)
	}
}

func TestIPRestrictCIDRWithForwardedHeader(t *testing.T) {
	t.Parallel()

	plugin, err := NewIPRestrict(IPRestrictConfig{
		Whitelist: []string{"203.0.113.0/24"},
	})
	if err != nil {
		t.Fatalf("NewIPRestrict error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "http://gateway.local/users", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	req.Header.Set("X-Forwarded-For", "203.0.113.8, 10.0.0.1")
	if err := plugin.Allow(req); err != nil {
		t.Fatalf("X-Forwarded-For CIDR match should be allowed: %v", err)
	}
}
