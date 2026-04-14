package ratelimit

import (
	"testing"
	"time"
)

func TestTokenBucket_PurgeStale(t *testing.T) {
	tb := NewTokenBucket(10, 20)

	// Access some keys
	tb.Allow("key1")
	tb.Allow("key2")
	tb.Allow("key3")

	// Count entries
	count := 0
	tb.buckets.Range(func(_, _ any) bool { count++; return true })
	if count != 3 {
		t.Fatalf("expected 3 entries, got %d", count)
	}

	// Purge with cutoff in the future (removes all)
	tb.PurgeStale(time.Now().Add(time.Hour))

	count = 0
	tb.buckets.Range(func(_, _ any) bool { count++; return true })
	if count != 0 {
		t.Fatalf("expected 0 entries after purge, got %d", count)
	}
}

func TestTokenBucket_PurgeStale_Partial(t *testing.T) {
	tb := NewTokenBucket(10, 20)

	// Access key1 "earlier"
	tb.Allow("key1")

	// Wait a tiny bit then access key2
	time.Sleep(2 * time.Millisecond)
	cutoff := time.Now()
	time.Sleep(2 * time.Millisecond)

	tb.Allow("key2")

	// Purge entries older than cutoff — only key1 should be removed
	tb.PurgeStale(cutoff)

	_, hasKey1 := tb.buckets.Load("key1")
	_, hasKey2 := tb.buckets.Load("key2")
	if hasKey1 {
		t.Error("expected key1 to be purged")
	}
	if !hasKey2 {
		t.Error("expected key2 to survive")
	}
}

func TestTokenBucket_PurgeStale_Nil(t *testing.T) {
	var tb *TokenBucket
	tb.PurgeStale(time.Now()) // should not panic
}

func TestFixedWindow_PurgeStale(t *testing.T) {
	fw := NewFixedWindow(10, time.Second)

	fw.Allow("key1")
	fw.Allow("key2")

	count := 0
	fw.windows.Range(func(_, _ any) bool { count++; return true })
	if count != 2 {
		t.Fatalf("expected 2 entries, got %d", count)
	}

	// Purge with current time — windows are current, so nothing removed
	fw.PurgeStale(time.Now())
	count = 0
	fw.windows.Range(func(_, _ any) bool { count++; return true })
	if count != 2 {
		t.Fatalf("expected 2 entries (current windows), got %d", count)
	}
}

func TestFixedWindow_PurgeStale_Nil(t *testing.T) {
	var fw *FixedWindow
	fw.PurgeStale(time.Now())
}

func TestSlidingWindow_PurgeStale(t *testing.T) {
	sw := NewSlidingWindow(10, 100*time.Millisecond)

	sw.Allow("key1")
	sw.Allow("key2")

	// Wait for window to expire
	time.Sleep(200 * time.Millisecond)

	// Now purge — all counts should be expired
	sw.PurgeStale(time.Now())

	count := 0
	sw.buckets.Range(func(_, _ any) bool { count++; return true })
	if count != 0 {
		t.Fatalf("expected 0 entries after window expiry, got %d", count)
	}
}

func TestSlidingWindow_PurgeStale_ActiveKey(t *testing.T) {
	sw := NewSlidingWindow(10, time.Hour) // long window

	sw.Allow("key1")

	// Purge immediately — key is still active
	sw.PurgeStale(time.Now())

	_, hasKey := sw.buckets.Load("key1")
	if !hasKey {
		t.Error("expected key1 to survive (window still active)")
	}
}

func TestSlidingWindow_PurgeStale_Nil(t *testing.T) {
	var sw *SlidingWindow
	sw.PurgeStale(time.Now())
}

func TestLeakyBucket_PurgeStale(t *testing.T) {
	lb := NewLeakyBucket(10, 10)

	lb.Allow("key1")
	lb.Allow("key2")

	// Purge with cutoff in the future (removes all)
	lb.PurgeStale(time.Now().Add(time.Hour))

	count := 0
	lb.buckets.Range(func(_, _ any) bool { count++; return true })
	if count != 0 {
		t.Fatalf("expected 0 entries after purge, got %d", count)
	}
}

func TestLeakyBucket_PurgeStale_Partial(t *testing.T) {
	lb := NewLeakyBucket(10, 10)

	lb.Allow("key1")
	time.Sleep(2 * time.Millisecond)
	cutoff := time.Now()
	time.Sleep(2 * time.Millisecond)
	lb.Allow("key2")

	lb.PurgeStale(cutoff)

	_, hasKey1 := lb.buckets.Load("key1")
	_, hasKey2 := lb.buckets.Load("key2")
	if hasKey1 {
		t.Error("expected key1 to be purged")
	}
	if !hasKey2 {
		t.Error("expected key2 to survive")
	}
}

func TestLeakyBucket_PurgeStale_Nil(t *testing.T) {
	var lb *LeakyBucket
	lb.PurgeStale(time.Now())
}
