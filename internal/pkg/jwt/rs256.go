package jwt

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"math/big"
)

var ErrInvalidJWK = errors.New("invalid jwk")

// JWK represents one key inside a JWKS document.
type JWK struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
	Crv string `json:"crv,omitempty"`
	X   string `json:"x,omitempty"`
	Y   string `json:"y,omitempty"`
}

// JWKS is a JSON Web Key Set document.
type JWKS struct {
	Keys []JWK `json:"keys"`
}

// VerifyRS256 validates JWT signature using RSA-SHA256.
func VerifyRS256(signingInput string, signature []byte, publicKey *rsa.PublicKey) bool {
	if publicKey == nil {
		return false
	}
	digest := sha256.Sum256([]byte(signingInput))
	return rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, digest[:], signature) == nil
}

// ParseRSAPublicKeyFromJWK builds RSA public key from JWK n/e fields.
func ParseRSAPublicKeyFromJWK(jwk JWK) (*rsa.PublicKey, error) {
	if jwk.Kty != "" && jwk.Kty != "RSA" {
		return nil, fmt.Errorf("%w: unsupported kty %q", ErrInvalidJWK, jwk.Kty)
	}
	if jwk.N == "" || jwk.E == "" {
		return nil, fmt.Errorf("%w: missing n/e", ErrInvalidJWK)
	}

	nBytes, err := base64.RawURLEncoding.DecodeString(jwk.N)
	if err != nil {
		return nil, fmt.Errorf("%w: decode n: %v", ErrInvalidJWK, err)
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(jwk.E)
	if err != nil {
		return nil, fmt.Errorf("%w: decode e: %v", ErrInvalidJWK, err)
	}
	if len(nBytes) == 0 || len(eBytes) == 0 {
		return nil, fmt.Errorf("%w: empty modulus/exponent", ErrInvalidJWK)
	}

	modulus := new(big.Int).SetBytes(nBytes)
	exponent := new(big.Int).SetBytes(eBytes)
	if !exponent.IsInt64() {
		return nil, fmt.Errorf("%w: exponent too large", ErrInvalidJWK)
	}

	e := int(exponent.Int64())
	if e <= 1 {
		return nil, fmt.Errorf("%w: invalid exponent", ErrInvalidJWK)
	}

	return &rsa.PublicKey{
		N: modulus,
		E: e,
	}, nil
}
