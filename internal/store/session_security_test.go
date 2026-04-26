package store

import (
	"encoding/base64"
	"testing"
)

// TestGenerateSessionToken tests session token generation.
func TestGenerateSessionToken(t *testing.T) {
	t.Run("generates 43-character token", func(t *testing.T) {
		t.Parallel()
		token, err := GenerateSessionToken()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// base64.RawURLEncoding of 32 bytes = 43 characters (no padding)
		if len(token) != 43 {
			t.Errorf("token length = %d, want 43", len(token))
		}
	})

	t.Run("generates unique tokens", func(t *testing.T) {
		t.Parallel()
		token1, err := GenerateSessionToken()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		token2, err := GenerateSessionToken()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if token1 == token2 {
			t.Error("expected unique tokens on each call")
		}
	})

	t.Run("token is valid base64url", func(t *testing.T) {
		t.Parallel()
		token, err := GenerateSessionToken()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Should decode without error
		_, err = base64.RawURLEncoding.DecodeString(token)
		if err != nil {
			t.Errorf("token is not valid base64url: %v", err)
		}
	})
}

// TestHashSessionToken tests session token hashing.
func TestHashSessionToken(t *testing.T) {
	t.Run("generates hex hash", func(t *testing.T) {
		t.Parallel()
		hash := HashSessionToken("test-token")
		// SHA256 produces 64 hex characters
		if len(hash) != 64 {
			t.Errorf("hash length = %d, want 64", len(hash))
		}
	})

	t.Run("consistent hashing", func(t *testing.T) {
		t.Parallel()
		token := "test-token-123"
		hash1 := HashSessionToken(token)
		hash2 := HashSessionToken(token)
		if hash1 != hash2 {
			t.Error("same token should produce same hash")
		}
	})

	t.Run("different tokens produce different hashes", func(t *testing.T) {
		t.Parallel()
		hash1 := HashSessionToken("token-a")
		hash2 := HashSessionToken("token-b")
		if hash1 == hash2 {
			t.Error("different tokens should produce different hashes")
		}
	})

	t.Run("trims whitespace", func(t *testing.T) {
		t.Parallel()
		hash1 := HashSessionToken("token")
		hash2 := HashSessionToken("  token  ")
		if hash1 != hash2 {
			t.Error("token hashing should trim whitespace")
		}
	})
}
