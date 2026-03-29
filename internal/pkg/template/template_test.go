package template

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestRenderVariables(t *testing.T) {
	t.Parallel()

	ctx := Context{
		Body:            `{"hello":"world"}`,
		Timestamp:       time.Unix(1700000000, 0).UTC(),
		UpstreamLatency: 125 * time.Millisecond,
		ConsumerID:      "consumer-1",
		RouteName:       "users-route",
		RequestID:       "req-abc",
		RemoteAddr:      "203.0.113.10",
		Headers: http.Header{
			"X-Custom": []string{"custom-value"},
		},
	}

	input := "$body|$timestamp_ms|$timestamp_iso|$upstream_latency_ms|$consumer_id|$route_name|$request_id|$remote_addr|$header.X-Custom"
	got := Render(input, ctx)

	expected := `{"hello":"world"}|1700000000000|2023-11-14T22:13:20Z|125|consumer-1|users-route|req-abc|203.0.113.10|custom-value`
	if got != expected {
		t.Fatalf("unexpected render output:\nwant: %s\ngot:  %s", expected, got)
	}
}

func TestApplyJSONPatchAddRemoveRenameNested(t *testing.T) {
	t.Parallel()

	body := []byte(`{"a":1,"nested":{"b":2,"c":3}}`)
	out, err := ApplyJSONPatch(body, JSONPatch{
		Add: map[string]any{
			"nested.new":      "x",
			"metadata.source": "gateway",
		},
		Remove: []string{"nested.c"},
		Rename: map[string]string{
			"a":        "a_renamed",
			"nested.b": "nested.renamed",
		},
	})
	if err != nil {
		t.Fatalf("ApplyJSONPatch error: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}

	if _, exists := got["a"]; exists {
		t.Fatalf("expected key a to be renamed")
	}
	if got["a_renamed"].(float64) != 1 {
		t.Fatalf("expected a_renamed=1")
	}

	nested := got["nested"].(map[string]any)
	if _, exists := nested["b"]; exists {
		t.Fatalf("expected nested.b to be renamed")
	}
	if nested["renamed"].(float64) != 2 {
		t.Fatalf("expected nested.renamed=2")
	}
	if _, exists := nested["c"]; exists {
		t.Fatalf("expected nested.c removed")
	}
	if nested["new"].(string) != "x" {
		t.Fatalf("expected nested.new=x")
	}

	meta := got["metadata"].(map[string]any)
	if meta["source"].(string) != "gateway" {
		t.Fatalf("expected metadata.source=gateway")
	}
}

func TestApplyJSONPatchNestedPathCreation(t *testing.T) {
	t.Parallel()

	out, err := ApplyJSONPatch([]byte(`{}`), JSONPatch{
		Add: map[string]any{
			"deep.inner.value": 42,
		},
	})
	if err != nil {
		t.Fatalf("ApplyJSONPatch error: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}

	deep := got["deep"].(map[string]any)
	inner := deep["inner"].(map[string]any)
	if inner["value"].(float64) != 42 {
		t.Fatalf("expected deep.inner.value=42")
	}
}

func TestTransformTemplateModeFullReplacement(t *testing.T) {
	t.Parallel()

	opts := TransformOptions{
		Mode:     ModeTemplate,
		Template: "wrapped=$body consumer=$consumer_id header=$header.X-Test",
	}
	ctx := Context{
		ConsumerID: "consumer-z",
		Headers: http.Header{
			"X-Test": []string{"ok"},
		},
	}
	out, err := Transform([]byte(`{"k":"v"}`), opts, ctx)
	if err != nil {
		t.Fatalf("Transform error: %v", err)
	}
	if got := string(out); got != `wrapped={"k":"v"} consumer=consumer-z header=ok` {
		t.Fatalf("unexpected template transform output: %q", got)
	}
}

func TestTransformUnsupportedMode(t *testing.T) {
	t.Parallel()

	_, err := Transform([]byte(`{}`), TransformOptions{Mode: Mode("unknown")}, Context{})
	if err == nil || !strings.Contains(err.Error(), "unsupported transform mode") {
		t.Fatalf("expected unsupported mode error, got %v", err)
	}
}
