package gateway

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/APICerberus/APICerebrus/internal/config"
)

func TestRouterExactMatch(t *testing.T) {
	t.Parallel()

	router := mustRouter(t,
		[]config.Route{
			{
				Name:    "users-exact",
				Service: "svc-users",
				Paths:   []string{"/api/v1/users"},
				Methods: []string{http.MethodGet},
			},
		},
		baseServices(),
	)

	req := httptest.NewRequest(http.MethodGet, "http://example.com/api/v1/users", nil)
	route, service, err := router.Match(req)
	if err != nil {
		t.Fatalf("Match error: %v", err)
	}
	if route.Name != "users-exact" {
		t.Fatalf("expected route users-exact got %q", route.Name)
	}
	if service.Name != "svc-users" {
		t.Fatalf("expected service svc-users got %q", service.Name)
	}
}

func TestRouterPrefixAndWildcardMatch(t *testing.T) {
	t.Parallel()

	router := mustRouter(t,
		[]config.Route{
			{
				Name:    "api-prefix",
				Service: "svc-users",
				Paths:   []string{"/api/*"},
				Methods: []string{http.MethodGet},
			},
		},
		baseServices(),
	)

	req := httptest.NewRequest(http.MethodGet, "http://example.com/api/v1/users/42", nil)
	route, _, err := router.Match(req)
	if err != nil {
		t.Fatalf("Match error: %v", err)
	}
	if route.Name != "api-prefix" {
		t.Fatalf("expected api-prefix got %q", route.Name)
	}
}

func TestRouterParamMatch(t *testing.T) {
	t.Parallel()

	router := mustRouter(t,
		[]config.Route{
			{
				Name:    "order-detail",
				Service: "svc-users",
				Paths:   []string{"/orders/:id"},
				Methods: []string{http.MethodGet},
			},
		},
		baseServices(),
	)

	req := httptest.NewRequest(http.MethodGet, "http://example.com/orders/42", nil)
	route, _, err := router.Match(req)
	if err != nil {
		t.Fatalf("Match error: %v", err)
	}
	if route.Name != "order-detail" {
		t.Fatalf("expected route order-detail got %q", route.Name)
	}

	params := Params(req)
	if params["id"] != "42" {
		t.Fatalf("expected param id=42 got %#v", params)
	}
}

func TestRouterHostRouting(t *testing.T) {
	t.Parallel()

	router := mustRouter(t,
		[]config.Route{
			{
				Name:    "default-health",
				Service: "svc-users",
				Paths:   []string{"/health"},
				Methods: []string{http.MethodGet},
			},
			{
				Name:    "host-health",
				Service: "svc-alt",
				Hosts:   []string{"api.example.com"},
				Paths:   []string{"/health"},
				Methods: []string{http.MethodGet},
			},
		},
		append(baseServices(), config.Service{Name: "svc-alt", Upstream: "up-users"}),
	)

	req := httptest.NewRequest(http.MethodGet, "http://api.example.com/health", nil)
	route, service, err := router.Match(req)
	if err != nil {
		t.Fatalf("Match error: %v", err)
	}
	if route.Name != "host-health" || service.Name != "svc-alt" {
		t.Fatalf("expected host-health/svc-alt got %s/%s", route.Name, service.Name)
	}

	reqDefault := httptest.NewRequest(http.MethodGet, "http://other.example.com/health", nil)
	route, service, err = router.Match(reqDefault)
	if err != nil {
		t.Fatalf("Match error: %v", err)
	}
	if route.Name != "default-health" || service.Name != "svc-users" {
		t.Fatalf("expected default-health/svc-users got %s/%s", route.Name, service.Name)
	}
}

func TestRouterPriorityExactThenPrefixThenRegex(t *testing.T) {
	t.Parallel()

	router := mustRouter(t,
		[]config.Route{
			{
				Name:     "regex-users",
				Service:  "svc-users",
				Paths:    []string{"/api/v[0-9]+/users"},
				Methods:  []string{http.MethodGet},
				Priority: 100,
			},
			{
				Name:     "prefix-api",
				Service:  "svc-users",
				Paths:    []string{"/api/*"},
				Methods:  []string{http.MethodGet},
				Priority: 10,
			},
			{
				Name:     "exact-v1",
				Service:  "svc-users",
				Paths:    []string{"/api/v1/users"},
				Methods:  []string{http.MethodGet},
				Priority: 1,
			},
		},
		baseServices(),
	)

	reqExact := httptest.NewRequest(http.MethodGet, "http://example.com/api/v1/users", nil)
	route, _, err := router.Match(reqExact)
	if err != nil {
		t.Fatalf("Match exact error: %v", err)
	}
	if route.Name != "exact-v1" {
		t.Fatalf("expected exact-v1 got %q", route.Name)
	}

	reqPrefix := httptest.NewRequest(http.MethodGet, "http://example.com/api/v2/users", nil)
	route, _, err = router.Match(reqPrefix)
	if err != nil {
		t.Fatalf("Match prefix error: %v", err)
	}
	if route.Name != "prefix-api" {
		t.Fatalf("expected prefix-api before regex, got %q", route.Name)
	}
}

func TestRouterStripPath(t *testing.T) {
	t.Parallel()

	router := mustRouter(t,
		[]config.Route{
			{
				Name:      "strip-api",
				Service:   "svc-users",
				Paths:     []string{"/api/*"},
				Methods:   []string{http.MethodGet},
				StripPath: true,
			},
		},
		baseServices(),
	)

	req := httptest.NewRequest(http.MethodGet, "http://example.com/api/v1/users", nil)
	route, _, err := router.Match(req)
	if err != nil {
		t.Fatalf("Match error: %v", err)
	}
	if route.Name != "strip-api" {
		t.Fatalf("expected strip-api got %q", route.Name)
	}
	if req.URL.Path != "/v1/users" {
		t.Fatalf("expected stripped path /v1/users got %q", req.URL.Path)
	}
}

func TestRouterAnyMethodTree(t *testing.T) {
	t.Parallel()

	router := mustRouter(t,
		[]config.Route{
			{
				Name:    "any-method",
				Service: "svc-users",
				Paths:   []string{"/ping"},
				Methods: []string{"*"},
			},
		},
		baseServices(),
	)

	for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodDelete} {
		req := httptest.NewRequest(method, "http://example.com/ping", nil)
		route, _, err := router.Match(req)
		if err != nil {
			t.Fatalf("Match error for %s: %v", method, err)
		}
		if route.Name != "any-method" {
			t.Fatalf("expected any-method for %s got %q", method, route.Name)
		}
	}
}

func TestRouterRebuild(t *testing.T) {
	t.Parallel()

	router := mustRouter(t,
		[]config.Route{
			{
				Name:    "route-a",
				Service: "svc-users",
				Paths:   []string{"/a"},
				Methods: []string{http.MethodGet},
			},
		},
		baseServices(),
	)

	reqA := httptest.NewRequest(http.MethodGet, "http://example.com/a", nil)
	if _, _, err := router.Match(reqA); err != nil {
		t.Fatalf("expected /a to match before rebuild: %v", err)
	}

	err := router.Rebuild(
		[]config.Route{
			{
				Name:    "route-b",
				Service: "svc-users",
				Paths:   []string{"/b"},
				Methods: []string{http.MethodGet},
			},
		},
		baseServices(),
	)
	if err != nil {
		t.Fatalf("Rebuild error: %v", err)
	}

	reqOld := httptest.NewRequest(http.MethodGet, "http://example.com/a", nil)
	if _, _, err := router.Match(reqOld); !errors.Is(err, ErrNoRouteMatched) {
		t.Fatalf("expected no route for old path, got: %v", err)
	}

	reqNew := httptest.NewRequest(http.MethodGet, "http://example.com/b", nil)
	route, _, err := router.Match(reqNew)
	if err != nil {
		t.Fatalf("expected /b to match after rebuild: %v", err)
	}
	if route.Name != "route-b" {
		t.Fatalf("expected route-b got %q", route.Name)
	}
}

func mustRouter(t *testing.T, routes []config.Route, services []config.Service) *Router {
	t.Helper()

	router, err := NewRouter(routes, services)
	if err != nil {
		t.Fatalf("NewRouter error: %v", err)
	}
	return router
}

func baseServices() []config.Service {
	return []config.Service{
		{
			Name:     "svc-users",
			Upstream: "up-users",
		},
	}
}
