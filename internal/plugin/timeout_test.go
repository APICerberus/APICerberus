package plugin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestTimeoutApply(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "http://gateway.local/x", nil)
	pctx := &PipelineContext{Request: req}
	timeout := NewTimeout(TimeoutConfig{Duration: 80 * time.Millisecond})
	timeout.Apply(pctx)

	deadline, ok := pctx.Request.Context().Deadline()
	if !ok {
		t.Fatalf("expected context deadline to be set")
	}
	if time.Until(deadline) > 120*time.Millisecond {
		t.Fatalf("unexpected deadline far in future: %v", deadline)
	}
	if len(pctx.Cleanup) != 1 {
		t.Fatalf("expected one cleanup callback")
	}
}

func TestTimeoutContextExpires(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "http://gateway.local/x", nil)
	pctx := &PipelineContext{Request: req}
	timeout := NewTimeout(TimeoutConfig{Duration: 40 * time.Millisecond})
	timeout.Apply(pctx)
	defer pctx.Cleanup[0]()

	select {
	case <-pctx.Request.Context().Done():
		t.Fatalf("context expired too early")
	case <-time.After(20 * time.Millisecond):
	}

	select {
	case <-pctx.Request.Context().Done():
		if pctx.Request.Context().Err() != context.DeadlineExceeded {
			t.Fatalf("expected deadline exceeded, got %v", pctx.Request.Context().Err())
		}
	case <-time.After(120 * time.Millisecond):
		t.Fatalf("expected timeout to expire")
	}
}
