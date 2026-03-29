package jwt

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
)

// VerifyHS256 validates JWT signature using HMAC-SHA256.
func VerifyHS256(signingInput string, signature []byte, secret []byte) bool {
	if len(secret) == 0 {
		return false
	}
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(signingInput))
	expected := mac.Sum(nil)
	return subtle.ConstantTimeCompare(expected, signature) == 1
}
