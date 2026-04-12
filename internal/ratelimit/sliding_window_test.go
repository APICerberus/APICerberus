package ratelimit

import (
	"testing"
	"time"
)

func TestSlidingWindowBasicLimit(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_000_000, 0).UTC()
	sw := NewSlidingWindow(3, time.Second)
	sw.now = func() time.Time { return now }

	if ok, _, _ := sw.Allow("a"); !ok {
		t.Fatalf("first should pass")
	}
	if ok, _, _ := sw.Allow("a"); !ok {
		t.Fatalf("second should pass")
	}
	if ok, _, _ := sw.Allow("a"); !ok {
		t.Fatalf("third should pass")
	}
	if ok, _, _ := sw.Allow("a"); ok {
		t.Fatalf("fourth should be limited")
	}
}

func TestSlidingWindowRotationAndBoundaryAccuracy(t *testing.T) {
	t.Parallel()

	current := time.Unix(1_700_000_000, 0).UTC()
	sw := NewSlidingWindow(2, time.Second)
	sw.subWindow = 100 * time.Millisecond
	sw.windowSpan = 10
	sw.now = func() time.Time { return current }

	// Two requests at start => full.
	if ok, _, _ := sw.Allow("a"); !ok {
		t.Fatalf("first should pass")
	}
	if ok, _, _ := sw.Allow("a"); !ok {
		t.Fatalf("second should pass")
	}
	if ok, _, _ := sw.Allow("a"); ok {
		t.Fatalf("third should be limited before rotation")
	}

	// Move near boundary but still inside full rolling window.
	current = current.Add(850 * time.Millisecond)
	if ok, _, _ := sw.Allow("a"); ok {
		t.Fatalf("should still be limited near boundary")
	}

	// Move beyond rolling window; old counts should expire.
	current = current.Add(300 * time.Millisecond) // total +1.15s
	if ok, rem, _ := sw.Allow("a"); !ok {
		t.Fatalf("should pass after old buckets expire")
	} else if rem != 1 {
		t.Fatalf("unexpected remaining: %d", rem)
	}
}
