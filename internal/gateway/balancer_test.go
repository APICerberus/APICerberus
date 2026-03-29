package gateway

import (
	"testing"

	"github.com/APICerberus/APICerebrus/internal/config"
)

func TestRoundRobinDistribution(t *testing.T) {
	t.Parallel()

	rr := NewRoundRobin([]config.UpstreamTarget{
		{ID: "a", Address: "10.0.0.1:8080"},
		{ID: "b", Address: "10.0.0.2:8080"},
	})

	counts := map[string]int{"a": 0, "b": 0}
	for i := 0; i < 100; i++ {
		target, err := rr.Next(nil)
		if err != nil {
			t.Fatalf("Next error: %v", err)
		}
		counts[target.ID]++
	}

	if counts["a"] != 50 || counts["b"] != 50 {
		t.Fatalf("unexpected round robin distribution: %#v", counts)
	}
}

func TestWeightedRoundRobinDistribution(t *testing.T) {
	t.Parallel()

	wrr := NewWeightedRoundRobin([]config.UpstreamTarget{
		{ID: "a", Address: "10.0.0.1:8080", Weight: 1},
		{ID: "b", Address: "10.0.0.2:8080", Weight: 3},
	})

	counts := map[string]int{"a": 0, "b": 0}
	for i := 0; i < 400; i++ {
		target, err := wrr.Next(nil)
		if err != nil {
			t.Fatalf("Next error: %v", err)
		}
		counts[target.ID]++
	}

	if counts["a"] != 100 || counts["b"] != 300 {
		t.Fatalf("unexpected weighted distribution: %#v", counts)
	}
}

func TestBalancerFactory(t *testing.T) {
	t.Parallel()

	targets := []config.UpstreamTarget{{ID: "a", Address: "10.0.0.1:8080"}}
	if _, ok := NewBalancer("weighted_round_robin", targets).(*WeightedRoundRobin); !ok {
		t.Fatalf("expected weighted round robin balancer")
	}
	if _, ok := NewBalancer("unknown", targets).(*RoundRobin); !ok {
		t.Fatalf("expected fallback round robin balancer")
	}
}

func TestUpstreamPoolSkipsUnhealthyTargets(t *testing.T) {
	t.Parallel()

	pool := NewUpstreamPool(config.Upstream{
		Name:      "up-users",
		Algorithm: "round_robin",
		Targets: []config.UpstreamTarget{
			{ID: "a", Address: "10.0.0.1:8080"},
			{ID: "b", Address: "10.0.0.2:8080"},
		},
	})

	pool.ReportHealth("a", false, 0)
	for i := 0; i < 20; i++ {
		target, err := pool.Next(nil)
		if err != nil {
			t.Fatalf("Next error: %v", err)
		}
		if target.ID != "b" {
			t.Fatalf("expected only healthy target b, got %q", target.ID)
		}
	}
}
