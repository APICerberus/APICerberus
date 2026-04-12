package test

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/APICerberus/APICerebrus/internal/config"
	"github.com/APICerberus/APICerebrus/internal/gateway"
)

func BenchmarkE2EAllFeatures10K(b *testing.B) {
	fast := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("bench-ok"))
	}))
	defer fast.Close()

	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			time.Sleep(5 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("bench-ok"))
	}))
	defer slow.Close()

	gwAddr := freeAddrB(b)
	cfg := &config.Config{
		Gateway: config.GatewayConfig{
			HTTPAddr:       gwAddr,
			ReadTimeout:    2 * time.Second,
			WriteTimeout:   2 * time.Second,
			IdleTimeout:    10 * time.Second,
			MaxHeaderBytes: 1 << 20,
			MaxBodyBytes:   1 << 20,
		},
		Services: []config.Service{
			{ID: "svc-bench", Name: "svc-bench", Protocol: "http", Upstream: "up-bench"},
		},
		Routes: []config.Route{
			{
				ID:      "route-bench",
				Name:    "route-bench",
				Service: "svc-bench",
				Paths:   []string{"/v003/bench"},
				Methods: []string{http.MethodGet, http.MethodOptions},
			},
		},
		Upstreams: []config.Upstream{
			{
				ID:        "up-bench",
				Name:      "up-bench",
				Algorithm: "least_latency",
				Targets: []config.UpstreamTarget{
					{ID: "t-fast", Address: mustHostB(b, fast.URL), Weight: 1},
					{ID: "t-slow", Address: mustHostB(b, slow.URL), Weight: 1},
				},
				HealthCheck: config.HealthCheckConfig{
					Active: config.ActiveHealthCheckConfig{
						Path:               "/health",
						Interval:           200 * time.Millisecond,
						Timeout:            100 * time.Millisecond,
						HealthyThreshold:   1,
						UnhealthyThreshold: 1,
					},
				},
			},
		},
		Consumers: []config.Consumer{
			{
				ID:   "bench-consumer",
				Name: "bench-consumer",
				APIKeys: []config.ConsumerAPIKey{
					{ID: "k-bench", Key: "ck_bench_v003"},
				},
			},
		},
		GlobalPlugins: []config.PluginConfig{
			{
				Name: "cors",
				Config: map[string]any{
					"allowed_origins": []any{"*"},
					"allowed_methods": []any{"GET", "OPTIONS"},
				},
			},
			{Name: "auth-apikey"},
			{
				Name: "rate-limit",
				Config: map[string]any{
					"algorithm": "fixed_window",
					"scope":     "consumer",
					"limit":     1000000,
					"window":    "1h",
				},
			},
			{
				Name: "retry",
				Config: map[string]any{
					"max_retries":   1,
					"base_delay":    "1ms",
					"max_delay":     "2ms",
					"jitter":        false,
					"retry_methods": []any{"GET"},
				},
			},
			{
				Name: "timeout",
				Config: map[string]any{
					"timeout": "1s",
				},
			},
			{
				Name: "circuit-breaker",
				Config: map[string]any{
					"error_threshold":    1.0,
					"volume_threshold":   10000,
					"sleep_window":       "1s",
					"half_open_requests": 1,
					"window":             "10s",
				},
			},
		},
	}

	gw, err := gateway.New(cfg)
	if err != nil {
		b.Fatalf("gateway.New error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errCh := make(chan error, 1)
	go func() { errCh <- gw.Start(ctx) }()

	waitForGatewayListenerB(b, gwAddr)

	b.ReportAllocs()
	b.ResetTimer()
	start := time.Now()
	b.RunParallel(func(pb *testing.PB) {
		rw := newBenchResponseWriter()
		for pb.Next() {
			rw.Reset()
			req := httptest.NewRequest(http.MethodGet, "http://gateway.local/v003/bench", nil)
			req.RemoteAddr = "198.51.100.42:54321"
			req.Header.Set("X-API-Key", "ck_bench_v003")
			req.Header.Set("Origin", "https://bench.local")
			gw.ServeHTTP(rw, req)
			if rw.status != http.StatusOK {
				b.Fatalf("unexpected status=%d", rw.status)
			}
		}
	})
	elapsed := time.Since(start)
	if elapsed > 0 {
		b.ReportMetric(float64(b.N)/elapsed.Seconds(), "req/s")
	}
	b.StopTimer()

	cancel()
	if err := <-errCh; err != nil {
		b.Fatalf("gateway runtime error: %v", err)
	}
}

func freeAddrB(b *testing.B) string {
	b.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		b.Fatalf("listen: %v", err)
	}
	addr := l.Addr().String()
	_ = l.Close()
	return addr
}

func waitForGatewayListenerB(b *testing.B, addr string) {
	b.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	b.Fatalf("gateway did not start listening on %s", addr)
}

func mustHostB(b *testing.B, rawURL string) string {
	b.Helper()
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		b.Fatalf("parse url %q: %v", rawURL, err)
	}
	return req.URL.Host
}

type benchResponseWriter struct {
	header http.Header
	status int
}

func newBenchResponseWriter() *benchResponseWriter {
	return &benchResponseWriter{header: make(http.Header)}
}

func (w *benchResponseWriter) Header() http.Header {
	return w.header
}

func (w *benchResponseWriter) Write(p []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return len(p), nil
}

func (w *benchResponseWriter) WriteHeader(statusCode int) {
	w.status = statusCode
}

func (w *benchResponseWriter) Reset() {
	for key := range w.header {
		delete(w.header, key)
	}
	w.status = 0
}
