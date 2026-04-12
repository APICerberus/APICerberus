package plugin

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/APICerberus/APICerebrus/internal/config"
)

func TestEndpointPermissionWhitelistHitMiss(t *testing.T) {
	t.Parallel()

	lookup := func(userID, routeID string) (*EndpointPermissionRecord, error) {
		if userID == "u1" && routeID == "route-1" {
			return &EndpointPermissionRecord{
				ID:      "perm-1",
				UserID:  "u1",
				RouteID: "route-1",
				Allowed: true,
				Methods: []string{"GET"},
			}, nil
		}
		return nil, nil
	}
	plugin := NewEndpointPermission(lookup)

	allowedCtx := permissionPipelineContext("u1", "route-1", http.MethodGet)
	if err := plugin.Evaluate(allowedCtx); err != nil {
		t.Fatalf("expected allowed permission, got error: %v", err)
	}

	deniedCtx := permissionPipelineContext("u1", "route-missing", http.MethodGet)
	err := plugin.Evaluate(deniedCtx)
	if err == nil {
		t.Fatalf("expected denied error for missing permission")
	}
	permErr, ok := err.(*EndpointPermissionError)
	if !ok || permErr.Status != http.StatusForbidden || permErr.Code != "permission_denied" {
		t.Fatalf("unexpected permission error: %#v", err)
	}
}

func TestEndpointPermissionMethodRestriction(t *testing.T) {
	t.Parallel()

	plugin := NewEndpointPermission(func(userID, routeID string) (*EndpointPermissionRecord, error) {
		return &EndpointPermissionRecord{
			ID:      "perm-2",
			UserID:  userID,
			RouteID: routeID,
			Allowed: true,
			Methods: []string{"GET"},
		}, nil
	})

	ctx := permissionPipelineContext("u1", "route-1", http.MethodPost)
	err := plugin.Evaluate(ctx)
	if err == nil {
		t.Fatalf("expected method denied error")
	}
	permErr, ok := err.(*EndpointPermissionError)
	if !ok || permErr.Code != "permission_method_denied" {
		t.Fatalf("unexpected permission error: %#v", err)
	}
}

func TestEndpointPermissionTimeRestrictionAndOverrides(t *testing.T) {
	t.Parallel()

	cost := int64(9)
	plugin := NewEndpointPermission(func(userID, routeID string) (*EndpointPermissionRecord, error) {
		return &EndpointPermissionRecord{
			ID:      "perm-3",
			UserID:  userID,
			RouteID: routeID,
			Allowed: true,
			Methods: []string{"GET"},
			AllowedDays: []int{
				1, // monday
			},
			AllowedHours: []string{"09:00-12:00"},
			RateLimits: map[string]any{
				"algorithm": "fixed_window",
				"limit":     2,
				"window":    "1s",
			},
			CreditCost: &cost,
		}, nil
	})

	plugin.now = func() time.Time {
		return time.Date(2026, time.March, 30, 10, 15, 0, 0, time.UTC) // Monday
	}
	ctxAllowed := permissionPipelineContext("u1", "route-1", http.MethodGet)
	if err := plugin.Evaluate(ctxAllowed); err != nil {
		t.Fatalf("expected allowed time window, got %v", err)
	}
	if ctxAllowed.Metadata[metadataPermissionRateLimitOverride] == nil {
		t.Fatalf("expected rate limit override metadata")
	}
	if ctxAllowed.Metadata[metadataPermissionCreditCost] != int64(9) {
		t.Fatalf("expected credit cost override metadata 9, got %#v", ctxAllowed.Metadata[metadataPermissionCreditCost])
	}

	plugin.now = func() time.Time {
		return time.Date(2026, time.March, 30, 18, 30, 0, 0, time.UTC) // Monday, out of range
	}
	ctxDenied := permissionPipelineContext("u1", "route-1", http.MethodGet)
	err := plugin.Evaluate(ctxDenied)
	if err == nil {
		t.Fatalf("expected hour denied error")
	}
	permErr, ok := err.(*EndpointPermissionError)
	if !ok || permErr.Code != "permission_hour_denied" {
		t.Fatalf("unexpected permission error: %#v", err)
	}
}

func permissionPipelineContext(userID, routeID, method string) *PipelineContext {
	return &PipelineContext{
		Request:        httptest.NewRequest(method, "http://gateway.local/x", nil),
		ResponseWriter: httptest.NewRecorder(),
		Route: &config.Route{
			ID:   routeID,
			Name: routeID,
		},
		Consumer: &config.Consumer{
			ID:   userID,
			Name: "user",
		},
		Metadata: map[string]any{},
	}
}
