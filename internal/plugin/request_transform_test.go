package plugin

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/APICerberus/APICerebrus/internal/config"
)

func TestRequestTransformApplyMutations(t *testing.T) {
	t.Parallel()

	transform, err := NewRequestTransform(RequestTransformConfig{
		AddHeaders: map[string]string{
			"X-Added": "v-added",
		},
		RemoveHeaders: []string{"X-Remove"},
		RenameHeaders: map[string]string{
			"X-Old": "X-New",
		},
		AddQuery: map[string]string{
			"added": "yes",
		},
		RemoveQuery: []string{"drop"},
		RenameQuery: map[string]string{
			"keep": "renamed",
		},
		Method:          "post",
		PathPattern:     `^/original$`,
		PathReplacement: "/transformed/path",
	})
	if err != nil {
		t.Fatalf("NewRequestTransform error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "http://gateway.local/original?keep=1&drop=2", nil)
	req.Header.Set("X-Old", "old-value")
	req.Header.Set("X-Remove", "remove-value")

	ctx := &PipelineContext{Request: req}
	if err := transform.Apply(ctx); err != nil {
		t.Fatalf("Apply error: %v", err)
	}

	if got := ctx.Request.Method; got != http.MethodPost {
		t.Fatalf("expected method %s got %s", http.MethodPost, got)
	}
	if got := ctx.Request.URL.Path; got != "/transformed/path" {
		t.Fatalf("expected transformed path, got %q", got)
	}

	if got := ctx.Request.Header.Get("X-New"); got != "old-value" {
		t.Fatalf("expected renamed header value old-value, got %q", got)
	}
	if got := ctx.Request.Header.Get("X-Old"); got != "" {
		t.Fatalf("expected old header removed, got %q", got)
	}
	if got := ctx.Request.Header.Get("X-Remove"); got != "" {
		t.Fatalf("expected removed header to be empty, got %q", got)
	}
	if got := ctx.Request.Header.Get("X-Added"); got != "v-added" {
		t.Fatalf("expected added header value v-added, got %q", got)
	}

	query := ctx.Request.URL.Query()
	if got := query.Get("renamed"); got != "1" {
		t.Fatalf("expected renamed query value 1, got %q", got)
	}
	if got := query.Get("keep"); got != "" {
		t.Fatalf("expected original query key removed, got %q", got)
	}
	if got := query.Get("drop"); got != "" {
		t.Fatalf("expected removed query key absent, got %q", got)
	}
	if got := query.Get("added"); got != "yes" {
		t.Fatalf("expected added query value yes, got %q", got)
	}
}

func TestRequestTransformBodyHookPlaceholderNoop(t *testing.T) {
	t.Parallel()

	transform, err := NewRequestTransform(RequestTransformConfig{
		BodyHooks: map[string]any{
			"mode": "json_patch",
		},
	})
	if err != nil {
		t.Fatalf("NewRequestTransform error: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "http://gateway.local/body", bytes.NewBufferString(`{"a":1}`))
	ctx := &PipelineContext{Request: req}
	if err := transform.Apply(ctx); err != nil {
		t.Fatalf("Apply error: %v", err)
	}

	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		t.Fatalf("read request body: %v", err)
	}
	if got := string(body); got != `{"a":1}` {
		t.Fatalf("expected body to remain unchanged, got %q", got)
	}
}

func TestRequestTransformInvalidMethod(t *testing.T) {
	t.Parallel()

	_, err := NewRequestTransform(RequestTransformConfig{Method: "BAD METHOD"})
	if err == nil {
		t.Fatalf("expected invalid method error")
	}
}

func TestRequestTransformPathRegexCaptureGroups(t *testing.T) {
	t.Parallel()

	transform, err := NewRequestTransform(RequestTransformConfig{
		PathPattern:     `^/v1/users/([0-9]+)$`,
		PathReplacement: `/internal/u/$1`,
	})
	if err != nil {
		t.Fatalf("NewRequestTransform error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "http://gateway.local/v1/users/42", nil)
	ctx := &PipelineContext{Request: req}
	if err := transform.Apply(ctx); err != nil {
		t.Fatalf("Apply error: %v", err)
	}
	if got := ctx.Request.URL.Path; got != "/internal/u/42" {
		t.Fatalf("expected rewritten path /internal/u/42, got %q", got)
	}
}

func TestBuildRoutePipelinesRequestTransform(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Routes: []config.Route{
			{
				ID:      "route-transform",
				Name:    "route-transform",
				Service: "svc",
				Paths:   []string{"/a"},
				Methods: []string{http.MethodGet},
				Plugins: []config.PluginConfig{
					{
						Name: "request-transform",
						Config: map[string]any{
							"path_pattern":     "^/a$",
							"path_replacement": "/mutated",
							"method":           "POST",
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
	chain := pipelines["route-transform"]
	if len(chain) != 1 {
		t.Fatalf("expected 1 plugin in chain, got %d", len(chain))
	}
	if chain[0].Name() != "request-transform" {
		t.Fatalf("expected request-transform plugin, got %q", chain[0].Name())
	}

	req := httptest.NewRequest(http.MethodGet, "http://gateway.local/a", nil)
	pipelineCtx := &PipelineContext{
		Request:        req,
		ResponseWriter: httptest.NewRecorder(),
		Route:          &cfg.Routes[0],
	}
	handled, runErr := chain[0].Run(pipelineCtx)
	if runErr != nil {
		t.Fatalf("run error: %v", runErr)
	}
	if handled {
		t.Fatalf("request-transform should not fully handle request")
	}
	if got := pipelineCtx.Request.URL.Path; got != "/mutated" {
		t.Fatalf("expected mutated path, got %q", got)
	}
	if got := pipelineCtx.Request.Method; got != http.MethodPost {
		t.Fatalf("expected mutated method POST, got %q", got)
	}
}
