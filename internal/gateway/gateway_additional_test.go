package gateway

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/APICerberus/APICerebrus/internal/config"
)

// Test NewBalancer with different algorithms
func TestNewBalancer(t *testing.T) {
	tests := []struct {
		name      string
		algorithm string
		wantType  string
	}{
		{
			name:      "round robin default",
			algorithm: "",
			wantType:  "*gateway.RoundRobin",
		},
		{
			name:      "round robin explicit",
			algorithm: "round_robin",
			wantType:  "*gateway.RoundRobin",
		},
		{
			name:      "least_conn",
			algorithm: "least_conn",
			wantType:  "*gateway.LeastConn",
		},
		{
			name:      "ip_hash",
			algorithm: "ip_hash",
			wantType:  "*gateway.IPHash",
		},
		{
			name:      "random",
			algorithm: "random",
			wantType:  "*gateway.RandomBalancer",
		},
		{
			name:      "consistent_hash",
			algorithm: "consistent_hash",
			wantType:  "*gateway.ConsistentHash",
		},
		{
			name:      "weighted_round_robin",
			algorithm: "weighted_round_robin",
			wantType:  "*gateway.WeightedRoundRobin",
		},
		{
			name:      "least_latency",
			algorithm: "least_latency",
			wantType:  "*gateway.LeastLatency",
		},
		{
			name:      "adaptive",
			algorithm: "adaptive",
			wantType:  "*gateway.Adaptive",
		},
		{
			name:      "geo_aware",
			algorithm: "geo_aware",
			wantType:  "*gateway.GeoAware",
		},
		{
			name:      "health_weighted",
			algorithm: "health_weighted",
			wantType:  "*gateway.HealthWeighted",
		},
		{
			name:      "unknown algorithm defaults to round robin",
			algorithm: "unknown",
			wantType:  "*gateway.RoundRobin",
		},
		{
			name:      "case insensitive",
			algorithm: "ROUND_ROBIN",
			wantType:  "*gateway.RoundRobin",
		},
		{
			name:      "with whitespace",
			algorithm: "  round_robin  ",
			wantType:  "*gateway.RoundRobin",
		},
	}

	targets := []config.UpstreamTarget{
		{ID: "a", Address: "10.0.0.1:8080"},
		{ID: "b", Address: "10.0.0.2:8080"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			balancer := NewBalancer(tt.algorithm, targets)
			if balancer == nil {
				t.Fatal("NewBalancer returned nil")
			}

			// Verify the balancer can return targets
			target, err := balancer.Next(nil)
			if err != nil {
				t.Fatalf("Next() error = %v", err)
			}
			if target == nil {
				t.Fatal("Next() returned nil target")
			}
		})
	}
}

// Test NewBalancer with empty targets
func TestNewBalancer_EmptyTargets(t *testing.T) {
	balancer := NewBalancer("round_robin", []config.UpstreamTarget{})
	if balancer == nil {
		t.Fatal("NewBalancer returned nil for empty targets")
	}

	// Should return error when no targets available
	_, err := balancer.Next(nil)
	if err != ErrNoHealthyTargets {
		t.Errorf("Next() error = %v, want ErrNoHealthyTargets", err)
	}
}

// Test compiledRoute.matches
func TestCompiledRoute_Matches(t *testing.T) {
	re := regexp.MustCompile("^/api/v1/users$")
	cr := &compiledRoute{
		host:    "example.com",
		methods: map[string]struct{}{"GET": {}, "POST": {}},
		re:      re,
	}

	// Match correct host, method, and path
	if !cr.matches("example.com", "GET", "/api/v1/users") {
		t.Error("Expected match for correct host, method, and path")
	}

	// Wrong host
	if cr.matches("wrong.com", "GET", "/api/v1/users") {
		t.Error("Expected no match for wrong host")
	}

	// Wrong method
	if cr.matches("example.com", "DELETE", "/api/v1/users") {
		t.Error("Expected no match for wrong method")
	}

	// Wrong path
	if cr.matches("example.com", "GET", "/api/v1/products") {
		t.Error("Expected no match for wrong path")
	}
}

// Test compiledRoute.matches with wildcard method
func TestCompiledRoute_Matches_WildcardMethod(t *testing.T) {
	re := regexp.MustCompile("^/api/.*")
	cr := &compiledRoute{
		host:    "",
		methods: map[string]struct{}{"*": {}},
		re:      re,
	}

	// Any method should match with wildcard
	if !cr.matches("any.com", "GET", "/api/users") {
		t.Error("Expected match with wildcard method")
	}
	if !cr.matches("any.com", "POST", "/api/users") {
		t.Error("Expected match with wildcard method")
	}
	if !cr.matches("any.com", "DELETE", "/api/users") {
		t.Error("Expected match with wildcard method")
	}
}

// Test compiledRoute.matches with empty host
func TestCompiledRoute_Matches_EmptyHost(t *testing.T) {
	re := regexp.MustCompile("^/health$")
	cr := &compiledRoute{
		host:    "",
		methods: map[string]struct{}{"GET": {}},
		re:      re,
	}

	// Empty host should match any host
	if !cr.matches("example.com", "GET", "/health") {
		t.Error("Expected match with empty host pattern")
	}
	if !cr.matches("api.example.com", "GET", "/health") {
		t.Error("Expected match with empty host pattern")
	}
}

// Test extractAPIKey from header
func TestExtractAPIKey_FromHeader(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/users", nil)
	req.Header.Set("X-API-Key", "test-api-key-123")

	key := extractAPIKey(req)
	if key != "test-api-key-123" {
		t.Errorf("Expected API key 'test-api-key-123', got %q", key)
	}
}

// Test extractAPIKey from Authorization header
func TestExtractAPIKey_FromAuthorization(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/users", nil)
	req.Header.Set("Authorization", "Bearer bearer-token-456")

	key := extractAPIKey(req)
	if key != "bearer-token-456" {
		t.Errorf("Expected API key 'bearer-token-456', got %q", key)
	}
}

// Test extractAPIKey from query parameter
func TestExtractAPIKey_FromQuery(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/users?apikey=query-key-789", nil)

	key := extractAPIKey(req)
	if key != "query-key-789" {
		t.Errorf("Expected API key 'query-key-789', got %q", key)
	}
}

// Test extractAPIKey from query parameter (api_key)
func TestExtractAPIKey_FromQueryUnderscore(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/users?api_key=underscore-key-abc", nil)

	key := extractAPIKey(req)
	if key != "underscore-key-abc" {
		t.Errorf("Expected API key 'underscore-key-abc', got %q", key)
	}
}

// Test extractAPIKey from cookie
func TestExtractAPIKey_FromCookie(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/users", nil)
	req.AddCookie(&http.Cookie{Name: "apikey", Value: "cookie-key-def"})

	key := extractAPIKey(req)
	if key != "cookie-key-def" {
		t.Errorf("Expected API key 'cookie-key-def', got %q", key)
	}
}

// Test extractAPIKey with no key
func TestExtractAPIKey_NoKey(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/users", nil)

	key := extractAPIKey(req)
	if key != "" {
		t.Errorf("Expected empty API key, got %q", key)
	}
}

// Test extractAPIKey with nil request
func TestExtractAPIKey_NilRequest(t *testing.T) {
	key := extractAPIKey(nil)
	if key != "" {
		t.Errorf("Expected empty API key for nil request, got %q", key)
	}
}

// Test extractAPIKey priority (header over query)
func TestExtractAPIKey_Priority(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/users?apikey=query-key", nil)
	req.Header.Set("X-API-Key", "header-key")

	// Header should take priority
	key := extractAPIKey(req)
	if key != "header-key" {
		t.Errorf("Expected header key 'header-key', got %q", key)
	}
}

// Test extractAPIKey with trimmed whitespace
func TestExtractAPIKey_Trimmed(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/users", nil)
	req.Header.Set("X-API-Key", "  spaced-key  ")

	key := extractAPIKey(req)
	if key != "spaced-key" {
		t.Errorf("Expected trimmed API key 'spaced-key', got %q", key)
	}
}

// Test New with nil config
func TestNew_NilConfig(t *testing.T) {
	_, err := New(nil)
	if err == nil {
		t.Error("Expected error for nil config")
	}
}

// Test New with empty config
func TestNew_EmptyConfig(t *testing.T) {
	cfg := &config.Config{}
	g, err := New(cfg)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if g == nil {
		t.Error("Expected non-nil Gateway")
	}
}

// Test New with valid config
func TestNew_ValidConfig(t *testing.T) {
	cfg := &config.Config{
		Gateway: config.GatewayConfig{
			HTTPAddr: ":8080",
		},
	}
	g, err := New(cfg)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if g == nil {
		t.Error("Expected non-nil Gateway")
	}
}

// Test ConsumerFromRequest
func TestConsumerFromRequest(t *testing.T) {
	// Create request with consumer
	consumer := &config.Consumer{ID: "consumer-123", Name: "Test Consumer"}
	req := httptest.NewRequest("GET", "/api/users", nil)
	setRequestConsumer(req, consumer)

	// Retrieve consumer
	retrieved := ConsumerFromRequest(req)
	if retrieved == nil {
		t.Fatal("Expected non-nil consumer")
	}
	if retrieved.ID != "consumer-123" {
		t.Errorf("Expected consumer ID 'consumer-123', got %q", retrieved.ID)
	}
}

// Test ConsumerFromRequest with no consumer
func TestConsumerFromRequest_NoConsumer(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/users", nil)

	retrieved := ConsumerFromRequest(req)
	if retrieved != nil {
		t.Error("Expected nil consumer when none set")
	}
}

// Test ConsumerFromRequest with nil request
func TestConsumerFromRequest_NilRequest(t *testing.T) {
	retrieved := ConsumerFromRequest(nil)
	if retrieved != nil {
		t.Error("Expected nil consumer for nil request")
	}
}

// Test populateCertificateLeaf with nil cert
func TestPopulateCertificateLeaf_Nil(t *testing.T) {
	err := populateCertificateLeaf(nil)
	if err != nil {
		t.Errorf("Expected nil error for nil cert, got %v", err)
	}
}

// Test populateCertificateLeaf with existing leaf
func TestPopulateCertificateLeaf_ExistingLeaf(t *testing.T) {
	cert := &tls.Certificate{
		Leaf: &x509.Certificate{},
	}
	err := populateCertificateLeaf(cert)
	if err != nil {
		t.Errorf("Expected nil error for cert with existing leaf, got %v", err)
	}
}

// Test certificateIsValidNow with nil cert
func TestCertificateIsValidNow_Nil(t *testing.T) {
	valid := certificateIsValidNow(nil)
	if valid {
		t.Error("Expected false for nil cert")
	}
}

// Test certificateNeedsRenewal with nil cert
func TestCertificateNeedsRenewal_Nil(t *testing.T) {
	needsRenewal := certificateNeedsRenewal(nil, 30*24*time.Hour)
	if !needsRenewal {
		t.Error("Expected true for nil cert (needs renewal)")
	}
}

// Test populateCertificateLeaf with empty certificate
func TestPopulateCertificateLeaf_Empty(t *testing.T) {
	cert := &tls.Certificate{
		Certificate: [][]byte{},
	}
	err := populateCertificateLeaf(cert)
	if err != nil {
		t.Errorf("Expected nil error for empty cert, got %v", err)
	}
}

// Test certificateIsValidNow with empty certificate
func TestCertificateIsValidNow_Empty(t *testing.T) {
	cert := &tls.Certificate{
		Certificate: [][]byte{},
	}
	valid := certificateIsValidNow(cert)
	// Should return true if Leaf is nil (defaults to valid)
	if !valid {
		t.Error("Expected true for empty cert")
	}
}

// Test certificateNeedsRenewal with empty certificate
func TestCertificateNeedsRenewal_Empty(t *testing.T) {
	cert := &tls.Certificate{
		Certificate: [][]byte{},
	}
	needsRenewal := certificateNeedsRenewal(cert, 30*24*time.Hour)
	// Should return false if Leaf is nil
	if needsRenewal {
		t.Error("Expected false for empty cert")
	}
}

// Test RoundRobin Done (no-op function)
func TestRoundRobin_Done(t *testing.T) {
	rr := NewRoundRobin([]config.UpstreamTarget{
		{ID: "a", Address: "10.0.0.1:8080"},
	})

	// Done should not panic
	rr.Done("a")
}

// Test WeightedRoundRobin Done (no-op function)
func TestWeightedRoundRobin_Done(t *testing.T) {
	wrr := NewWeightedRoundRobin([]config.UpstreamTarget{
		{ID: "a", Address: "10.0.0.1:8080", Weight: 1},
	})

	// Done should not panic
	wrr.Done("a")
}

// Test NewTLSManager with empty config
func TestNewTLSManager_EmptyConfig(t *testing.T) {
	cfg := config.TLSConfig{}
	tm, err := NewTLSManager(cfg)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if tm == nil {
		t.Error("Expected non-nil TLSManager")
	}
}

// Test NewTLSManager with only cert file
func TestNewTLSManager_OnlyCertFile(t *testing.T) {
	cfg := config.TLSConfig{
		CertFile: "/path/to/cert.pem",
	}
	_, err := NewTLSManager(cfg)
	if err == nil {
		t.Error("Expected error when only cert_file is provided")
	}
}

// Test NewTLSManager with only key file
func TestNewTLSManager_OnlyKeyFile(t *testing.T) {
	cfg := config.TLSConfig{
		KeyFile: "/path/to/key.pem",
	}
	_, err := NewTLSManager(cfg)
	if err == nil {
		t.Error("Expected error when only key_file is provided")
	}
}

// Test NewTLSManager with auto and ACME
func TestNewTLSManager_AutoACME(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.TLSConfig{
		Auto:      true,
		ACMEEmail: "test@example.com",
		ACMEDir:   tmpDir,
	}
	tm, err := NewTLSManager(cfg)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if tm == nil {
		t.Error("Expected non-nil TLSManager")
	}
}

// Test New with federation enabled
func TestNew_WithFederation(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Gateway: config.GatewayConfig{
			HTTPAddr: ":18080",
		},
		Store: config.StoreConfig{
			Path: tmpDir + "/test.db",
		},
		Federation: config.FederationConfig{
			Enabled: true,
		},
	}
	g, err := New(cfg)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if g == nil {
		t.Fatal("Expected non-nil Gateway")
	}
	if !g.federationEnabled {
		t.Error("Expected federation to be enabled")
	}
	if g.subgraphs == nil {
		t.Error("Expected subgraphs manager to be initialized")
	}
	if g.federationComposer == nil {
		t.Error("Expected federation composer to be initialized")
	}
	if g.federationExecutor == nil {
		t.Error("Expected federation executor to be initialized")
	}
	// Shutdown to release database lock before temp dir cleanup
	if g != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		g.Shutdown(ctx)
	}
}

// Test Health Checker Snapshot
func TestHealthChecker_Snapshot(t *testing.T) {
	upstream := config.Upstream{
		Name: "upstream1",
		Targets: []config.UpstreamTarget{
			{ID: "target1", Address: "10.0.0.1:8080"},
			{ID: "target2", Address: "10.0.0.2:8080"},
		},
	}

	checker := NewChecker([]config.Upstream{upstream}, map[string]*UpstreamPool{})

	// Get snapshot - use Name not ID
	snapshot := checker.Snapshot("upstream1")
	if len(snapshot) != 2 {
		t.Errorf("Expected 2 targets in snapshot, got %d", len(snapshot))
	}

	// Check non-existent upstream
	emptySnapshot := checker.Snapshot("nonexistent")
	if len(emptySnapshot) != 0 {
		t.Error("Expected empty snapshot for non-existent upstream")
	}
}

// Test Health Checker IsHealthy
func TestHealthChecker_IsHealthy(t *testing.T) {
	upstream := config.Upstream{
		Name: "upstream1",
		Targets: []config.UpstreamTarget{
			{ID: "target1", Address: "10.0.0.1:8080"},
		},
	}

	checker := NewChecker([]config.Upstream{upstream}, map[string]*UpstreamPool{})

	// Should return true if not explicitly marked unhealthy (defaults to healthy)
	// Use Name not ID for upstream, and target ID for target
	healthy := checker.IsHealthy("upstream1", "target1")
	if !healthy {
		t.Error("Expected target to be healthy by default")
	}

	// Non-existent upstream should return false
	healthy = checker.IsHealthy("nonexistent", "target1")
	if healthy {
		t.Error("Expected non-existent upstream to return false")
	}
}

// Test Balancer ReportHealth methods
func TestRoundRobin_ReportHealth(t *testing.T) {
	rr := NewRoundRobin([]config.UpstreamTarget{
		{ID: "a", Address: "10.0.0.1:8080"},
		{ID: "b", Address: "10.0.0.2:8080"},
	})

	// Report health - should not panic (includes duration parameter)
	rr.ReportHealth("a", true, 0)
	rr.ReportHealth("b", false, 0)
}

// Test WeightedRoundRobin ReportHealth
func TestWeightedRoundRobin_ReportHealth(t *testing.T) {
	wrr := NewWeightedRoundRobin([]config.UpstreamTarget{
		{ID: "a", Address: "10.0.0.1:8080", Weight: 1},
		{ID: "b", Address: "10.0.0.2:8080", Weight: 2},
	})

	// Report health - should not panic
	wrr.ReportHealth("a", true, 0)
	wrr.ReportHealth("b", false, 0)
}

// Test Gateway Uptime
func TestGateway_Uptime(t *testing.T) {
	cfg := &config.Config{}
	g, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer g.Shutdown(context.Background())

	uptime := g.Uptime()
	if uptime < 0 {
		t.Error("Uptime should be non-negative")
	}
}

// Test Gateway UpstreamHealth
func TestGateway_UpstreamHealth(t *testing.T) {
	cfg := &config.Config{}
	g, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer g.Shutdown(context.Background())

	// Get upstream health (should return empty map)
	health := g.UpstreamHealth("nonexistent")
	if health == nil {
		t.Error("UpstreamHealth should return non-nil map")
	}
}

// Test Gateway Federation getters
func TestGateway_FederationGetters(t *testing.T) {
	cfg := &config.Config{}
	g, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer g.Shutdown(context.Background())

	// Subgraphs (should return nil when disabled)
	subgraphs := g.Subgraphs()
	if subgraphs != nil {
		t.Error("Subgraphs should return nil when federation is disabled")
	}

	// FederationComposer (should return nil when disabled)
	composer := g.FederationComposer()
	if composer != nil {
		t.Error("FederationComposer should return nil when federation is disabled")
	}

	// FederationEnabled
	enabled := g.FederationEnabled()
	if enabled {
		t.Error("FederationEnabled should return false when not configured")
	}
}
