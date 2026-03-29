package plugin

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestSizeLimitAllowsAndPreservesBody(t *testing.T) {
	t.Parallel()

	plugin := NewRequestSizeLimit(RequestSizeLimitConfig{MaxBytes: 8})
	req := httptest.NewRequest(http.MethodPost, "http://gateway.local/upload", bytes.NewBufferString("1234567"))
	ctx := &PipelineContext{Request: req}

	if err := plugin.Enforce(ctx); err != nil {
		t.Fatalf("Enforce error: %v", err)
	}
	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if string(body) != "1234567" {
		t.Fatalf("expected body to be preserved, got %q", string(body))
	}
}

func TestRequestSizeLimitRejectsByContentLength(t *testing.T) {
	t.Parallel()

	plugin := NewRequestSizeLimit(RequestSizeLimitConfig{MaxBytes: 5})
	req := httptest.NewRequest(http.MethodPost, "http://gateway.local/upload", bytes.NewBufferString("123456"))
	req.ContentLength = 6
	ctx := &PipelineContext{Request: req}

	err := plugin.Enforce(ctx)
	if err == nil {
		t.Fatalf("expected payload too large error")
	}
	rlErr, ok := err.(*RequestSizeLimitError)
	if !ok {
		t.Fatalf("expected RequestSizeLimitError got %T", err)
	}
	if rlErr.Status != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413 got %d", rlErr.Status)
	}
}

func TestRequestSizeLimitRejectsByReadSize(t *testing.T) {
	t.Parallel()

	plugin := NewRequestSizeLimit(RequestSizeLimitConfig{MaxBytes: 5})
	req := httptest.NewRequest(http.MethodPost, "http://gateway.local/upload", bytes.NewBufferString("123456"))
	req.ContentLength = -1
	ctx := &PipelineContext{Request: req}

	err := plugin.Enforce(ctx)
	if err == nil {
		t.Fatalf("expected payload too large error")
	}
	if _, ok := err.(*RequestSizeLimitError); !ok {
		t.Fatalf("expected RequestSizeLimitError got %T", err)
	}
}

func TestBuildRoutePipelinesRequestSizeLimit(t *testing.T) {
	t.Parallel()

	cfg := RequestSizeLimitConfig{MaxBytes: 10}
	plugin := NewRequestSizeLimit(cfg)
	if plugin.Name() != "request-size-limit" {
		t.Fatalf("unexpected name %q", plugin.Name())
	}
	if plugin.Phase() != PhasePreProxy {
		t.Fatalf("unexpected phase %q", plugin.Phase())
	}
	if plugin.Priority() != 25 {
		t.Fatalf("unexpected priority %d", plugin.Priority())
	}
}
