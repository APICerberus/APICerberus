package audit

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCaptureRequestBodyRestoresBody(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPost, "http://gateway.local/test", bytes.NewBufferString("hello-world"))
	captured, err := CaptureRequestBody(req, 5)
	if err != nil {
		t.Fatalf("CaptureRequestBody error: %v", err)
	}
	if string(captured) != "hello" {
		t.Fatalf("unexpected captured body: %q", string(captured))
	}

	replayed, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("read restored body: %v", err)
	}
	if string(replayed) != "hello-world" {
		t.Fatalf("unexpected restored body: %q", string(replayed))
	}

	if req.GetBody == nil {
		t.Fatalf("expected GetBody to be set")
	}
	rc, err := req.GetBody()
	if err != nil {
		t.Fatalf("GetBody error: %v", err)
	}
	defer rc.Close()
	second, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("read GetBody: %v", err)
	}
	if string(second) != "hello-world" {
		t.Fatalf("unexpected GetBody content: %q", string(second))
	}
}

func TestResponseCaptureWriterCapturesStatusAndBody(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()
	capture := NewResponseCaptureWriter(rr, 4)
	capture.Header().Set("X-Test", "ok")
	capture.WriteHeader(http.StatusAccepted)
	_, _ = capture.Write([]byte("abcdef"))

	if rr.Code != http.StatusAccepted {
		t.Fatalf("unexpected recorder code: %d", rr.Code)
	}
	if rr.Body.String() != "abcdef" {
		t.Fatalf("unexpected recorder body: %q", rr.Body.String())
	}
	if capture.StatusCode() != http.StatusAccepted {
		t.Fatalf("unexpected capture status: %d", capture.StatusCode())
	}
	if capture.BytesWritten() != int64(len("abcdef")) {
		t.Fatalf("unexpected bytes written: %d", capture.BytesWritten())
	}
	if string(capture.BodyBytes()) != "abcd" {
		t.Fatalf("unexpected captured body: %q", string(capture.BodyBytes()))
	}
}
