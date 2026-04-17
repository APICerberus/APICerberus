package federation

import (
	"testing"
)

func TestValidateSubgraphURL_Valid(t *testing.T) {
	t.Parallel()
	// Use public IP literals so the test does not depend on outbound DNS —
	// the gate's hostname path is exercised by the Invalid suite below via
	// DNS-fail cases. SEC-GQL-005 made DNS failures fail-closed, so any
	// test that relied on unresolved hostnames being "valid" has moved.
	tests := []struct {
		name string
		url  string
	}{
		{"public ipv4 http", "http://8.8.8.8/graphql"},
		{"public ipv4 https", "https://1.1.1.1/graphql"},
		{"public ipv4 with port", "https://8.8.4.4:443/graphql"},
		{"public ipv6", "http://[2001:4860:4860::8888]/graphql"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if err := validateSubgraphURL(tt.url); err != nil {
				t.Errorf("validateSubgraphURL(%q) unexpected error: %v", tt.url, err)
			}
		})
	}
}

func TestValidateSubgraphURL_Invalid(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		url  string
	}{
		{"bad scheme", "ftp://example.com/graphql"},
		{"no host", "http:///graphql"},
		{"loopback ip", "http://127.0.0.1/graphql"},
		{"loopback ipv6", "http://[::1]/graphql"},
		{"private 10.x", "http://10.0.0.1/graphql"},
		{"private 172.16", "http://172.16.0.1/graphql"},
		{"private 192.168", "http://192.168.1.1/graphql"},
		{"link-local metadata", "http://169.254.169.254/graphql"},
		{"multicast", "http://224.0.0.1/graphql"},
		{"unspecified", "http://0.0.0.0/graphql"},
		// SEC-GQL-005 additions:
		{"v4_mapped_ipv6_metadata", "http://[::ffff:169.254.169.254]/graphql"},
		{"v4_mapped_ipv6_rfc1918", "http://[::ffff:10.0.0.1]/graphql"},
		{"ipv6_link_local_unicast", "http://[fe80::1]/graphql"},
		{"ipv6_unique_local", "http://[fc00::1]/graphql"},
		{"unresolvable_host_fails_closed", "http://this-host-should-never-resolve.invalid/graphql"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if err := validateSubgraphURL(tt.url); err == nil {
				t.Errorf("validateSubgraphURL(%q) expected error, got nil", tt.url)
			}
		})
	}
}

// TestFetchSchemaReValidatesURL asserts SEC-GQL-005: even if AddSubgraph
// accepted a URL at registration time, FetchSchema must re-check before
// issuing the outbound introspection request so a DNS rebinding flip after
// registration cannot reach private space.
func TestFetchSchemaReValidatesURL(t *testing.T) {
	t.Parallel()
	m := NewSubgraphManager() // validateURLs defaults to true
	// Manually install the subgraph to bypass AddSubgraph's own check;
	// this simulates the post-registration DNS-flip scenario where the
	// host no longer resolves to a public IP.
	sg := &Subgraph{ID: "sg1", Name: "sg1", URL: "http://127.0.0.1:1/graphql"}
	m.mu.Lock()
	m.subgraphs[sg.ID] = sg
	m.mu.Unlock()
	if _, err := m.FetchSchema(sg); err == nil {
		t.Fatal("FetchSchema must reject loopback URL even if manager state already holds it")
	}
}

// TestCheckHealthReValidatesURL asserts the same re-check for CheckHealth.
func TestCheckHealthReValidatesURL(t *testing.T) {
	t.Parallel()
	m := NewSubgraphManager()
	sg := &Subgraph{ID: "sg1", Name: "sg1", URL: "http://10.0.0.1:80/graphql"}
	m.mu.Lock()
	m.subgraphs[sg.ID] = sg
	m.mu.Unlock()
	if err := m.CheckHealth(sg); err == nil {
		t.Fatal("CheckHealth must reject RFC1918 URL post-registration")
	}
}
