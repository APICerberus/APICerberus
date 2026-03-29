package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/APICerberus/APICerebrus/internal/config"
	"github.com/APICerberus/APICerebrus/internal/gateway"
)

func TestAdminAuthMiddleware(t *testing.T) {
	t.Parallel()

	serverURL, _ := newAdminTestServer(t)
	req, _ := http.NewRequest(http.MethodGet, serverURL+"/admin/api/v1/status", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", resp.StatusCode)
	}
}

func TestAdminEndpointsIntegration(t *testing.T) {
	t.Parallel()

	baseURL, upstreamURL := newAdminTestServer(t)

	// status
	resp := mustJSONRequest(t, http.MethodGet, baseURL+"/admin/api/v1/status", "secret-admin", nil)
	assertStatus(t, resp, http.StatusOK)
	assertJSONField(t, resp, "status", "ok")

	// info
	resp = mustJSONRequest(t, http.MethodGet, baseURL+"/admin/api/v1/info", "secret-admin", nil)
	assertStatus(t, resp, http.StatusOK)
	assertHasJSONField(t, resp, "version")

	// services list
	resp = mustJSONRequest(t, http.MethodGet, baseURL+"/admin/api/v1/services", "secret-admin", nil)
	assertStatus(t, resp, http.StatusOK)
	assertJSONArrayLenAtLeast(t, resp, 1)

	// create service
	servicePayload := map[string]any{
		"id":       "svc-orders",
		"name":     "svc-orders",
		"protocol": "http",
		"upstream": "up-users",
	}
	resp = mustJSONRequest(t, http.MethodPost, baseURL+"/admin/api/v1/services", "secret-admin", servicePayload)
	assertStatus(t, resp, http.StatusCreated)

	// get/update/delete service
	resp = mustJSONRequest(t, http.MethodGet, baseURL+"/admin/api/v1/services/svc-orders", "secret-admin", nil)
	assertStatus(t, resp, http.StatusOK)

	servicePayload["name"] = "svc-orders-v2"
	resp = mustJSONRequest(t, http.MethodPut, baseURL+"/admin/api/v1/services/svc-orders", "secret-admin", servicePayload)
	assertStatus(t, resp, http.StatusOK)

	resp = mustJSONRequest(t, http.MethodDelete, baseURL+"/admin/api/v1/services/svc-orders", "secret-admin", nil)
	assertStatus(t, resp, http.StatusNoContent)

	// routes CRUD
	routePayload := map[string]any{
		"id":      "route-extra",
		"name":    "route-extra",
		"service": "svc-users",
		"paths":   []string{"/extra"},
		"methods": []string{"GET"},
	}
	resp = mustJSONRequest(t, http.MethodPost, baseURL+"/admin/api/v1/routes", "secret-admin", routePayload)
	assertStatus(t, resp, http.StatusCreated)

	resp = mustJSONRequest(t, http.MethodGet, baseURL+"/admin/api/v1/routes/route-extra", "secret-admin", nil)
	assertStatus(t, resp, http.StatusOK)

	routePayload["paths"] = []string{"/extra-v2"}
	resp = mustJSONRequest(t, http.MethodPut, baseURL+"/admin/api/v1/routes/route-extra", "secret-admin", routePayload)
	assertStatus(t, resp, http.StatusOK)

	resp = mustJSONRequest(t, http.MethodDelete, baseURL+"/admin/api/v1/routes/route-extra", "secret-admin", nil)
	assertStatus(t, resp, http.StatusNoContent)

	// upstream CRUD
	upstreamHost := mustHost(t, upstreamURL)
	upstreamPayload := map[string]any{
		"id":        "up-extra",
		"name":      "up-extra",
		"algorithm": "round_robin",
		"targets": []map[string]any{
			{
				"id":      "up-extra-t1",
				"address": upstreamHost,
				"weight":  1,
			},
		},
		"health_check": map[string]any{
			"active": map[string]any{
				"path":                "/health",
				"interval":            int64(time.Second),
				"timeout":             int64(time.Second),
				"healthy_threshold":   1,
				"unhealthy_threshold": 1,
			},
		},
	}
	resp = mustJSONRequest(t, http.MethodPost, baseURL+"/admin/api/v1/upstreams", "secret-admin", upstreamPayload)
	assertStatus(t, resp, http.StatusCreated)

	resp = mustJSONRequest(t, http.MethodGet, baseURL+"/admin/api/v1/upstreams/up-extra", "secret-admin", nil)
	assertStatus(t, resp, http.StatusOK)

	upstreamPayload["algorithm"] = "weighted_round_robin"
	resp = mustJSONRequest(t, http.MethodPut, baseURL+"/admin/api/v1/upstreams/up-extra", "secret-admin", upstreamPayload)
	assertStatus(t, resp, http.StatusOK)

	// target management
	targetPayload := map[string]any{
		"id":      "up-extra-t2",
		"address": upstreamHost,
		"weight":  2,
	}
	resp = mustJSONRequest(t, http.MethodPost, baseURL+"/admin/api/v1/upstreams/up-extra/targets", "secret-admin", targetPayload)
	assertStatus(t, resp, http.StatusCreated)

	resp = mustJSONRequest(t, http.MethodGet, baseURL+"/admin/api/v1/upstreams/up-extra/health", "secret-admin", nil)
	assertStatus(t, resp, http.StatusOK)
	assertHasJSONField(t, resp, "targets")

	resp = mustJSONRequest(t, http.MethodDelete, baseURL+"/admin/api/v1/upstreams/up-extra/targets/up-extra-t2", "secret-admin", nil)
	assertStatus(t, resp, http.StatusNoContent)

	resp = mustJSONRequest(t, http.MethodDelete, baseURL+"/admin/api/v1/upstreams/up-extra", "secret-admin", nil)
	assertStatus(t, resp, http.StatusNoContent)

	// reload endpoint
	resp = mustJSONRequest(t, http.MethodPost, baseURL+"/admin/api/v1/config/reload", "secret-admin", map[string]any{})
	assertStatus(t, resp, http.StatusOK)
	assertJSONField(t, resp, "reloaded", true)
}

func newAdminTestServer(t *testing.T) (adminBaseURL string, upstreamURL string) {
	t.Helper()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	t.Cleanup(upstream.Close)

	cfg := &config.Config{
		Gateway: config.GatewayConfig{
			HTTPAddr:       "127.0.0.1:0",
			ReadTimeout:    2 * time.Second,
			WriteTimeout:   2 * time.Second,
			IdleTimeout:    10 * time.Second,
			MaxHeaderBytes: 1 << 20,
			MaxBodyBytes:   1 << 20,
		},
		Admin: config.AdminConfig{
			APIKey: "secret-admin",
		},
		Services: []config.Service{
			{
				ID:       "svc-users",
				Name:     "svc-users",
				Protocol: "http",
				Upstream: "up-users",
			},
		},
		Routes: []config.Route{
			{
				ID:      "route-users",
				Name:    "route-users",
				Service: "svc-users",
				Paths:   []string{"/users"},
				Methods: []string{http.MethodGet},
			},
		},
		Upstreams: []config.Upstream{
			{
				ID:        "up-users",
				Name:      "up-users",
				Algorithm: "round_robin",
				Targets: []config.UpstreamTarget{
					{
						ID:      "up-users-t1",
						Address: mustHost(t, upstream.URL),
						Weight:  1,
					},
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
	adminSrv, err := NewServer(cfg, gw)
	if err != nil {
		t.Fatalf("NewServer error: %v", err)
	}
	httpSrv := httptest.NewServer(adminSrv)
	t.Cleanup(httpSrv.Close)

	return httpSrv.URL, upstream.URL
}

func mustJSONRequest(t *testing.T, method, rawURL, adminKey string, payload any) map[string]any {
	t.Helper()

	var bodyBytes []byte
	if payload != nil {
		var err error
		bodyBytes, err = json.Marshal(payload)
		if err != nil {
			t.Fatalf("json marshal: %v", err)
		}
	}

	req, err := http.NewRequest(method, rawURL, bytes.NewReader(bodyBytes))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("X-Admin-Key", adminKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	result := map[string]any{
		"status_code": float64(resp.StatusCode),
	}
	if resp.ContentLength == 0 || resp.StatusCode == http.StatusNoContent {
		return result
	}

	var body any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	result["body"] = body
	return result
}

func assertStatus(t *testing.T, resp map[string]any, want int) {
	t.Helper()
	got := int(resp["status_code"].(float64))
	if got != want {
		t.Fatalf("expected status %d got %d (resp=%#v)", want, got, resp)
	}
}

func assertJSONField(t *testing.T, resp map[string]any, key string, want any) {
	t.Helper()
	body, ok := resp["body"].(map[string]any)
	if !ok {
		t.Fatalf("response body is not object: %#v", resp)
	}
	if body[key] != want {
		t.Fatalf("expected body[%q]=%v got %v (body=%#v)", key, want, body[key], body)
	}
}

func assertHasJSONField(t *testing.T, resp map[string]any, key string) {
	t.Helper()
	body, ok := resp["body"].(map[string]any)
	if !ok {
		t.Fatalf("response body is not object: %#v", resp)
	}
	if _, exists := body[key]; !exists {
		t.Fatalf("expected field %q in body %#v", key, body)
	}
}

func assertJSONArrayLenAtLeast(t *testing.T, resp map[string]any, min int) {
	t.Helper()
	body, ok := resp["body"].([]any)
	if !ok {
		t.Fatalf("response body is not array: %#v", resp)
	}
	if len(body) < min {
		t.Fatalf("expected array len >= %d got %d", min, len(body))
	}
}

func mustHost(t *testing.T, rawURL string) string {
	t.Helper()
	u, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	return u.Host
}
