package jwt

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
)

const minHS256SecretLength = 32 // minimum 256 bits for HS256 (NIST SP 800-117)

var ErrWeakHS256Secret = errors.New("weak HS256 secret")

// EncodeSegment encodes bytes to base64url without padding.
func EncodeSegment(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

// SignHS256 creates a JWT signature for signingInput using HMAC-SHA256.
func SignHS256(signingInput string, secret []byte) ([]byte, error) {
	if len(secret) == 0 {
		return nil, fmt.Errorf("%w: secret is empty", ErrWeakHS256Secret)
	}
	if len(secret) < minHS256SecretLength {
		return nil, fmt.Errorf("%w: secret length %d is below minimum %d bytes", ErrWeakHS256Secret, len(secret), minHS256SecretLength)
	}
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(signingInput))
	return mac.Sum(nil), nil
}

// VerifyHS256 validates JWT signature using HMAC-SHA256.
func VerifyHS256(signingInput string, signature []byte, secret []byte) bool {
	if len(secret) < minHS256SecretLength {
		return false
	}
	expected, err := SignHS256(signingInput, secret)
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare(expected, signature) == 1
}
