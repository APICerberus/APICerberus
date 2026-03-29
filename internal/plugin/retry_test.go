package plugin

import (
	"net/http"
	"testing"
	"time"
)

func TestRetryShouldRetry(t *testing.T) {
	t.Parallel()

	r := NewRetry(RetryConfig{
		MaxRetries:   2,
		BaseDelay:    10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Jitter:       false,
		RetryMethods: []string{http.MethodGet},
		RetryOnStatus: []int{
			http.StatusBadGateway,
			http.StatusServiceUnavailable,
			http.StatusGatewayTimeout,
		},
	})

	if !r.ShouldRetry(http.MethodGet, 0, http.StatusBadGateway, nil) {
		t.Fatalf("expected retry for GET 502")
	}
	if !r.ShouldRetry(http.MethodGet, 0, 0, assertErr{}) {
		t.Fatalf("expected retry on proxy error")
	}
	if r.ShouldRetry(http.MethodPost, 0, http.StatusBadGateway, nil) {
		t.Fatalf("expected no retry for POST by default")
	}
	if r.ShouldRetry(http.MethodGet, 2, http.StatusBadGateway, nil) {
		t.Fatalf("expected no retry once max retries reached")
	}
}

func TestRetryBackoff(t *testing.T) {
	t.Parallel()

	r := NewRetry(RetryConfig{
		MaxRetries: 2,
		BaseDelay:  50 * time.Millisecond,
		MaxDelay:   200 * time.Millisecond,
		Jitter:     false,
	})

	if got := r.Backoff(0); got != 50*time.Millisecond {
		t.Fatalf("attempt0 delay mismatch: %v", got)
	}
	if got := r.Backoff(1); got != 100*time.Millisecond {
		t.Fatalf("attempt1 delay mismatch: %v", got)
	}
	if got := r.Backoff(3); got != 200*time.Millisecond {
		t.Fatalf("attempt3 should cap at max delay: %v", got)
	}
}

type assertErr struct{}

func (assertErr) Error() string { return "x" }
