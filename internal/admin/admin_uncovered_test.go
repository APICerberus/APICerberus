package admin

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestExtractClientIPVarious tests extractClientIP with various headers
func TestExtractClientIPVarious(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		headers    map[string]string
		expected   string
	}{
		{
			name:       "x-forwarded-for",
			remoteAddr: "192.168.1.1:1234",
			headers:    map[string]string{"X-Forwarded-For": "10.0.0.1, 10.0.0.2"},
			expected:   "10.0.0.1",
		},
		{
			name:       "x-real-ip",
			remoteAddr: "192.168.1.1:1234",
			headers:    map[string]string{"X-Real-Ip": "10.0.0.5"},
			expected:   "10.0.0.5",
		},
		{
			name:       "cf-connecting-ip-not-supported",
			remoteAddr: "192.168.1.1:1234",
			headers:    map[string]string{"Cf-Connecting-Ip": "10.0.0.6"},
			expected:   "192.168.1.1", // falls back to RemoteAddr since Cf-Connecting-Ip is not checked
		},
		{
			name:       "remote-addr-fallback",
			remoteAddr: "192.168.1.1:1234",
			headers:    map[string]string{},
			expected:   "192.168.1.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = tt.remoteAddr
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			result := extractClientIP(req)
			if result != tt.expected {
				t.Errorf("extractClientIP() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestFederationEndpointsDisabled tests federation endpoints when disabled
func TestFederationEndpointsDisabled(t *testing.T) {
	t.Parallel()
	baseURL, _, _ := newAdminTestServer(t)

	tests := []struct {
		name   string
		method string
		path   string
		body   map[string]any
	}{
		{
			name:   "addSubgraph",
			method: http.MethodPost,
			path:   "/admin/api/v1/federation/subgraphs",
			body:   map[string]any{"name": "test", "url": "http://localhost:4001"},
		},
		{
			name:   "getSubgraph",
			method: http.MethodGet,
			path:   "/admin/api/v1/federation/subgraphs/test-id",
		},
		{
			name:   "removeSubgraph",
			method: http.MethodDelete,
			path:   "/admin/api/v1/federation/subgraphs/test-id",
		},
		{
			name:   "composeSubgraphs",
			method: http.MethodPost,
			path:   "/admin/api/v1/federation/compose",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := mustJSONRequest(t, tt.method, baseURL+tt.path, "secret-admin", tt.body)
			status := resp["status_code"].(float64)
			// When federation is disabled, should return 400 or 404
			if status != http.StatusBadRequest && status != http.StatusNotFound {
				t.Errorf("expected %d or %d, got %v", http.StatusBadRequest, http.StatusNotFound, status)
			}
		})
	}
}
