package jsonutil

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWriteJSON(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()
	err := WriteJSON(rr, http.StatusCreated, map[string]string{"ok": "true"})
	if err != nil {
		t.Fatalf("WriteJSON error: %v", err)
	}

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status 201 got %d", rr.Code)
	}
	if got := rr.Header().Get("Content-Type"); !strings.Contains(got, "application/json") {
		t.Fatalf("unexpected content-type: %q", got)
	}
	if body := rr.Body.String(); !strings.Contains(body, `"ok":"true"`) {
		t.Fatalf("unexpected body: %s", body)
	}
}

func TestReadJSON(t *testing.T) {
	t.Parallel()

	type payload struct {
		Name string `json:"name"`
	}

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"cerberus"}`))
	var got payload
	if err := ReadJSON(req, &got, 1024); err != nil {
		t.Fatalf("ReadJSON error: %v", err)
	}
	if got.Name != "cerberus" {
		t.Fatalf("unexpected payload: %#v", got)
	}
}

func TestReadJSONUnknownField(t *testing.T) {
	t.Parallel()

	type payload struct {
		Name string `json:"name"`
	}

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"cerberus","extra":"x"}`))
	var got payload
	if err := ReadJSON(req, &got, 1024); err == nil {
		t.Fatalf("expected unknown field error")
	}
}

func TestReadJSON_DefaultMaxBytes(t *testing.T) {
	t.Parallel()

	type payload struct {
		Name string `json:"name"`
	}

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"cerberus"}`))
	var got payload
	// Pass 0 or negative maxBytes to use default
	if err := ReadJSON(req, &got, 0); err != nil {
		t.Fatalf("ReadJSON with default maxBytes error: %v", err)
	}
	if got.Name != "cerberus" {
		t.Fatalf("unexpected payload: %#v", got)
	}

	// Test negative maxBytes
	req2 := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"test2"}`))
	var got2 payload
	if err := ReadJSON(req2, &got2, -1); err != nil {
		t.Fatalf("ReadJSON with negative maxBytes error: %v", err)
	}
	if got2.Name != "test2" {
		t.Fatalf("unexpected payload: %#v", got2)
	}
}

func TestMarshalJSON(t *testing.T) {
	t.Parallel()

	type payload struct {
		Name string `json:"name"`
	}

	got := MarshalJSON(payload{Name: "cerberus"})
	if got != `{"name":"cerberus"}` {
		t.Fatalf("unexpected marshaled payload: %s", got)
	}

	if MarshalJSON(map[string]any{"bad": make(chan int)}) != "{}" {
		t.Fatalf("marshal failure fallback should be {}")
	}
}
