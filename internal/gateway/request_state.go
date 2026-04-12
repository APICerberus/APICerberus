package gateway

import (
	"net/http"
	"strings"
	"time"

	"github.com/APICerberus/APICerebrus/internal/audit"
	"github.com/APICerberus/APICerebrus/internal/config"
	"github.com/APICerberus/APICerebrus/internal/plugin"
)

// requestState holds the mutable state for a single request through the
// ServeHTTP pipeline. Each phase reads/writes the fields it cares about.
type requestState struct {
	route               *config.Route
	service             *config.Service
	consumer            *config.Consumer
	requestBodySnapshot []byte
	proxyErrForAudit    error
	blocked             bool
	blockReason         string
	auditWriter         *audit.ResponseCaptureWriter
	responseWriter      *audit.ResponseCaptureWriter
	pipelineCtx         *plugin.PipelineContext
	pipeline            *plugin.Pipeline
	billingState        *billingRequestState
	requestStartedAt    time.Time
}

func newRequestState() *requestState {
	return &requestState{
		requestStartedAt: time.Now(),
	}
}

// markBlocked sets blocked=true with the given reason.
func (rs *requestState) markBlocked(reason string) {
	rs.blocked = true
	rs.blockReason = reason
}

// runPipelineCleanup executes registered cleanup functions in LIFO order.
func (rs *requestState) runPipelineCleanup() {
	if rs.pipelineCtx == nil || len(rs.pipelineCtx.Cleanup) == 0 {
		return
	}
	for i := len(rs.pipelineCtx.Cleanup) - 1; i >= 0; i-- {
		if rs.pipelineCtx.Cleanup[i] != nil {
			rs.pipelineCtx.Cleanup[i]()
		}
	}
}

// writeResponseConsumer handles post-auth response consumer setup.
func (rs *requestState) writeResponseConsumer(r *http.Request) {
	if rs.consumer != nil {
		setRequestConsumer(r, rs.consumer)
	}
}

// setPipelineResponse updates the pipeline context with the upstream response.
func (rs *requestState) setPipelineResponse(resp *http.Response) {
	if rs.pipelineCtx != nil {
		rs.pipelineCtx.Response = resp
	}
}

// getDownstreamWriter returns the pipeline response writer, falling back to
// the original response writer.
func (rs *requestState) getDownstreamWriter(fallback http.ResponseWriter) http.ResponseWriter {
	if rs.pipelineCtx == nil || rs.pipelineCtx.ResponseWriter == nil {
		return fallback
	}
	return rs.pipelineCtx.ResponseWriter
}

// runAfterProxy executes post-proxy pipeline hooks and captures the consumer.
func (rs *requestState) runAfterProxy(proxyErr error) {
	if rs.pipelineCtx == nil || rs.pipeline == nil {
		return
	}
	rs.pipelineCtx.Consumer = rs.consumer
	rs.pipeline.ExecutePostProxy(rs.pipelineCtx, proxyErr)
	if capture, ok := rs.pipelineCtx.ResponseWriter.(*plugin.TransformCaptureWriter); ok && capture.HasCaptured() && !capture.IsFlushed() {
		_ = capture.Flush()
	}
	if capture, ok := rs.pipelineCtx.ResponseWriter.(*plugin.CaptureResponseWriter); ok && !capture.IsFlushed() {
		_ = capture.Flush()
	}
	rs.consumer = rs.pipelineCtx.Consumer
}

// routePipelineKey returns the key for looking up route-specific plugin pipelines.
func (rs *requestState) routePipelineKey() string {
	if rs.route == nil {
		return ""
	}
	if value := strings.TrimSpace(rs.route.ID); value != "" {
		return value
	}
	return strings.TrimSpace(rs.route.Name)
}
