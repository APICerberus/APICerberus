package graphql

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

// Proxy proxies GraphQL requests to upstream servers.
type Proxy struct {
	target     *url.URL
	client     *http.Client
	reverseProxy *httputil.ReverseProxy
}

// ProxyConfig configures the GraphQL proxy.
type ProxyConfig struct {
	TargetURL      string
	Timeout        time.Duration
}

// NewProxy creates a new GraphQL proxy.
func NewProxy(cfg *ProxyConfig) (*Proxy, error) {
	target, err := url.Parse(cfg.TargetURL)
	if err != nil {
		return nil, err
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	client := &http.Client{
		Timeout: timeout,
	}

	reverseProxy := httputil.NewSingleHostReverseProxy(target)
	reverseProxy.Transport = client.Transport

	return &Proxy{
		target:       target,
		client:       client,
		reverseProxy: reverseProxy,
	}, nil
}

// ServeHTTP implements http.Handler.
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.reverseProxy.ServeHTTP(w, r)
}

// Forward forwards a GraphQL request to the upstream.
func (p *Proxy) Forward(req *Request) (*Response, error) {
	// Build request body
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	// Create HTTP request
	httpReq, err := http.NewRequest("POST", p.target.String(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// Execute request
	httpResp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	// Parse response
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, err
	}

	var resp Response
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// IntrospectionChecker checks if a request is an introspection query.
type IntrospectionChecker struct {
	allowed bool // if false, introspection is blocked
}

// NewIntrospectionChecker creates a new introspection checker.
func NewIntrospectionChecker(allowed bool) *IntrospectionChecker {
	return &IntrospectionChecker{allowed: allowed}
}

// Check checks if a query is allowed.
func (c *IntrospectionChecker) Check(query string) bool {
	if c.allowed {
		return true
	}
	return !IsIntrospectionQuery(query)
}
