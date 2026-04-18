package plugin

import (
	"net/http"
	"net/url"
	"strings"
)

// RedirectRule maps one path to a redirect target and status code.
type RedirectRule struct {
	Path       string
	TargetURL  string
	StatusCode int
}

// RedirectConfig configures redirect plugin behavior.
type RedirectConfig struct {
	Rules          []RedirectRule
	AllowedDomains []string // Optional domain allowlist for external redirects; if set, only these domains are allowed
}

// Redirect performs early redirect response based on request path.
type Redirect struct {
	rules          []RedirectRule
	allowedDomains map[string]bool // lookup map for O(1) domain check
}

// isValidRedirectTarget validates that a redirect target is safe.
// Rejects: javascript:, data:, file:, vbscript: and other schemes.
// Rejects: relative paths starting with // (proto-relative to different host).
// Allows: absolute paths (/foo), https://, http:// (with warning).
// If allowedDomains is non-empty, external https:///http:// redirects are
// restricted to those domains only (M-003 fix).
func isValidRedirectTarget(target string, allowedDomains map[string]bool) bool {
	target = strings.TrimSpace(target)
	if target == "" {
		return false
	}

	// Block protocol-relative // attacks (redirects to different host)
	if strings.HasPrefix(target, "//") {
		return false
	}

	u, err := url.Parse(target)
	if err != nil {
		return false
	}

	// If scheme is empty, it's an absolute path — always allowed
	if u.Scheme == "" {
		return true
	}

	// Explicitly block dangerous schemes
	switch strings.ToLower(u.Scheme) {
	case "https", "http":
		// M-003: if allowedDomains is configured, restrict external redirects
		if len(allowedDomains) > 0 {
			host := strings.ToLower(u.Host)
			if !allowedDomains[host] {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func NewRedirect(cfg RedirectConfig) *Redirect {
	rules := make([]RedirectRule, 0, len(cfg.Rules))
	allowedDomains := make(map[string]bool, len(cfg.AllowedDomains))
	for _, d := range cfg.AllowedDomains {
		allowedDomains[strings.ToLower(strings.TrimSpace(d))] = true
	}

	for _, rule := range cfg.Rules {
		path := strings.TrimSpace(rule.Path)
		target := strings.TrimSpace(rule.TargetURL)
		if path == "" || target == "" {
			continue
		}
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
		if !isValidRedirectTarget(target, allowedDomains) {
			continue
		}
		status := normalizeRedirectStatus(rule.StatusCode)
		rules = append(rules, RedirectRule{
			Path:       path,
			TargetURL:  target,
			StatusCode: status,
		})
	}
	return &Redirect{rules: rules, allowedDomains: allowedDomains}
}

func (r *Redirect) Name() string  { return "redirect" }
func (r *Redirect) Phase() Phase  { return PhasePreProxy }
func (r *Redirect) Priority() int { return 15 }

func (r *Redirect) Handle(w http.ResponseWriter, req *http.Request) bool {
	if r == nil || w == nil || req == nil || req.URL == nil {
		return false
	}
	for _, rule := range r.rules {
		if req.URL.Path != rule.Path {
			continue
		}
		// Do not append original query parameters to redirect target.
		// Original query may contain tokens, session IDs, or other
		// sensitive data that should not be leaked to external domains.
		http.Redirect(w, req, rule.TargetURL, rule.StatusCode)
		return true
	}
	return false
}

func normalizeRedirectStatus(code int) int {
	switch code {
	case http.StatusMovedPermanently, http.StatusFound, http.StatusTemporaryRedirect, http.StatusPermanentRedirect:
		return code
	default:
		return http.StatusFound
	}
}
