package ratelimit

import (
	"testing"
	"time"
)

func TestFixedWindowWithinLimitAndExceed(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_000_000, 0).UTC()
	fw := NewFixedWindow(2, time.Second)
	fw.now = func() time.Time { return now }

	ok1, rem1, reset1 := fw.Allow("client-a")
	ok2, rem2, reset2 := fw.Allow("client-a")
	ok3, rem3, reset3 := fw.Allow("client-a")

	if !ok1 || !ok2 {
		t.Fatalf("first two requests should be allowed")
	}
	if ok3 {
		t.Fatalf("third request should exceed fixed window")
	}
	if rem1 != 1 || rem2 != 0 || rem3 != 0 {
		t.Fatalf("unexpected remaining values: %d %d %d", rem1, rem2, rem3)
	}
	if !reset1.Equal(reset2) || !reset2.Equal(reset3) {
		t.Fatalf("resetAt should be identical within same window")
	}
}

func TestFixedWindowReset(t *testing.T) {
	t.Parallel()

	current := time.Unix(1_700_000_000, 0).UTC()
	fw := NewFixedWindow(1, time.Second)
	fw.now = func() time.Time { return current }

	ok1, _, _ := fw.Allow("client-a")
	ok2, _, _ := fw.Allow("client-a")
	if !ok1 || ok2 {
		t.Fatalf("expected allow then deny in first window")
	}

	current = current.Add(1 * time.Second)
	ok3, rem3, _ := fw.Allow("client-a")
	if !ok3 {
		t.Fatalf("expected window reset to allow request")
	}
	if rem3 != 0 {
		t.Fatalf("unexpected remaining after reset: %d", rem3)
	}
}

func TestFixedWindowMultipleKeys(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_000_000, 0).UTC()
	fw := NewFixedWindow(1, time.Second)
	fw.now = func() time.Time { return now }

	if ok, _, _ := fw.Allow("a"); !ok {
		t.Fatalf("key a first request should be allowed")
	}
	if ok, _, _ := fw.Allow("a"); ok {
		t.Fatalf("key a second request should be denied")
	}
	if ok, _, _ := fw.Allow("b"); !ok {
		t.Fatalf("key b should be independent and allowed")
	}
}
