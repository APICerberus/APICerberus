package plugin

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRedirectHandleConfiguredPath(t *testing.T) {
	t.Parallel()

	plugin := NewRedirect(RedirectConfig{
		Rules: []RedirectRule{
			{Path: "/old", TargetURL: "https://example.com/new", StatusCode: http.StatusMovedPermanently},
		},
	})

	req := httptest.NewRequest(http.MethodGet, "http://gateway.local/old?x=1", nil)
	rr := httptest.NewRecorder()
	handled := plugin.Handle(rr, req)
	if !handled {
		t.Fatalf("expected redirect to handle request")
	}
	if rr.Code != http.StatusMovedPermanently {
		t.Fatalf("expected 301 got %d", rr.Code)
	}
	if rr.Header().Get("Location") != "https://example.com/new" {
		t.Fatalf("unexpected location header %q (query params intentionally not forwarded for security)", rr.Header().Get("Location"))
	}
}

func TestRedirectHandleSkipsNonMatchingPath(t *testing.T) {
	t.Parallel()

	plugin := NewRedirect(RedirectConfig{
		Rules: []RedirectRule{
			{Path: "/old", TargetURL: "https://example.com/new", StatusCode: http.StatusMovedPermanently},
		},
	})

	req := httptest.NewRequest(http.MethodGet, "http://gateway.local/other", nil)
	rr := httptest.NewRecorder()
	handled := plugin.Handle(rr, req)
	if handled {
		t.Fatalf("expected non-matching path not to be handled")
	}
}

func TestBuildRoutePipelinesRedirectPlugin(t *testing.T) {
	t.Parallel()

	cfg := RedirectConfig{
		Rules: []RedirectRule{
			{Path: "/from", TargetURL: "https://example.com/to", StatusCode: http.StatusTemporaryRedirect},
		},
	}
	plugin := NewRedirect(cfg)
	if plugin.Name() != "redirect" {
		t.Fatalf("unexpected plugin name %q", plugin.Name())
	}
	if plugin.Phase() != PhasePreProxy {
		t.Fatalf("unexpected plugin phase %q", plugin.Phase())
	}
}
