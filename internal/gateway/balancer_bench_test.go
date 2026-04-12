package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/APICerberus/APICerebrus/internal/config"
)

func BenchmarkBalancerSelection(b *testing.B) {
	targets := []config.UpstreamTarget{
		{ID: "t1", Address: "10.0.0.1:8080", Weight: 1},
		{ID: "t2", Address: "10.0.0.2:8080", Weight: 2},
		{ID: "t3", Address: "10.0.0.3:8080", Weight: 3},
		{ID: "t4", Address: "10.0.0.4:8080", Weight: 4},
	}
	req := httptest.NewRequest(http.MethodGet, "http://gateway.local/api/items/42", nil)
	req.RemoteAddr = "203.0.113.77:5555"
	ctx := &RequestContext{Request: req}

	b.Run("round_robin", func(b *testing.B) {
		benchmarkBalancerNext(b, NewRoundRobin(targets), ctx)
	})
	b.Run("weighted_round_robin", func(b *testing.B) {
		benchmarkBalancerNext(b, NewWeightedRoundRobin(targets), ctx)
	})
	b.Run("least_conn", func(b *testing.B) {
		benchmarkBalancerNext(b, NewLeastConn(targets), ctx)
	})
	b.Run("ip_hash", func(b *testing.B) {
		benchmarkBalancerNext(b, NewIPHash(targets), ctx)
	})
	b.Run("random", func(b *testing.B) {
		benchmarkBalancerNext(b, NewRandomBalancer(targets), ctx)
	})
	b.Run("consistent_hash", func(b *testing.B) {
		benchmarkBalancerNext(b, NewConsistentHash(targets), ctx)
	})
	b.Run("least_latency", func(b *testing.B) {
		ll := NewLeastLatency(targets)
		ll.ReportHealth("t1", true, 120*time.Millisecond)
		ll.ReportHealth("t2", true, 60*time.Millisecond)
		ll.ReportHealth("t3", true, 30*time.Millisecond)
		ll.ReportHealth("t4", true, 90*time.Millisecond)
		benchmarkBalancerNext(b, ll, ctx)
	})
	b.Run("health_weighted", func(b *testing.B) {
		benchmarkBalancerNext(b, NewHealthWeighted(targets), ctx)
	})
}

func benchmarkBalancerNext(b *testing.B, balancer Balancer, ctx *RequestContext) {
	b.Helper()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		target, err := balancer.Next(ctx)
		if err != nil {
			b.Fatalf("Next error: %v", err)
		}
		balancer.Done(targetKey(*target))
	}
}
