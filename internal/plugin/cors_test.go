package plugin

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORSPreflight(t *testing.T) {
	t.Parallel()

	cors := NewCORS(CORSConfig{
		AllowedOrigins: []string{"https://app.example.com"},
		AllowedMethods: []string{"GET", "POST"},
		AllowedHeaders: []string{"Authorization", "Content-Type"},
		MaxAge:         600,
	})

	req := httptest.NewRequest(http.MethodOptions, "http://gateway.local/users", nil)
	req.Header.Set("Origin", "https://app.example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rr := httptest.NewRecorder()

	handled := cors.Handle(rr, req)
	if !handled {
		t.Fatalf("preflight should be handled")
	}
	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204 got %d", rr.Code)
	}
	if rr.Header().Get("Access-Control-Allow-Origin") != "https://app.example.com" {
		t.Fatalf("unexpected allow origin header")
	}
	if rr.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Fatalf("missing Access-Control-Allow-Methods")
	}
	if rr.Header().Get("Access-Control-Allow-Headers") == "" {
		t.Fatalf("missing Access-Control-Allow-Headers")
	}
}

func TestCORSActualRequest(t *testing.T) {
	t.Parallel()

	cors := NewCORS(CORSConfig{
		AllowedOrigins: []string{"*"},
	})

	req := httptest.NewRequest(http.MethodGet, "http://gateway.local/users", nil)
	req.Header.Set("Origin", "https://other.example.com")
	rr := httptest.NewRecorder()

	handled := cors.Handle(rr, req)
	if handled {
		t.Fatalf("actual request should continue in pipeline")
	}
	if rr.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("expected wildcard allow origin")
	}
}

func TestCORSDisallowedOrigin(t *testing.T) {
	t.Parallel()

	cors := NewCORS(CORSConfig{
		AllowedOrigins: []string{"https://app.example.com"},
	})

	req := httptest.NewRequest(http.MethodGet, "http://gateway.local/users", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	rr := httptest.NewRecorder()

	handled := cors.Handle(rr, req)
	if !handled {
		t.Fatalf("disallowed origin should be handled with rejection")
	}
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 got %d", rr.Code)
	}
}
