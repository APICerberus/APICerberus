package jwt

import (
	"context"
	"crypto"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	t.Parallel()

	raw := buildHS256Token(t, map[string]any{"alg": "HS256", "typ": "JWT"}, map[string]any{"sub": "u1"}, []byte("s3cr3t-long-enough-for-hs256!!"))
	token, err := Parse(raw)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	alg, ok := token.HeaderString("alg")
	if !ok || alg != "HS256" {
		t.Fatalf("unexpected alg: %q", alg)
	}
	sub, ok := token.ClaimString("sub")
	if !ok || sub != "u1" {
		t.Fatalf("unexpected sub: %q", sub)
	}
}

func TestVerifyHS256(t *testing.T) {
	t.Parallel()

	secret := []byte("test-secret-long-enough-for-hs256-min!!")
	raw := buildHS256Token(t, map[string]any{"alg": "HS256"}, map[string]any{"sub": "u1"}, secret)
	token, err := Parse(raw)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if !VerifyHS256(token.SigningInput, token.Signature, secret) {
		t.Fatalf("expected HS256 verification to succeed")
	}
	if VerifyHS256(token.SigningInput, token.Signature, []byte("wrong-secret-long-enough-for-min!!")) {
		t.Fatalf("expected HS256 verification to fail with wrong key")
	}
}

func TestVerifyRS256AndJWK(t *testing.T) {
	t.Parallel()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey error: %v", err)
	}
	header := map[string]any{"alg": "RS256", "kid": "k1"}
	payload := map[string]any{"sub": "u1"}
	raw := buildRS256Token(t, header, payload, privateKey)
	token, err := Parse(raw)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if !VerifyRS256(token.SigningInput, token.Signature, &privateKey.PublicKey) {
		t.Fatalf("expected RS256 verification to succeed")
	}

	jwk := JWK{
		Kty: "RSA",
		Kid: "k1",
		N:   base64.RawURLEncoding.EncodeToString(privateKey.PublicKey.N.Bytes()),
		E:   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(privateKey.PublicKey.E)).Bytes()),
	}
	pub, err := ParseRSAPublicKeyFromJWK(jwk)
	if err != nil {
		t.Fatalf("ParseRSAPublicKeyFromJWK error: %v", err)
	}
	if !VerifyRS256(token.SigningInput, token.Signature, pub) {
		t.Fatalf("expected RS256 verification with JWK key to succeed")
	}
}

func TestJWKSClientCaching(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey error: %v", err)
	}
	var hits int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&hits, 1)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"keys": []map[string]any{
				{
					"kty": "RSA",
					"kid": "k1",
					"n":   base64.RawURLEncoding.EncodeToString(privateKey.PublicKey.N.Bytes()),
					"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(privateKey.PublicKey.E)).Bytes()),
				},
			},
		})
	}))
	defer server.Close()

	client := NewJWKSClient(server.URL, time.Minute)
	pub, err := client.GetRSAKey(context.Background(), "k1")
	if err != nil {
		t.Fatalf("GetRSAKey first call error: %v", err)
	}
	if pub == nil {
		t.Fatalf("expected rsa public key")
	}
	pub2, err := client.GetRSAKey(context.Background(), "k1")
	if err != nil {
		t.Fatalf("GetRSAKey second call error: %v", err)
	}
	if pub2 == nil {
		t.Fatalf("expected rsa public key")
	}
	if atomic.LoadInt64(&hits) != 1 {
		t.Fatalf("expected single JWKS fetch, got %d", atomic.LoadInt64(&hits))
	}
}

func buildHS256Token(t *testing.T, header, payload map[string]any, secret []byte) string {
	t.Helper()
	headerSeg := mustJSONSegment(t, header)
	payloadSeg := mustJSONSegment(t, payload)
	signingInput := headerSeg + "." + payloadSeg

	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(signingInput))
	signature := mac.Sum(nil)
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(signature)
}

func buildRS256Token(t *testing.T, header, payload map[string]any, privateKey *rsa.PrivateKey) string {
	t.Helper()
	headerSeg := mustJSONSegment(t, header)
	payloadSeg := mustJSONSegment(t, payload)
	signingInput := headerSeg + "." + payloadSeg
	hash := sha256.Sum256([]byte(signingInput))
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hash[:])
	if err != nil {
		t.Fatalf("rsa.SignPKCS1v15 error: %v", err)
	}
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(signature)
}

func mustJSONSegment(t *testing.T, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal error: %v", err)
	}
	return base64.RawURLEncoding.EncodeToString(data)
}
