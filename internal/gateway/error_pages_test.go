package gateway

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/APICerberus/APICerebrus/internal/config"
)

func TestHTMLErrorPage_Renders(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	htmlErrorPage(rec, http.StatusNotFound, "not_found", "The requested resource was not found")

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rec.Code)
	}

	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Errorf("expected Content-Type text/html, got %s", ct)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "404") {
		t.Error("expected status code 404 in HTML body")
	}
	if !strings.Contains(body, "Not Found") {
		t.Error("expected 'Not Found' in HTML body")
	}
	if !strings.Contains(body, "The requested resource was not found") {
		t.Error("expected error message in HTML body")
	}
}

func TestHTMLErrorPage_XSSPrevention(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	htmlErrorPage(rec, http.StatusBadRequest, "bad_input", "<script>alert('xss')</script>")

	body := rec.Body.String()
	if strings.Contains(body, "<script>") {
		t.Error("HTML error page should escape script tags")
	}
	if !strings.Contains(body, "&lt;script&gt;") {
		t.Error("HTML error page should have escaped script tags")
	}
}

func TestWriteErrorRoute_HTML(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Gateway: config.GatewayConfig{
			HTTPAddr:   ":0",
			HTMLErrors: true, // Global HTML errors enabled
		},
		Store: config.StoreConfig{Path: ":memory:"},
	}

	g, err := New(cfg)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	defer func() { _ = g.Shutdown(context.Background()) }()

	rec := httptest.NewRecorder()
	g.writeErrorRoute(rec, http.StatusBadGateway, "upstream_error", "Upstream failed", &config.Route{
		ID:   "test-route",
		Name: "Test Route",
	})

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected status 502, got %d", rec.Code)
	}

	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Errorf("expected HTML response, got %s", ct)
	}
}

func TestWriteErrorRoute_JSON(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Gateway: config.GatewayConfig{HTTPAddr: ":0"},
		Store:   config.StoreConfig{Path: ":memory:"},
	}

	g, err := New(cfg)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	defer func() { _ = g.Shutdown(context.Background()) }()

	rec := httptest.NewRecorder()
	g.writeErrorRoute(rec, http.StatusBadGateway, "upstream_error", "Upstream failed", &config.Route{
		ID:         "test-route",
		Name:       "Test Route",
		HTMLErrors: false, // Route-level JSON
	})

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected status 502, got %d", rec.Code)
	}

	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Errorf("expected JSON response, got %s", ct)
	}
}

func TestWriteErrorRoute_RouteLevelOverride(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Gateway: config.GatewayConfig{HTTPAddr: ":0"},
		Store:   config.StoreConfig{Path: ":memory:"},
	}

	g, err := New(cfg)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	defer func() { _ = g.Shutdown(context.Background()) }()

	// Global is off, route-level overrides to HTML
	rec := httptest.NewRecorder()
	g.writeErrorRoute(rec, http.StatusForbidden, "access_denied", "Forbidden", &config.Route{
		ID:         "test-route",
		Name:       "Test Route",
		HTMLErrors: true, // Route-level HTML override
	})

	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Errorf("expected HTML response with route-level override, got %s", ct)
	}
}

func TestWriteErrorRoute_NilRoute(t *testing.T) {
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

	// Nil route should default to JSON (global off)
	rec := httptest.NewRecorder()
	g.writeErrorRoute(rec, http.StatusNotFound, "not_found", "Not found", nil)

	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Errorf("expected JSON response with nil route, got %s", ct)
	}
}
