package observability

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// Exporter defines the interface for span exporters.
type Exporter interface {
	// Export sends a finished span to the configured destination.
	Export(span *Span) error
	// Shutdown gracefully shuts down the exporter, flushing any pending spans.
	Shutdown() error
}

// ExporterConfig configures the span exporter.
type ExporterConfig struct {
	Type          string        `yaml:"type"`           // "otlp", "stdout", "none"
	Endpoint      string        `yaml:"endpoint"`       // OTLP/HTTP endpoint URL
	BatchSize     int           `yaml:"batch_size"`     // Number of spans per batch
	FlushInterval time.Duration `yaml:"flush_interval"` // How often to flush batches
}

// DefaultExporterConfig returns a default exporter configuration.
func DefaultExporterConfig() *ExporterConfig {
	return &ExporterConfig{
		Type:          "none",
		BatchSize:     64,
		FlushInterval: 5 * time.Second,
	}
}

// --- OTLP Exporter ---

// otlpResource describes the service emitting spans.
type otlpResource struct {
	Attributes []otlpKeyValue `json:"attributes"`
}

// otlpKeyValue is an OTLP attribute key-value pair.
type otlpKeyValue struct {
	Key   string         `json:"key"`
	Value otlpAnyValue   `json:"value"`
}

// otlpAnyValue wraps a string value for OTLP JSON encoding.
type otlpAnyValue struct {
	StringValue string `json:"stringValue"`
}

// otlpScopeSpans groups spans under an instrumentation scope.
type otlpScopeSpans struct {
	Spans []otlpSpan `json:"spans"`
}

// otlpResourceSpans is the top-level OTLP export structure.
type otlpResourceSpans struct {
	Resource   otlpResource     `json:"resource"`
	ScopeSpans []otlpScopeSpans `json:"scopeSpans"`
}

// otlpExportRequest is the JSON body sent to the OTLP/HTTP endpoint.
type otlpExportRequest struct {
	ResourceSpans []otlpResourceSpans `json:"resourceSpans"`
}

// otlpSpan is the OTLP span representation.
type otlpSpan struct {
	TraceID           string         `json:"traceId"`
	SpanID            string         `json:"spanId"`
	ParentSpanID      string         `json:"parentSpanId,omitempty"`
	Name              string         `json:"name"`
	StartTimeUnixNano string         `json:"startTimeUnixNano"`
	EndTimeUnixNano   string         `json:"endTimeUnixNano"`
	Status            otlpStatus     `json:"status"`
	Attributes        []otlpKeyValue `json:"attributes,omitempty"`
}

// otlpStatus is the OTLP span status.
type otlpStatus struct {
	Code int `json:"code"` // 0=Unset, 1=OK, 2=Error
}

// OTLPExporter sends spans to an OTLP/HTTP endpoint.
type OTLPExporter struct {
	endpoint    string
	serviceName string
	client      *http.Client
}

// NewOTLPExporter creates an exporter that sends spans via HTTP POST.
func NewOTLPExporter(endpoint, serviceName string) *OTLPExporter {
	return &OTLPExporter{
		endpoint:    endpoint,
		serviceName: serviceName,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Export sends a single span to the OTLP endpoint.
func (e *OTLPExporter) Export(span *Span) error {
	return e.exportSpans([]*Span{span})
}

// exportSpans sends a batch of spans to the OTLP endpoint.
func (e *OTLPExporter) exportSpans(spans []*Span) error {
	if len(spans) == 0 {
		return nil
	}

	otlpSpans := make([]otlpSpan, 0, len(spans))
	for _, s := range spans {
		attrs := make([]otlpKeyValue, 0, len(s.Tags))
		for k, v := range s.Tags {
			attrs = append(attrs, otlpKeyValue{
				Key:   k,
				Value: otlpAnyValue{StringValue: v},
			})
		}

		statusCode := 1 // OK
		if s.Status == SpanStatusError {
			statusCode = 2 // Error
		}

		otlpSpans = append(otlpSpans, otlpSpan{
			TraceID:           s.TraceID,
			SpanID:            s.SpanID,
			ParentSpanID:      s.ParentID,
			Name:              s.Name,
			StartTimeUnixNano: fmt.Sprintf("%d", s.StartTime.UnixNano()),
			EndTimeUnixNano:   fmt.Sprintf("%d", s.EndTime.UnixNano()),
			Status:            otlpStatus{Code: statusCode},
			Attributes:        attrs,
		})
	}

	payload := otlpExportRequest{
		ResourceSpans: []otlpResourceSpans{
			{
				Resource: otlpResource{
					Attributes: []otlpKeyValue{
						{
							Key:   "service.name",
							Value: otlpAnyValue{StringValue: e.serviceName},
						},
					},
				},
				ScopeSpans: []otlpScopeSpans{
					{Spans: otlpSpans},
				},
			},
		},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal OTLP payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, e.endpoint, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("create OTLP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return fmt.Errorf("send OTLP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("OTLP endpoint returned status %d", resp.StatusCode)
	}
	return nil
}

// Shutdown is a no-op for the direct OTLP exporter.
func (e *OTLPExporter) Shutdown() error {
	return nil
}

// --- Stdout Exporter ---

// StdoutExporter writes span data to slog for development.
type StdoutExporter struct {
	logger *slog.Logger
}

// NewStdoutExporter creates an exporter that writes spans to slog.
func NewStdoutExporter(logger *slog.Logger) *StdoutExporter {
	if logger == nil {
		logger = slog.Default()
	}
	return &StdoutExporter{logger: logger}
}

// Export logs the span to slog.
func (e *StdoutExporter) Export(span *Span) error {
	if span == nil {
		return nil
	}
	e.logger.Info("span finished",
		"trace_id", span.TraceID,
		"span_id", span.SpanID,
		"parent_id", span.ParentID,
		"name", span.Name,
		"duration", span.Duration.String(),
		"status", span.Status,
		"tags", span.Tags,
	)
	return nil
}

// Shutdown is a no-op for the stdout exporter.
func (e *StdoutExporter) Shutdown() error {
	return nil
}

// --- Batch Exporter ---

// BatchExporter wraps another Exporter and batches spans before flushing.
type BatchExporter struct {
	mu            sync.Mutex
	inner         Exporter
	batchSize     int
	flushInterval time.Duration
	pending       []*Span
	done          chan struct{}
	stopped       bool
}

// NewBatchExporter wraps an exporter with batching. Spans are accumulated and
// flushed either when batchSize is reached or every flushInterval.
func NewBatchExporter(inner Exporter, batchSize int, flushInterval time.Duration) *BatchExporter {
	if batchSize <= 0 {
		batchSize = 64
	}
	if flushInterval <= 0 {
		flushInterval = 5 * time.Second
	}

	be := &BatchExporter{
		inner:         inner,
		batchSize:     batchSize,
		flushInterval: flushInterval,
		pending:       make([]*Span, 0, batchSize),
		done:          make(chan struct{}),
	}
	go be.flushLoop()
	return be
}

// Export queues a span for batched export.
func (be *BatchExporter) Export(span *Span) error {
	if span == nil {
		return nil
	}

	be.mu.Lock()
	defer be.mu.Unlock()

	if be.stopped {
		return fmt.Errorf("batch exporter is shut down")
	}

	be.pending = append(be.pending, span)
	if len(be.pending) >= be.batchSize {
		return be.flushLocked()
	}
	return nil
}

// Shutdown flushes remaining spans and stops the background loop.
func (be *BatchExporter) Shutdown() error {
	be.mu.Lock()
	if be.stopped {
		be.mu.Unlock()
		return nil
	}
	be.stopped = true
	err := be.flushLocked()
	be.mu.Unlock()

	close(be.done)
	return err
}

// flushLoop periodically flushes pending spans.
func (be *BatchExporter) flushLoop() {
	ticker := time.NewTicker(be.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			be.mu.Lock()
			if !be.stopped {
				_ = be.flushLocked()
			}
			be.mu.Unlock()
		case <-be.done:
			return
		}
	}
}

// flushLocked exports all pending spans via the inner exporter. Caller must hold mu.
func (be *BatchExporter) flushLocked() error {
	if len(be.pending) == 0 {
		return nil
	}

	batch := be.pending
	be.pending = make([]*Span, 0, be.batchSize)

	var lastErr error
	for _, span := range batch {
		if err := be.inner.Export(span); err != nil {
			lastErr = err
		}
	}
	return lastErr
}
