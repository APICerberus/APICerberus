package gateway

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/APICerberus/APICerebrus/internal/config"
)

func TestProxyForwardBasic(t *testing.T) {
	t.Parallel()

	type seenRequest struct {
		path           string
		host           string
		method         string
		customHeader   string
		removedHeader  string
		forwardedFor   string
		forwardedHost  string
		forwardedProto string
	}

	seenCh := make(chan seenRequest, 1)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenCh <- seenRequest{
			path:           r.URL.Path,
			host:           r.Host,
			method:         r.Method,
			customHeader:   r.Header.Get("X-Custom"),
			removedHeader:  r.Header.Get("X-Remove-Me"),
			forwardedFor:   r.Header.Get("X-Forwarded-For"),
			forwardedHost:  r.Header.Get("X-Forwarded-Host"),
			forwardedProto: r.Header.Get("X-Forwarded-Proto"),
		}

		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Keep-Alive", "timeout=5")
		w.Header().Set("X-Upstream", "ok")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("upstream-response"))
	}))
	defer upstream.Close()

	req := httptest.NewRequest(http.MethodPost, "http://gateway.local/api/resource?x=1", strings.NewReader("payload"))
	req.Header.Set("Connection", "keep-alive, X-Remove-Me")
	req.Header.Set("X-Remove-Me", "should-not-forward")
	req.Header.Set("X-Custom", "custom-value")
	req.RemoteAddr = "203.0.113.10:54000"

	rr := httptest.NewRecorder()
	p := NewProxy(config.PoolConfig{})
	err := p.Forward(&RequestContext{
		Request:        req,
		ResponseWriter: rr,
		Route:          &config.Route{},
	}, &config.UpstreamTarget{Address: upstream.URL})
	if err != nil {
		t.Fatalf("Forward error: %v", err)
	}

	seen := <-seenCh
	if seen.path != "/api/resource" {
		t.Fatalf("expected path /api/resource got %q", seen.path)
	}
	if seen.method != http.MethodPost {
		t.Fatalf("expected method POST got %q", seen.method)
	}
	if seen.customHeader != "custom-value" {
		t.Fatalf("expected X-Custom to forward")
	}
	if seen.removedHeader != "" {
		t.Fatalf("expected connection-token header to be removed, got %q", seen.removedHeader)
	}
	if !strings.Contains(seen.forwardedFor, "203.0.113.10") {
		t.Fatalf("expected X-Forwarded-For to contain client IP, got %q", seen.forwardedFor)
	}
	if seen.forwardedHost != "gateway.local" {
		t.Fatalf("expected X-Forwarded-Host gateway.local got %q", seen.forwardedHost)
	}
	if seen.forwardedProto != "http" {
		t.Fatalf("expected X-Forwarded-Proto http got %q", seen.forwardedProto)
	}

	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected status 202 got %d", rr.Code)
	}
	if rr.Body.String() != "upstream-response" {
		t.Fatalf("unexpected body: %s", rr.Body.String())
	}
	if rr.Header().Get("Connection") != "" || rr.Header().Get("Keep-Alive") != "" {
		t.Fatalf("hop-by-hop response headers should be stripped")
	}
	if rr.Header().Get("X-Upstream") != "ok" {
		t.Fatalf("expected upstream response header to be preserved")
	}
}

func TestProxyStripPath(t *testing.T) {
	t.Parallel()

	pathCh := make(chan string, 1)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pathCh <- r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	req := httptest.NewRequest(http.MethodGet, "http://gateway.local/api/v1/items", nil)
	rr := httptest.NewRecorder()

	p := NewProxy(config.PoolConfig{})
	err := p.Forward(&RequestContext{
		Request:        req,
		ResponseWriter: rr,
		Route: &config.Route{
			StripPath: true,
			Paths:     []string{"/api/*"},
		},
	}, &config.UpstreamTarget{Address: upstream.URL})
	if err != nil {
		t.Fatalf("Forward error: %v", err)
	}

	if got := <-pathCh; got != "/v1/items" {
		t.Fatalf("expected stripped upstream path /v1/items got %q", got)
	}
}

func TestProxyPreserveHostToggle(t *testing.T) {
	t.Parallel()

	hostCh := make(chan string, 2)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hostCh <- r.Host
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	upURL, _ := url.Parse(upstream.URL)
	upHost := upURL.Host

	p := NewProxy(config.PoolConfig{})

	reqPreserve := httptest.NewRequest(http.MethodGet, "http://client.example.com/check", nil)
	reqPreserve.Host = "client.example.com"
	rrPreserve := httptest.NewRecorder()
	if err := p.Forward(&RequestContext{
		Request:        reqPreserve,
		ResponseWriter: rrPreserve,
		Route: &config.Route{
			PreserveHost: true,
		},
	}, &config.UpstreamTarget{Address: upstream.URL}); err != nil {
		t.Fatalf("Forward preserve host error: %v", err)
	}

	reqOverride := httptest.NewRequest(http.MethodGet, "http://client.example.com/check", nil)
	reqOverride.Host = "client.example.com"
	rrOverride := httptest.NewRecorder()
	if err := p.Forward(&RequestContext{
		Request:        reqOverride,
		ResponseWriter: rrOverride,
		Route: &config.Route{
			PreserveHost: false,
		},
	}, &config.UpstreamTarget{Address: upstream.URL}); err != nil {
		t.Fatalf("Forward override host error: %v", err)
	}

	preservedHost := <-hostCh
	overriddenHost := <-hostCh
	if preservedHost != "client.example.com" {
		t.Fatalf("expected preserved host client.example.com got %q", preservedHost)
	}
	if overriddenHost != upHost {
		t.Fatalf("expected overridden host %q got %q", upHost, overriddenHost)
	}
}

func TestProxyError502(t *testing.T) {
	t.Parallel()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen error: %v", err)
	}
	addr := l.Addr().String()
	_ = l.Close()

	req := httptest.NewRequest(http.MethodGet, "http://gateway.local/fail", nil)
	rr := httptest.NewRecorder()

	p := NewProxy(config.PoolConfig{})
	err = p.Forward(&RequestContext{
		Request:        req,
		ResponseWriter: rr,
		Route:          &config.Route{},
	}, &config.UpstreamTarget{Address: addr})
	if err == nil {
		t.Fatalf("expected forwarding error")
	}
	if rr.Code != http.StatusBadGateway {
		t.Fatalf("expected 502 got %d", rr.Code)
	}
}

func TestProxyError504OnContextTimeout(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	baseReq := httptest.NewRequest(http.MethodGet, "http://gateway.local/slow", nil)
	ctx, cancel := context.WithTimeout(baseReq.Context(), 20*time.Millisecond)
	defer cancel()
	req := baseReq.WithContext(ctx)

	rr := httptest.NewRecorder()
	p := NewProxy(config.PoolConfig{})
	err := p.Forward(&RequestContext{
		Request:        req,
		ResponseWriter: rr,
		Route:          &config.Route{},
	}, &config.UpstreamTarget{Address: upstream.URL})
	if err == nil {
		t.Fatalf("expected timeout error")
	}
	if rr.Code != http.StatusGatewayTimeout {
		t.Fatalf("expected 504 got %d", rr.Code)
	}
}

func TestProxyResponseStreaming(t *testing.T) {
	t.Parallel()

	payload := strings.Repeat("x", 200*1024)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(payload))
	}))
	defer upstream.Close()

	req := httptest.NewRequest(http.MethodGet, "http://gateway.local/stream", nil)
	rr := httptest.NewRecorder()

	p := NewProxy(config.PoolConfig{})
	if err := p.Forward(&RequestContext{
		Request:        req,
		ResponseWriter: rr,
		Route:          &config.Route{},
	}, &config.UpstreamTarget{Address: upstream.URL}); err != nil {
		t.Fatalf("Forward error: %v", err)
	}

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", rr.Code)
	}
	if rr.Body.Len() != len(payload) {
		t.Fatalf("expected body size %d got %d", len(payload), rr.Body.Len())
	}
	if rr.Body.String() != payload {
		t.Fatalf("payload mismatch")
	}
}
