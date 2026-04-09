package jwt

import "errors"

const minHS256SecretLength = 32 // minimum 256 bits for HS256 (NIST SP 800-117)

var ErrWeakHS256Secret = errors.New("weak HS256 secret")
