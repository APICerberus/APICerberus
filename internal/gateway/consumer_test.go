package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/APICerberus/APICerebrus/internal/config"
)

func TestGatewayConsumerResolutionByAPIKeyHeader(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer upstream.Close()

	cfg := &config.Config{
		Gateway: config.GatewayConfig{
			HTTPAddr:       "127.0.0.1:0",
			ReadTimeout:    2 * time.Second,
			WriteTimeout:   2 * time.Second,
			IdleTimeout:    10 * time.Second,
			MaxHeaderBytes: 1 << 20,
			MaxBodyBytes:   1 << 20,
		},
		Services: []config.Service{
			{ID: "svc-1", Name: "svc-users", Upstream: "up-users", Protocol: "http"},
		},
		Routes: []config.Route{
			{ID: "rt-1", Name: "users", Service: "svc-users", Paths: []string{"/users"}, Methods: []string{"GET"}},
		},
		Upstreams: []config.Upstream{
			{
				ID:        "up-1",
				Name:      "up-users",
				Algorithm: "round_robin",
				Targets: []config.UpstreamTarget{
					{ID: "t1", Address: mustHost(t, upstream.URL), Weight: 1},
				},
				HealthCheck: config.HealthCheckConfig{
					Active: config.ActiveHealthCheckConfig{
						Path:               "/health",
						Interval:           time.Second,
						Timeout:            time.Second,
						HealthyThreshold:   1,
						UnhealthyThreshold: 1,
					},
				},
			},
		},
		Consumers: []config.Consumer{
			{
				ID:   "c1",
				Name: "mobile-app",
				APIKeys: []config.ConsumerAPIKey{
					{ID: "k1", Key: "ck_live_abc123"},
				},
			},
		},
	}

	gw, err := New(cfg)
	if err != nil {
		t.Fatalf("gateway.New error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "http://gateway.local/users", nil)
	req.Header.Set("X-API-Key", "ck_live_abc123")
	rr := httptest.NewRecorder()
	gw.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200 got %d", rr.Code)
	}

	consumer := ConsumerFromRequest(req)
	if consumer == nil {
		t.Fatalf("expected consumer to be resolved")
	}
	if consumer.Name != "mobile-app" {
		t.Fatalf("expected mobile-app got %q", consumer.Name)
	}
}

func TestExtractAPIKeyFallbacks(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "http://example.com/path?apikey=query-key", nil)
	if got := extractAPIKey(req); got != "query-key" {
		t.Fatalf("expected query key, got %q", got)
	}

	req2 := httptest.NewRequest(http.MethodGet, "http://example.com/path", nil)
	req2.Header.Set("Authorization", "Bearer bearer-key")
	if got := extractAPIKey(req2); got != "bearer-key" {
		t.Fatalf("expected bearer key, got %q", got)
	}
}
