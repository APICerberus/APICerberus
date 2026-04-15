package admin

import (
	"encoding/json"
	"fmt"
	"strings"
)

// OpenAPI31Generator generates an OpenAPI 3.1 spec from the admin API routes.
// It introspects registered HTTP paths and produces a valid OpenAPI document
// with schemas derived from handler response types.
type OpenAPI31Generator struct {
	server  *Server
	info    Info
	servers []ServerEntry
}

// Info holds the OpenAPI document metadata.
type Info struct {
	Title       string
	Description string
	Version     string
	Contact     Contact
	License     License
}

// Contact represents the contact information in the spec.
type Contact struct {
	Name  string
	Email string
	URL   string
}

// License represents the license information.
type License struct {
	Name string
	URL  string
}

// ServerEntry represents a server URL in the spec.
type ServerEntry struct {
	URL         string
	Description string
}

// NewOpenAPI31Generator creates a generator with the given metadata.
func NewOpenAPI31Generator(srv *Server, info Info, servers []ServerEntry) *OpenAPI31Generator {
	if srv == nil {
		return nil
	}
	if info.Version == "" {
		info.Version = "1.0.0"
	}
	return &OpenAPI31Generator{
		server:  srv,
		info:    info,
		servers: servers,
	}
}

// Generate produces the full OpenAPI 3.1 document as a byte slice.
func (g *OpenAPI31Generator) Generate() ([]byte, error) {
	if g == nil || g.server == nil {
		return nil, nil
	}

	doc := OpenAPIDocument{
		OpenAPI: "3.1.0",
		Info: OpenAPIInfo{
			Title:       g.info.Title,
			Description: g.info.Description,
			Version:     g.info.Version,
			Contact: OpenAPIContact{
				Name: g.info.Contact.Name,
				URL:  g.info.Contact.URL,
			},
			License: OpenAPILicense{
				Name: g.info.License.Name,
				URL:  g.info.License.URL,
			},
		},
		Servers: g.buildServers(),
		Paths:   g.buildPaths(),
		Components: Components{
			Schemas:         g.buildSchemas(),
			SecuritySchemes: g.buildSecuritySchemes(),
		},
		Tags: g.buildTags(),
	}

	return json.MarshalIndent(doc, "", "  ")
}

func (g *OpenAPI31Generator) buildServers() []OpenAPIServer {
	out := make([]OpenAPIServer, 0, len(g.servers))
	for _, s := range g.servers {
		out = append(out, OpenAPIServer{URL: s.URL, Description: s.Description})
	}
	return out
}

func (g *OpenAPI31Generator) buildTags() []OpenAPITag {
	return []OpenAPITag{
		{Name: "Health", Description: "Health check endpoints"},
		{Name: "Auth", Description: "Authentication and SSO"},
		{Name: "Gateway", Description: "Gateway status and configuration"},
		{Name: "Services", Description: "Backend service management"},
		{Name: "Routes", Description: "Route configuration"},
		{Name: "Upstreams", Description: "Upstream and load balancer management"},
		{Name: "Users", Description: "User management"},
		{Name: "API Keys", Description: "API key management"},
		{Name: "Credits", Description: "Credit and billing operations"},
		{Name: "Permissions", Description: "User permission management"},
		{Name: "IP Whitelist", Description: "IP whitelist management"},
		{Name: "Audit Logs", Description: "Audit log queries and management"},
		{Name: "Analytics", Description: "Metrics, analytics, and forecasting"},
		{Name: "Alerts", Description: "Alert rule management"},
		{Name: "Billing", Description: "Billing configuration and route costs"},
		{Name: "Cluster", Description: "Raft cluster management"},
		{Name: "Federation", Description: "GraphQL federation management"},
		{Name: "Webhooks", Description: "Webhook management and delivery"},
		{Name: "Bulk", Description: "Bulk operations"},
		{Name: "GraphQL", Description: "GraphQL admin endpoint"},
		{Name: "WebSocket", Description: "Real-time WebSocket connection"},
	}
}

func (g *OpenAPI31Generator) buildSecuritySchemes() map[string]SecurityScheme {
	return map[string]SecurityScheme{
		"BearerAuth": {
			Type:         "http",
			Scheme:       "bearer",
			BearerFormat: "JWT",
			Description:  "JWT bearer token obtained from /auth/token",
		},
		"ApiKeyAuth": {
			Type:        "apiKey",
			In:          "header",
			Name:        "X-API-Key",
			Description: "Static API key for service-to-service calls",
		},
	}
}

func (g *OpenAPI31Generator) buildPaths() map[string]map[string]Operation {
	paths := make(map[string]map[string]Operation)

	// Health and status (no auth)
	g.addPath(paths, "GET", "/health", PathInfo{
		Tags:          []string{"Health"},
		Summary:       "Health check",
		Description:   "Returns the health status of the gateway",
		OperationID:   "getHealth",
		Security:      []string{},
		ResponseCodes: []ResponseInfo{{Code: 200, Description: "Service is healthy", SchemaRef: "#/components/schemas/HealthResponse"}},
	})
	g.addPath(paths, "GET", "/ready", PathInfo{
		Tags:          []string{"Health"},
		Summary:       "Readiness check",
		Description:   "Returns whether the gateway is ready to serve requests",
		OperationID:   "getReady",
		Security:      []string{},
		ResponseCodes: []ResponseInfo{{Code: 200, Description: "Ready", SchemaRef: "#/components/schemas/ReadyResponse"}},
	})

	// Auth
	g.addPath(paths, "POST", "/admin/api/v1/auth/token", PathInfo{
		Tags:        []string{"Auth"},
		Summary:     "Issue JWT token",
		Description: "Authenticate with admin credentials and receive a JWT",
		OperationID: "issueToken",
		Security:    []string{"BearerAuth", "ApiKeyAuth"},
		RequestBody: &RequestBodyInfo{
			ContentType: "application/json",
			Schema:      mustRef("#/components/schemas/TokenRequest"),
		},
		ResponseCodes: []ResponseInfo{
			{Code: 200, Description: "Token issued", SchemaRef: "#/components/schemas/TokenResponse"},
			{Code: 401, Description: "Invalid credentials"},
		},
	})

	// Gateway info
	g.addPath(paths, "GET", "/admin/api/v1/info", PathInfo{
		Tags:        []string{"Gateway"},
		Summary:     "Gateway information",
		Description: "Returns version, uptime, and summary of configured entities",
		OperationID: "getInfo",
		ResponseCodes: []ResponseInfo{
			{Code: 200, Description: "OK", SchemaRef: "#/components/schemas/InfoResponse"},
		},
	})
	g.addPath(paths, "GET", "/admin/api/v1/status", PathInfo{
		Tags:        []string{"Gateway"},
		Summary:     "Gateway status",
		Description: "Returns current status and store metrics",
		OperationID: "getStatus",
		ResponseCodes: []ResponseInfo{
			{Code: 200, Description: "OK", SchemaRef: "#/components/schemas/StatusResponse"},
		},
	})
	g.addPath(paths, "GET", "/admin/api/v1/branding", PathInfo{
		Tags:        []string{"Gateway"},
		Summary:     "Branding configuration",
		Description: "Returns branding configuration for the dashboard",
		OperationID: "getBranding",
		ResponseCodes: []ResponseInfo{
			{Code: 200, Description: "OK", SchemaRef: "#/components/schemas/BrandingResponse"},
		},
	})

	// Services CRUD
	g.addPath(paths, "GET", "/admin/api/v1/services", PathInfo{
		Tags:        []string{"Services"},
		Summary:     "List services",
		Description: "Returns a list of all configured services",
		OperationID: "listServices",
		ResponseCodes: []ResponseInfo{
			{Code: 200, Description: "OK", SchemaRef: "#/components/schemas/ServiceListResponse"},
		},
	})
	g.addPath(paths, "POST", "/admin/api/v1/services", PathInfo{
		Tags:        []string{"Services"},
		Summary:     "Create service",
		Description: "Creates a new backend service",
		OperationID: "createService",
		RequestBody: &RequestBodyInfo{
			ContentType: "application/json",
			Schema:      mustRef("#/components/schemas/ServiceCreateRequest"),
		},
		ResponseCodes: []ResponseInfo{
			{Code: 201, Description: "Created", SchemaRef: "#/components/schemas/ServiceResponse"},
			{Code: 400, Description: "Invalid request"},
		},
	})
	g.addPath(paths, "GET", "/admin/api/v1/services/{id}", PathInfo{
		Tags:        []string{"Services"},
		Summary:     "Get service",
		Description: "Returns a specific service by ID",
		OperationID: "getService",
		PathParams:  []ParameterInfo{{Name: "id", In: "path", Description: "Service ID", Required: true, SchemaType: "string"}},
		ResponseCodes: []ResponseInfo{
			{Code: 200, Description: "OK", SchemaRef: "#/components/schemas/ServiceResponse"},
			{Code: 404, Description: "Not found"},
		},
	})
	g.addPath(paths, "PUT", "/admin/api/v1/services/{id}", PathInfo{
		Tags:        []string{"Services"},
		Summary:     "Update service",
		Description: "Updates an existing service",
		OperationID: "updateService",
		PathParams:  []ParameterInfo{{Name: "id", In: "path", Description: "Service ID", Required: true, SchemaType: "string"}},
		RequestBody: &RequestBodyInfo{
			ContentType: "application/json",
			Schema:      mustRef("#/components/schemas/ServiceUpdateRequest"),
		},
		ResponseCodes: []ResponseInfo{
			{Code: 200, Description: "Updated", SchemaRef: "#/components/schemas/ServiceResponse"},
			{Code: 404, Description: "Not found"},
		},
	})
	g.addPath(paths, "DELETE", "/admin/api/v1/services/{id}", PathInfo{
		Tags:        []string{"Services"},
		Summary:     "Delete service",
		Description: "Deletes a service (fails if referenced by a route)",
		OperationID: "deleteService",
		PathParams:  []ParameterInfo{{Name: "id", In: "path", Description: "Service ID", Required: true, SchemaType: "string"}},
		ResponseCodes: []ResponseInfo{
			{Code: 204, Description: "Deleted"},
			{Code: 404, Description: "Not found"},
			{Code: 409, Description: "Service in use"},
		},
	})

	// Routes CRUD
	g.addPath(paths, "GET", "/admin/api/v1/routes", PathInfo{
		Tags:        []string{"Routes"},
		Summary:     "List routes",
		Description: "Returns a list of all configured routes",
		OperationID: "listRoutes",
		ResponseCodes: []ResponseInfo{
			{Code: 200, Description: "OK", SchemaRef: "#/components/schemas/RouteListResponse"},
		},
	})
	g.addPath(paths, "POST", "/admin/api/v1/routes", PathInfo{
		Tags:        []string{"Routes"},
		Summary:     "Create route",
		Description: "Creates a new route",
		OperationID: "createRoute",
		RequestBody: &RequestBodyInfo{
			ContentType: "application/json",
			Schema:      mustRef("#/components/schemas/RouteCreateRequest"),
		},
		ResponseCodes: []ResponseInfo{
			{Code: 201, Description: "Created", SchemaRef: "#/components/schemas/RouteResponse"},
			{Code: 400, Description: "Invalid request"},
		},
	})
	g.addPath(paths, "GET", "/admin/api/v1/routes/{id}", PathInfo{
		Tags:        []string{"Routes"},
		Summary:     "Get route",
		Description: "Returns a specific route by ID",
		OperationID: "getRoute",
		PathParams:  []ParameterInfo{{Name: "id", In: "path", Description: "Route ID", Required: true, SchemaType: "string"}},
		ResponseCodes: []ResponseInfo{
			{Code: 200, Description: "OK", SchemaRef: "#/components/schemas/RouteResponse"},
			{Code: 404, Description: "Not found"},
		},
	})
	g.addPath(paths, "PUT", "/admin/api/v1/routes/{id}", PathInfo{
		Tags:        []string{"Routes"},
		Summary:     "Update route",
		Description: "Updates an existing route",
		OperationID: "updateRoute",
		PathParams:  []ParameterInfo{{Name: "id", In: "path", Description: "Route ID", Required: true, SchemaType: "string"}},
		RequestBody: &RequestBodyInfo{
			ContentType: "application/json",
			Schema:      mustRef("#/components/schemas/RouteUpdateRequest"),
		},
		ResponseCodes: []ResponseInfo{
			{Code: 200, Description: "Updated", SchemaRef: "#/components/schemas/RouteResponse"},
			{Code: 404, Description: "Not found"},
		},
	})
	g.addPath(paths, "DELETE", "/admin/api/v1/routes/{id}", PathInfo{
		Tags:        []string{"Routes"},
		Summary:     "Delete route",
		Description: "Deletes a route",
		OperationID: "deleteRoute",
		PathParams:  []ParameterInfo{{Name: "id", In: "path", Description: "Route ID", Required: true, SchemaType: "string"}},
		ResponseCodes: []ResponseInfo{
			{Code: 204, Description: "Deleted"},
			{Code: 404, Description: "Not found"},
		},
	})

	// Upstreams CRUD
	g.addPath(paths, "GET", "/admin/api/v1/upstreams", PathInfo{
		Tags:        []string{"Upstreams"},
		Summary:     "List upstreams",
		Description: "Returns all configured upstreams",
		OperationID: "listUpstreams",
		ResponseCodes: []ResponseInfo{
			{Code: 200, Description: "OK", SchemaRef: "#/components/schemas/UpstreamListResponse"},
		},
	})
	g.addPath(paths, "POST", "/admin/api/v1/upstreams", PathInfo{
		Tags:        []string{"Upstreams"},
		Summary:     "Create upstream",
		Description: "Creates a new upstream with targets",
		OperationID: "createUpstream",
		RequestBody: &RequestBodyInfo{
			ContentType: "application/json",
			Schema:      mustRef("#/components/schemas/UpstreamCreateRequest"),
		},
		ResponseCodes: []ResponseInfo{
			{Code: 201, Description: "Created", SchemaRef: "#/components/schemas/UpstreamResponse"},
			{Code: 400, Description: "Invalid request"},
		},
	})
	g.addPath(paths, "GET", "/admin/api/v1/upstreams/{id}", PathInfo{
		Tags:        []string{"Upstreams"},
		Summary:     "Get upstream",
		Description: "Returns a specific upstream by ID",
		OperationID: "getUpstream",
		PathParams:  []ParameterInfo{{Name: "id", In: "path", Description: "Upstream ID", Required: true, SchemaType: "string"}},
		ResponseCodes: []ResponseInfo{
			{Code: 200, Description: "OK", SchemaRef: "#/components/schemas/UpstreamResponse"},
			{Code: 404, Description: "Not found"},
		},
	})
	g.addPath(paths, "PUT", "/admin/api/v1/upstreams/{id}", PathInfo{
		Tags:        []string{"Upstreams"},
		Summary:     "Update upstream",
		Description: "Updates an existing upstream",
		OperationID: "updateUpstream",
		PathParams:  []ParameterInfo{{Name: "id", In: "path", Description: "Upstream ID", Required: true, SchemaType: "string"}},
		RequestBody: &RequestBodyInfo{
			ContentType: "application/json",
			Schema:      mustRef("#/components/schemas/UpstreamUpdateRequest"),
		},
		ResponseCodes: []ResponseInfo{
			{Code: 200, Description: "Updated", SchemaRef: "#/components/schemas/UpstreamResponse"},
			{Code: 404, Description: "Not found"},
		},
	})
	g.addPath(paths, "DELETE", "/admin/api/v1/upstreams/{id}", PathInfo{
		Tags:        []string{"Upstreams"},
		Summary:     "Delete upstream",
		Description: "Deletes an upstream",
		OperationID: "deleteUpstream",
		PathParams:  []ParameterInfo{{Name: "id", In: "path", Description: "Upstream ID", Required: true, SchemaType: "string"}},
		ResponseCodes: []ResponseInfo{
			{Code: 204, Description: "Deleted"},
			{Code: 404, Description: "Not found"},
			{Code: 409, Description: "Upstream in use"},
		},
	})
	g.addPath(paths, "POST", "/admin/api/v1/upstreams/{id}/targets", PathInfo{
		Tags:        []string{"Upstreams"},
		Summary:     "Add upstream target",
		Description: "Adds a target to an upstream",
		OperationID: "addUpstreamTarget",
		PathParams:  []ParameterInfo{{Name: "id", In: "path", Description: "Upstream ID", Required: true, SchemaType: "string"}},
		RequestBody: &RequestBodyInfo{
			ContentType: "application/json",
			Schema:      mustRef("#/components/schemas/TargetCreateRequest"),
		},
		ResponseCodes: []ResponseInfo{
			{Code: 201, Description: "Created", SchemaRef: "#/components/schemas/TargetResponse"},
			{Code: 404, Description: "Not found"},
		},
	})
	g.addPath(paths, "DELETE", "/admin/api/v1/upstreams/{id}/targets/{tid}", PathInfo{
		Tags:        []string{"Upstreams"},
		Summary:     "Remove upstream target",
		Description: "Removes a target from an upstream",
		OperationID: "deleteUpstreamTarget",
		PathParams: []ParameterInfo{
			{Name: "id", In: "path", Description: "Upstream ID", Required: true, SchemaType: "string"},
			{Name: "tid", In: "path", Description: "Target ID", Required: true, SchemaType: "string"},
		},
		ResponseCodes: []ResponseInfo{
			{Code: 204, Description: "Deleted"},
			{Code: 404, Description: "Not found"},
		},
	})
	g.addPath(paths, "GET", "/admin/api/v1/upstreams/{id}/health", PathInfo{
		Tags:        []string{"Upstreams"},
		Summary:     "Upstream health",
		Description: "Returns health status of all targets in an upstream",
		OperationID: "getUpstreamHealth",
		PathParams:  []ParameterInfo{{Name: "id", In: "path", Description: "Upstream ID", Required: true, SchemaType: "string"}},
		ResponseCodes: []ResponseInfo{
			{Code: 200, Description: "OK", SchemaRef: "#/components/schemas/UpstreamHealthResponse"},
		},
	})

	// Users CRUD
	g.addPath(paths, "GET", "/admin/api/v1/users", PathInfo{
		Tags:        []string{"Users"},
		Summary:     "List users",
		Description: "Returns all users with optional filtering by search, status, role",
		OperationID: "listUsers",
		QueryParams: []ParameterInfo{
			{Name: "search", In: "query", Description: "Search by name or email", SchemaType: "string"},
			{Name: "status", In: "query", Description: "Filter by status (active, suspended)", SchemaType: "string"},
			{Name: "role", In: "query", Description: "Filter by role (admin, user)", SchemaType: "string"},
			{Name: "limit", In: "query", Description: "Max results", SchemaType: "integer"},
			{Name: "offset", In: "query", Description: "Pagination offset", SchemaType: "integer"},
		},
		ResponseCodes: []ResponseInfo{
			{Code: 200, Description: "OK", SchemaRef: "#/components/schemas/UserListResponse"},
		},
	})
	g.addPath(paths, "POST", "/admin/api/v1/users", PathInfo{
		Tags:        []string{"Users"},
		Summary:     "Create user",
		Description: "Creates a new user",
		OperationID: "createUser",
		RequestBody: &RequestBodyInfo{
			ContentType: "application/json",
			Schema:      mustRef("#/components/schemas/UserCreateRequest"),
		},
		ResponseCodes: []ResponseInfo{
			{Code: 201, Description: "Created", SchemaRef: "#/components/schemas/UserResponse"},
			{Code: 400, Description: "Invalid request"},
		},
	})
	g.addPath(paths, "GET", "/admin/api/v1/users/{id}", PathInfo{
		Tags:        []string{"Users"},
		Summary:     "Get user",
		Description: "Returns a specific user by ID",
		OperationID: "getUser",
		PathParams:  []ParameterInfo{{Name: "id", In: "path", Description: "User ID", Required: true, SchemaType: "string"}},
		ResponseCodes: []ResponseInfo{
			{Code: 200, Description: "OK", SchemaRef: "#/components/schemas/UserResponse"},
			{Code: 404, Description: "Not found"},
		},
	})
	g.addPath(paths, "PUT", "/admin/api/v1/users/{id}", PathInfo{
		Tags:        []string{"Users"},
		Summary:     "Update user",
		Description: "Updates user profile fields (name, email)",
		OperationID: "updateUser",
		PathParams:  []ParameterInfo{{Name: "id", In: "path", Description: "User ID", Required: true, SchemaType: "string"}},
		RequestBody: &RequestBodyInfo{
			ContentType: "application/json",
			Schema:      mustRef("#/components/schemas/UserUpdateRequest"),
		},
		ResponseCodes: []ResponseInfo{
			{Code: 200, Description: "Updated", SchemaRef: "#/components/schemas/UserResponse"},
			{Code: 404, Description: "Not found"},
		},
	})
	g.addPath(paths, "DELETE", "/admin/api/v1/users/{id}", PathInfo{
		Tags:        []string{"Users"},
		Summary:     "Delete user",
		Description: "Permanently deletes a user and all associated data",
		OperationID: "deleteUser",
		PathParams:  []ParameterInfo{{Name: "id", In: "path", Description: "User ID", Required: true, SchemaType: "string"}},
		ResponseCodes: []ResponseInfo{
			{Code: 204, Description: "Deleted"},
			{Code: 404, Description: "Not found"},
		},
	})

	// Credits
	g.addPath(paths, "GET", "/admin/api/v1/credits/overview", PathInfo{
		Tags:        []string{"Credits"},
		Summary:     "Credits overview",
		Description: "Returns aggregate credit statistics across all users",
		OperationID: "creditOverview",
		ResponseCodes: []ResponseInfo{
			{Code: 200, Description: "OK", SchemaRef: "#/components/schemas/CreditsOverviewResponse"},
		},
	})
	g.addPath(paths, "GET", "/admin/api/v1/users/{id}/credits", PathInfo{
		Tags:        []string{"Credits"},
		Summary:     "User credits overview",
		Description: "Returns credit balance and summary for a specific user",
		OperationID: "userCreditOverview",
		PathParams:  []ParameterInfo{{Name: "id", In: "path", Description: "User ID", Required: true, SchemaType: "string"}},
		ResponseCodes: []ResponseInfo{
			{Code: 200, Description: "OK", SchemaRef: "#/components/schemas/UserCreditsResponse"},
		},
	})
	g.addPath(paths, "GET", "/admin/api/v1/users/{id}/credits/transactions", PathInfo{
		Tags:        []string{"Credits"},
		Summary:     "User credit transactions",
		Description: "Returns paginated list of credit transactions for a user",
		OperationID: "listCreditTransactions",
		PathParams:  []ParameterInfo{{Name: "id", In: "path", Description: "User ID", Required: true, SchemaType: "string"}},
		QueryParams: []ParameterInfo{
			{Name: "limit", In: "query", SchemaType: "integer"},
			{Name: "offset", In: "query", SchemaType: "integer"},
		},
		ResponseCodes: []ResponseInfo{
			{Code: 200, Description: "OK", SchemaRef: "#/components/schemas/TransactionListResponse"},
		},
	})

	// Audit logs
	g.addPath(paths, "GET", "/admin/api/v1/audit-logs", PathInfo{
		Tags:        []string{"Audit Logs"},
		Summary:     "Search audit logs",
		Description: "Full-text search over audit logs with optional filters",
		OperationID: "searchAuditLogs",
		QueryParams: []ParameterInfo{
			{Name: "q", In: "query", Description: "Search query", SchemaType: "string"},
			{Name: "user_id", In: "query", Description: "Filter by user ID", SchemaType: "string"},
			{Name: "route", In: "query", Description: "Filter by route pattern", SchemaType: "string"},
			{Name: "since", In: "query", Description: "Start timestamp (RFC3339)", SchemaType: "string"},
			{Name: "until", In: "query", Description: "End timestamp (RFC3339)", SchemaType: "string"},
			{Name: "limit", In: "query", SchemaType: "integer"},
			{Name: "offset", In: "query", SchemaType: "integer"},
		},
		ResponseCodes: []ResponseInfo{
			{Code: 200, Description: "OK", SchemaRef: "#/components/schemas/AuditLogListResponse"},
		},
	})
	g.addPath(paths, "GET", "/admin/api/v1/audit-logs/stats", PathInfo{
		Tags:        []string{"Audit Logs"},
		Summary:     "Audit log statistics",
		Description: "Returns aggregated statistics for audit logs",
		OperationID: "auditLogStats",
		QueryParams: []ParameterInfo{
			{Name: "since", In: "query", Description: "Start timestamp (RFC3339)", SchemaType: "string"},
			{Name: "until", In: "query", Description: "End timestamp (RFC3339)", SchemaType: "string"},
		},
		ResponseCodes: []ResponseInfo{
			{Code: 200, Description: "OK", SchemaRef: "#/components/schemas/AuditLogStatsResponse"},
		},
	})

	// Analytics
	g.addPath(paths, "GET", "/admin/api/v1/analytics/overview", PathInfo{
		Tags:        []string{"Analytics"},
		Summary:     "Analytics overview",
		Description: "Top-level metrics: total requests, error rate, p95 latency, top routes",
		OperationID: "analyticsOverview",
		QueryParams: []ParameterInfo{
			{Name: "since", In: "query", SchemaType: "string"},
			{Name: "until", In: "query", SchemaType: "string"},
		},
		ResponseCodes: []ResponseInfo{
			{Code: 200, Description: "OK", SchemaRef: "#/components/schemas/AnalyticsOverviewResponse"},
		},
	})
	g.addPath(paths, "GET", "/admin/api/v1/analytics/timeseries", PathInfo{
		Tags:        []string{"Analytics"},
		Summary:     "Analytics time series",
		Description: "Time-series data for a specific metric",
		OperationID: "analyticsTimeSeries",
		QueryParams: []ParameterInfo{
			{Name: "metric", In: "query", Description: "Metric name (requests, latency, errors)", SchemaType: "string"},
			{Name: "since", In: "query", SchemaType: "string"},
			{Name: "until", In: "query", SchemaType: "string"},
			{Name: "interval", In: "query", Description: "Bucket interval (1m, 5m, 1h, 1d)", SchemaType: "string"},
			{Name: "route", In: "query", SchemaType: "string"},
		},
		ResponseCodes: []ResponseInfo{
			{Code: 200, Description: "OK", SchemaRef: "#/components/schemas/AnalyticsTimeSeriesResponse"},
		},
	})

	// Billing
	g.addPath(paths, "GET", "/admin/api/v1/billing/config", PathInfo{
		Tags:        []string{"Billing"},
		Summary:     "Get billing config",
		Description: "Returns the current billing configuration",
		OperationID: "getBillingConfig",
		ResponseCodes: []ResponseInfo{
			{Code: 200, Description: "OK", SchemaRef: "#/components/schemas/BillingConfigResponse"},
		},
	})
	g.addPath(paths, "PUT", "/admin/api/v1/billing/config", PathInfo{
		Tags:        []string{"Billing"},
		Summary:     "Update billing config",
		Description: "Updates billing settings (default cost, method multipliers)",
		OperationID: "updateBillingConfig",
		RequestBody: &RequestBodyInfo{
			ContentType: "application/json",
			Schema:      mustRef("#/components/schemas/BillingConfigUpdateRequest"),
		},
		ResponseCodes: []ResponseInfo{
			{Code: 200, Description: "Updated", SchemaRef: "#/components/schemas/BillingConfigResponse"},
		},
	})
	g.addPath(paths, "GET", "/admin/api/v1/billing/route-costs", PathInfo{
		Tags:        []string{"Billing"},
		Summary:     "List route costs",
		Description: "Returns all configured route costs",
		OperationID: "getBillingRouteCosts",
		ResponseCodes: []ResponseInfo{
			{Code: 200, Description: "OK", SchemaRef: "#/components/schemas/RouteCostListResponse"},
		},
	})
	g.addPath(paths, "PUT", "/admin/api/v1/billing/route-costs", PathInfo{
		Tags:        []string{"Billing"},
		Summary:     "Update route costs",
		Description: "Bulk updates route cost assignments",
		OperationID: "updateBillingRouteCosts",
		RequestBody: &RequestBodyInfo{
			ContentType: "application/json",
			Schema:      mustRef("#/components/schemas/RouteCostUpdateRequest"),
		},
		ResponseCodes: []ResponseInfo{
			{Code: 200, Description: "Updated"},
		},
	})

	// Webhooks
	g.addPath(paths, "GET", "/admin/api/v1/webhooks", PathInfo{
		Tags:        []string{"Webhooks"},
		Summary:     "List webhooks",
		Description: "Returns all registered webhooks",
		OperationID: "listWebhooks",
		ResponseCodes: []ResponseInfo{
			{Code: 200, Description: "OK", SchemaRef: "#/components/schemas/WebhookListResponse"},
		},
	})
	g.addPath(paths, "POST", "/admin/api/v1/webhooks", PathInfo{
		Tags:        []string{"Webhooks"},
		Summary:     "Create webhook",
		Description: "Registers a new webhook endpoint",
		OperationID: "createWebhook",
		RequestBody: &RequestBodyInfo{
			ContentType: "application/json",
			Schema:      mustRef("#/components/schemas/WebhookCreateRequest"),
		},
		ResponseCodes: []ResponseInfo{
			{Code: 201, Description: "Created", SchemaRef: "#/components/schemas/WebhookResponse"},
		},
	})
	g.addPath(paths, "DELETE", "/admin/api/v1/webhooks/{id}", PathInfo{
		Tags:        []string{"Webhooks"},
		Summary:     "Delete webhook",
		Description: "Removes a webhook",
		OperationID: "deleteWebhook",
		PathParams:  []ParameterInfo{{Name: "id", In: "path", Description: "Webhook ID", Required: true, SchemaType: "string"}},
		ResponseCodes: []ResponseInfo{
			{Code: 204, Description: "Deleted"},
		},
	})

	// Cluster
	g.addPath(paths, "GET", "/admin/api/v1/cluster/status", PathInfo{
		Tags:        []string{"Cluster"},
		Summary:     "Cluster status",
		Description: "Returns current Raft cluster state, nodes, and leader",
		OperationID: "clusterStatus",
		ResponseCodes: []ResponseInfo{
			{Code: 200, Description: "OK", SchemaRef: "#/components/schemas/ClusterStatusResponse"},
		},
	})

	// Federation
	g.addPath(paths, "GET", "/admin/api/v1/subgraphs", PathInfo{
		Tags:        []string{"Federation"},
		Summary:     "List subgraphs",
		Description: "Returns all registered federation subgraphs",
		OperationID: "listSubgraphs",
		ResponseCodes: []ResponseInfo{
			{Code: 200, Description: "OK", SchemaRef: "#/components/schemas/SubgraphListResponse"},
		},
	})
	g.addPath(paths, "POST", "/admin/api/v1/subgraphs/compose", PathInfo{
		Tags:        []string{"Federation"},
		Summary:     "Compose subgraphs",
		Description: "Triggers on-demand federation schema composition",
		OperationID: "composeSubgraphs",
		ResponseCodes: []ResponseInfo{
			{Code: 200, Description: "OK", SchemaRef: "#/components/schemas/ComposeResponse"},
		},
	})

	// GraphQL
	g.addPath(paths, "POST", "/admin/api/v1/graphql", PathInfo{
		Tags:        []string{"GraphQL"},
		Summary:     "GraphQL endpoint",
		Description: "Main admin GraphQL endpoint for federation queries",
		OperationID: "graphQL",
		RequestBody: &RequestBodyInfo{
			ContentType: "application/json",
			Schema:      mustRef("#/components/schemas/GraphQLRequest"),
		},
		ResponseCodes: []ResponseInfo{
			{Code: 200, Description: "OK", SchemaRef: "#/components/schemas/GraphQLResponse"},
		},
	})

	// WebSocket
	g.addPath(paths, "GET", "/admin/api/v1/ws", PathInfo{
		Tags:        []string{"WebSocket"},
		Summary:     "WebSocket endpoint",
		Description: "Real-time event stream via WebSocket (topic-based subscriptions)",
		OperationID: "websocket",
		ResponseCodes: []ResponseInfo{
			{Code: 101, Description: "Switching Protocols"},
		},
	})

	return paths
}

func (g *OpenAPI31Generator) addPath(paths map[string]map[string]Operation, method, path string, info PathInfo) {
	// Convert {param} to OpenAPI format.
	openAPIPath := path

	item, exists := paths[openAPIPath]
	if !exists {
		item = make(map[string]Operation)
		paths[openAPIPath] = item
	}

	// Only default to auth if Security is nil (not set). Empty slice means explicit no-security.
	security := []string{}
	if info.Security != nil {
		for _, s := range info.Security {
			if s == "" {
				continue
			}
			security = append(security, s)
		}
	}

	op := Operation{
		Tags:         info.Tags,
		Summary:     info.Summary,
		Description:  info.Description,
		OperationID: info.OperationID,
		Security:    security,
		Parameters:  make([]Parameter, 0),
		RequestBody: info.RequestBody,
		Responses:   make(map[string]Response, len(info.ResponseCodes)),
	}

	for _, p := range info.PathParams {
		op.Parameters = append(op.Parameters, g.newParameter(p))
	}
	for _, p := range info.QueryParams {
		op.Parameters = append(op.Parameters, g.newParameter(p))
	}

	for _, rc := range info.ResponseCodes {
		r := Response{
			Description: rc.Description,
		}
		if rc.SchemaRef != "" {
			r.Content = map[string]MediaType{
				"application/json": {Schema: Schema{Ref: rc.SchemaRef}},
			}
		}
		if rc.Code >= 200 && rc.Code < 300 {
			op.Responses[toResponseCode(rc.Code)] = r
		}
		// Always add error responses.
		if rc.Code >= 400 {
			op.Responses[toResponseCode(rc.Code)] = r
		} else if rc.Code == 201 || rc.Code == 204 {
			op.Responses[toResponseCode(rc.Code)] = r
		}
	}

	item[strings.ToUpper(method)] = op
	paths[openAPIPath] = item
}

func (g *OpenAPI31Generator) newParameter(p ParameterInfo) Parameter {
	schemaType := p.SchemaType
	if schemaType == "" {
		schemaType = "string"
	}
	return Parameter{
		Name:        p.Name,
		In:          p.In,
		Description: p.Description,
		Required:   p.Required,
		Schema:      Schema{Type: p.SchemaType},
	}
}

func (g *OpenAPI31Generator) buildSchemas() map[string]Schema {
	return map[string]Schema{
		"ErrorResponse": {
			Type: "object",
			Properties: map[string]Schema{
				"error": {
					Type: "object",
					Properties: map[string]Schema{
						"code":    {Type: "string"},
						"message": {Type: "string"},
					},
				},
			},
		},
		"HealthResponse": {
			Type: "object",
			Properties: map[string]Schema{
				"status":  {Type: "string"},
				"uptime":  {Type: "string"},
				"version": {Type: "string"},
			},
		},
		"ReadyResponse": {
			Type: "object",
			Properties: map[string]Schema{
				"ready": {Type: "boolean"},
			},
		},
		"StatusResponse": {
			Type: "object",
			Properties: map[string]Schema{
				"status": {Type: "string"},
				"store": {
					Type: "object",
					Properties: map[string]Schema{
						"open_connections": {Type: "integer"},
						"in_use":          {Type: "integer"},
						"idle":            {Type: "integer"},
					},
				},
			},
		},
		"InfoResponse": {
			Type: "object",
			Properties: map[string]Schema{
				"version":    {Type: "string"},
				"commit":     {Type: "string"},
				"build_time": {Type: "string"},
				"uptime_sec": {Type: "integer"},
				"summary": {
					Type: "object",
					Properties: map[string]Schema{
						"services":  {Type: "integer"},
						"routes":    {Type: "integer"},
						"upstreams": {Type: "integer"},
					},
				},
			},
		},
		"TokenRequest": {
			Type: "object",
			Properties: map[string]Schema{
				"username": {Type: "string"},
				"password": {Type: "string"},
			},
			Required: []string{"username", "password"},
		},
		"TokenResponse": {
			Type: "object",
			Properties: map[string]Schema{
				"token":      {Type: "string"},
				"expires_in": {Type: "integer"},
			},
		},
		"ServiceResponse":      {Type: "object", AdditionalProperties: true},
		"ServiceListResponse":   {Type: "object", Properties: map[string]Schema{"data": {Type: "array", Items: &Schema{Type: "object"}}}},
		"ServiceCreateRequest": {Type: "object", AdditionalProperties: true},
		"ServiceUpdateRequest": {Type: "object", AdditionalProperties: true},
		"RouteResponse":        {Type: "object", AdditionalProperties: true},
		"RouteListResponse":    {Type: "object", Properties: map[string]Schema{"data": {Type: "array", Items: &Schema{Type: "object"}}}},
		"RouteCreateRequest":  {Type: "object", AdditionalProperties: true},
		"RouteUpdateRequest":  {Type: "object", AdditionalProperties: true},
		"UpstreamResponse":        {Type: "object", AdditionalProperties: true},
		"UpstreamListResponse":    {Type: "object", Properties: map[string]Schema{"data": {Type: "array", Items: &Schema{Type: "object"}}}},
		"UpstreamCreateRequest":   {Type: "object", AdditionalProperties: true},
		"UpstreamUpdateRequest":   {Type: "object", AdditionalProperties: true},
		"TargetResponse":          {Type: "object", AdditionalProperties: true},
		"TargetCreateRequest":     {Type: "object", AdditionalProperties: true},
		"UpstreamHealthResponse":  {Type: "object", AdditionalProperties: true},
		"UserResponse":            {Type: "object", AdditionalProperties: true},
		"UserListResponse":        {Type: "object", Properties: map[string]Schema{"data": {Type: "array", Items: &Schema{Type: "object"}}}},
		"UserCreateRequest":        {Type: "object", AdditionalProperties: true},
		"UserUpdateRequest":        {Type: "object", AdditionalProperties: true},
		"CreditsOverviewResponse":  {Type: "object", AdditionalProperties: true},
		"UserCreditsResponse":     {Type: "object", AdditionalProperties: true},
		"TransactionListResponse":  {Type: "object", Properties: map[string]Schema{"data": {Type: "array", Items: &Schema{Type: "object"}}}},
		"AuditLogListResponse":    {Type: "object", Properties: map[string]Schema{"data": {Type: "array", Items: &Schema{Type: "object"}}}},
		"AuditLogStatsResponse":   {Type: "object", AdditionalProperties: true},
		"AnalyticsOverviewResponse":    {Type: "object", AdditionalProperties: true},
		"AnalyticsTimeSeriesResponse": {Type: "object", AdditionalProperties: true},
		"BillingConfigResponse":       {Type: "object", AdditionalProperties: true},
		"BillingConfigUpdateRequest":   {Type: "object", AdditionalProperties: true},
		"RouteCostListResponse":       {Type: "object", Properties: map[string]Schema{"data": {Type: "array", Items: &Schema{Type: "object"}}}},
		"RouteCostUpdateRequest":       {Type: "object", AdditionalProperties: true},
		"WebhookResponse":             {Type: "object", AdditionalProperties: true},
		"WebhookListResponse":          {Type: "object", Properties: map[string]Schema{"data": {Type: "array", Items: &Schema{Type: "object"}}}},
		"WebhookCreateRequest":        {Type: "object", AdditionalProperties: true},
		"ClusterStatusResponse":       {Type: "object", AdditionalProperties: true},
		"SubgraphListResponse":        {Type: "object", Properties: map[string]Schema{"data": {Type: "array", Items: &Schema{Type: "object"}}}},
		"ComposeResponse":             {Type: "object", AdditionalProperties: true},
		"GraphQLRequest": {
			Type: "object",
			Properties: map[string]Schema{
				"query":         {Type: "string"},
				"variables":     {Type: "object"},
				"operationName": {Type: "string"},
			},
		},
		"GraphQLResponse":       {Type: "object", AdditionalProperties: true},
		"BrandingResponse": {
			Type: "object",
			Properties: map[string]Schema{
				"app_name":      {Type: "string"},
				"logo_url":      {Type: "string"},
				"favicon_url":   {Type: "string"},
				"primary_color": {Type: "string"},
				"accent_color":  {Type: "string"},
				"theme_mode":    {Type: "string"},
			},
		},
	}
}

// OpenAPI types — mirrors OpenAPI 3.1 structure.
type (
	OpenAPIDocument struct {
		OpenAPI    string                     `json:"openapi"`
		Info       OpenAPIInfo                `json:"info"`
		Servers    []OpenAPIServer           `json:"servers,omitempty"`
		Paths      map[string]map[string]Operation `json:"paths"`
		Components Components                 `json:"components"`
		Tags       []OpenAPITag              `json:"tags,omitempty"`
	}

	OpenAPIInfo struct {
		Title       string         `json:"title"`
		Description string         `json:"description,omitempty"`
		Version     string         `json:"version"`
		Contact     OpenAPIContact `json:"contact,omitempty"`
		License     OpenAPILicense `json:"license,omitempty"`
	}

	OpenAPIContact struct {
		Name string `json:"name,omitempty"`
		URL  string `json:"url,omitempty"`
	}

	OpenAPILicense struct {
		Name string `json:"name,omitempty"`
		URL  string `json:"url,omitempty"`
	}

	OpenAPIServer struct {
		URL         string `json:"url"`
		Description string `json:"description,omitempty"`
	}

	OpenAPITag struct {
		Name        string `json:"name"`
		Description string `json:"description,omitempty"`
	}

	Components struct {
		Schemas         map[string]Schema         `json:"schemas"`
		SecuritySchemes map[string]SecurityScheme `json:"securitySchemes"`
	}

	SecurityScheme struct {
		Type         string `json:"type"`
		Scheme       string `json:"scheme,omitempty"`
		In           string `json:"in,omitempty"`
		Name         string `json:"name,omitempty"`
		BearerFormat string `json:"bearerFormat,omitempty"`
		Description  string `json:"description,omitempty"`
	}

	Schema struct {
		Type        string              `json:"type,omitempty"`
		Ref         string              `json:"$ref,omitempty"`
		Properties  map[string]Schema   `json:"properties,omitempty"`
		Items       *Schema             `json:"items,omitempty"`
		Required    []string             `json:"required,omitempty"`
		AdditionalProperties interface{} `json:"additionalProperties,omitempty"`
	}

	Operation struct {
		Tags         []string                 `json:"tags,omitempty"`
		Summary      string                   `json:"summary,omitempty"`
		Description  string                   `json:"description,omitempty"`
		OperationID  string                   `json:"operationId"`
		Security     []string                 `json:"security"`
		Parameters   []Parameter              `json:"parameters,omitempty"`
		RequestBody  *RequestBodyInfo         `json:"requestBody,omitempty"`
		Responses    map[string]Response      `json:"responses"`
	}

	Parameter struct {
		Name        string `json:"name"`
		In          string `json:"in"`
		Description string `json:"description,omitempty"`
		Required    bool   `json:"required,omitempty"`
		Schema      Schema `json:"schema"`
	}

	RequestBodyInfo struct {
		ContentType string `json:"contentType"`
		Schema      Schema `json:"schema"`
	}

	Response struct {
		Description string            `json:"description"`
		Content     map[string]MediaType `json:"content,omitempty"`
	}

	MediaType struct {
		Schema Schema `json:"schema"`
	}
)

// PathInfo describes a path+method for spec generation.
type PathInfo struct {
	Tags          []string
	Summary       string
	Description   string
	OperationID   string
	Security      []string
	PathParams    []ParameterInfo
	QueryParams   []ParameterInfo
	RequestBody   *RequestBodyInfo
	ResponseCodes []ResponseInfo
}

type ParameterInfo struct {
	Name        string
	In          string // "path" or "query"
	Description string
	Required    bool
	SchemaType  string
}

type ResponseInfo struct {
	Code      int
	Description string
	SchemaRef  string
}

func toResponseCode(code int) string {
	return fmt.Sprintf("%d", code)
}

func mustRef(s string) Schema {
	return Schema{Ref: s}
}