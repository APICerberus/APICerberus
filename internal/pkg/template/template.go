package template

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Mode determines how body transformation is applied.
type Mode string

const (
	ModeJSON     Mode = "json"
	ModeTemplate Mode = "template"
)

// Context contains runtime variables that can be injected into templates.
type Context struct {
	Body            string
	Timestamp       time.Time
	UpstreamLatency time.Duration
	ConsumerID      string
	RouteName       string
	RequestID       string
	RemoteAddr      string
	Headers         http.Header
}

// JSONPatch configures object-level modifications for JSON body mode.
type JSONPatch struct {
	Add    map[string]any
	Remove []string
	Rename map[string]string
}

// TransformOptions controls transformation behavior.
type TransformOptions struct {
	Mode     Mode
	Template string
	JSON     JSONPatch
}

var headerVarPattern = regexp.MustCompile(`\$header\.([A-Za-z0-9\-]+)`)

// Render performs variable substitution on input template string.
func Render(input string, ctx Context) string {
	now := ctx.Timestamp
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}

	values := map[string]string{
		"$body":                ctx.Body,
		"$timestamp_ms":        strconv.FormatInt(now.UnixMilli(), 10),
		"$timestamp_iso":       now.Format(time.RFC3339),
		"$upstream_latency_ms": strconv.FormatInt(ctx.UpstreamLatency.Milliseconds(), 10),
		"$consumer_id":         ctx.ConsumerID,
		"$route_name":          ctx.RouteName,
		"$request_id":          ctx.RequestID,
		"$remote_addr":         ctx.RemoteAddr,
	}

	out := input
	for variable, value := range values {
		out = strings.ReplaceAll(out, variable, value)
	}

	out = headerVarPattern.ReplaceAllStringFunc(out, func(match string) string {
		groups := headerVarPattern.FindStringSubmatch(match)
		if len(groups) != 2 {
			return ""
		}
		headerName := strings.TrimSpace(groups[1])
		if headerName == "" || ctx.Headers == nil {
			return ""
		}
		return strings.TrimSpace(ctx.Headers.Get(headerName))
	})

	return out
}

// Transform applies either JSON patch mode or full template mode.
func Transform(body []byte, opts TransformOptions, ctx Context) ([]byte, error) {
	switch opts.Mode {
	case "", ModeJSON:
		return ApplyJSONPatch(body, opts.JSON)
	case ModeTemplate:
		if ctx.Body == "" {
			ctx.Body = string(body)
		}
		templateValue := opts.Template
		if strings.TrimSpace(templateValue) == "" {
			templateValue = "$body"
		}
		return []byte(Render(templateValue, ctx)), nil
	default:
		return nil, fmt.Errorf("unsupported transform mode %q", opts.Mode)
	}
}

// ApplyJSONPatch applies add/remove/rename operations with dot-notation paths.
func ApplyJSONPatch(body []byte, patch JSONPatch) ([]byte, error) {
	root := map[string]any{}
	trimmed := strings.TrimSpace(string(body))
	if trimmed != "" {
		var decoded any
		if err := json.Unmarshal(body, &decoded); err != nil {
			return nil, fmt.Errorf("parse json body: %w", err)
		}
		asMap, ok := decoded.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("json body must be an object")
		}
		root = asMap
	}

	for path, value := range patch.Add {
		if err := setPath(root, path, value); err != nil {
			return nil, err
		}
	}
	for _, path := range patch.Remove {
		_ = deletePath(root, path)
	}
	for from, to := range patch.Rename {
		value, found := getPath(root, from)
		if !found {
			continue
		}
		_ = deletePath(root, from)
		if err := setPath(root, to, value); err != nil {
			return nil, err
		}
	}

	out, err := json.Marshal(root)
	if err != nil {
		return nil, fmt.Errorf("marshal transformed json: %w", err)
	}
	return out, nil
}

func setPath(root map[string]any, path string, value any) error {
	parts := splitPath(path)
	if len(parts) == 0 {
		return fmt.Errorf("path is empty")
	}

	current := root
	for i := 0; i < len(parts)-1; i++ {
		key := parts[i]
		next, exists := current[key]
		if !exists {
			child := map[string]any{}
			current[key] = child
			current = child
			continue
		}
		asMap, ok := next.(map[string]any)
		if !ok {
			child := map[string]any{}
			current[key] = child
			current = child
			continue
		}
		current = asMap
	}

	current[parts[len(parts)-1]] = value
	return nil
}

func getPath(root map[string]any, path string) (any, bool) {
	parts := splitPath(path)
	if len(parts) == 0 {
		return nil, false
	}

	current := root
	for i := 0; i < len(parts)-1; i++ {
		next, ok := current[parts[i]]
		if !ok {
			return nil, false
		}
		asMap, ok := next.(map[string]any)
		if !ok {
			return nil, false
		}
		current = asMap
	}

	value, ok := current[parts[len(parts)-1]]
	return value, ok
}

func deletePath(root map[string]any, path string) bool {
	parts := splitPath(path)
	if len(parts) == 0 {
		return false
	}

	current := root
	for i := 0; i < len(parts)-1; i++ {
		next, ok := current[parts[i]]
		if !ok {
			return false
		}
		asMap, ok := next.(map[string]any)
		if !ok {
			return false
		}
		current = asMap
	}

	last := parts[len(parts)-1]
	if _, ok := current[last]; !ok {
		return false
	}
	delete(current, last)
	return true
}

func splitPath(path string) []string {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}
	parts := strings.Split(path, ".")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		out = append(out, part)
	}
	return out
}
