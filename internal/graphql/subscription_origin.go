package graphql

import (
	"net/http"
	"net/url"
	"strings"
)

// isSubscriptionOriginAllowed implements the CSWSH gate shared by the
// WebSocket (HandleSubscription) and SSE (HandleSSE) subscription transports.
//
// SEC-GQL-007 policy:
//
//   - An empty / nil allow-list is the "compat mode" (no operator opt-in);
//     any Origin passes and a missing Origin also passes so non-browser
//     clients keep working. Operators exposing subscriptions on the public
//     internet MUST populate the list.
//   - A non-empty list is "strict mode":
//       - Missing Origin is rejected — browsers always set Origin on a
//         WebSocket handshake per RFC 6455 §10.2, so missing Origin here
//         indicates a misconfigured or malicious client bypassing the
//         browser trust boundary.
//       - "null" origin (sandbox iframe, data:) is rejected.
//       - Non-http(s) schemes are rejected.
//       - The scheme + host + port must match one allow-list entry.
//
// Allow-list grammar (aligned with the admin WS handler):
//
//   - "https://app.example.com"  — exact URL match (scheme + host + default port).
//   - "app.example.com"          — host-only (any scheme ok); optional :port.
//   - "*.example.com"            — single-label wildcard on the leftmost label.
func isSubscriptionOriginAllowed(r *http.Request, allowed []string) bool {
	if r == nil {
		return false
	}

	origin := strings.TrimSpace(r.Header.Get("Origin"))

	// Compat mode: no operator-declared allow-list, accept everything.
	// Documented failure mode; operators are expected to opt in.
	if len(allowed) == 0 {
		return true
	}

	if origin == "" || origin == "null" {
		return false
	}

	u, err := url.Parse(origin)
	if err != nil || u.Host == "" {
		return false
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return false
	}

	host := strings.ToLower(u.Hostname())
	port := u.Port()

	for _, entry := range allowed {
		entry = strings.ToLower(strings.TrimSpace(entry))
		if entry == "" {
			continue
		}
		if matchSubscriptionOrigin(host, port, scheme, entry) {
			return true
		}
	}
	return false
}

// matchSubscriptionOrigin checks a parsed origin against one allow-list entry.
func matchSubscriptionOrigin(originHost, originPort, originScheme, entry string) bool {
	// Exact URL form: scheme + host [+ port].
	if strings.HasPrefix(entry, "http://") || strings.HasPrefix(entry, "https://") {
		entryURL, err := url.Parse(entry)
		if err != nil || entryURL.Host == "" {
			return false
		}
		entryHost := strings.ToLower(entryURL.Hostname())
		entryPort := entryURL.Port()
		entryScheme := strings.ToLower(entryURL.Scheme)
		if originScheme != entryScheme {
			return false
		}
		if entryPort == "" {
			// Allow default ports (empty, 80, 443) when the entry pins scheme only.
			if originPort != "" && originPort != "80" && originPort != "443" {
				return false
			}
		} else if originPort != entryPort {
			return false
		}
		return matchSubscriptionHost(originHost, entryHost)
	}

	// Host-only (optionally with :port). Scheme is unconstrained.
	entryHost, entryPort := entry, ""
	if i := strings.LastIndex(entry, ":"); i != -1 && !strings.Contains(entry[i+1:], ".") {
		entryHost = entry[:i]
		entryPort = entry[i+1:]
	}
	if entryPort != "" && entryPort != originPort {
		return false
	}
	if entryPort == "" && originPort != "" && originPort != "80" && originPort != "443" {
		// If operator pinned a bare host (no port), treat it as "default port".
		return false
	}
	return matchSubscriptionHost(originHost, entryHost)
}

// matchSubscriptionHost matches a host against an allow-list host spec.
// Supports a single leftmost "*." wildcard matching exactly one subdomain label.
func matchSubscriptionHost(originHost, entryHost string) bool {
	if entryHost == originHost {
		return true
	}
	if strings.HasPrefix(entryHost, "*.") {
		suffix := entryHost[1:] // ".example.com"
		// The origin must end with the suffix and have exactly one extra
		// label — "*.example.com" matches "a.example.com" but not "example.com"
		// and not "x.y.example.com".
		if !strings.HasSuffix(originHost, suffix) {
			return false
		}
		prefix := strings.TrimSuffix(originHost, suffix)
		if prefix == "" || strings.Contains(prefix, ".") {
			return false
		}
		return true
	}
	return false
}
