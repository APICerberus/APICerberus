package plugin

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBotDetectBlocksKnownBot(t *testing.T) {
	t.Parallel()

	plugin := NewBotDetect(BotDetectConfig{})
	req := httptest.NewRequest(http.MethodGet, "http://gateway.local/x", nil)
	req.Header.Set("User-Agent", "Googlebot/2.1")
	ctx := &PipelineContext{
		Request:        req,
		ResponseWriter: httptest.NewRecorder(),
	}

	err := plugin.Evaluate(ctx)
	if err == nil {
		t.Fatalf("expected bot to be blocked")
	}
	berr, ok := err.(*BotDetectError)
	if !ok {
		t.Fatalf("expected BotDetectError got %T", err)
	}
	if berr.Status != http.StatusForbidden {
		t.Fatalf("expected 403 got %d", berr.Status)
	}
}

func TestBotDetectAllowListBypassesDetection(t *testing.T) {
	t.Parallel()

	plugin := NewBotDetect(BotDetectConfig{
		AllowList: []string{"googlebot"},
		Action:    "block",
	})
	req := httptest.NewRequest(http.MethodGet, "http://gateway.local/x", nil)
	req.Header.Set("User-Agent", "Googlebot/2.1")
	ctx := &PipelineContext{
		Request:        req,
		ResponseWriter: httptest.NewRecorder(),
	}

	if err := plugin.Evaluate(ctx); err != nil {
		t.Fatalf("expected allow-list bot to pass, got %v", err)
	}
}

func TestBotDetectFlagAction(t *testing.T) {
	t.Parallel()

	plugin := NewBotDetect(BotDetectConfig{Action: "flag"})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://gateway.local/x", nil)
	req.Header.Set("User-Agent", "SomeCrawler/1.0")
	ctx := &PipelineContext{
		Request:        req,
		ResponseWriter: rr,
	}

	if err := plugin.Evaluate(ctx); err != nil {
		t.Fatalf("expected flag mode to not block, got %v", err)
	}
	if rr.Header().Get("X-Bot-Detected") != "true" {
		t.Fatalf("expected X-Bot-Detected header in flag mode")
	}
	if detected, ok := ctx.Metadata["bot_detected"].(bool); !ok || !detected {
		t.Fatalf("expected bot_detected metadata flag")
	}
}
