package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	if r == nil {
		t.Fatal("NewRegistry() returned nil")
	}
}

func TestCounter(t *testing.T) {
	r := NewRegistry()
	c := r.NewCounter("test_counter", "Test counter", []string{})

	// Test initial value
	if c.Value() != 0 {
		t.Errorf("Initial value = %v, want 0", c.Value())
	}

	// Test Inc
	c.Inc()
	if c.Value() != 1 {
		t.Errorf("After Inc, value = %v, want 1", c.Value())
	}

	// Test Add
	c.Add(5)
	if c.Value() != 6 {
		t.Errorf("After Add(5), value = %v, want 6", c.Value())
	}
}

func TestGauge(t *testing.T) {
	r := NewRegistry()
	g := r.NewGauge("test_gauge", "Test gauge", []string{})

	// Test Set
	g.Set(10)
	if g.Value() != 10 {
		t.Errorf("After Set(10), value = %v, want 10", g.Value())
	}

	// Test Inc
	g.Inc()
	if g.Value() != 11 {
		t.Errorf("After Inc, value = %v, want 11", g.Value())
	}

	// Test Dec
	g.Dec()
	if g.Value() != 10 {
		t.Errorf("After Dec, value = %v, want 10", g.Value())
	}

	// Test Add
	g.Add(5)
	if g.Value() != 15 {
		t.Errorf("After Add(5), value = %v, want 15", g.Value())
	}

	// Test Sub
	g.Sub(3)
	if g.Value() != 12 {
		t.Errorf("After Sub(3), value = %v, want 12", g.Value())
	}
}

func TestHistogram(t *testing.T) {
	r := NewRegistry()
	h := r.NewHistogram("test_histogram", "Test histogram", []string{},
		[]float64{0.1, 0.5, 1.0, 2.0, 5.0})

	// Test Observe
	h.Observe(0.05)
	h.Observe(0.3)
	h.Observe(1.5)
	h.Observe(3.0)

	// Just verify it doesn't panic - full histogram testing would be complex
}

func TestPrometheusHandler(t *testing.T) {
	r := NewRegistry()

	// Create some metrics
	c := r.NewCounter("requests_total", "Total requests", []string{})
	c.Inc()
	c.Inc()

	g := r.NewGauge("active_connections", "Active connections", []string{})
	g.Set(10)

	// Create handler
	handler := r.PrometheusHandler()

	// Make request
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Check status
	if rec.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", rec.Code, http.StatusOK)
	}

	// Check content type
	contentType := rec.Header().Get("Content-Type")
	if contentType != "text/plain; version=0.0.4" {
		t.Errorf("Content-Type = %v, want text/plain; version=0.0.4", contentType)
	}

	// Check body contains metrics
	body := rec.Body.String()
	if !contains(body, "requests_total") {
		t.Error("Body should contain requests_total")
	}
	if !contains(body, "active_connections") {
		t.Error("Body should contain active_connections")
	}
	if !contains(body, "# TYPE requests_total counter") {
		t.Error("Body should contain TYPE annotation for counter")
	}
	if !contains(body, "# TYPE active_connections gauge") {
		t.Error("Body should contain TYPE annotation for gauge")
	}
}

func TestPrometheusHandlerMethodNotAllowed(t *testing.T) {
	r := NewRegistry()
	handler := r.PrometheusHandler()

	req := httptest.NewRequest(http.MethodPost, "/metrics", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("Status = %v, want %v", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestGatewayMetrics(t *testing.T) {
	r := NewRegistry()
	m := NewGatewayMetrics(r)

	if m.RequestsTotal == nil {
		t.Error("RequestsTotal should not be nil")
	}
	if m.RequestDuration == nil {
		t.Error("RequestDuration should not be nil")
	}
	if m.ActiveConnections == nil {
		t.Error("ActiveConnections should not be nil")
	}
	if m.AuthSuccess == nil {
		t.Error("AuthSuccess should not be nil")
	}
}

func TestGatewayMetricsRecordRequest(t *testing.T) {
	r := NewRegistry()
	m := NewGatewayMetrics(r)

	// Record a request
	m.RecordRequest("GET", "200", 100*time.Millisecond, 100, 1000)

	// Just verify it doesn't panic
}

func TestGatewayMetricsRecordBackendRequest(t *testing.T) {
	r := NewRegistry()
	m := NewGatewayMetrics(r)

	// Record successful backend request
	m.RecordBackendRequest("service-1", "target-1", 50*time.Millisecond, nil)

	// Record failed backend request
	m.RecordBackendRequest("service-1", "target-1", 100*time.Millisecond, http.ErrServerClosed)

	// Just verify it doesn't panic
}

// Test LabeledCounter
func TestLabeledCounter(t *testing.T) {
	r := NewRegistry()
	c := r.NewCounter("labeled_counter", "Test labeled counter", []string{"method", "status"})

	// Create labeled counter
	lc := c.WithLabels(map[string]string{
		"method": "GET",
		"status": "200",
	})

	// Test Inc
	lc.Inc()
	if c.Value() != 1 {
		t.Errorf("After labeled Inc, counter value = %v, want 1", c.Value())
	}

	// Test multiple labeled counters
	lc2 := c.WithLabels(map[string]string{
		"method": "POST",
		"status": "201",
	})
	lc2.Inc()
	lc2.Inc()

	if c.Value() != 3 {
		t.Errorf("After multiple labeled Inc, counter value = %v, want 3", c.Value())
	}
}

// Test LabeledGauge
func TestLabeledGauge(t *testing.T) {
	r := NewRegistry()
	g := r.NewGauge("labeled_gauge", "Test labeled gauge", []string{"service"})

	// Create labeled gauge
	lg := g.WithLabels(map[string]string{
		"service": "gateway",
	})

	// Test Set
	lg.Set(10)
	if g.Value() != 10 {
		t.Errorf("After labeled Set(10), gauge value = %v, want 10", g.Value())
	}

	// Test Inc
	lg.Inc()
	if g.Value() != 11 {
		t.Errorf("After labeled Inc, gauge value = %v, want 11", g.Value())
	}

	// Test Dec
	lg.Dec()
	if g.Value() != 10 {
		t.Errorf("After labeled Dec, gauge value = %v, want 10", g.Value())
	}
}

// Test LabeledHistogram
func TestLabeledHistogram(t *testing.T) {
	r := NewRegistry()
	h := r.NewHistogram("labeled_histogram", "Test labeled histogram", []string{"endpoint"},
		[]float64{0.1, 0.5, 1.0, 2.0, 5.0})

	// Create labeled histogram
	lh := h.WithLabels(map[string]string{
		"endpoint": "/api/v1/users",
	})

	// Test Observe
	lh.Observe(0.05)
	lh.Observe(0.3)
	lh.Observe(1.5)

	// Just verify it doesn't panic
}

// Test Registry Snapshot
func TestRegistry_Snapshot(t *testing.T) {
	r := NewRegistry()
	r.NewCounter("snapshot_counter", "Test counter", []string{})
	r.NewGauge("snapshot_gauge", "Test gauge", []string{})
	r.NewHistogram("snapshot_histogram", "Test histogram", []string{}, []float64{0.1, 0.5})

	counters, gauges, histograms := r.Snapshot()

	if len(counters) != 1 {
		t.Errorf("Counters length = %v, want 1", len(counters))
	}
	if len(gauges) != 1 {
		t.Errorf("Gauges length = %v, want 1", len(gauges))
	}
	if len(histograms) != 1 {
		t.Errorf("Histograms length = %v, want 1", len(histograms))
	}

	if counters[0].Name != "snapshot_counter" {
		t.Errorf("Counter name = %v, want snapshot_counter", counters[0].Name)
	}
	if gauges[0].Name != "snapshot_gauge" {
		t.Errorf("Gauge name = %v, want snapshot_gauge", gauges[0].Name)
	}
	if histograms[0].Name != "snapshot_histogram" {
		t.Errorf("Histogram name = %v, want snapshot_histogram", histograms[0].Name)
	}
}

// Test AggregateCounters
func TestAggregateCounters(t *testing.T) {
	snapshots := []CounterSnapshot{
		{Name: "counter1", Value: 10, Help: "Counter 1"},
		{Name: "counter1", Value: 30, Help: "Counter 1"}, // Duplicate name
	}

	result := AggregateCounters(snapshots)

	if result != 40 {
		t.Errorf("Aggregated counter value = %v, want 40", result)
	}
}

// Test AggregateGauges
func TestAggregateGauges(t *testing.T) {
	snapshots := []GaugeSnapshot{
		{Name: "gauge1", Value: 10, Help: "Gauge 1"},
		{Name: "gauge1", Value: 30, Help: "Gauge 1"}, // Duplicate name
	}

	avg, min, max := AggregateGauges(snapshots)

	if avg != 20 {
		t.Errorf("Average = %v, want 20", avg)
	}
	if min != 10 {
		t.Errorf("Min = %v, want 10", min)
	}
	if max != 30 {
		t.Errorf("Max = %v, want 30", max)
	}
}

// Test Registry GetCounter
func TestRegistry_GetCounter(t *testing.T) {
	r := NewRegistry()
	r.NewCounter("existing_counter", "Test counter", []string{})

	// Get existing counter
	c := r.GetCounter("existing_counter")
	if c == nil {
		t.Error("GetCounter should return existing counter")
	} else if c.Name != "existing_counter" {
		t.Errorf("Counter name = %v, want existing_counter", c.Name)
	}

	// Get non-existing counter
	c = r.GetCounter("non_existing")
	if c != nil {
		t.Error("GetCounter should return nil for non-existing counter")
	}
}

// Test Registry GetGauge
func TestRegistry_GetGauge(t *testing.T) {
	r := NewRegistry()
	r.NewGauge("existing_gauge", "Test gauge", []string{})

	// Get existing gauge
	g := r.GetGauge("existing_gauge")
	if g == nil {
		t.Error("GetGauge should return existing gauge")
	} else if g.Name != "existing_gauge" {
		t.Errorf("Gauge name = %v, want existing_gauge", g.Name)
	}

	// Get non-existing gauge
	g = r.GetGauge("non_existing")
	if g != nil {
		t.Error("GetGauge should return nil for non-existing gauge")
	}
}

// Test Registry GetHistogram
func TestRegistry_GetHistogram(t *testing.T) {
	r := NewRegistry()
	r.NewHistogram("existing_histogram", "Test histogram", []string{}, []float64{0.1, 0.5})

	// Get existing histogram
	h := r.GetHistogram("existing_histogram")
	if h == nil {
		t.Error("GetHistogram should return existing histogram")
	} else if h.Name != "existing_histogram" {
		t.Errorf("Histogram name = %v, want existing_histogram", h.Name)
	}

	// Get non-existing histogram
	h = r.GetHistogram("non_existing")
	if h != nil {
		t.Error("GetHistogram should return nil for non-existing histogram")
	}
}

// Test Registry Reset
func TestRegistry_Reset(t *testing.T) {
	r := NewRegistry()
	r.NewCounter("reset_counter", "Test counter", []string{})
	r.NewGauge("reset_gauge", "Test gauge", []string{})

	r.Reset()

	// After reset, metrics should be cleared
	if r.GetCounter("reset_counter") != nil {
		t.Error("GetCounter should not find counter after Reset")
	}

	if r.GetGauge("reset_gauge") != nil {
		t.Error("GetGauge should not find gauge after Reset")
	}
}

// Test Histogram with empty buckets
func TestHistogram_EmptyBuckets(t *testing.T) {
	r := NewRegistry()
	h := r.NewHistogram("empty_histogram", "Test empty histogram", []string{}, []float64{})

	// Should handle empty buckets gracefully
	h.Observe(1.0)
	h.Observe(2.0)
}

// Test Counter with negative Add (implementation allows it)
func TestCounter_NegativeAdd(t *testing.T) {
	r := NewRegistry()
	c := r.NewCounter("negative_counter", "Test counter", []string{})

	c.Add(10)
	c.Add(-5) // Counter implementation allows negative values

	// Implementation allows negative values
	if c.Value() != 5 {
		t.Errorf("Counter value after negative Add = %v, want 5", c.Value())
	}
}

// Test PrometheusHandler with histograms
func TestPrometheusHandler_WithHistograms(t *testing.T) {
	r := NewRegistry()
	h := r.NewHistogram("request_duration", "Request duration", []string{},
		[]float64{0.1, 0.5, 1.0, 2.0, 5.0})

	h.Observe(0.05)
	h.Observe(0.3)
	h.Observe(1.5)

	handler := r.PrometheusHandler()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", rec.Code, http.StatusOK)
	}

	body := rec.Body.String()
	if !contains(body, "request_duration") {
		t.Error("Body should contain request_duration")
	}
	if !contains(body, "# TYPE request_duration histogram") {
		t.Error("Body should contain TYPE annotation for histogram")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
