package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

func (s *Server) resourceDefinitions() []resourceDefinition {
	return []resourceDefinition{
		{
			URI:         "apicerberus://services",
			Name:        "Services",
			Description: "Current gateway services.",
			MimeType:    "application/json",
		},
		{
			URI:         "apicerberus://routes",
			Name:        "Routes",
			Description: "Current gateway routes.",
			MimeType:    "application/json",
		},
		{
			URI:         "apicerberus://upstreams",
			Name:        "Upstreams",
			Description: "Current gateway upstreams.",
			MimeType:    "application/json",
		},
		{
			URI:         "apicerberus://users",
			Name:        "Users",
			Description: "Platform users.",
			MimeType:    "application/json",
		},
		{
			URI:         "apicerberus://credits/overview",
			Name:        "Credits Overview",
			Description: "Platform credit distribution and usage summary.",
			MimeType:    "application/json",
		},
		{
			URI:         "apicerberus://analytics/overview",
			Name:        "Analytics Overview",
			Description: "High-level analytics metrics.",
			MimeType:    "application/json",
		},
		{
			URI:         "apicerberus://config",
			Name:        "Runtime Config",
			Description: "Current runtime config snapshot.",
			MimeType:    "application/json",
		},
	}
}

func (s *Server) readResource(ctx context.Context, uri string) (map[string]any, error) {
	parsed, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("parse resource uri: %w", err)
	}
	if !strings.EqualFold(parsed.Scheme, "apicerberus") {
		return nil, fmt.Errorf("unsupported resource scheme: %s", parsed.Scheme)
	}
	resourceKey := parsed.Host + parsed.Path
	var value any
	switch resourceKey {
	case "services":
		value, err = s.callTool(ctx, "gateway.services.list", map[string]any{})
	case "routes":
		value, err = s.callTool(ctx, "gateway.routes.list", map[string]any{})
	case "upstreams":
		value, err = s.callTool(ctx, "gateway.upstreams.list", map[string]any{})
	case "users":
		value, err = s.callTool(ctx, "users.list", map[string]any{"limit": 100})
	case "credits/overview":
		value, err = s.callTool(ctx, "credits.overview", map[string]any{})
	case "analytics/overview":
		value, err = s.callTool(ctx, "analytics.overview", map[string]any{})
	case "config":
		value, err = s.callTool(ctx, "system.config.export", map[string]any{})
	default:
		return nil, fmt.Errorf("resource not found: %s", uri)
	}
	if err != nil {
		return nil, err
	}
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal resource: %w", err)
	}
	return map[string]any{
		"uri":      uri,
		"mimeType": "application/json",
		"text":     string(raw),
	}, nil
}
