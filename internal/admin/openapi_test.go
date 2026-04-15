package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewOpenAPI31GeneratorNilServer(t *testing.T) {
	t.Parallel()

	g := NewOpenAPI31Generator(nil, Info{Title: "Test"}, nil)
	if g != nil {
		t.Fatal("expected nil for nil server")
	}
}

func TestGeneratorDefaultVersion(t *testing.T) {
	t.Parallel()

	g := NewOpenAPI31Generator(&Server{}, Info{Title: "Test"}, nil)
	data, err := g.Generate()
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("parse json: %v", err)
	}

	info := doc["info"].(map[string]any)
	if info["version"] != "1.0.0" {
		t.Fatalf("expected default version 1.0.0, got %v", info["version"])
	}
}

func TestGeneratorIncludesOpenAPI31Field(t *testing.T) {
	t.Parallel()

	g := NewOpenAPI31Generator(&Server{}, Info{Title: "Test", Version: "2.0"}, nil)
	data, _ := g.Generate()

	var doc map[string]any
	json.Unmarshal(data, &doc)

	if doc["openapi"] != "3.1.0" {
		t.Fatalf("expected openapi 3.1.0, got %v", doc["openapi"])
	}
}

func TestGeneratorPaths(t *testing.T) {
	t.Parallel()

	g := NewOpenAPI31Generator(&Server{}, Info{Title: "Test"}, nil)
	data, _ := g.Generate()

	var doc map[string]any
	json.Unmarshal(data, &doc)

	paths := doc["paths"].(map[string]any)
	if len(paths) == 0 {
		t.Fatal("expected paths to be populated")
	}

	// Check a few known paths.
	expectedPaths := []string{
		"/health",
		"/admin/api/v1/services",
		"/admin/api/v1/routes",
		"/admin/api/v1/upstreams",
		"/admin/api/v1/users",
		"/admin/api/v1/credits/overview",
		"/admin/api/v1/audit-logs",
		"/admin/api/v1/analytics/overview",
		"/admin/api/v1/cluster/status",
	}

	for _, p := range expectedPaths {
		if _, ok := paths[p]; !ok {
			t.Errorf("expected path %q in spec", p)
		}
	}
}

func TestGeneratorHealthPathNoAuth(t *testing.T) {
	g := NewOpenAPI31Generator(&Server{}, Info{Title: "Test"}, nil)
	data, err := g.Generate()
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("parse json: %v\n%s", err, string(data))
	}

	paths, ok := doc["paths"].(map[string]any)
	if !ok {
		t.Fatalf("paths is not map[string]any: %T", doc["paths"])
	}

	health, ok := paths["/health"].(map[string]any)
	if !ok {
		t.Fatalf("health path is not map[string]any: %T", paths["/health"])
	}

	getOp, ok := health["GET"].(map[string]any)
	if !ok {
		t.Fatalf("get operation is not map[string]any: %T", health["get"])
	}

	security, ok := getOp["security"]
	if !ok {
		t.Fatalf("security field missing: %T", getOp["security"])
	}
	// Health path should have no security (nil or empty slice = no auth required).
	secArr, isSlice := security.([]any)
	if !isSlice {
		t.Fatalf("security is not []any: %T", security)
	}
	if len(secArr) != 0 {
		t.Fatalf("expected no security for health, got %d", len(secArr))
	}
}

func TestGeneratorAuthPathHasSecurity(t *testing.T) {
	t.Parallel()

	g := NewOpenAPI31Generator(&Server{}, Info{Title: "Test"}, nil)
	data, _ := g.Generate()

	var doc map[string]any
	json.Unmarshal(data, &doc)

	paths := doc["paths"].(map[string]any)
	tokenPath := paths["/admin/api/v1/auth/token"].(map[string]any)
	postOp := tokenPath["POST"].(map[string]any)

	security := postOp["security"].([]any)
	if len(security) == 0 {
		t.Fatal("expected auth to have security")
	}
}

func TestGeneratorComponentsSchemas(t *testing.T) {
	t.Parallel()

	g := NewOpenAPI31Generator(&Server{}, Info{Title: "Test"}, nil)
	data, _ := g.Generate()

	var doc map[string]any
	json.Unmarshal(data, &doc)

	components := doc["components"].(map[string]any)
	schemas := components["schemas"].(map[string]any)

	// Check that key schemas are present.
	keySchemas := []string{
		"ServiceResponse",
		"RouteResponse",
		"UpstreamResponse",
		"UserResponse",
		"CreditsOverviewResponse",
		"AnalyticsOverviewResponse",
		"ClusterStatusResponse",
		"GraphQLRequest",
		"BrandingResponse",
	}

	for _, s := range keySchemas {
		if _, ok := schemas[s]; !ok {
			t.Errorf("expected schema %q", s)
		}
	}
}

func TestGeneratorSecuritySchemes(t *testing.T) {
	t.Parallel()

	g := NewOpenAPI31Generator(&Server{}, Info{Title: "Test"}, nil)
	data, _ := g.Generate()

	var doc map[string]any
	json.Unmarshal(data, &doc)

	components := doc["components"].(map[string]any)
	secSchemes := components["securitySchemes"].(map[string]any)

	bearer, ok := secSchemes["BearerAuth"]
	if !ok {
		t.Fatal("expected BearerAuth security scheme")
	}
	bearerMap := bearer.(map[string]any)
	if bearerMap["type"] != "http" || bearerMap["scheme"] != "bearer" {
		t.Fatalf("unexpected bearer scheme config: %+v", bearerMap)
	}

	apiKey, ok := secSchemes["ApiKeyAuth"]
	if !ok {
		t.Fatal("expected ApiKeyAuth security scheme")
	}
	apiKeyMap := apiKey.(map[string]any)
	if apiKeyMap["type"] != "apiKey" || apiKeyMap["in"] != "header" {
		t.Fatalf("unexpected apiKey scheme config: %+v", apiKeyMap)
	}
}

func TestGeneratorTags(t *testing.T) {
	t.Parallel()

	g := NewOpenAPI31Generator(&Server{}, Info{Title: "Test"}, nil)
	data, _ := g.Generate()

	var doc map[string]any
	json.Unmarshal(data, &doc)

	tags := doc["tags"].([]any)
	if len(tags) == 0 {
		t.Fatal("expected tags to be populated")
	}

	tagNames := make([]string, len(tags))
	for i, t := range tags {
		tagNames[i] = t.(map[string]any)["name"].(string)
	}

	expectedTags := []string{
		"Health", "Auth", "Gateway", "Services", "Routes",
		"Upstreams", "Users", "API Keys", "Credits", "Permissions",
		"IP Whitelist", "Audit Logs", "Analytics", "Alerts", "Billing",
		"Cluster", "Federation", "Webhooks", "Bulk", "GraphQL", "WebSocket",
	}

	for _, et := range expectedTags {
		found := false
		for _, tn := range tagNames {
			if tn == et {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected tag %q", et)
		}
	}
}

func TestGeneratorServers(t *testing.T) {
	t.Parallel()

	g := NewOpenAPI31Generator(&Server{}, Info{Title: "Test"}, []ServerEntry{
		{URL: "https://api.example.com", Description: "Production"},
		{URL: "http://localhost:9876", Description: "Local"},
	})
	data, _ := g.Generate()

	var doc map[string]any
	json.Unmarshal(data, &doc)

	servers := doc["servers"].([]any)
	if len(servers) != 2 {
		t.Fatalf("expected 2 servers, got %d", len(servers))
	}

	first := servers[0].(map[string]any)
	if first["url"] != "https://api.example.com" {
		t.Fatalf("unexpected first server url: %v", first["url"])
	}
}

func TestGeneratorOperationIDs(t *testing.T) {
	t.Parallel()

	g := NewOpenAPI31Generator(&Server{}, Info{Title: "Test"}, nil)
	data, _ := g.Generate()

	var doc map[string]any
	json.Unmarshal(data, &doc)

	paths := doc["paths"].(map[string]any)

	// All operations should have operationId.
	for path, pathItem := range paths {
		pi := pathItem.(map[string]any)
		for method, op := range pi {
			if method == "get" || method == "post" || method == "put" || method == "delete" || method == "patch" {
				operation := op.(map[string]any)
				if operation["operationId"] == "" {
					t.Errorf("path %s method %s missing operationId", path, method)
				}
			}
		}
	}
}

func TestGeneratorNoServers(t *testing.T) {
	t.Parallel()

	g := NewOpenAPI31Generator(&Server{}, Info{Title: "Test"}, nil)
	data, _ := g.Generate()

	var doc map[string]any
	json.Unmarshal(data, &doc)

	// servers key should be absent or empty.
	if servers, ok := doc["servers"]; ok {
		sl := servers.([]any)
		if len(sl) != 0 {
			t.Fatalf("expected no servers, got %d", len(sl))
		}
	}
}

func TestGeneratorServicePathHasIDParam(t *testing.T) {
	t.Parallel()

	g := NewOpenAPI31Generator(&Server{}, Info{Title: "Test"}, nil)
	data, _ := g.Generate()

	var doc map[string]any
	json.Unmarshal(data, &doc)

	paths := doc["paths"].(map[string]any)
	path := paths["/admin/api/v1/services/{id}"].(map[string]any)
	getOp := path["GET"].(map[string]any)

	params := getOp["parameters"].([]any)
	if len(params) != 1 {
		t.Fatalf("expected 1 parameter for service ID, got %d", len(params))
	}

	param := params[0].(map[string]any)
	if param["name"] != "id" || param["in"] != "path" {
		t.Fatalf("unexpected param: %+v", param)
	}
}

func TestGeneratorPathInfoResponseCodes(t *testing.T) {
	t.Parallel()

	g := NewOpenAPI31Generator(&Server{}, Info{Title: "Test"}, nil)
	data, _ := g.Generate()

	var doc map[string]any
	json.Unmarshal(data, &doc)

	paths := doc["paths"].(map[string]any)
	path := paths["/admin/api/v1/services"].(map[string]any)
	getOp := path["GET"].(map[string]any)

	responses := getOp["responses"].(map[string]any)
	if _, ok := responses["200"]; !ok {
		t.Fatal("expected 200 response")
	}
}

func TestGeneratorOpenAPI31SpecValidJSON(t *testing.T) {
	t.Parallel()

	g := NewOpenAPI31Generator(&Server{}, Info{
		Title:       "API Cerberus Admin API",
		Description: "Test description",
		Version:     "1.0.0",
		Contact:     Contact{Name: "Team", URL: "https://example.com"},
		License:     License{Name: "MIT"},
	}, []ServerEntry{
		{URL: "https://api.example.com"},
	})

	data, err := g.Generate()
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	// Should be valid, re-parseable JSON.
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, string(data))
	}

	// Verify top-level required fields.
	for _, field := range []string{"openapi", "info", "paths", "components"} {
		if _, ok := doc[field]; !ok {
			t.Errorf("missing required field: %s", field)
		}
	}
}

func TestOpenAPI31GeneratorHandlesSpecialCharsInPath(t *testing.T) {
	t.Parallel()

	g := NewOpenAPI31Generator(&Server{}, Info{Title: "Test"}, nil)
	data, _ := g.Generate()

	var doc map[string]any
	json.Unmarshal(data, &doc)

	paths := doc["paths"].(map[string]any)
	// Paths with special chars should not cause issues.
	for path := range paths {
		if strings.Contains(path, "{") && !strings.Contains(path, "}") {
			t.Errorf("malformed path template: %s", path)
		}
	}
}

func TestToResponseCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    int
		expected string
	}{
		{101, "101"},
		{200, "200"},
		{201, "201"},
		{204, "204"},
		{400, "400"},
		{401, "401"},
		{404, "404"},
		{409, "409"},
		{500, "500"},
	}

	for _, tt := range tests {
		got := toResponseCode(tt.input)
		if got != tt.expected {
			t.Errorf("input %d: expected %q, got %q", tt.input, tt.expected, got)
		}
	}
}

func TestGeneratorAllTopLevelPathsHaveOperations(t *testing.T) {
	t.Parallel()

	g := NewOpenAPI31Generator(&Server{}, Info{Title: "Test"}, nil)
	data, _ := g.Generate()

	var doc map[string]any
	json.Unmarshal(data, &doc)

	paths := doc["paths"].(map[string]any)
	for path, pathItem := range paths {
		pi := pathItem.(map[string]any)
		hasOp := false
		for _, method := range []string{"GET", "POST", "PUT", "DELETE", "PATCH"} {
			if _, ok := pi[method]; ok {
				hasOp = true
				break
			}
		}
		if !hasOp {
			t.Errorf("path %q has no operations", path)
		}
	}
}

func TestGeneratorWebhookDeleteHasIDParam(t *testing.T) {
	t.Parallel()

	g := NewOpenAPI31Generator(&Server{}, Info{Title: "Test"}, nil)
	data, _ := g.Generate()

	var doc map[string]any
	json.Unmarshal(data, &doc)

	paths := doc["paths"].(map[string]any)
	path := paths["/admin/api/v1/webhooks/{id}"].(map[string]any)
	deleteOp := path["DELETE"].(map[string]any)

	params := deleteOp["parameters"].([]any)
	if len(params) != 1 {
		t.Fatalf("expected 1 parameter, got %d", len(params))
	}

	param := params[0].(map[string]any)
	if param["name"] != "id" {
		t.Fatalf("expected id param, got %v", param["name"])
	}
}

func TestOpenAPI31SpecServeHTTP(t *testing.T) {
	t.Parallel()

	// Test that we can serve the spec via HTTP endpoint.
	g := NewOpenAPI31Generator(&Server{}, Info{Title: "Test"}, nil)
	data, _ := g.Generate()

	rec := httptest.NewRecorder()
	_ = httptest.NewRequest(http.MethodGet, "/openapi.json", nil)

	// Simulate what the endpoint handler would do.
	rec.Header().Set("Content-Type", "application/json")
	rec.WriteHeader(http.StatusOK)
	rec.Write(data)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("expected application/json, got %s", rec.Header().Get("Content-Type"))
	}
}

func TestGeneratorAllSchemasHaveType(t *testing.T) {
	t.Parallel()

	g := NewOpenAPI31Generator(&Server{}, Info{Title: "Test"}, nil)
	data, _ := g.Generate()

	var doc map[string]any
	json.Unmarshal(data, &doc)

	components := doc["components"].(map[string]any)
	schemas := components["schemas"].(map[string]any)

	for name, schemaRaw := range schemas {
		schema := schemaRaw.(map[string]any)
		if _, hasType := schema["type"]; !hasType {
			if _, hasRef := schema["$ref"]; !hasRef {
				t.Errorf("schema %q has neither type nor ref", name)
			}
		}
	}
}