package plugin

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/APICerberus/APICerebrus/internal/config"
)

// Test GraphQLGuard plugin
func TestGraphQLGuard_Methods(t *testing.T) {
	t.Run("NewGraphQLGuard with nil config", func(t *testing.T) {
		guard := NewGraphQLGuard(nil)
		if guard == nil {
			t.Fatal("NewGraphQLGuard(nil) returned nil")
		}
		if guard.maxDepth != 15 {
			t.Errorf("maxDepth = %d, want 15", guard.maxDepth)
		}
		if guard.maxComplexity != 1000 {
			t.Errorf("maxComplexity = %d, want 1000", guard.maxComplexity)
		}
	})

	t.Run("NewGraphQLGuard with custom config", func(t *testing.T) {
		guard := NewGraphQLGuard(&GraphQLGuardConfig{
			MaxDepth:           10,
			MaxComplexity:      500,
			BlockIntrospection: true,
			FieldCosts:         map[string]int{"Query": 2},
		})
		if guard == nil {
			t.Fatal("NewGraphQLGuard returned nil")
		}
		if guard.maxDepth != 10 {
			t.Errorf("maxDepth = %d, want 10", guard.maxDepth)
		}
		if guard.maxComplexity != 500 {
			t.Errorf("maxComplexity = %d, want 500", guard.maxComplexity)
		}
		if !guard.blockIntrospection {
			t.Error("blockIntrospection should be true")
		}
	})

	t.Run("Name returns graphql_guard", func(t *testing.T) {
		guard := NewGraphQLGuard(nil)
		if guard.Name() != "graphql_guard" {
			t.Errorf("Name() = %q, want %q", guard.Name(), "graphql_guard")
		}
	})

	t.Run("Phase returns PhasePreAuth", func(t *testing.T) {
		guard := NewGraphQLGuard(nil)
		if guard.Phase() != PhasePreAuth {
			t.Errorf("Phase() = %v, want %v", guard.Phase(), PhasePreAuth)
		}
	})

	t.Run("Priority returns 2", func(t *testing.T) {
		guard := NewGraphQLGuard(nil)
		if guard.Priority() != 2 {
			t.Errorf("Priority() = %d, want 2", guard.Priority())
		}
	})

	t.Run("Handle with nil receiver", func(t *testing.T) {
		var guard *GraphQLGuard
		if guard.Handle(nil, nil) {
			t.Error("Handle should return false with nil receiver")
		}
	})

	t.Run("Handle with nil writer", func(t *testing.T) {
		guard := NewGraphQLGuard(nil)
		if guard.Handle(nil, nil) {
			t.Error("Handle should return false with nil writer")
		}
	})

	t.Run("Handle with nil request", func(t *testing.T) {
		guard := NewGraphQLGuard(nil)
		w := &mockResponseWriter{}
		if guard.Handle(w, nil) {
			t.Error("Handle should return false with nil request")
		}
	})
}

// Mock response writer for testing
type mockResponseWriter struct {
	header     http.Header
	statusCode int
	body       []byte
	writeErr   error
}

func (m *mockResponseWriter) Header() http.Header {
	if m.header == nil {
		m.header = make(http.Header)
	}
	return m.header
}

func (m *mockResponseWriter) Write(b []byte) (int, error) {
	if m.writeErr != nil {
		return 0, m.writeErr
	}
	m.body = append(m.body, b...)
	return len(b), nil
}

func (m *mockResponseWriter) WriteHeader(code int) {
	m.statusCode = code
}

// Test EndpointPermissionError Error method
func TestEndpointPermissionError_Error(t *testing.T) {
	err := &EndpointPermissionError{
		Code:    "test_code",
		Message: "test message",
	}

	if err.Error() != "test message" {
		t.Errorf("Error() = %q, want %q", err.Error(), "test message")
	}
}

// Test claimValueToHeader function
func TestClaimValueToHeader(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		wantVal  string
		wantBool bool
	}{
		{"empty string", "", "", false},
		{"whitespace string", "   ", "", false},
		{"valid string", "test", "test", true},
		{"string with spaces", " test ", "test", true},
		{"float64", float64(42), "42", true},
		{"float32", float32(42), "42", true},
		{"int", int(42), "42", true},
		{"int64", int64(42), "42", true},
		{"nil", nil, "<nil>", true},  // nil falls through to default case which uses fmt.Sprint
		{"empty []any", []any{}, "", false},
		{"[]any with values", []any{"a", "b"}, "a,b", true},
		{"[]any with nil", []any{nil, "a"}, "a", true},
		{"bool true", true, "true", true},
		{"bool false", false, "false", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, ok := claimValueToHeader(tt.input)
			if val != tt.wantVal {
				t.Errorf("claimValueToHeader() val = %q, want %q", val, tt.wantVal)
			}
			if ok != tt.wantBool {
				t.Errorf("claimValueToHeader() ok = %v, want %v", ok, tt.wantBool)
			}
		})
	}
}

// Test hasClaimValue function
func TestHasClaimValue(t *testing.T) {
	tests := []struct {
		name string
		input any
		want bool
	}{
		{"nil", nil, false},
		{"empty string", "", false},
		{"whitespace string", "   ", false},
		{"valid string", "test", true},
		{"empty []any", []any{}, false},
		{"[]any with values", []any{"a"}, true},
		{"empty []string", []string{}, false},
		{"[]string with values", []string{"a"}, true},
		{"int", 42, true},
		{"bool", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasClaimValue(tt.input)
			if got != tt.want {
				t.Errorf("hasClaimValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Test CircuitBreaker state transitions
func TestCircuitBreaker_StateTransitions(t *testing.T) {
	t.Run("closed state", func(t *testing.T) {
		cb := NewCircuitBreaker(CircuitBreakerConfig{
			ErrorThreshold: 0.5,
			SleepWindow:    time.Second,
		})
		if cb.State() != CircuitClosed {
			t.Errorf("Initial state = %v, want CLOSED", cb.State())
		}
	})

	t.Run("nil config uses defaults", func(t *testing.T) {
		cb := NewCircuitBreaker(CircuitBreakerConfig{})
		if cb == nil {
			t.Fatal("NewCircuitBreaker returned nil")
		}
		if cb.State() != CircuitClosed {
			t.Error("Should start in CLOSED state")
		}
	})
}

// Test AuthJWT plugin
func TestAuthJWT_Methods(t *testing.T) {
	t.Run("NewAuthJWT with empty options", func(t *testing.T) {
		jwtAuth := NewAuthJWT(AuthJWTOptions{})
		if jwtAuth == nil {
			t.Fatal("NewAuthJWT returned nil")
		}
		// Check defaults are applied
		if jwtAuth.clockSkew != 30*time.Second {
			t.Errorf("clockSkew = %v, want 30s", jwtAuth.clockSkew)
		}
	})

	t.Run("NewAuthJWT with custom clock skew", func(t *testing.T) {
		jwtAuth := NewAuthJWT(AuthJWTOptions{
			ClockSkew: 60 * time.Second,
		})
		if jwtAuth.clockSkew != 60*time.Second {
			t.Errorf("clockSkew = %v, want 60s", jwtAuth.clockSkew)
		}
	})

	t.Run("NewAuthJWT with negative clock skew uses default", func(t *testing.T) {
		jwtAuth := NewAuthJWT(AuthJWTOptions{
			ClockSkew: -10 * time.Second,
		})
		// Negative clock skew becomes 0, then 0 becomes default 30s
		if jwtAuth.clockSkew != 30*time.Second {
			t.Errorf("clockSkew = %v, want 30s", jwtAuth.clockSkew)
		}
	})

	t.Run("NewAuthJWT with claims to headers", func(t *testing.T) {
		jwtAuth := NewAuthJWT(AuthJWTOptions{
			ClaimsToHeaders: map[string]string{
				"sub": "X-User-ID",
				"email": "X-User-Email",
			},
		})
		if jwtAuth.claimsToHeaders["sub"] != "X-User-ID" {
			t.Error("claimsToHeaders not set correctly")
		}
	})

	t.Run("Name returns auth-jwt", func(t *testing.T) {
		jwtAuth := NewAuthJWT(AuthJWTOptions{})
		if jwtAuth.Name() != "auth-jwt" {
			t.Errorf("Name() = %q, want %q", jwtAuth.Name(), "auth-jwt")
		}
	})

	t.Run("Phase returns PhaseAuth", func(t *testing.T) {
		jwtAuth := NewAuthJWT(AuthJWTOptions{})
		if jwtAuth.Phase() != PhaseAuth {
			t.Errorf("Phase() = %v, want %v", jwtAuth.Phase(), PhaseAuth)
		}
	})

	t.Run("Priority returns 20", func(t *testing.T) {
		jwtAuth := NewAuthJWT(AuthJWTOptions{})
		if jwtAuth.Priority() != 20 {
			t.Errorf("Priority() = %d, want 20", jwtAuth.Priority())
		}
	})
}

// Test CircuitBreaker Allow method
func TestCircuitBreaker_Allow(t *testing.T) {
	t.Run("Allow in closed state", func(t *testing.T) {
		cb := NewCircuitBreaker(CircuitBreakerConfig{
			ErrorThreshold: 0.5,
			SleepWindow:    time.Second,
		})
		// In closed state, Allow should return nil (allow request)
		err := cb.Allow()
		if err != nil {
			t.Errorf("Allow() in closed state = %v, want nil", err)
		}
	})

	t.Run("Allow with nil config uses defaults", func(t *testing.T) {
		cb := NewCircuitBreaker(CircuitBreakerConfig{})
		if cb == nil {
			t.Fatal("NewCircuitBreaker returned nil")
		}
		err := cb.Allow()
		if err != nil {
			t.Errorf("Allow() with defaults = %v, want nil", err)
		}
	})
}

// Test compression Apply with various content types
func TestCompression_Apply(t *testing.T) {
	t.Run("with nil context", func(t *testing.T) {
		c := NewCompression(CompressionConfig{
			MinSize: 100,
		})

		// Should not panic with nil
		c.Apply(nil)
	})

	t.Run("with empty config", func(t *testing.T) {
		c := NewCompression(CompressionConfig{})
		if c == nil {
			t.Fatal("NewCompression returned nil")
		}
		if c.minSize != 0 {
			t.Errorf("minSize = %d, want 0", c.minSize)
		}
	})
}

// Test BotDetect
func TestBotDetect(t *testing.T) {
	t.Run("NewBotDetect", func(t *testing.T) {
		bd := NewBotDetect(BotDetectConfig{
			DenyList: []string{"badbot"},
			Action:   "block",
		})
		if bd == nil {
			t.Fatal("NewBotDetect returned nil")
		}
	})

	t.Run("NewBotDetect nil config", func(t *testing.T) {
		bd := NewBotDetect(BotDetectConfig{})
		if bd == nil {
			t.Fatal("NewBotDetect returned nil")
		}
	})
}

// Test CorrelationID
func TestCorrelationID(t *testing.T) {
	t.Run("NewCorrelationID", func(t *testing.T) {
		cid := NewCorrelationID()
		if cid == nil {
			t.Fatal("NewCorrelationID returned nil")
		}
	})

	t.Run("Apply with nil", func(t *testing.T) {
		cid := NewCorrelationID()
		// Should not panic with nil
		cid.Apply(nil)
	})
}

// Test AuthAPIKey additional methods
func TestAuthAPIKey_Additional(t *testing.T) {
	t.Run("Lookup method", func(t *testing.T) {
		auth := NewAuthAPIKey(nil, AuthAPIKeyOptions{
			KeyNames: []string{"X-API-Key"},
		})
		// Lookup with empty key should return nil
		result, err := auth.Lookup("")
		if err == nil && result != nil {
			t.Error("Lookup with empty key should return nil or error")
		}
	})

	t.Run("DebugSummary", func(t *testing.T) {
		auth := NewAuthAPIKey(nil, AuthAPIKeyOptions{
			KeyNames: []string{"X-API-Key"},
		})
		summary := auth.DebugSummary()
		if summary == "" {
			t.Error("DebugSummary should return non-empty string")
		}
	})
}

// Test BotDetect methods
func TestBotDetect_Methods(t *testing.T) {
	bd := NewBotDetect(BotDetectConfig{
		DenyList: []string{"badbot"},
		Action:   "block",
	})

	t.Run("Name returns bot-detect", func(t *testing.T) {
		if bd.Name() != "bot-detect" {
			t.Errorf("Name() = %q, want %q", bd.Name(), "bot-detect")
		}
	})

	t.Run("Phase returns PhasePreAuth", func(t *testing.T) {
		if bd.Phase() != PhasePreAuth {
			t.Errorf("Phase() = %v, want %v", bd.Phase(), PhasePreAuth)
		}
	})

	t.Run("Priority returns 3", func(t *testing.T) {
		if bd.Priority() != 3 {
			t.Errorf("Priority() = %d, want 3", bd.Priority())
		}
	})
}

// Test error types
func TestError_Types(t *testing.T) {
	t.Run("AuthError Error", func(t *testing.T) {
		err := &AuthError{
			Code:    "invalid_key",
			Message: "API key is invalid",
		}
		if err.Error() != "API key is invalid" {
			t.Errorf("Error() = %q, want %q", err.Error(), "API key is invalid")
		}
	})

	t.Run("JWTAuthError Error", func(t *testing.T) {
		err := &JWTAuthError{
			Code:    "invalid_token",
			Message: "token is expired",
		}
		if err.Error() != "token is expired" {
			t.Errorf("Error() = %q, want %q", err.Error(), "token is expired")
		}
	})

	t.Run("BotDetectError Error", func(t *testing.T) {
		err := &BotDetectError{
			Code:    "bot_detected",
			Message: "bot detected in request",
		}
		if err.Error() != "bot detected in request" {
			t.Errorf("Error() = %q, want %q", err.Error(), "bot detected in request")
		}
	})
}

// Test CircuitBreaker State method
func TestCircuitBreaker_State(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		ErrorThreshold: 0.5,
		SleepWindow:    time.Second,
	})

	// Initial state should be Closed
	if cb.State() != CircuitClosed {
		t.Errorf("Initial state = %v, want CircuitClosed", cb.State())
	}
}

// Test normalizeRedirectStatus function
func TestNormalizeRedirectStatus(t *testing.T) {
	tests := []struct {
		input    int
		expected int
	}{
		{301, 301},
		{302, 302},
		{307, 307},
		{308, 308},
		{200, 302}, // Invalid, should default to 302
		{404, 302}, // Invalid, should default to 302
		{500, 302}, // Invalid, should default to 302
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("status_%d", tt.input), func(t *testing.T) {
			got := normalizeRedirectStatus(tt.input)
			if got != tt.expected {
				t.Errorf("normalizeRedirectStatus(%d) = %d, want %d", tt.input, got, tt.expected)
			}
		})
	}
}

// Test appendQueryIfMissing function
func TestAppendQueryIfMissing(t *testing.T) {
	tests := []struct {
		url      string
		query    string
		expected string
	}{
		{"http://example.com", "foo=bar", "http://example.com?foo=bar"},
		{"http://example.com?existing=param", "foo=bar", "http://example.com?existing=param"},
		{"http://example.com", "", "http://example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got := appendQueryIfMissing(tt.url, tt.query)
			if got != tt.expected {
				t.Errorf("appendQueryIfMissing(%q, %q) = %q, want %q", tt.url, tt.query, got, tt.expected)
			}
		})
	}
}

// Test parseMinute function
func TestParseMinute(t *testing.T) {
	tests := []struct {
		input    string
		expected int
		wantErr  bool
	}{
		{"00:00", 0, false},
		{"00:30", 30, false},
		{"00:59", 59, false},
		{"12:00", 720, false},
		{"23:59", 1439, false},
		{"25:00", 0, true},     // Hour out of range
		{"12:60", 0, true},     // Minute out of range
		{"-1:00", 0, true},     // Negative hour
		{"12:-1", 0, true},     // Negative minute
		{"abc", 0, true},       // Invalid
		{"", 0, true},          // Empty
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseMinute(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseMinute(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.expected {
				t.Errorf("parseMinute(%q) = %d, want %d", tt.input, got, tt.expected)
			}
		})
	}
}

// Test parseTimeRange function
func TestParseTimeRange(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantErr  bool
	}{
		{"valid range", "00:00-23:59", false},
		{"business hours", "09:00-17:00", false},
		{"empty", "", true},
		{"invalid format", "invalid", true},
		{"no dash", "00:00 23:59", true},
		{"same time not allowed", "12:00-12:00", true}, // Same time is invalid
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			startMin, endMin, err := parseTimeRange(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseTimeRange() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				// Verify times are valid minutes
				if startMin < 0 || startMin > 1439 {
					t.Errorf("startMin = %d, want between 0 and 1439", startMin)
				}
				if endMin < 0 || endMin > 1439 {
					t.Errorf("endMin = %d, want between 0 and 1439", endMin)
				}
			}
		})
	}
}

// Test consumerKey function
func TestConsumerKey(t *testing.T) {
	tests := []struct {
		name     string
		consumer *config.Consumer
		expected string
	}{
		{
			name:     "nil consumer",
			consumer: nil,
			expected: "anonymous",
		},
		{
			name:     "consumer with ID",
			consumer: &config.Consumer{ID: "consumer-123"},
			expected: "consumer-123",
		},
		{
			name:     "consumer without ID",
			consumer: &config.Consumer{},
			expected: "anonymous",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := consumerKey(tt.consumer)
			if got != tt.expected {
				t.Errorf("consumerKey() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// Test routeKey function
func TestRouteKey(t *testing.T) {
	tests := []struct {
		name     string
		route    *config.Route
		req      *http.Request
		expected string
	}{
		{
			name:     "nil route",
			route:    nil,
			req:      nil,
			expected: "unknown", // Function returns "unknown" for nil route
		},
		{
			name:     "route with ID",
			route:    &config.Route{ID: "route-123"},
			req:      nil,
			expected: "route-123",
		},
		{
			name:     "route with Name",
			route:    &config.Route{Name: "route-name"},
			req:      nil,
			expected: "route-name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := routeKey(tt.route, tt.req)
			if got != tt.expected {
				t.Errorf("routeKey() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// Test extractBearerToken function
func TestExtractBearerToken(t *testing.T) {
	tests := []struct {
		name     string
		auth     string
		expected string
	}{
		{"Bearer token", "Bearer abc123", "abc123"},
		{"bearer lowercase", "bearer abc123", "abc123"}, // EqualFold makes it case-insensitive
		{"BEARER uppercase", "BEARER abc123", "abc123"},
		{"with extra spaces", "Bearer   abc123", "abc123"}, // Spaces are trimmed
		{"short token", "Bearer x", "x"}, // Just needs 8+ chars
		{"empty", "", ""},
		{"only bearer", "Bearer", ""},
		{"no space", "Bearerabc", ""}, // Must have space after Bearer
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &http.Request{
				Header: http.Header{
					"Authorization": []string{tt.auth},
				},
			}
			got := extractBearerToken(req)
			if got != tt.expected {
				t.Errorf("extractBearerToken(%q) = %q, want %q", tt.auth, got, tt.expected)
			}
		})
	}
}

// Test extractBearerToken with nil request
func TestExtractBearerToken_NilRequest(t *testing.T) {
	got := extractBearerToken(nil)
	if got != "" {
		t.Errorf("extractBearerToken(nil) = %q, want empty string", got)
	}
}

// Test ensureVaryAcceptEncoding function
func TestEnsureVaryAcceptEncoding(t *testing.T) {
	tests := []struct {
		name     string
		input    http.Header
		expected string
	}{
		{
			name:     "empty header",
			input:    http.Header{},
			expected: "Accept-Encoding",
		},
		{
			name: "existing vary header",
			input: http.Header{
				"Vary": []string{"Authorization"},
			},
			expected: "Authorization, Accept-Encoding",
		},
		{
			name: "already has accept-encoding",
			input: http.Header{
				"Vary": []string{"Accept-Encoding"},
			},
			expected: "Accept-Encoding",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ensureVaryAcceptEncoding(tt.input)
			got := tt.input.Get("Vary")
			if got != tt.expected {
				t.Errorf("Vary header = %q, want %q", got, tt.expected)
			}
		})
	}
}

// Test phaseOrder function
func TestPhaseOrder(t *testing.T) {
	tests := []struct {
		phase    Phase
		expected int
	}{
		{PhasePreAuth, 1},
		{PhaseAuth, 2},
		{PhasePreProxy, 3},
		{PhaseProxy, 4},
		{PhasePostProxy, 5},
		{Phase("unknown"), 999}, // Unknown phase
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("phase_%s", tt.phase), func(t *testing.T) {
			got := phaseOrder(tt.phase)
			if got != tt.expected {
				t.Errorf("phaseOrder(%v) = %d, want %d", tt.phase, got, tt.expected)
			}
		})
	}
}

// Test asString function
func TestAsString(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{"string", "hello", "hello"},
		{"int", 42, "42"},
		{"int64", int64(42), "42"},
		{"float64", 3.14, "3.14"},
		{"bool", true, "true"},
		{"nil", nil, ""},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := asString(tt.input)
			if got != tt.expected {
				t.Errorf("asString(%v) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

// Test asStringSlice function
func TestAsStringSlice(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected []string
	}{
		{
			name:     "[]string",
			input:    []string{"a", "b", "c"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "[]interface{}",
			input:    []any{"a", "b", "c"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "[]interface{} with ints",
			input:    []any{1, 2, 3},
			expected: []string{"1", "2", "3"},
		},
		{
			name:     "nil",
			input:    nil,
			expected: nil,
		},
		{
			name:     "string",
			input:    "not a slice",
			expected: []string{"not a slice"}, // String is wrapped in a slice
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := asStringSlice(tt.input)
			if len(got) != len(tt.expected) {
				t.Errorf("asStringSlice(%v) = %v, want %v", tt.input, got, tt.expected)
				return
			}
			for i := range tt.expected {
				if got[i] != tt.expected[i] {
					t.Errorf("asStringSlice(%v)[%d] = %q, want %q", tt.input, i, got[i], tt.expected[i])
				}
			}
		})
	}
}

// Test plugin interface methods
func TestCorrelationID_Methods(t *testing.T) {
	plugin := &CorrelationID{}

	if plugin.Name() != "correlation-id" {
		t.Errorf("Name() = %v, want correlation-id", plugin.Name())
	}

	if plugin.Phase() != PhasePreAuth {
		t.Errorf("Phase() = %v, want PhasePreAuth", plugin.Phase())
	}

	if plugin.Priority() != 0 {
		t.Errorf("Priority() = %v, want 0", plugin.Priority())
	}
}

func TestTimeout_Methods(t *testing.T) {
	timeout := NewTimeout(TimeoutConfig{Duration: time.Second})

	if timeout.Name() != "timeout" {
		t.Errorf("Name() = %v, want timeout", timeout.Name())
	}

	if timeout.Phase() != PhaseProxy {
		t.Errorf("Phase() = %v, want PhaseProxy", timeout.Phase())
	}

	if timeout.Priority() != 10 {
		t.Errorf("Priority() = %v, want 10", timeout.Priority())
	}
}

func TestCompression_Methods(t *testing.T) {
	comp := NewCompression(CompressionConfig{})

	if comp.Name() != "compression" {
		t.Errorf("Name() = %v, want compression", comp.Name())
	}

	if comp.Phase() != PhasePostProxy {
		t.Errorf("Phase() = %v, want PhasePostProxy", comp.Phase())
	}
}

func TestCompression_DefaultLevel(t *testing.T) {
	// Test with default level (0 means no compression)
	comp := NewCompression(CompressionConfig{})
	if comp == nil {
		t.Error("NewCompression should not return nil")
	}
}

// Test simple plugin interface methods
func TestPluginInterfaceMethods(t *testing.T) {
	t.Run("CorrelationID", func(t *testing.T) {
		p := &CorrelationID{}
		if p.Name() != "correlation-id" {
			t.Errorf("Name() = %v", p.Name())
		}
		if p.Phase() != PhasePreAuth {
			t.Errorf("Phase() = %v", p.Phase())
		}
		if p.Priority() != 0 {
			t.Errorf("Priority() = %v", p.Priority())
		}
	})

	t.Run("Compression", func(t *testing.T) {
		p := NewCompression(CompressionConfig{})
		if p.Name() != "compression" {
			t.Errorf("Name() = %v", p.Name())
		}
		if p.Phase() != PhasePostProxy {
			t.Errorf("Phase() = %v", p.Phase())
		}
	})

	t.Run("Timeout", func(t *testing.T) {
		p := NewTimeout(TimeoutConfig{Duration: time.Second})
		if p.Name() != "timeout" {
			t.Errorf("Name() = %v", p.Name())
		}
		if p.Phase() != PhaseProxy {
			t.Errorf("Phase() = %v", p.Phase())
		}
		if p.Priority() != 10 {
			t.Errorf("Priority() = %v", p.Priority())
		}
	})

	t.Run("BotDetect", func(t *testing.T) {
		p := NewBotDetect(BotDetectConfig{})
		if p.Name() != "bot-detect" {
			t.Errorf("Name() = %v", p.Name())
		}
		if p.Phase() != PhasePreAuth {
			t.Errorf("Phase() = %v", p.Phase())
		}
	})

	t.Run("CircuitBreaker", func(t *testing.T) {
		p := NewCircuitBreaker(CircuitBreakerConfig{})
		if p.Name() != "circuit-breaker" {
			t.Errorf("Name() = %v", p.Name())
		}
		if p.Phase() != PhaseProxy {
			t.Errorf("Phase() = %v", p.Phase())
		}
	})

	t.Run("CORS", func(t *testing.T) {
		p := NewCORS(CORSConfig{})
		if p.Name() != "cors" {
			t.Errorf("Name() = %v", p.Name())
		}
		if p.Phase() != PhasePreAuth {
			t.Errorf("Phase() = %v", p.Phase())
		}
	})

	t.Run("Redirect", func(t *testing.T) {
		p := NewRedirect(RedirectConfig{})
		if p.Name() != "redirect" {
			t.Errorf("Name() = %v", p.Name())
		}
		if p.Phase() != PhasePreProxy {
			t.Errorf("Phase() = %v", p.Phase())
		}
	})

	t.Run("RequestSizeLimit", func(t *testing.T) {
		p := NewRequestSizeLimit(RequestSizeLimitConfig{})
		if p.Name() != "request-size-limit" {
			t.Errorf("Name() = %v", p.Name())
		}
		if p.Phase() != PhasePreProxy {
			t.Errorf("Phase() = %v", p.Phase())
		}
	})

	t.Run("ResponseTransform", func(t *testing.T) {
		p := NewResponseTransform(ResponseTransformConfig{})
		if p.Name() != "response-transform" {
			t.Errorf("Name() = %v", p.Name())
		}
		if p.Phase() != PhasePostProxy {
			t.Errorf("Phase() = %v", p.Phase())
		}
	})

	t.Run("URLRewrite", func(t *testing.T) {
		p, _ := NewURLRewrite(URLRewriteConfig{})
		if p.Name() != "url-rewrite" {
			t.Errorf("Name() = %v", p.Name())
		}
		if p.Phase() != PhasePreProxy {
			t.Errorf("Phase() = %v", p.Phase())
		}
	})
}
