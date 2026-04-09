package plugin

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAuthJWTValidHS256(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_000_000, 0).UTC()
	secret := []byte("top-secret")
	token := buildHS256JWT(t,
		map[string]any{"alg": "HS256", "typ": "JWT"},
		map[string]any{
			"sub":  "consumer-1",
			"iss":  "apicerberus",
			"aud":  "public-api",
			"role": "gold",
			"exp":  now.Add(5 * time.Minute).Unix(),
		},
		secret,
	)

	auth := NewAuthJWT(AuthJWTOptions{
		Secret:         string(secret),
		Issuer:         "apicerberus",
		Audience:       []string{"public-api"},
		RequiredClaims: []string{"sub", "role"},
		ClaimsToHeaders: map[string]string{
			"sub": "X-Consumer-ID",
		},
		ClockSkew: 10 * time.Second,
	})
	auth.now = func() time.Time { return now }

	req := httptest.NewRequest(http.MethodGet, "http://gateway.local/users", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	claims, err := auth.Authenticate(req)
	if err != nil {
		t.Fatalf("Authenticate error: %v", err)
	}
	if claims["sub"] != "consumer-1" {
		t.Fatalf("unexpected claims: %#v", claims)
	}
	if req.Header.Get("X-Consumer-ID") != "consumer-1" {
		t.Fatalf("expected mapped header")
	}
}

func TestAuthJWTValidRS256FromJWKS(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey error: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"keys": []map[string]any{
				{
					"kty": "RSA",
					"kid": "kid-1",
					"n":   base64.RawURLEncoding.EncodeToString(privateKey.PublicKey.N.Bytes()),
					"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(privateKey.PublicKey.E)).Bytes()),
				},
			},
		})
	}))
	defer server.Close()

	now := time.Unix(1_700_000_000, 0).UTC()
	token := buildRS256JWT(t,
		map[string]any{"alg": "RS256", "kid": "kid-1"},
		map[string]any{
			"sub": "consumer-rsa",
			"exp": now.Add(5 * time.Minute).Unix(),
		},
		privateKey,
	)

	auth := NewAuthJWT(AuthJWTOptions{
		JWKSURL: server.URL,
	})
	auth.now = func() time.Time { return now }

	req := httptest.NewRequest(http.MethodGet, "http://gateway.local/users", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	claims, err := auth.Authenticate(req)
	if err != nil {
		t.Fatalf("Authenticate error: %v", err)
	}
	if claims["sub"] != "consumer-rsa" {
		t.Fatalf("unexpected claims: %#v", claims)
	}
}

func TestAuthJWTExpired(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_000_000, 0).UTC()
	secret := []byte("top-secret")
	token := buildHS256JWT(t,
		map[string]any{"alg": "HS256"},
		map[string]any{
			"sub": "consumer-1",
			"exp": now.Add(-time.Minute).Unix(),
		},
		secret,
	)

	auth := NewAuthJWT(AuthJWTOptions{
		Secret:    string(secret),
		ClockSkew: 0,
	})
	auth.now = func() time.Time { return now }

	req := httptest.NewRequest(http.MethodGet, "http://gateway.local/users", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	_, err := auth.Authenticate(req)
	if err != ErrExpiredJWT {
		t.Fatalf("expected ErrExpiredJWT got %v", err)
	}
}

func TestAuthJWTWrongIssuer(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_000_000, 0).UTC()
	secret := []byte("top-secret")
	token := buildHS256JWT(t,
		map[string]any{"alg": "HS256"},
		map[string]any{
			"sub": "consumer-1",
			"iss": "issuer-a",
			"exp": now.Add(5 * time.Minute).Unix(),
		},
		secret,
	)

	auth := NewAuthJWT(AuthJWTOptions{
		Secret: string(secret),
		Issuer: "issuer-b",
	})
	auth.now = func() time.Time { return now }

	req := httptest.NewRequest(http.MethodGet, "http://gateway.local/users", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	_, err := auth.Authenticate(req)
	assertJWTErrorCode(t, err, "invalid_jwt_claims")
}

func TestAuthJWTWrongAudience(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_000_000, 0).UTC()
	secret := []byte("top-secret")
	token := buildHS256JWT(t,
		map[string]any{"alg": "HS256"},
		map[string]any{
			"sub": "consumer-1",
			"aud": []string{"private-api"},
			"exp": now.Add(5 * time.Minute).Unix(),
		},
		secret,
	)

	auth := NewAuthJWT(AuthJWTOptions{
		Secret:   string(secret),
		Audience: []string{"public-api"},
	})
	auth.now = func() time.Time { return now }

	req := httptest.NewRequest(http.MethodGet, "http://gateway.local/users", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	_, err := auth.Authenticate(req)
	assertJWTErrorCode(t, err, "invalid_jwt_claims")
}

func TestAuthJWTMissingRequiredClaim(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_000_000, 0).UTC()
	secret := []byte("top-secret")
	token := buildHS256JWT(t,
		map[string]any{"alg": "HS256"},
		map[string]any{
			"sub": "consumer-1",
			"exp": now.Add(5 * time.Minute).Unix(),
		},
		secret,
	)

	auth := NewAuthJWT(AuthJWTOptions{
		Secret:         string(secret),
		RequiredClaims: []string{"sub", "tenant_id"},
	})
	auth.now = func() time.Time { return now }

	req := httptest.NewRequest(http.MethodGet, "http://gateway.local/users", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	_, err := auth.Authenticate(req)
	assertJWTErrorCode(t, err, "invalid_jwt_claims")
}

func assertJWTErrorCode(t *testing.T, err error, expectedCode string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error")
	}
	authErr, ok := err.(*JWTAuthError)
	if !ok {
		t.Fatalf("expected *JWTAuthError got %T", err)
	}
	if authErr.Code != expectedCode {
		t.Fatalf("expected error code %q got %q", expectedCode, authErr.Code)
	}
}

func TestAuthJWTNoneAlgorithmRejected(t *testing.T) {
	t.Parallel()

	// Build a JWT with "alg": "none" - this should be explicitly rejected
	// even though it's technically unsupported, to prevent algorithm confusion attacks
	now := time.Unix(1_700_000_000, 0).UTC()
	header := map[string]any{"alg": "none", "typ": "JWT"}
	payload := map[string]any{
		"sub": "attacker",
		"exp": now.Add(5 * time.Minute).Unix(),
	}
	headerSeg := mustJSONSegment(t, header)
	payloadSeg := mustJSONSegment(t, payload)
	// For "none" algorithm, signature should be empty
	token := headerSeg + "." + payloadSeg + "."

	auth := NewAuthJWT(AuthJWTOptions{
		Secret: "any-secret",
	})
	auth.now = func() time.Time { return now }

	req := httptest.NewRequest(http.MethodGet, "http://gateway.local/users", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	_, err := auth.Authenticate(req)
	assertJWTErrorCode(t, err, "unsupported_jwt_algorithm")
}

func TestAuthJWTNoneAlgorithmUpperCaseRejected(t *testing.T) {
	t.Parallel()

	// Test that "NONE" (uppercase) is also rejected
	now := time.Unix(1_700_000_000, 0).UTC()
	header := map[string]any{"alg": "NONE", "typ": "JWT"}
	payload := map[string]any{
		"sub": "attacker",
		"exp": now.Add(5 * time.Minute).Unix(),
	}
	headerSeg := mustJSONSegment(t, header)
	payloadSeg := mustJSONSegment(t, payload)
	token := headerSeg + "." + payloadSeg + "."

	auth := NewAuthJWT(AuthJWTOptions{
		Secret: "any-secret",
	})
	auth.now = func() time.Time { return now }

	req := httptest.NewRequest(http.MethodGet, "http://gateway.local/users", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	_, err := auth.Authenticate(req)
	assertJWTErrorCode(t, err, "unsupported_jwt_algorithm")
}

func buildHS256JWT(t *testing.T, header, payload map[string]any, secret []byte) string {
	t.Helper()
	headerSeg := mustJSONSegment(t, header)
	payloadSeg := mustJSONSegment(t, payload)
	signingInput := headerSeg + "." + payloadSeg
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(signingInput))
	sig := mac.Sum(nil)
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(sig)
}

func buildRS256JWT(t *testing.T, header, payload map[string]any, privateKey *rsa.PrivateKey) string {
	t.Helper()
	headerSeg := mustJSONSegment(t, header)
	payloadSeg := mustJSONSegment(t, payload)
	signingInput := headerSeg + "." + payloadSeg
	hash := sha256.Sum256([]byte(signingInput))
	sig, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hash[:])
	if err != nil {
		t.Fatalf("rsa.SignPKCS1v15 error: %v", err)
	}
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(sig)
}

func mustJSONSegment(t *testing.T, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal error: %v", err)
	}
	return base64.RawURLEncoding.EncodeToString(data)
}

// --- nbf validation tests ---

func TestAuthJWTValidNBF(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_000_000, 0).UTC()
	secret := []byte("top-secret")
	token := buildHS256JWT(t,
		map[string]any{"alg": "HS256"},
		map[string]any{
			"sub": "consumer-1",
			"exp": now.Add(5 * time.Minute).Unix(),
			"nbf": now.Add(-time.Minute).Unix(),
		},
		secret,
	)

	auth := NewAuthJWT(AuthJWTOptions{Secret: string(secret)})
	auth.now = func() time.Time { return now }

	req := httptest.NewRequest(http.MethodGet, "http://gateway.local/users", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	claims, err := auth.Authenticate(req)
	if err != nil {
		t.Fatalf("Authenticate error: %v", err)
	}
	if claims["sub"] != "consumer-1" {
		t.Fatalf("unexpected claims: %#v", claims)
	}
}

func TestAuthJWTNBFNotYetValid(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_000_000, 0).UTC()
	secret := []byte("top-secret")
	token := buildHS256JWT(t,
		map[string]any{"alg": "HS256"},
		map[string]any{
			"sub": "consumer-1",
			"exp": now.Add(10 * time.Minute).Unix(),
			"nbf": now.Add(5 * time.Minute).Unix(),
		},
		secret,
	)

	auth := NewAuthJWT(AuthJWTOptions{Secret: string(secret), ClockSkew: 10 * time.Second})
	auth.now = func() time.Time { return now }

	req := httptest.NewRequest(http.MethodGet, "http://gateway.local/users", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	_, err := auth.Authenticate(req)
	assertJWTErrorCode(t, err, "invalid_jwt_claims")
}

func TestAuthJWTNBFWithinClockSkew(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_000_000, 0).UTC()
	secret := []byte("top-secret")
	token := buildHS256JWT(t,
		map[string]any{"alg": "HS256"},
		map[string]any{
			"sub": "consumer-1",
			"exp": now.Add(10 * time.Minute).Unix(),
			"nbf": now.Add(5 * time.Second).Unix(),
		},
		secret,
	)

	auth := NewAuthJWT(AuthJWTOptions{Secret: string(secret), ClockSkew: 30 * time.Second})
	auth.now = func() time.Time { return now }

	req := httptest.NewRequest(http.MethodGet, "http://gateway.local/users", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	claims, err := auth.Authenticate(req)
	if err != nil {
		t.Fatalf("Authenticate error: %v", err)
	}
	if claims["sub"] != "consumer-1" {
		t.Fatalf("unexpected claims: %#v", claims)
	}
}

// --- ES256 tests ---

func TestAuthJWTValidES256(t *testing.T) {
	t.Parallel()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("ecdsa.GenerateKey error: %v", err)
	}

	now := time.Unix(1_700_000_000, 0).UTC()
	token := buildES256JWT(t,
		map[string]any{"alg": "ES256", "kid": "ec-key-1"},
		map[string]any{
			"sub": "consumer-es256",
			"exp": now.Add(5 * time.Minute).Unix(),
		},
		key,
	)

	auth := NewAuthJWT(AuthJWTOptions{ECDSAPublicKey: &key.PublicKey})
	auth.now = func() time.Time { return now }

	req := httptest.NewRequest(http.MethodGet, "http://gateway.local/users", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	claims, err := auth.Authenticate(req)
	if err != nil {
		t.Fatalf("Authenticate error: %v", err)
	}
	if claims["sub"] != "consumer-es256" {
		t.Fatalf("unexpected claims: %#v", claims)
	}
}

func TestAuthJWTES256InvalidSignature(t *testing.T) {
	t.Parallel()

	key1, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	key2, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)

	now := time.Unix(1_700_000_000, 0).UTC()
	token := buildES256JWT(t,
		map[string]any{"alg": "ES256"},
		map[string]any{
			"sub": "consumer-es256",
			"exp": now.Add(5 * time.Minute).Unix(),
		},
		key1,
	)

	auth := NewAuthJWT(AuthJWTOptions{ECDSAPublicKey: &key2.PublicKey})
	auth.now = func() time.Time { return now }

	req := httptest.NewRequest(http.MethodGet, "http://gateway.local/users", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	_, err := auth.Authenticate(req)
	if err != ErrInvalidJWTSignature {
		t.Fatalf("expected ErrInvalidJWTSignature got %v", err)
	}
}

// --- EdDSA tests ---

func TestAuthJWTValidEdDSA(t *testing.T) {
	t.Parallel()

	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("ed25519.GenerateKey error: %v", err)
	}

	now := time.Unix(1_700_000_000, 0).UTC()
	token := buildEdDSAJWT(t,
		map[string]any{"alg": "EdDSA"},
		map[string]any{
			"sub": "consumer-eddsa",
			"exp": now.Add(5 * time.Minute).Unix(),
		},
		privateKey,
	)

	auth := NewAuthJWT(AuthJWTOptions{EdDSAPublicKey: publicKey})
	auth.now = func() time.Time { return now }

	req := httptest.NewRequest(http.MethodGet, "http://gateway.local/users", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	claims, err := auth.Authenticate(req)
	if err != nil {
		t.Fatalf("Authenticate error: %v", err)
	}
	if claims["sub"] != "consumer-eddsa" {
		t.Fatalf("unexpected claims: %#v", claims)
	}
}

func TestAuthJWTEdDSANoKeyConfigured(t *testing.T) {
	t.Parallel()

	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("ed25519.GenerateKey error: %v", err)
	}

	now := time.Unix(1_700_000_000, 0).UTC()
	token := buildEdDSAJWT(t,
		map[string]any{"alg": "EdDSA"},
		map[string]any{
			"sub": "consumer-eddsa",
			"exp": now.Add(5 * time.Minute).Unix(),
		},
		privateKey,
	)

	auth := NewAuthJWT(AuthJWTOptions{Secret: "unused"})
	auth.now = func() time.Time { return now }

	req := httptest.NewRequest(http.MethodGet, "http://gateway.local/users", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	_, err = auth.Authenticate(req)
	assertJWTErrorCode(t, err, "unsupported_jwt_algorithm")
}

// --- JTI Replay Cache tests ---

func TestJTIReplayCachePreventsReplay(t *testing.T) {
	t.Parallel()

	cache := NewJTIReplayCache()
	jti := "unique-jti-123"

	// First use should succeed (not seen)
	if cache.Seen(jti) {
		t.Fatal("expected jti to not be seen on first check")
	}

	// Register the jti
	cache.Add(jti, 5*time.Minute)

	// Second use should be detected as replay
	if !cache.Seen(jti) {
		t.Fatal("expected jti to be seen after registration")
	}
}

func TestJTIReplayCacheExpiresEntries(t *testing.T) {
	t.Parallel()

	cache := &JTIReplayCache{
		entries: make(map[string]time.Time),
		now:     func() time.Time { return time.Unix(1_700_000_000, 0) },
	}

	jti := "expiring-jti"
	cache.Add(jti, 1*time.Minute)

	if !cache.Seen(jti) {
		t.Fatal("expected jti to be seen before expiry")
	}

	// Advance time past expiry
	cache.now = func() time.Time { return time.Unix(1_700_000_000+120, 0) }

	if cache.Seen(jti) {
		t.Fatal("expected jti to not be seen after expiry")
	}
}

func TestAuthJWTWithJTIReplayPreventsReuse(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_000_000, 0).UTC()
	secret := []byte("top-secret")
	jti := "replay-test-jti"
	token := buildHS256JWT(t,
		map[string]any{"alg": "HS256"},
		map[string]any{
			"sub": "consumer-1",
			"exp": now.Add(5 * time.Minute).Unix(),
			"jti": jti,
		},
		secret,
	)

	cache := NewJTIReplayCache()
	auth := NewAuthJWT(AuthJWTOptions{
		Secret:         string(secret),
		JTIReplayCache: cache,
	})
	auth.now = func() time.Time { return now }

	// First request should succeed
	req1 := httptest.NewRequest(http.MethodGet, "http://gateway.local/users", nil)
	req1.Header.Set("Authorization", "Bearer "+token)
	claims, err := auth.Authenticate(req1)
	if err != nil {
		t.Fatalf("first Authenticate error: %v", err)
	}
	if claims["sub"] != "consumer-1" {
		t.Fatalf("unexpected claims: %#v", claims)
	}

	// Second request with same token should fail
	req2 := httptest.NewRequest(http.MethodGet, "http://gateway.local/users", nil)
	req2.Header.Set("Authorization", "Bearer "+token)
	_, err = auth.Authenticate(req2)
	if err == nil {
		t.Fatal("expected replay error on second request")
	}
	authErr, ok := err.(*JWTAuthError)
	if !ok || authErr.Code != "jti_replay" {
		t.Fatalf("expected jti_replay error got %v", err)
	}
}

// --- JWT builder helpers ---

func buildES256JWT(t *testing.T, header, payload map[string]any, key *ecdsa.PrivateKey) string {
	t.Helper()
	headerSeg := mustJSONSegment(t, header)
	payloadSeg := mustJSONSegment(t, payload)
	signingInput := headerSeg + "." + payloadSeg

	hash := sha256.Sum256([]byte(signingInput))
	r, s, err := ecdsa.Sign(rand.Reader, key, hash[:])
	if err != nil {
		t.Fatalf("ecdsa.Sign error: %v", err)
	}
	curveBytes := key.Params().BitSize / 8
	sig := make([]byte, 0, curveBytes*2)
	sig = append(sig, padBigInt(r, curveBytes)...)
	sig = append(sig, padBigInt(s, curveBytes)...)
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(sig)
}

func buildEdDSAJWT(t *testing.T, header, payload map[string]any, key ed25519.PrivateKey) string {
	t.Helper()
	headerSeg := mustJSONSegment(t, header)
	payloadSeg := mustJSONSegment(t, payload)
	signingInput := headerSeg + "." + payloadSeg

	sig := ed25519.Sign(key, []byte(signingInput))
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(sig)
}

func padBigInt(n *big.Int, size int) []byte {
	buf := n.Bytes()
	if len(buf) >= size {
		return buf[len(buf)-size:]
	}
	padded := make([]byte, size)
	copy(padded[size-len(buf):], buf)
	return padded
}
