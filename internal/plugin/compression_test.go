package plugin

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCompressionGzipAppliedWhenAcceptedAndAboveMinSize(t *testing.T) {
	t.Parallel()

	out := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://gateway.local/resource", nil)
	req.Header.Set("Accept-Encoding", "gzip, deflate")

	compression := NewCompression(CompressionConfig{MinSize: 5})
	ctx := &PipelineContext{
		Request:        req,
		ResponseWriter: out,
	}
	compression.Apply(ctx)
	ctx.ResponseWriter.Header().Set("Content-Type", "application/json")
	ctx.ResponseWriter.WriteHeader(http.StatusOK)
	_, _ = ctx.ResponseWriter.Write([]byte(`{"hello":"world"}`))
	compression.AfterProxy(ctx, nil)

	capture := ctx.ResponseWriter.(*CaptureResponseWriter)
	if err := capture.Flush(); err != nil {
		t.Fatalf("flush error: %v", err)
	}

	if out.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("expected gzip content encoding")
	}
	if out.Header().Get("Vary") == "" {
		t.Fatalf("expected Vary header")
	}
	body, err := gunzipBytes(out.Body.Bytes())
	if err != nil {
		t.Fatalf("gunzip response body: %v", err)
	}
	if string(body) != `{"hello":"world"}` {
		t.Fatalf("unexpected decompressed body %q", string(body))
	}
}

func TestCompressionSkipsWhenBelowThreshold(t *testing.T) {
	t.Parallel()

	out := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://gateway.local/resource", nil)
	req.Header.Set("Accept-Encoding", "gzip")

	compression := NewCompression(CompressionConfig{MinSize: 100})
	ctx := &PipelineContext{
		Request:        req,
		ResponseWriter: out,
	}
	compression.Apply(ctx)
	ctx.ResponseWriter.WriteHeader(http.StatusOK)
	_, _ = ctx.ResponseWriter.Write([]byte("tiny"))
	compression.AfterProxy(ctx, nil)

	capture := ctx.ResponseWriter.(*CaptureResponseWriter)
	if err := capture.Flush(); err != nil {
		t.Fatalf("flush error: %v", err)
	}

	if out.Header().Get("Content-Encoding") != "" {
		t.Fatalf("did not expect content encoding for tiny body")
	}
	if out.Body.String() != "tiny" {
		t.Fatalf("expected uncompressed body, got %q", out.Body.String())
	}
}

func TestCompressionSkipsWhenClientDoesNotAcceptGzip(t *testing.T) {
	t.Parallel()

	out := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://gateway.local/resource", nil)

	compression := NewCompression(CompressionConfig{MinSize: 1})
	ctx := &PipelineContext{
		Request:        req,
		ResponseWriter: out,
	}
	compression.Apply(ctx)
	ctx.ResponseWriter.WriteHeader(http.StatusOK)
	_, _ = ctx.ResponseWriter.Write([]byte("plain"))
	compression.AfterProxy(ctx, nil)

	capture := ctx.ResponseWriter.(*CaptureResponseWriter)
	if err := capture.Flush(); err != nil {
		t.Fatalf("flush error: %v", err)
	}

	if out.Header().Get("Content-Encoding") != "" {
		t.Fatalf("expected no compression without Accept-Encoding gzip")
	}
}
