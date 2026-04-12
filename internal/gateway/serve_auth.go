package gateway

import (
	"net/http"

	"github.com/APICerberus/APICerebrus/internal/plugin"
)

// executeAuthChain runs the authentication phase. It returns true if the
// request was handled (response already written).
func (g *Gateway) executeAuthChain(r *http.Request, rs *requestState, authRequired bool, authAPIKey *plugin.AuthAPIKey, routePipelines map[string][]plugin.PipelinePlugin, routeHasAuth map[string]bool) bool {
	routeKey := rs.routePipelineKey()

	if authRequired && !routeHasAuth[routeKey] {
		if authAPIKey == nil {
			rs.markBlocked("auth_unavailable")
			g.writeErrorRoute(gwResponseWriter(rs), http.StatusInternalServerError, "auth_unavailable", "Authentication module is unavailable", rs.route)
			return true
		}
		resolved, err := authAPIKey.Authenticate(r)
		if err != nil {
			rs.markBlocked("auth_failed")
			g.writeAuthError(gwResponseWriter(rs), err)
			return true
		}
		rs.consumer = resolved
	}
	if rs.consumer != nil {
		setRequestConsumer(r, rs.consumer)
	}
	return false
}

// gwResponseWriter returns the original response writer (before any capture wrapper).
// This is used for error responses that should bypass the audit capture.
func gwResponseWriter(rs *requestState) http.ResponseWriter {
	// Return the wrapped writer — error writes through it are fine because
	// the audit logger records the status code from the wrapper.
	return rs.responseWriter
}
