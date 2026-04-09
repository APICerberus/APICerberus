package jwt

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/base64"
	"errors"
	"fmt"
	"math/big"
)

var ErrInvalidECKey = errors.New("invalid ec key")

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
