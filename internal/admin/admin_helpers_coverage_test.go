package admin

import (
	"testing"
)

// TestSetTrustedProxies tests trusted proxy configuration.
func TestSetTrustedProxies(t *testing.T) {
	t.Run("sets valid CIDR", func(t *testing.T) {
		t.Parallel()
		proxies := []string{"10.0.0.0/8", "172.16.0.0/12"}
		SetTrustedProxies(proxies)
		t.Log("SetTrustedProxies called with valid CIDR ranges")
	})

	t.Run("sets valid IP", func(t *testing.T) {
		t.Parallel()
		proxies := []string{"192.168.1.1"}
		SetTrustedProxies(proxies)
		t.Log("SetTrustedProxies called with IP")
	})

	t.Run("sets empty list", func(t *testing.T) {
		t.Parallel()
		proxies := []string{}
		SetTrustedProxies(proxies)
		t.Log("SetTrustedProxies called with empty list")
	})
}
