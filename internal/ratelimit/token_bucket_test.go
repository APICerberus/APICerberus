package ratelimit

import (
	"testing"
	"time"
)

func TestTokenBucketBurst(t *testing.T) {
	t.Parallel()

	start := time.Unix(1_700_000_000, 0).UTC()
	tb := NewTokenBucket(2, 3)
	tb.now = func() time.Time { return start }

	allowed1, remaining1, _ := tb.Allow("client-a")
	allowed2, remaining2, _ := tb.Allow("client-a")
	allowed3, remaining3, _ := tb.Allow("client-a")
	allowed4, remaining4, _ := tb.Allow("client-a")

	if !allowed1 || !allowed2 || !allowed3 {
		t.Fatalf("first three requests should be allowed")
	}
	if allowed4 {
		t.Fatalf("fourth request should be denied due to burst limit")
	}
	if remaining1 != 2 || remaining2 != 1 || remaining3 != 0 || remaining4 != 0 {
		t.Fatalf("unexpected remaining values: %d %d %d %d", remaining1, remaining2, remaining3, remaining4)
	}
}

func TestTokenBucketRefillTiming(t *testing.T) {
	t.Parallel()

	current := time.Unix(1_700_000_000, 0).UTC()
	tb := NewTokenBucket(2, 2) // 2 tokens/s
	tb.now = func() time.Time { return current }

	if ok, _, _ := tb.Allow("client-a"); !ok {
		t.Fatalf("first request should be allowed")
	}
	if ok, _, _ := tb.Allow("client-a"); !ok {
		t.Fatalf("second request should be allowed")
	}
	if ok, _, resetAt := tb.Allow("client-a"); ok {
		t.Fatalf("third request should be denied")
	} else if !resetAt.After(current) {
		t.Fatalf("resetAt should be in the future")
	}

	current = current.Add(500 * time.Millisecond) // refill 1 token
	if ok, remaining, _ := tb.Allow("client-a"); !ok {
		t.Fatalf("request should be allowed after refill")
	} else if remaining != 0 {
		t.Fatalf("expected remaining 0 got %d", remaining)
	}
}

func TestTokenBucketMultipleKeys(t *testing.T) {
	t.Parallel()

	start := time.Unix(1_700_000_000, 0).UTC()
	tb := NewTokenBucket(1, 1)
	tb.now = func() time.Time { return start }

	if ok, _, _ := tb.Allow("a"); !ok {
		t.Fatalf("key a first request should be allowed")
	}
	if ok, _, _ := tb.Allow("a"); ok {
		t.Fatalf("key a second request should be denied")
	}
	if ok, _, _ := tb.Allow("b"); !ok {
		t.Fatalf("key b should be independent and allowed")
	}
}
