package audit

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestMaskerMaskHeaders(t *testing.T) {
	t.Parallel()

	masker := NewMasker([]string{"Authorization", "X-API-Key"}, nil, "***")
	headers := http.Header{}
	headers.Set("Authorization", "Bearer abc")
	headers.Set("X-API-Key", "key-123")
	headers.Set("X-Trace", "trace-1")

	masked := masker.MaskHeaders(headers)
	if masked["Authorization"] != "***" {
		t.Fatalf("authorization not masked: %#v", masked["Authorization"])
	}
	apiKeyValue := masked["X-API-Key"]
	if apiKeyValue == nil {
		apiKeyValue = masked["X-Api-Key"]
	}
	if apiKeyValue != "***" {
		t.Fatalf("x-api-key not masked: %#v", apiKeyValue)
	}
	if masked["X-Trace"] != "trace-1" {
		t.Fatalf("unexpected trace header: %#v", masked["X-Trace"])
	}
}

func TestMaskerMaskBodyNestedFields(t *testing.T) {
	t.Parallel()

	masker := NewMasker(nil, []string{"password", "user.token", "items.secret"}, "REDACTED")
	raw := []byte(`{"password":"abc","user":{"token":"t-1"},"items":[{"secret":"s-1"},{"secret":"s-2"}]}`)

	maskedRaw := masker.MaskBody(raw)
	var payload map[string]any
	if err := json.Unmarshal(maskedRaw, &payload); err != nil {
		t.Fatalf("unmarshal masked body: %v", err)
	}
	if payload["password"] != "REDACTED" {
		t.Fatalf("password field not masked: %#v", payload["password"])
	}
	user := payload["user"].(map[string]any)
	if user["token"] != "REDACTED" {
		t.Fatalf("nested token not masked: %#v", user["token"])
	}
	items := payload["items"].([]any)
	for _, item := range items {
		entry := item.(map[string]any)
		if entry["secret"] != "REDACTED" {
			t.Fatalf("array secret not masked: %#v", entry["secret"])
		}
	}
}
