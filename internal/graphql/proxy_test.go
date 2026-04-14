package graphql

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewProxy(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		cfg := &ProxyConfig{
			TargetURL: "http://localhost:8080/graphql",
			Timeout:   30 * time.Second,
		}
		proxy, err := NewProxy(cfg)
		if err != nil {
			t.Fatalf("NewProxy(config.PoolConfig{}) error = %v", err)
		}
		if proxy == nil {
			t.Fatal("NewProxy(config.PoolConfig{}) returned nil")
		}
		if proxy.target == nil {
			t.Error("target is nil")
		}
		if proxy.client == nil {
			t.Error("client is nil")
		}
		if proxy.reverseProxy == nil {
			t.Error("reverseProxy is nil")
		}
		if proxy.subscriptionProxy == nil {
			t.Error("subscriptionProxy is nil")
		}
	})

	t.Run("invalid URL", func(t *testing.T) {
		cfg := &ProxyConfig{
			TargetURL: "://invalid-url",
		}
		_, err := NewProxy(cfg)
		if err == nil {
			t.Error("NewProxy(config.PoolConfig{}) should return error for invalid URL")
		}
	})

	t.Run("default timeout", func(t *testing.T) {
		cfg := &ProxyConfig{
			TargetURL: "http://localhost:8080/graphql",
			Timeout:   0, // Should use default
		}
		proxy, err := NewProxy(cfg)
		if err != nil {
			t.Fatalf("NewProxy(config.PoolConfig{}) error = %v", err)
		}
		if proxy.client.Timeout != 30*time.Second {
			t.Errorf("Timeout = %v, want 30s", proxy.client.Timeout)
		}
	})
}

func TestProxy_Forward(t *testing.T) {
	// Create a mock upstream server
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", ct)
		}

		// Read and verify body
		var req Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request: %v", err)
		}

		// Send response
		resp := Response{
			Data: json.RawMessage(`{"users":[{"id":"1","name":"Alice"}]}`),
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer upstream.Close()

	cfg := &ProxyConfig{
		TargetURL: upstream.URL,
		Timeout:   5 * time.Second,
	}
	proxy, err := NewProxy(cfg)
	if err != nil {
		t.Fatalf("NewProxy(config.PoolConfig{}) error = %v", err)
	}

	t.Run("forward query", func(t *testing.T) {
		req := &Request{
			Query: "{ users { id name } }",
		}
		resp, err := proxy.Forward(req)
		if err != nil {
			t.Errorf("Forward() error = %v", err)
		}
		if resp == nil {
			t.Fatal("Forward() returned nil")
		}
		if resp.Data == nil {
			t.Error("Response Data is nil")
		}
	})

	t.Run("subscription query rejected", func(t *testing.T) {
		req := &Request{
			Query: "subscription { messageAdded { id } }",
		}
		_, err := proxy.Forward(req)
		if err == nil {
			t.Error("Forward() should return error for subscription")
		}
		if !strings.Contains(err.Error(), "WebSocket") {
			t.Errorf("Error should mention WebSocket, got: %v", err)
		}
	})
}

func TestIntrospectionChecker(t *testing.T) {
	t.Run("introspection allowed", func(t *testing.T) {
		checker := NewIntrospectionChecker(true)
		if checker == nil {
			t.Fatal("NewIntrospectionChecker returned nil")
		}
		if !checker.Check("{ __schema { types { name } } }") {
			t.Error("Check should return true when introspection allowed")
		}
	})

	t.Run("introspection blocked", func(t *testing.T) {
		checker := NewIntrospectionChecker(false)
		if checker.Check("{ __schema { types { name } } }") {
			t.Error("Check should return false for introspection query when blocked")
		}
	})

	t.Run("regular query allowed even when blocked", func(t *testing.T) {
		checker := NewIntrospectionChecker(false)
		if !checker.Check("{ users { id name } }") {
			t.Error("Check should return true for regular query")
		}
	})

	t.Run("query with __typename blocked when introspection disabled", func(t *testing.T) {
		checker := NewIntrospectionChecker(false)
		// __typename is considered an introspection field
		if checker.Check("{ users { id __typename } }") {
			t.Error("Check should return false for query with __typename when introspection blocked")
		}
	})
}

func TestProxy_Forward_WithVariables(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req Request
		_ = json.NewDecoder(r.Body).Decode(&req)

		if req.Variables == nil {
			t.Error("Variables should not be nil")
		}

		resp := Response{
			Data: json.RawMessage(`{"user":{"id":"1"}}`),
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer upstream.Close()

	cfg := &ProxyConfig{
		TargetURL: upstream.URL,
		Timeout:   5 * time.Second,
	}
	proxy, _ := NewProxy(cfg)

	req := &Request{
		Query:     "query GetUser($id: ID!) { user(id: $id) { id } }",
		Variables: map[string]interface{}{"id": "1"},
	}
	resp, err := proxy.Forward(req)
	if err != nil {
		t.Errorf("Forward() error = %v", err)
	}
	if resp == nil {
		t.Error("Forward() returned nil response")
	}
}

func TestProxy_Forward_WithOperationName(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req Request
		_ = json.NewDecoder(r.Body).Decode(&req)

		if req.OperationName != "GetUser" {
			t.Errorf("OperationName = %s, want GetUser", req.OperationName)
		}

		resp := Response{
			Data: json.RawMessage(`{"user":{"id":"1"}}`),
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer upstream.Close()

	cfg := &ProxyConfig{
		TargetURL: upstream.URL,
		Timeout:   5 * time.Second,
	}
	proxy, _ := NewProxy(cfg)

	req := &Request{
		Query:         "query GetUser { user { id } }",
		OperationName: "GetUser",
	}
	resp, err := proxy.Forward(req)
	if err != nil {
		t.Errorf("Forward() error = %v", err)
	}
	if resp == nil {
		t.Error("Forward() returned nil response")
	}
}

func TestProxy_Forward_ErrorResponse(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`invalid json`))
	}))
	defer upstream.Close()

	cfg := &ProxyConfig{
		TargetURL: upstream.URL,
		Timeout:   5 * time.Second,
	}
	proxy, _ := NewProxy(cfg)

	req := &Request{
		Query: "{ users { id } }",
	}
	_, err := proxy.Forward(req)
	if err == nil {
		t.Error("Forward() should return error for invalid JSON response")
	}
}

func TestProxy_Forward_NetworkError(t *testing.T) {
	cfg := &ProxyConfig{
		TargetURL: "http://localhost:1", // Unlikely to be open
		Timeout:   100 * time.Millisecond,
	}
	proxy, _ := NewProxy(cfg)

	req := &Request{
		Query: "{ users { id } }",
	}
	_, err := proxy.Forward(req)
	if err == nil {
		t.Error("Forward() should return error for network failure")
	}
}

func TestProxy_Forward_Mutation(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := Response{
			Data: json.RawMessage(`{"createUser":{"id":"123"}}`),
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer upstream.Close()

	cfg := &ProxyConfig{
		TargetURL: upstream.URL,
		Timeout:   5 * time.Second,
	}
	proxy, _ := NewProxy(cfg)

	req := &Request{
		Query: "mutation { createUser(name: \"Alice\") { id } }",
	}
	resp, err := proxy.Forward(req)
	if err != nil {
		t.Errorf("Forward() error = %v", err)
	}
	if resp == nil {
		t.Error("Forward() returned nil response")
	}
}

func TestProxyConfig(t *testing.T) {
	cfg := &ProxyConfig{
		TargetURL: "http://example.com/graphql",
		Timeout:   time.Minute,
	}

	if cfg.TargetURL != "http://example.com/graphql" {
		t.Errorf("TargetURL = %s, want http://example.com/graphql", cfg.TargetURL)
	}
	if cfg.Timeout != time.Minute {
		t.Errorf("Timeout = %v, want 1m", cfg.Timeout)
	}
}
