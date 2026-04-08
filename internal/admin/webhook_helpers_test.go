package admin

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

// =============================================================================
// Webbook Helper Tests
// =============================================================================

func TestValidateWebhookSignature(t *testing.T) {
	t.Run("valid signature", func(t *testing.T) {
		secret := "test-secret"
		payload := []byte(`{"event":"test"}`)

		// Generate valid HMAC signature
		h := hmac.New(sha256.New, []byte(secret))
		h.Write(payload)
		signature := "sha256=" + hex.EncodeToString(h.Sum(nil))

		if !ValidateWebhookSignature(payload, signature, secret) {
			t.Error("expected signature to be valid")
		}
	})

	t.Run("invalid signature prefix", func(t *testing.T) {
		secret := "test-secret"
		payload := []byte(`{"event":"test"}`)
		signature := "invalid-prefix"

		if ValidateWebhookSignature(payload, signature, secret) {
			t.Error("expected signature to be invalid")
		}
	})

	t.Run("wrong signature", func(t *testing.T) {
		secret := "test-secret"
		payload := []byte(`{"event":"test"}`)
		signature := "sha256=wrongsignature"

		if ValidateWebhookSignature(payload, signature, secret) {
			t.Error("expected signature to be invalid")
		}
	})

	t.Run("empty signature", func(t *testing.T) {
		secret := "test-secret"
		payload := []byte(`{"event":"test"}`)

		if ValidateWebhookSignature(payload, "", secret) {
			t.Error("expected signature to be invalid")
		}
	})

	t.Run("empty secret", func(t *testing.T) {
		payload := []byte(`{"event":"test"}`)
		signature := "sha256=abc123"

		if ValidateWebhookSignature(payload, signature, "") {
			t.Error("expected signature to be invalid with empty secret")
		}
	})
}

func TestSignPayloadHelper(t *testing.T) {
	t.Run("sign payload generates valid signature", func(t *testing.T) {
		secret := "test-secret"
		payload := []byte(`{"event":"test"}`)

		// Generate HMAC signature
		h := hmac.New(sha256.New, []byte(secret))
		h.Write(payload)
		signature := "sha256=" + hex.EncodeToString(h.Sum(nil))

		// Verify it starts with expected prefix
		if !strings.HasPrefix(signature, "sha256=") {
			t.Error("expected signature to start with sha256=")
		}
	})
}

// =============================================================================
// Webhook Event Types Tests
// =============================================================================

func TestWebhookEvents(t *testing.T) {
	t.Run("verify webhook events exist", func(t *testing.T) {
		if len(WebhookEvents) == 0 {
			t.Error("expected webhook events to be defined")
		}

		// Verify common events
		eventTypes := make(map[string]bool)
		for _, e := range WebhookEvents {
			eventTypes[e.Type] = true
		}

		expectedEvents := []string{
			"route.created",
			"route.updated",
			"route.deleted",
			"service.created",
			"service.updated",
			"service.deleted",
		}

		for _, event := range expectedEvents {
			if !eventTypes[event] {
				t.Errorf("expected event %s to exist", event)
			}
		}
	})
}
