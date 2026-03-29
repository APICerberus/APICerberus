package jwt

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

var ErrJWKSKeyNotFound = errors.New("jwks key not found")

// JWKSClient fetches JWKS documents and caches parsed RSA keys.
type JWKSClient struct {
	url        string
	ttl        time.Duration
	httpClient *http.Client

	now func() time.Time

	mu       sync.RWMutex
	keysByID map[string]*rsaKeyRef
	keys     []*rsaKeyRef
	fetched  time.Time
}

type rsaKeyRef struct {
	kid string
	key *rsa.PublicKey
}

// NewJWKSClient creates a JWKS client with cache TTL (defaults to 1h).
func NewJWKSClient(url string, ttl time.Duration) *JWKSClient {
	if ttl <= 0 {
		ttl = time.Hour
	}
	return &JWKSClient{
		url:        strings.TrimSpace(url),
		ttl:        ttl,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		now:        time.Now,
		keysByID:   make(map[string]*rsaKeyRef),
	}
}

// GetRSAKey resolves an RSA public key by kid.
func (c *JWKSClient) GetRSAKey(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	if c == nil {
		return nil, errors.New("jwks client is nil")
	}
	kid = strings.TrimSpace(kid)
	if !c.isFresh() {
		if err := c.refresh(ctx); err != nil {
			// Attempt stale fallback.
			if key, ok := c.lookup(kid); ok {
				return key, nil
			}
			return nil, err
		}
	}
	if key, ok := c.lookup(kid); ok {
		return key, nil
	}
	return nil, ErrJWKSKeyNotFound
}

func (c *JWKSClient) isFresh() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.fetched.IsZero() {
		return false
	}
	return c.now().Sub(c.fetched) < c.ttl
}

func (c *JWKSClient) lookup(kid string) (*rsa.PublicKey, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if kid != "" {
		ref, ok := c.keysByID[kid]
		if !ok || ref == nil || ref.key == nil {
			return nil, false
		}
		return ref.key, true
	}
	if len(c.keys) == 1 && c.keys[0] != nil && c.keys[0].key != nil {
		return c.keys[0].key, true
	}
	return nil, false
}

func (c *JWKSClient) refresh(ctx context.Context) error {
	if strings.TrimSpace(c.url) == "" {
		return errors.New("jwks url is empty")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url, nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("jwks request failed: status %d", resp.StatusCode)
	}

	var doc JWKS
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return err
	}

	keysByID := make(map[string]*rsaKeyRef)
	keys := make([]*rsaKeyRef, 0, len(doc.Keys))
	for _, jwk := range doc.Keys {
		pub, err := ParseRSAPublicKeyFromJWK(jwk)
		if err != nil {
			continue
		}
		ref := &rsaKeyRef{
			kid: strings.TrimSpace(jwk.Kid),
			key: pub,
		}
		keys = append(keys, ref)
		if ref.kid != "" {
			keysByID[ref.kid] = ref
		}
	}
	if len(keys) == 0 {
		return errors.New("jwks has no usable rsa keys")
	}

	c.mu.Lock()
	c.keysByID = keysByID
	c.keys = keys
	c.fetched = c.now()
	c.mu.Unlock()
	return nil
}
