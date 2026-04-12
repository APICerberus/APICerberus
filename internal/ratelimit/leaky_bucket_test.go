package ratelimit

import (
	"testing"
	"time"
)

func TestLeakyBucketBurstAndRejection(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_000_000, 0).UTC()
	lb := NewLeakyBucket(2, 1)
	lb.now = func() time.Time { return now }

	if ok, _, _ := lb.Allow("a"); !ok {
		t.Fatalf("first should pass")
	}
	if ok, _, _ := lb.Allow("a"); !ok {
		t.Fatalf("second should pass")
	}
	if ok, _, _ := lb.Allow("a"); ok {
		t.Fatalf("third should be rejected when queue full")
	}
}

func TestLeakyBucketDrain(t *testing.T) {
	t.Parallel()

	current := time.Unix(1_700_000_000, 0).UTC()
	lb := NewLeakyBucket(2, 2) // drain 2 req/s
	lb.now = func() time.Time { return current }

	if ok, _, _ := lb.Allow("a"); !ok {
		t.Fatalf("first should pass")
	}
	if ok, _, _ := lb.Allow("a"); !ok {
		t.Fatalf("second should pass")
	}
	if ok, _, _ := lb.Allow("a"); ok {
		t.Fatalf("third should fail before drain")
	}

	current = current.Add(600 * time.Millisecond) // drain about 1.2
	if ok, _, _ := lb.Allow("a"); !ok {
		t.Fatalf("should pass after enough drain")
	}
}
