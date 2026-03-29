package plugin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/APICerberus/APICerebrus/internal/config"
)

func TestURLRewriteApplyCaptureGroupsAndQueryPreserved(t *testing.T) {
	t.Parallel()

	rewrite, err := NewURLRewrite(URLRewriteConfig{
		Pattern:     `^/api/v1/users/([0-9]+)$`,
		Replacement: `/internal/users/$1`,
	})
	if err != nil {
		t.Fatalf("NewURLRewrite error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "http://gateway.local/api/v1/users/42?sort=desc&limit=5", nil)
	ctx := &PipelineContext{Request: req}
	if err := rewrite.Apply(ctx); err != nil {
		t.Fatalf("Apply error: %v", err)
	}

	if got := ctx.Request.URL.Path; got != "/internal/users/42" {
		t.Fatalf("expected rewritten path /internal/users/42 got %q", got)
	}
	if got := ctx.Request.URL.RawQuery; got != "sort=desc&limit=5" && got != "limit=5&sort=desc" {
		t.Fatalf("expected query string preserved, got %q", got)
	}
}

func TestURLRewriteInvalidPattern(t *testing.T) {
	t.Parallel()

	_, err := NewURLRewrite(URLRewriteConfig{Pattern: "["})
	if err == nil {
		t.Fatalf("expected invalid regex error")
	}
}

func TestBuildRoutePipelinesURLRewrite(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Routes: []config.Route{
			{
				ID:      "route-url-rewrite",
				Name:    "route-url-rewrite",
				Service: "svc",
				Paths:   []string{"/api/v1/users/*"},
				Methods: []string{http.MethodGet},
				Plugins: []config.PluginConfig{
					{
						Name: "url-rewrite",
						Config: map[string]any{
							"pattern":     "^/api/v1/users/(.+)$",
							"replacement": "/users/$1",
						},
					},
				},
			},
		},
	}

	pipelines, _, err := BuildRoutePipelines(cfg, nil)
	if err != nil {
		t.Fatalf("BuildRoutePipelines error: %v", err)
	}
	chain := pipelines["route-url-rewrite"]
	if len(chain) != 1 {
		t.Fatalf("expected one plugin got %d", len(chain))
	}
	if chain[0].Name() != "url-rewrite" {
		t.Fatalf("expected url-rewrite plugin got %q", chain[0].Name())
	}
	if chain[0].Phase() != PhasePreProxy {
		t.Fatalf("expected pre-proxy phase got %q", chain[0].Phase())
	}

	req := httptest.NewRequest(http.MethodGet, "http://gateway.local/api/v1/users/abc", nil)
	ctx := &PipelineContext{
		Request:        req,
		ResponseWriter: httptest.NewRecorder(),
		Route:          &cfg.Routes[0],
	}
	handled, runErr := chain[0].Run(ctx)
	if runErr != nil {
		t.Fatalf("run error: %v", runErr)
	}
	if handled {
		t.Fatalf("url-rewrite should not fully handle response")
	}
	if got := ctx.Request.URL.Path; got != "/users/abc" {
		t.Fatalf("expected rewritten path /users/abc got %q", got)
	}
}
