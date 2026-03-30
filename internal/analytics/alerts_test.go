package analytics

import (
	"testing"
	"time"
)

func TestAlertEngineErrorRateAndCooldown(t *testing.T) {
	engine := NewAlertEngine(AlertEngineOptions{})
	_, err := engine.UpsertRule(AlertRule{
		ID:        "r1",
		Name:      "High Error Rate",
		Enabled:   true,
		Type:      AlertRuleErrorRate,
		Threshold: 30,
		Window:    "5m",
		Cooldown:  "2m",
		Action: AlertAction{
			Type: AlertActionLog,
		},
	})
	if err != nil {
		t.Fatalf("upsert rule: %v", err)
	}

	now := time.Now().UTC()
	metrics := []RequestMetric{
		{Timestamp: now.Add(-30 * time.Second), StatusCode: 500, Error: true, LatencyMS: 90},
		{Timestamp: now.Add(-20 * time.Second), StatusCode: 502, Error: true, LatencyMS: 100},
		{Timestamp: now.Add(-10 * time.Second), StatusCode: 200, Error: false, LatencyMS: 80},
	}

	first := engine.Evaluate(metrics, 100, now)
	if len(first) != 1 {
		t.Fatalf("expected first evaluation to trigger once, got %d", len(first))
	}
	if first[0].RuleID != "r1" {
		t.Fatalf("expected rule id r1, got %s", first[0].RuleID)
	}

	second := engine.Evaluate(metrics, 100, now.Add(30*time.Second))
	if len(second) != 0 {
		t.Fatalf("expected cooldown to suppress second trigger, got %d", len(second))
	}

	third := engine.Evaluate(metrics, 100, now.Add(3*time.Minute))
	if len(third) != 1 {
		t.Fatalf("expected cooldown expiry to allow trigger, got %d", len(third))
	}
}

func TestAlertEngineUpstreamHealth(t *testing.T) {
	engine := NewAlertEngine(AlertEngineOptions{})
	_, err := engine.UpsertRule(AlertRule{
		ID:        "r2",
		Name:      "Upstream Health Low",
		Enabled:   true,
		Type:      AlertRuleUpstreamHealth,
		Threshold: 80,
		Window:    "1m",
		Cooldown:  "30s",
		Action: AlertAction{
			Type: AlertActionLog,
		},
	})
	if err != nil {
		t.Fatalf("upsert rule: %v", err)
	}

	now := time.Now().UTC()
	triggered := engine.Evaluate(nil, 55, now)
	if len(triggered) != 1 {
		t.Fatalf("expected upstream health alert trigger, got %d", len(triggered))
	}

	notTriggered := engine.Evaluate(nil, 99, now.Add(time.Minute))
	if len(notTriggered) != 0 {
		t.Fatalf("expected no alert with healthy upstream percent, got %d", len(notTriggered))
	}
}
