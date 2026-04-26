package jwt

import (
	"crypto/rsa"
	"encoding/base64"
	"math/big"
	"testing"
)

// TestParseRSAPublicKeyFromJWK tests RSA public key parsing from JWK for LOW-003 fix verification.
func TestParseRSAPublicKeyFromJWK(t *testing.T) {
	t.Run("valid RSA JWK", func(t *testing.T) {
		t.Parallel()
		// 2048-bit RSA key n/e values (from a known test key)
		jwk := JWK{
			Kty: "RSA",
			Kid: "test-key-1",
			Alg: "RS256",
			N:   "0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbISD08qDuyr6ClmxGEOty82kuPE5CPb9eWQMbD_CZk0grQTY3DNt2wBcUlDmG7DrmqwHmacXPVbLrkRN_U6XAqA",
			E:   "AQAB",
		}
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
		jwk := JWK{Kty: "RSA", N: "0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbISD08qDuyr6ClmxGEOty82kuPE5CPb9eWQMbD_CZk0grQTY3DNt2wBcUlDmG7DrmqwHmacXPVbLrkRN_U6XAqA"}
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
		// Use valid n but e=0 which is invalid
		jwk := JWK{
			Kty: "RSA",
			N:   "0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbISD08qDuyr6ClmxGEOty82kuPE5CPb9eWQMbD_CZk0grQTY3DNt2wBcUlDmG7DrmqwHmacXPVbLrkRN_U6XAqA",
			E:   "AAEC", // base64 for 0
		}
		_, err := ParseRSAPublicKeyFromJWK(jwk)
		if err == nil {
			t.Error("expected error for invalid exponent")
		}
	})

	t.Run("rejects RSA key below 2048 bits", func(t *testing.T) {
		t.Parallel()
		// 1024-bit RSA key for testing
		jwk := JWK{
			Kty: "RSA",
			N:   base64.RawURLEncoding.EncodeToString(big.NewInt(0).Bytes()),
			E:   "AQAB",
		}
		// Use a small but valid-looking key that decodes correctly but is too small
		smallN := base64.RawURLEncoding.EncodeToString([]byte{0x01, 0x00, 0x01}) // 65537 as bytes
		jwk.N = smallN
		// This creates a small key that will fail the 2048-bit check
		// We need actual small RSA values
		bigN, _ := new(big.Int).SetString("10487441958381657719827032009606280474076115513813994883979294892099717856127", 10)
		jwk.N = base64.RawURLEncoding.EncodeToString(bigN.Bytes())
		_, err := ParseRSAPublicKeyFromJWK(jwk)
		if err == nil {
			t.Error("expected error for key below 2048 bits")
		}
	})
}

// TestParseRSAPublicKeyFromJWK_EmptyKty tests that empty Kty accepts RSA.
func TestParseRSAPublicKeyFromJWK_EmptyKty(t *testing.T) {
	t.Parallel()
	jwk := JWK{
		Kty: "", // empty Kty should default to RSA
		N:   "0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbISD08qDuyr6ClmxGEOty82kuPE5CPb9eWQMbD_CZk0grQTY3DNt2wBcUlDmG7DrmqwHmacXPVbLrkRN_U6XAqA",
		E:   "AQAB",
	}
	key, err := ParseRSAPublicKeyFromJWK(jwk)
	if err != nil {
		t.Fatalf("empty Kty should default to RSA: %v", err)
	}
	if key == nil {
		t.Error("expected valid key with empty Kty")
	}
}
