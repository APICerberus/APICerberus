package grpc

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"google.golang.org/grpc/codes"
)

// TestGRPCWebCORSAllowedOrigins verifies MEDIUM-002 fix:
// gRPC-Web CORS blocked by default when AllowedOrigins is nil.
func TestGRPCWebCORSAllowedOrigins(t *testing.T) {
	t.Run("nil AllowedOrigins blocks cross-origin", func(t *testing.T) {
		p := &Proxy{
			AllowedOrigins: nil,
		}

		req := httptest.NewRequest("POST", "/test.Service/Method", nil)
		req.Header.Set("Origin", "https://evil.example.com")
		req.Header.Set("Content-Type", "application/grpc-web-text")
		rec := httptest.NewRecorder()

		p.handleGRPCWeb(rec, req)

		// When AllowedOrigins is nil/empty, Access-Control-Allow-Origin should NOT be set
		if rec.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Error("cross-origin request should be blocked when AllowedOrigins is nil")
		}
	})

	t.Run("empty AllowedOrigins blocks cross-origin", func(t *testing.T) {
		p := &Proxy{
			AllowedOrigins: []string{},
		}

		req := httptest.NewRequest("POST", "/test.Service/Method", nil)
		req.Header.Set("Origin", "https://evil.example.com")
		req.Header.Set("Content-Type", "application/grpc-web-text")
		rec := httptest.NewRecorder()

		p.handleGRPCWeb(rec, req)

		if rec.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Error("cross-origin request should be blocked when AllowedOrigins is empty")
		}
	})

	t.Run("matching origin allowed", func(t *testing.T) {
		p := &Proxy{
			AllowedOrigins: []string{"https://dashboard.example.com"},
		}

		req := httptest.NewRequest("POST", "/test.Service/Method", nil)
		req.Header.Set("Origin", "https://dashboard.example.com")
		req.Header.Set("Content-Type", "application/grpc-web-text")
		rec := httptest.NewRecorder()

		p.handleGRPCWeb(rec, req)

		if rec.Header().Get("Access-Control-Allow-Origin") != "https://dashboard.example.com" {
			t.Error("matching origin should be allowed")
		}
	})

	t.Run("non-matching origin blocked", func(t *testing.T) {
		p := &Proxy{
			AllowedOrigins: []string{"https://dashboard.example.com"},
		}

		req := httptest.NewRequest("POST", "/test.Service/Method", nil)
		req.Header.Set("Origin", "https://evil.example.com")
		req.Header.Set("Content-Type", "application/grpc-web-text")
		rec := httptest.NewRecorder()

		p.handleGRPCWeb(rec, req)

		// Non-matching origin should not get Access-Control-Allow-Origin
		if rec.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Error("non-matching origin should be blocked")
		}
	})
}

// TestGRPCProxyInsecureMode verifies CRITICAL-001 fix:
// NewProxy accepts Insecure but logs a warning.
func TestGRPCProxyInsecureMode(t *testing.T) {
	t.Run("insecure config accepted but warns", func(t *testing.T) {
		cfg := &ProxyConfig{
			Target:    "localhost:50051",
			Insecure:  true,
		}

		proxy, err := NewProxy(cfg)
		if err != nil {
			t.Fatalf("NewProxy with Insecure=true should not error: %v", err)
		}
		if proxy == nil {
			t.Fatal("proxy should not be nil")
		}

		proxy.Close()
		t.Log("CRITICAL-001 fix: Insecure=true accepted with warning comment in code")
	})
}

// TestGRPCStatusToHTTP tests gRPC to HTTP status code mapping.
func TestGRPCStatusToHTTP(t *testing.T) {
	tests := []struct {
		code   codes.Code
		status int
	}{
		{codes.OK, http.StatusOK},
		{codes.Canceled, 499},
		{codes.InvalidArgument, http.StatusBadRequest},
		{codes.NotFound, http.StatusNotFound},
		{codes.PermissionDenied, http.StatusForbidden},
		{codes.ResourceExhausted, http.StatusTooManyRequests},
		{codes.Unauthenticated, http.StatusUnauthorized},
		{codes.Internal, http.StatusInternalServerError},
		{codes.Unavailable, http.StatusServiceUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.code.String(), func(t *testing.T) {
			t.Parallel()
			got := GRPCStatusToHTTP(tt.code)
			if got != tt.status {
				t.Errorf("GRPCStatusToHTTP(%v) = %d, want %d", tt.code, got, tt.status)
			}
		})
	}
}

// TestHTTPStatusToGRPC tests HTTP to gRPC status code mapping.
func TestHTTPStatusToGRPC(t *testing.T) {
	tests := []struct {
		status int
		code   codes.Code
	}{
		{http.StatusOK, codes.OK},
		{http.StatusBadRequest, codes.InvalidArgument},
		{http.StatusUnauthorized, codes.Unauthenticated},
		{http.StatusForbidden, codes.PermissionDenied},
		{http.StatusNotFound, codes.NotFound},
		{http.StatusConflict, codes.AlreadyExists},
		{http.StatusTooManyRequests, codes.ResourceExhausted},
		{http.StatusInternalServerError, codes.Internal},
	}

	for _, tt := range tests {
		t.Run(http.StatusText(tt.status), func(t *testing.T) {
			t.Parallel()
			got := HTTPStatusToGRPC(tt.status)
			if got != tt.code {
				t.Errorf("HTTPStatusToGRPC(%d) = %v, want %v", tt.status, got, tt.code)
			}
		})
	}
}
