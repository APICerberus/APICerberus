package jwt

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"math/big"
	"testing"
)

// testJWK returns a valid 2048-bit RSA JWK for testing.
func testJWK() JWK {
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic("failed to generate test RSA key: " + err.Error())
	}
	return JWK{
		Kty: "RSA",
		Kid: "test-key-1",
		Alg: "RS256",
		N:   base64.RawURLEncoding.EncodeToString(privKey.N.Bytes()),
		E:   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(privKey.E)).Bytes()),
	}
}

// TestParseRSAPublicKeyFromJWK tests RSA public key parsing from JWK for LOW-003 fix verification.
func TestParseRSAPublicKeyFromJWK(t *testing.T) {
	t.Run("valid RSA JWK", func(t *testing.T) {
		t.Parallel()
		jwk := testJWK()
		key, err := ParseRSAPublicKeyFromJWK(jwk)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if key.N.BitLen() < 2048 {
			t.Errorf("expected at least 2048-bit key, got %d bits", key.N.BitLen())
		}
		if key.E != 65537 {
			t.Errorf("expected exponent 65537, got %d", key.E)
		}
	})

	t.Run("rejects unsupported kty", func(t *testing.T) {
		t.Parallel()
		jwk := JWK{Kty: "EC", N: "abc", E: "AQAB"}
		_, err := ParseRSAPublicKeyFromJWK(jwk)
		if err == nil {
			t.Error("expected error for unsupported kty")
		}
	})

	t.Run("rejects missing n", func(t *testing.T) {
		t.Parallel()
		jwk := JWK{Kty: "RSA", E: "AQAB"}
		_, err := ParseRSAPublicKeyFromJWK(jwk)
		if err == nil {
			t.Error("expected error for missing n")
		}
	})

	t.Run("rejects missing e", func(t *testing.T) {
		t.Parallel()
		jwk := testJWK()
		jwk.E = ""
		_, err := ParseRSAPublicKeyFromJWK(jwk)
		if err == nil {
			t.Error("expected error for missing e")
		}
	})

	t.Run("rejects invalid base64 in n", func(t *testing.T) {
		t.Parallel()
		jwk := JWK{Kty: "RSA", N: "!!!invalid!!!", E: "AQAB"}
		_, err := ParseRSAPublicKeyFromJWK(jwk)
		if err == nil {
			t.Error("expected error for invalid base64 in n")
		}
	})

	t.Run("rejects empty modulus", func(t *testing.T) {
		t.Parallel()
		jwk := JWK{Kty: "RSA", N: "", E: "AQAB"}
		_, err := ParseRSAPublicKeyFromJWK(jwk)
		if err == nil {
			t.Error("expected error for empty modulus")
		}
	})

	t.Run("rejects invalid exponent", func(t *testing.T) {
		t.Parallel()
		jwk := testJWK()
		jwk.E = "AA==" // base64 for byte 0 (e=0)
		_, err := ParseRSAPublicKeyFromJWK(jwk)
		if err == nil {
			t.Error("expected error for invalid exponent")
		}
	})

	t.Run("rejects RSA key below 2048 bits", func(t *testing.T) {
		t.Parallel()
		// Use a 1024-bit modulus (big.NewInt(2)^1024)
		smallN := new(big.Int).Lsh(big.NewInt(1), 1024)
		jwk := JWK{
			Kty: "RSA",
			N:   base64.RawURLEncoding.EncodeToString(smallN.Bytes()),
			E:   "AQAB",
		}
		_, err := ParseRSAPublicKeyFromJWK(jwk)
		if err == nil {
			t.Error("expected error for key below 2048 bits")
		}
	})
}

// TestParseRSAPublicKeyFromJWK_EmptyKty tests that empty Kty accepts RSA.
func TestParseRSAPublicKeyFromJWK_EmptyKty(t *testing.T) {
	t.Parallel()
	jwk := testJWK()
	jwk.Kty = "" // empty Kty should default to RSA
	key, err := ParseRSAPublicKeyFromJWK(jwk)
	if err != nil {
		t.Fatalf("empty Kty should default to RSA: %v", err)
	}
	if key == nil {
		t.Error("expected valid key with empty Kty")
	}
}
