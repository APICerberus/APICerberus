package test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/APICerberus/APICerebrus/internal/config"
	"github.com/APICerberus/APICerebrus/internal/gateway"
)

func TestE2ERequestTransformProxyResponseTransformPipeline(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/v1" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("path mismatch"))
			return
		}
		if r.Header.Get("X-Req-Stage") != "transformed" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("header mismatch"))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("upstream-body"))
	}))
	defer upstream.Close()

	gwAddr := freeAddr(t)
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
			{ID: "svc-v004", Name: "svc-v004", Protocol: "http", Upstream: "up-v004"},
		},
		Routes: []config.Route{
			{
				ID:      "route-v004",
				Name:    "route-v004",
				Service: "svc-v004",
				Paths:   []string{"/v004/transform"},
				Methods: []string{http.MethodGet},
				Plugins: []config.PluginConfig{
					{
						Name: "request-transform",
						Config: map[string]any{
							"path_pattern":     "^/v004/transform$",
							"path_replacement": "/internal/v1",
							"add_headers": map[string]any{
								"X-Req-Stage": "transformed",
							},
						},
					},
					{
						Name: "response-transform",
						Config: map[string]any{
							"add_headers": map[string]any{
								"X-Resp-Stage": "transformed",
							},
							"replace_body": "response-transformed",
						},
					},
				},
			},
		},
		Upstreams: []config.Upstream{
			{
				ID:        "up-v004",
				Name:      "up-v004",
				Algorithm: "round_robin",
				Targets: []config.UpstreamTarget{
					{ID: "t-v004", Address: mustHost(t, upstream.URL), Weight: 1},
				},
				HealthCheck: config.HealthCheckConfig{
					Active: config.ActiveHealthCheckConfig{
						Path:               "/health",
						Interval:           1 * time.Second,
						Timeout:            1 * time.Second,
						HealthyThreshold:   1,
						UnhealthyThreshold: 1,
					},
				},
			},
		},
	}

	gw, err := gateway.New(cfg)
	if err != nil {
		t.Fatalf("gateway.New error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errCh := make(chan error, 1)
	go func() { errCh <- gw.Start(ctx) }()

	waitForHTTPReady(t, "http://"+gwAddr+"/v004/transform", nil)

	resp, err := http.Get("http://" + gwAddr + "/v004/transform")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body := readAllAndClose(t, resp.Body)
	if resp.StatusCode != http.StatusOK || body != "response-transformed" {
		t.Fatalf("unexpected response status=%d body=%q", resp.StatusCode, body)
	}
	if resp.Header.Get("X-Resp-Stage") != "transformed" {
		t.Fatalf("expected transformed response header")
	}

	cancel()
	if err := <-errCh; err != nil {
		t.Fatalf("gateway runtime error: %v", err)
	}
}

func TestE2EJSONSchemaValidationAndCorrelationIDPropagation(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("missing request id"))
			return
		}
		w.Header().Set("X-Upstream-Request-ID", id)
		_, _ = w.Write([]byte(id))
	}))
	defer upstream.Close()

	gwAddr := freeAddr(t)
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
			{ID: "svc-v004-validate", Name: "svc-v004-validate", Protocol: "http", Upstream: "up-v004-validate"},
		},
		Routes: []config.Route{
			{
				ID:      "route-v004-validate",
				Name:    "route-v004-validate",
				Service: "svc-v004-validate",
				Paths:   []string{"/v004/validate"},
				Methods: []string{http.MethodPost},
				Plugins: []config.PluginConfig{
					{Name: "correlation-id"},
					{
						Name: "request-validator",
						Config: map[string]any{
							"schema": map[string]any{
								"type":     "object",
								"required": []any{"name", "email"},
								"properties": map[string]any{
									"name":  map[string]any{"type": "string"},
									"email": map[string]any{"type": "string", "format": "email"},
								},
							},
						},
					},
				},
			},
		},
		Upstreams: []config.Upstream{
			{
				ID:        "up-v004-validate",
				Name:      "up-v004-validate",
				Algorithm: "round_robin",
				Targets: []config.UpstreamTarget{
					{ID: "t-v004-validate", Address: mustHost(t, upstream.URL), Weight: 1},
				},
				HealthCheck: config.HealthCheckConfig{
					Active: config.ActiveHealthCheckConfig{
						Path:               "/health",
						Interval:           1 * time.Second,
						Timeout:            1 * time.Second,
						HealthyThreshold:   1,
						UnhealthyThreshold: 1,
					},
				},
			},
		},
	}

	gw, err := gateway.New(cfg)
	if err != nil {
		t.Fatalf("gateway.New error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errCh := make(chan error, 1)
	go func() { errCh <- gw.Start(ctx) }()

	waitForHTTPReady(t, "http://"+gwAddr+"/v004/validate", nil)

	validReq, _ := http.NewRequest(http.MethodPost, "http://"+gwAddr+"/v004/validate", bytes.NewBufferString(`{"name":"Alice","email":"alice@example.com"}`))
	validReq.Header.Set("Content-Type", "application/json")
	validResp, err := http.DefaultClient.Do(validReq)
	if err != nil {
		t.Fatalf("valid request failed: %v", err)
	}
	validBody := readAllAndClose(t, validResp.Body)
	if validResp.StatusCode != http.StatusOK {
		t.Fatalf("expected valid request 200 got %d body=%q", validResp.StatusCode, validBody)
	}
	correlationID := validResp.Header.Get("X-Request-ID")
	if correlationID == "" {
		t.Fatalf("expected generated X-Request-ID header")
	}
	if validBody != correlationID {
		t.Fatalf("expected upstream to receive same correlation id, body=%q header=%q", validBody, correlationID)
	}

	invalidReq, _ := http.NewRequest(http.MethodPost, "http://"+gwAddr+"/v004/validate", bytes.NewBufferString(`{"name":123,"email":"invalid"}`))
	invalidReq.Header.Set("Content-Type", "application/json")
	invalidResp, err := http.DefaultClient.Do(invalidReq)
	if err != nil {
		t.Fatalf("invalid request failed: %v", err)
	}
	invalidBody := readAllAndClose(t, invalidResp.Body)
	if invalidResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected invalid request 400 got %d body=%q", invalidResp.StatusCode, invalidBody)
	}

	cancel()
	if err := <-errCh; err != nil {
		t.Fatalf("gateway runtime error: %v", err)
	}
}
