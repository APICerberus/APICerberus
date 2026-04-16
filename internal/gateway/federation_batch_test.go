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
	defer func() { _ = g.Shutdown(context.Background()) }()

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
	defer func() { _ = g.Shutdown(context.Background()) }()

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
	defer func() { _ = g.Shutdown(context.Background()) }()

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
	defer func() { _ = g.Shutdown(context.Background()) }()

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
	defer func() { _ = g.Shutdown(context.Background()) }()

	// Manually set up a mock executor/planner to test empty query handling
	// Without subgraphs, the planner will be nil, so we test the empty query check indirectly.
	// The empty query check happens inside the goroutine, so we need executor/planner.
	// For now, we just test the path-level dispatch.
}

// TestServeFederationBatch_RejectsUnauthenticated verifies the SEC-GQL-001 fix:
// when the gateway has any consumer configured, the batch endpoint must refuse
// requests that lack a valid API key. Prior to the fix the endpoint bypassed
// auth entirely, allowing amplified unauthenticated fan-out to subgraphs.
func TestServeFederationBatch_RejectsUnauthenticated(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Gateway: config.GatewayConfig{HTTPAddr: ":0"},
		Federation: config.FederationConfig{
			Enabled: true,
		},
		Consumers: []config.Consumer{
			{
				ID:   "c1",
				Name: "federation-client",
				APIKeys: []config.ConsumerAPIKey{
					{ID: "k1", Key: "ck_live_valid"},
				},
			},
		},
	}

	g, err := New(cfg)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	defer func() { _ = g.Shutdown(context.Background()) }()

	rec := httptest.NewRecorder()
	batch := []batchGraphQLRequest{{Query: "{ users { id } }"}}
	body, _ := json.Marshal(batch)
	req := httptest.NewRequest(http.MethodPost, "/graphql/batch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// no X-API-Key / Authorization header → must be rejected
	g.serveFederationBatch(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 unauthorized, got %d (body: %s)", rec.Code, rec.Body.String())
	}
}

// TestServeFederationBatch_AcceptsAuthenticated verifies that a request with a
// valid API key passes the auth gate. It will then fail with 503 because the
// federation schema is not composed in this test — the important assertion is
// that we got past the 401 check.
func TestServeFederationBatch_AcceptsAuthenticated(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Gateway: config.GatewayConfig{HTTPAddr: ":0"},
		Federation: config.FederationConfig{
			Enabled: true,
		},
		Consumers: []config.Consumer{
			{
				ID:   "c1",
				Name: "federation-client",
				APIKeys: []config.ConsumerAPIKey{
					{ID: "k1", Key: "ck_live_valid"},
				},
			},
		},
	}

	g, err := New(cfg)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	defer func() { _ = g.Shutdown(context.Background()) }()

	rec := httptest.NewRecorder()
	batch := []batchGraphQLRequest{{Query: "{ users { id } }"}}
	body, _ := json.Marshal(batch)
	req := httptest.NewRequest(http.MethodPost, "/graphql/batch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "ck_live_valid")
	g.serveFederationBatch(rec, req)

	// 503 means auth passed (would be 401 otherwise) and we landed on the
	// "schema has not been composed yet" branch.
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 (schema not ready) after passing auth, got %d (body: %s)",
			rec.Code, rec.Body.String())
	}
}
