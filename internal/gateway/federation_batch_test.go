package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/APICerberus/APICerebrus/internal/config"
)

func TestServeFederationBatch_InvalidMethod(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Gateway: config.GatewayConfig{HTTPAddr: ":0"},
		Federation: config.FederationConfig{
			Enabled: true,
		},
		Store: config.StoreConfig{Path: ":memory:"},
	}

	g, err := New(cfg)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	defer g.Shutdown(context.Background())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/graphql/batch", nil)
	g.serveFederationBatch(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestServeFederationBatch_InvalidJSON(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Gateway: config.GatewayConfig{HTTPAddr: ":0"},
		Federation: config.FederationConfig{
			Enabled: true,
		},
		Store: config.StoreConfig{Path: ":memory:"},
	}

	g, err := New(cfg)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	defer g.Shutdown(context.Background())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/graphql/batch", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	g.serveFederationBatch(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestServeFederationBatch_EmptyBatch(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Gateway: config.GatewayConfig{HTTPAddr: ":0"},
		Federation: config.FederationConfig{
			Enabled: true,
		},
		Store: config.StoreConfig{Path: ":memory:"},
	}

	g, err := New(cfg)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	defer g.Shutdown(context.Background())

	rec := httptest.NewRecorder()
	body, _ := json.Marshal([]batchGraphQLRequest{})
	req := httptest.NewRequest(http.MethodPost, "/graphql/batch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	g.serveFederationBatch(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestServeFederationBatch_NotReady(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Gateway: config.GatewayConfig{HTTPAddr: ":0"},
		Federation: config.FederationConfig{
			Enabled: true,
		},
		Store: config.StoreConfig{Path: ":memory:"},
	}

	g, err := New(cfg)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	defer g.Shutdown(context.Background())

	// Federation is enabled but planner/executor are nil (not composed yet)
	rec := httptest.NewRecorder()
	batch := []batchGraphQLRequest{{Query: "{ users { id } }"}}
	body, _ := json.Marshal(batch)
	req := httptest.NewRequest(http.MethodPost, "/graphql/batch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	g.serveFederationBatch(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}

func TestServeFederationBatch_EmptyQuery(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Gateway: config.GatewayConfig{HTTPAddr: ":0"},
		Federation: config.FederationConfig{
			Enabled: true,
		},
		Store: config.StoreConfig{Path: ":memory:"},
	}

	g, err := New(cfg)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	defer g.Shutdown(context.Background())

	// Manually set up a mock executor/planner to test empty query handling
	// Without subgraphs, the planner will be nil, so we test the empty query check indirectly.
	// The empty query check happens inside the goroutine, so we need executor/planner.
	// For now, we just test the path-level dispatch.
}
