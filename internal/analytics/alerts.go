package analytics

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

type AlertRuleType string

const (
	AlertRuleErrorRate      AlertRuleType = "error_rate"
	AlertRuleP99Latency     AlertRuleType = "p99_latency"
	AlertRuleUpstreamHealth AlertRuleType = "upstream_health"
)

type AlertActionType string

const (
	AlertActionLog     AlertActionType = "log"
	AlertActionWebhook AlertActionType = "webhook"
)

type AlertAction struct {
	Type       AlertActionType `json:"type"`
	WebhookURL string          `json:"webhook_url,omitempty"`
}

type AlertRule struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	Enabled   bool          `json:"enabled"`
	Type      AlertRuleType `json:"type"`
	Threshold float64       `json:"threshold"`
	Window    string        `json:"window"`
	Cooldown  string        `json:"cooldown"`
	Action    AlertAction   `json:"action"`
}

type AlertHistoryEntry struct {
	ID          string          `json:"id"`
	RuleID      string          `json:"rule_id"`
	RuleName    string          `json:"rule_name"`
	RuleType    AlertRuleType   `json:"rule_type"`
	TriggeredAt time.Time       `json:"triggered_at"`
	Value       float64         `json:"value"`
	Threshold   float64         `json:"threshold"`
	ActionType  AlertActionType `json:"action_type"`
	Success     bool            `json:"success"`
	Error       string          `json:"error,omitempty"`
}

type AlertEngineOptions struct {
	WebhookTimeout time.Duration
	MaxHistory     int
}

type AlertEngine struct {
	mu            sync.RWMutex
	rules         map[string]AlertRule
	history       []AlertHistoryEntry
	lastTriggered map[string]time.Time
	httpClient    *http.Client
	maxHistory    int
}

func NewAlertEngine(options AlertEngineOptions) *AlertEngine {
	if options.WebhookTimeout <= 0 {
		options.WebhookTimeout = 5 * time.Second
	}
	if options.MaxHistory <= 0 {
		options.MaxHistory = 500
	}
	return &AlertEngine{
		rules:         make(map[string]AlertRule),
		history:       make([]AlertHistoryEntry, 0, options.MaxHistory),
		lastTriggered: make(map[string]time.Time),
		httpClient:    &http.Client{Timeout: options.WebhookTimeout},
		maxHistory:    options.MaxHistory,
	}
}

func (engine *AlertEngine) ListRules() []AlertRule {
	if engine == nil {
		return nil
	}
	engine.mu.RLock()
	defer engine.mu.RUnlock()

	out := make([]AlertRule, 0, len(engine.rules))
	for _, rule := range engine.rules {
		out = append(out, rule)
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out
}

func (engine *AlertEngine) GetRule(id string) (AlertRule, bool) {
	if engine == nil {
		return AlertRule{}, false
	}
	engine.mu.RLock()
	defer engine.mu.RUnlock()
	rule, ok := engine.rules[strings.TrimSpace(id)]
	return rule, ok
}

func (engine *AlertEngine) UpsertRule(rule AlertRule) (AlertRule, error) {
	if engine == nil {
		return AlertRule{}, fmt.Errorf("alert engine is nil")
	}
	normalized, err := normalizeRule(rule)
	if err != nil {
		return AlertRule{}, err
	}
	engine.mu.Lock()
	engine.rules[normalized.ID] = normalized
	engine.mu.Unlock()
	return normalized, nil
}

func (engine *AlertEngine) DeleteRule(id string) bool {
	if engine == nil {
		return false
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return false
	}
	engine.mu.Lock()
	defer engine.mu.Unlock()
	if _, ok := engine.rules[id]; !ok {
		return false
	}
	delete(engine.rules, id)
	delete(engine.lastTriggered, id)
	return true
}

func (engine *AlertEngine) History(limit int) []AlertHistoryEntry {
	if engine == nil {
		return nil
	}
	engine.mu.RLock()
	defer engine.mu.RUnlock()
	if limit <= 0 || limit > len(engine.history) {
		limit = len(engine.history)
	}
	out := make([]AlertHistoryEntry, 0, limit)
	for i := len(engine.history) - 1; i >= 0 && len(out) < limit; i-- {
		out = append(out, engine.history[i])
	}
	return out
}

func (engine *AlertEngine) Evaluate(metrics []RequestMetric, upstreamHealthPercent float64, now time.Time) []AlertHistoryEntry {
	if engine == nil {
		return nil
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}

	rules := engine.ListRules()
	triggered := make([]AlertHistoryEntry, 0, len(rules))

	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}

		value, shouldTrigger, ok := evaluateRule(rule, metrics, upstreamHealthPercent, now)
		if !ok || !shouldTrigger {
			continue
		}

		if !engine.canTrigger(rule.ID, now, parseDuration(rule.Cooldown, time.Minute)) {
			continue
		}

		history := AlertHistoryEntry{
			ID:          fmt.Sprintf("%d-%s", now.UnixNano(), rule.ID),
			RuleID:      rule.ID,
			RuleName:    rule.Name,
			RuleType:    rule.Type,
			TriggeredAt: now,
			Value:       value,
			Threshold:   rule.Threshold,
			ActionType:  rule.Action.Type,
			Success:     true,
		}

		if err := engine.executeAction(rule, history); err != nil {
			history.Success = false
			history.Error = err.Error()
		}

		engine.recordHistory(history)
		triggered = append(triggered, history)
	}

	return triggered
}

func (engine *AlertEngine) canTrigger(ruleID string, now time.Time, cooldown time.Duration) bool {
	engine.mu.Lock()
	defer engine.mu.Unlock()

	last := engine.lastTriggered[ruleID]
	if !last.IsZero() && now.Sub(last) < cooldown {
		return false
	}
	engine.lastTriggered[ruleID] = now
	return true
}

func (engine *AlertEngine) recordHistory(entry AlertHistoryEntry) {
	engine.mu.Lock()
	engine.history = append(engine.history, entry)
	if len(engine.history) > engine.maxHistory {
		engine.history = engine.history[len(engine.history)-engine.maxHistory:]
	}
	engine.mu.Unlock()
}

func (engine *AlertEngine) executeAction(rule AlertRule, entry AlertHistoryEntry) error {
	switch rule.Action.Type {
	case AlertActionWebhook:
		url := strings.TrimSpace(rule.Action.WebhookURL)
		if url == "" {
			return fmt.Errorf("webhook_url is required for webhook action")
		}
		payload, err := json.Marshal(map[string]any{
			"alert": entry,
			"rule":  rule,
		})
		if err != nil {
			return err
		}
		resp, err := engine.httpClient.Post(url, "application/json", bytes.NewReader(payload))
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode >= http.StatusMultipleChoices {
			return fmt.Errorf("webhook returned status %d", resp.StatusCode)
		}
		return nil
	case AlertActionLog:
		fallthrough
	default:
		return nil
	}
}

func normalizeRule(rule AlertRule) (AlertRule, error) {
	rule.ID = strings.TrimSpace(rule.ID)
	rule.Name = strings.TrimSpace(rule.Name)
	rule.Window = strings.TrimSpace(rule.Window)
	rule.Cooldown = strings.TrimSpace(rule.Cooldown)
	rule.Action.WebhookURL = strings.TrimSpace(rule.Action.WebhookURL)

	if rule.ID == "" {
		return AlertRule{}, fmt.Errorf("rule id is required")
	}
	if rule.Name == "" {
		return AlertRule{}, fmt.Errorf("rule name is required")
	}
	switch rule.Type {
	case AlertRuleErrorRate, AlertRuleP99Latency, AlertRuleUpstreamHealth:
	default:
		return AlertRule{}, fmt.Errorf("invalid rule type")
	}
	if rule.Threshold < 0 {
		return AlertRule{}, fmt.Errorf("threshold must be non-negative")
	}
	if rule.Window == "" {
		rule.Window = "5m"
	}
	if _, err := time.ParseDuration(rule.Window); err != nil {
		return AlertRule{}, fmt.Errorf("invalid window duration")
	}
	if rule.Cooldown == "" {
		rule.Cooldown = "1m"
	}
	if _, err := time.ParseDuration(rule.Cooldown); err != nil {
		return AlertRule{}, fmt.Errorf("invalid cooldown duration")
	}
	if rule.Action.Type == "" {
		rule.Action.Type = AlertActionLog
	}
	switch rule.Action.Type {
	case AlertActionLog:
	case AlertActionWebhook:
		if rule.Action.WebhookURL == "" {
			return AlertRule{}, fmt.Errorf("webhook_url is required for webhook action")
		}
	default:
		return AlertRule{}, fmt.Errorf("invalid action type")
	}
	return rule, nil
}

func evaluateRule(rule AlertRule, metrics []RequestMetric, upstreamHealthPercent float64, now time.Time) (float64, bool, bool) {
	window := parseDuration(rule.Window, 5*time.Minute)
	recent := metricsInWindow(metrics, now.Add(-window), now)

	switch rule.Type {
	case AlertRuleErrorRate:
		if len(recent) == 0 {
			return 0, false, false
		}
		errors := 0
		for _, metric := range recent {
			if metric.Error || metric.StatusCode >= 500 {
				errors++
			}
		}
		rate := (float64(errors) / float64(len(recent))) * 100
		return rate, rate > rule.Threshold, true
	case AlertRuleP99Latency:
		if len(recent) == 0 {
			return 0, false, false
		}
		latencies := make([]int64, 0, len(recent))
		for _, metric := range recent {
			latencies = append(latencies, metric.LatencyMS)
		}
		p99 := float64(percentile(latencies, 99))
		return p99, p99 > rule.Threshold, true
	case AlertRuleUpstreamHealth:
		return upstreamHealthPercent, upstreamHealthPercent < rule.Threshold, true
	default:
		return 0, false, false
	}
}

func metricsInWindow(metrics []RequestMetric, from, to time.Time) []RequestMetric {
	if len(metrics) == 0 {
		return nil
	}
	out := make([]RequestMetric, 0, len(metrics))
	for _, metric := range metrics {
		ts := metric.Timestamp.UTC()
		if ts.Before(from) || ts.After(to) {
			continue
		}
		out = append(out, metric)
	}
	return out
}

func parseDuration(raw string, fallback time.Duration) time.Duration {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(raw)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}
