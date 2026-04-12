package gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/APICerberus/APICerebrus/internal/config"
)

func TestGatewayHandleHealth_HealthEndpoint(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Gateway: config.GatewayConfig{
			HTTPAddr: ":0",
		},
		Admin: config.AdminConfig{
			Addr:        ":0",
			APIKey:      "test-admin-api-key-at-least-32-chars!!",
			TokenSecret: "test-admin-token-secret-at-least-32-chars",
		},
		Store: config.StoreConfig{
			Path:        ":memory:",
			BusyTimeout: 5 * time.Second,
			JournalMode: "MEMORY",
			ForeignKeys: true,
		},
	}

	g, err := New(cfg)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	defer func() { _ = g.Shutdown(context.Background()) }()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	g.handleHealth(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp["status"] != "ok" {
		t.Errorf("expected status=ok, got %v", resp["status"])
	}
	if _, ok := resp["uptime"]; !ok {
		t.Error("expected uptime in response")
	}
}

func TestGatewayHandleHealth_ReadyEndpoint(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Gateway: config.GatewayConfig{
			HTTPAddr: ":0",
		},
		Admin: config.AdminConfig{
			Addr:        ":0",
			APIKey:      "test-admin-api-key-at-least-32-chars!!",
			TokenSecret: "test-admin-token-secret-at-least-32-chars",
		},
		Store: config.StoreConfig{
			Path:        ":memory:",
			BusyTimeout: 5 * time.Second,
			JournalMode: "MEMORY",
			ForeignKeys: true,
		},
	}

	g, err := New(cfg)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	defer func() { _ = g.Shutdown(context.Background()) }()

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()
	g.handleHealth(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp["status"] != "ok" {
		t.Errorf("expected status=ok, got %v", resp["status"])
	}
}

func TestGatewayHandleHealth_AuditDropsEndpoint(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Gateway: config.GatewayConfig{
			HTTPAddr: ":0",
		},
		Admin: config.AdminConfig{
			Addr:        ":0",
			APIKey:      "test-admin-api-key-at-least-32-chars!!",
			TokenSecret: "test-admin-token-secret-at-least-32-chars",
		},
		Store: config.StoreConfig{
			Path:        ":memory:",
			BusyTimeout: 5 * time.Second,
			JournalMode: "MEMORY",
			ForeignKeys: true,
		},
	}

	g, err := New(cfg)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	defer func() { _ = g.Shutdown(context.Background()) }()

	req := httptest.NewRequest(http.MethodGet, "/health/audit-drops", nil)
	rec := httptest.NewRecorder()
	g.handleHealth(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if _, ok := resp["dropped_entries"]; !ok {
		t.Error("expected dropped_entries in response")
	}
	if _, ok := resp["audit_enabled"]; !ok {
		t.Error("expected audit_enabled in response")
	}
}

func TestGatewayHandleHealth_UnknownPath(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Gateway: config.GatewayConfig{HTTPAddr: ":0"},
		Store: config.StoreConfig{
			Path:        ":memory:",
			BusyTimeout: 5 * time.Second,
			JournalMode: "MEMORY",
		},
	}

	g, err := New(cfg)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	defer func() { _ = g.Shutdown(context.Background()) }()

	req := httptest.NewRequest(http.MethodGet, "/not-health", nil)
	rec := httptest.NewRecorder()

	handled := g.handleHealth(rec, req)
	if handled {
		t.Fatal("expected handleHealth to return false for non-health path")
	}
	if rec.Body.Len() > 0 {
		t.Errorf("expected no response body, got: %s", rec.Body.String())
	}
}

func TestGatewayHandleMetrics_MetricsEndpoint(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Gateway: config.GatewayConfig{
			HTTPAddr: ":0",
		},
		Admin: config.AdminConfig{
			Addr:        ":0",
			APIKey:      "test-admin-api-key-at-least-32-chars!!",
			TokenSecret: "test-admin-token-secret-at-least-32-chars",
		},
		Store: config.StoreConfig{
			Path:        ":memory:",
			BusyTimeout: 5 * time.Second,
			JournalMode: "MEMORY",
			ForeignKeys: true,
		},
	}

	g, err := New(cfg)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	defer func() { _ = g.Shutdown(context.Background()) }()

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()

	handled := g.handleMetrics(rec, req)
	if !handled {
		t.Fatal("expected handleMetrics to return true for /metrics")
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()

	// Verify all expected metrics are present
	expectedMetrics := []string{
		"gateway_requests_total",
		"gateway_active_connections",
		"gateway_audit_dropped_total",
		"gateway_database_ready",
		"gateway_uptime_seconds",
	}
	for _, metric := range expectedMetrics {
		if !containsString(body, metric) {
			t.Errorf("expected metric %q in response body", metric)
		}
	}

	// Verify Prometheus format
	if !containsString(body, "# HELP") {
		t.Error("expected HELP comment in response")
	}
	if !containsString(body, "# TYPE") {
		t.Error("expected TYPE comment in response")
	}
}

func TestGatewayHandleMetrics_NonMetricsPath(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Gateway: config.GatewayConfig{HTTPAddr: ":0"},
		Admin: config.AdminConfig{
			Addr:        ":0",
			APIKey:      "test-admin-api-key-at-least-32-chars!!",
			TokenSecret: "test-admin-token-secret-at-least-32-chars",
		},
		Store: config.StoreConfig{Path: ":memory:", BusyTimeout: 5 * time.Second},
	}

	g, err := New(cfg)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	defer func() { _ = g.Shutdown(context.Background()) }()

	req := httptest.NewRequest(http.MethodGet, "/not-metrics", nil)
	rec := httptest.NewRecorder()

	handled := g.handleMetrics(rec, req)
	if handled {
		t.Fatal("expected handleMetrics to return false for non-/metrics path")
	}
}

func TestGatewayHandleMetrics_PostRejected(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Gateway: config.GatewayConfig{HTTPAddr: ":0"},
		Admin: config.AdminConfig{
			Addr:        ":0",
			APIKey:      "test-admin-api-key-at-least-32-chars!!",
			TokenSecret: "test-admin-token-secret-at-least-32-chars",
		},
		Store: config.StoreConfig{Path: ":memory:", BusyTimeout: 5 * time.Second},
	}

	g, err := New(cfg)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	defer func() { _ = g.Shutdown(context.Background()) }()

	req := httptest.NewRequest(http.MethodPost, "/metrics", nil)
	rec := httptest.NewRecorder()

	handled := g.handleMetrics(rec, req)
	if handled {
		t.Fatal("expected handleMetrics to return false for POST /metrics")
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
