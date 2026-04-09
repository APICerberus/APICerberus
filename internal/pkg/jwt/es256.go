package jwt

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	cryptorand "crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"math/big"
)

var ErrInvalidECKey = errors.New("invalid ec key")

// VerifyES256 validates JWT signature using ECDSA P-256 with SHA-256.
func VerifyES256(signingInput string, signature []byte, publicKey *ecdsa.PublicKey) bool {
	if publicKey == nil || len(signature) != 64 {
		return false
	}
	digest := sha256.Sum256([]byte(signingInput))
	r := new(big.Int).SetBytes(signature[:32])
	s := new(big.Int).SetBytes(signature[32:])
	return ecdsa.Verify(publicKey, digest[:], r, s)
}

// ParseECDSAPublicKeyFromJWK builds an ECDSA public key from JWK fields.
// Supports P-256 (ES256) only.
func ParseECDSAPublicKeyFromJWK(jwk JWK) (*ecdsa.PublicKey, error) {
	if jwk.Kty != "" && jwk.Kty != "EC" {
		return nil, fmt.Errorf("%w: unsupported kty %q", ErrInvalidECKey, jwk.Kty)
	}
	if jwk.Crv == "" {
		return nil, fmt.Errorf("%w: missing crv", ErrInvalidECKey)
	}
	curve := curveByName(jwk.Crv)
	if curve == nil {
		return nil, fmt.Errorf("%w: unsupported curve %q", ErrInvalidECKey, jwk.Crv)
	}
	if jwk.X == "" || jwk.Y == "" {
		return nil, fmt.Errorf("%w: missing x/y", ErrInvalidECKey)
	}

	xBytes, err := base64.RawURLEncoding.DecodeString(jwk.X)
	if err != nil {
		return nil, fmt.Errorf("%w: decode x: %v", ErrInvalidECKey, err)
	}
	yBytes, err := base64.RawURLEncoding.DecodeString(jwk.Y)
	if err != nil {
		return nil, fmt.Errorf("%w: decode y: %v", ErrInvalidECKey, err)
	}

	x := new(big.Int).SetBytes(xBytes)
	y := new(big.Int).SetBytes(yBytes)

	if !curve.IsOnCurve(x, y) {
		return nil, fmt.Errorf("%w: point not on curve", ErrInvalidECKey)
	}

	return &ecdsa.PublicKey{Curve: curve, X: x, Y: y}, nil
}

// curveByName maps JWK crv names to elliptic.Curve implementations.
func curveByName(name string) elliptic.Curve {
	switch name {
	case "P-256":
		return elliptic.P256()
	default:
		return nil
	}
}

// SignES256 creates a JWT signature for signingInput using ECDSA P-256.
// Returns the 64-byte concatenated r||s signature per RFC 7518.
func SignES256(signingInput string, privateKey *ecdsa.PrivateKey) ([]byte, error) {
	if privateKey == nil {
		return nil, ErrInvalidECKey
	}
	digest := sha256.Sum256([]byte(signingInput))
	r, s, err := ecdsa.Sign(cryptorand.Reader, privateKey, digest[:])
	if err != nil {
		return nil, err
	}
	curveBytes := privateKey.Params().BitSize / 8
	sig := make([]byte, 0, curveBytes*2)
	sig = append(sig, padToSize(r.Bytes(), curveBytes)...)
	sig = append(sig, padToSize(s.Bytes(), curveBytes)...)
	return sig, nil
}

func padToSize(buf []byte, size int) []byte {
	if len(buf) >= size {
		return buf[len(buf)-size:]
	}
	padded := make([]byte, size)
	copy(padded[size-len(buf):], buf)
	return padded
}

// VerifyEdDSA validates JWT signature using Ed25519 (EdDSA with PureEdDSA variant).
func VerifyEdDSA(signingInput string, signature []byte, publicKey any) bool {
	pk, ok := publicKey.(ed25519.PublicKey)
	if !ok {
		return false
	}
	return ed25519.Verify(pk, []byte(signingInput), signature)
}
