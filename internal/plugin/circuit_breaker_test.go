package plugin

import (
	"testing"
	"time"
)

func TestCircuitBreakerOpenHalfOpenClosedFlow(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_000_000, 0).UTC()
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		ErrorThreshold:   0.5,
		VolumeThreshold:  2,
		SleepWindow:      200 * time.Millisecond,
		HalfOpenRequests: 2,
		Window:           time.Second,
	})
	cb.now = func() time.Time { return now }

	if err := cb.Allow(); err != nil {
		t.Fatalf("Allow should pass in closed state: %v", err)
	}
	cb.Report(false)

	if err := cb.Allow(); err != nil {
		t.Fatalf("second Allow should still pass before trip: %v", err)
	}
	cb.Report(false) // 2/2 failures => open

	if cb.State() != CircuitOpen {
		t.Fatalf("expected circuit open, got %s", cb.State())
	}
	if err := cb.Allow(); err != ErrCircuitOpen {
		t.Fatalf("expected ErrCircuitOpen, got %v", err)
	}

	now = now.Add(250 * time.Millisecond) // pass sleep window
	if err := cb.Allow(); err != nil {
		t.Fatalf("allow should pass in half-open trial: %v", err)
	}
	if cb.State() != CircuitHalfOpen {
		t.Fatalf("expected half-open state")
	}
	cb.Report(true)

	if err := cb.Allow(); err != nil {
		t.Fatalf("second half-open trial should pass: %v", err)
	}
	cb.Report(true)

	if cb.State() != CircuitClosed {
		t.Fatalf("expected closed after successful half-open trials, got %s", cb.State())
	}
}

func TestCircuitBreakerHalfOpenFailureReopens(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_000_000, 0).UTC()
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		ErrorThreshold:   0.5,
		VolumeThreshold:  2,
		SleepWindow:      100 * time.Millisecond,
		HalfOpenRequests: 1,
		Window:           time.Second,
	})
	cb.now = func() time.Time { return now }

	// trip open
	_ = cb.Allow()
	cb.Report(false)
	_ = cb.Allow()
	cb.Report(false)
	if cb.State() != CircuitOpen {
		t.Fatalf("expected open state")
	}

	now = now.Add(150 * time.Millisecond)
	if err := cb.Allow(); err != nil {
		t.Fatalf("half-open trial should be allowed: %v", err)
	}
	cb.Report(false) // fail in half-open => reopen

	if cb.State() != CircuitOpen {
		t.Fatalf("expected reopen to open state")
	}
	if err := cb.Allow(); err != ErrCircuitOpen {
		t.Fatalf("expected open circuit reject after half-open failure, got %v", err)
	}
}

func TestCircuitBreakerVolumeThreshold(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_000_000, 0).UTC()
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		ErrorThreshold:   0.2,
		VolumeThreshold:  5,
		SleepWindow:      time.Second,
		HalfOpenRequests: 1,
		Window:           5 * time.Second,
	})
	cb.now = func() time.Time { return now }

	sequence := []bool{false, false, false, false} // 4 failures but volume<5
	for _, success := range sequence {
		if err := cb.Allow(); err != nil {
			t.Fatalf("allow should pass before volume threshold: %v", err)
		}
		cb.Report(success)
	}
	if cb.State() != CircuitClosed {
		t.Fatalf("circuit should remain closed until volume threshold reached")
	}

	if err := cb.Allow(); err != nil {
		t.Fatalf("allow should pass on 5th call before report")
	}
	cb.Report(false)
	if cb.State() != CircuitOpen {
		t.Fatalf("expected open when threshold reached")
	}
}
