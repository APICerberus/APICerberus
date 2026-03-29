package plugin

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCorrelationIDGeneratesWhenMissing(t *testing.T) {
	t.Parallel()

	plugin := NewCorrelationID()
	req := httptest.NewRequest(http.MethodGet, "http://gateway.local/x", nil)
	rr := httptest.NewRecorder()
	ctx := &PipelineContext{
		Request:        req,
		ResponseWriter: rr,
	}
	plugin.Apply(ctx)

	id := req.Header.Get("X-Request-ID")
	if id == "" {
		t.Fatalf("expected generated request id")
	}
	if ctx.CorrelationID != id {
		t.Fatalf("expected context correlation id %q got %q", id, ctx.CorrelationID)
	}
	if rr.Header().Get("X-Request-ID") != id {
		t.Fatalf("expected response header request id to be set")
	}
}

func TestCorrelationIDPassesThroughExisting(t *testing.T) {
	t.Parallel()

	plugin := NewCorrelationID()
	req := httptest.NewRequest(http.MethodGet, "http://gateway.local/x", nil)
	req.Header.Set("X-Request-ID", "req-existing-123")
	rr := httptest.NewRecorder()
	ctx := &PipelineContext{
		Request:        req,
		ResponseWriter: rr,
	}
	plugin.Apply(ctx)

	if req.Header.Get("X-Request-ID") != "req-existing-123" {
		t.Fatalf("expected existing request id preserved")
	}
	if ctx.CorrelationID != "req-existing-123" {
		t.Fatalf("expected context to keep existing request id")
	}
	if rr.Header().Get("X-Request-ID") != "req-existing-123" {
		t.Fatalf("expected response header to keep existing request id")
	}
}
