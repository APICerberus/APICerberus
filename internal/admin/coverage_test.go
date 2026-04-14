package admin

import (
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestParseBoolString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input    string
		expected bool
		wantErr  bool
	}{
		{"true", true, false},
		{"1", true, false},
		{"yes", true, false},
		{"on", true, false},
		{"TRUE", true, false},
		{"True", true, false},
		{"false", false, false},
		{"0", false, false},
		{"no", false, false},
		{"off", false, false},
		{"FALSE", false, false},
		{"", false, true},
		{"maybe", false, true},
		{"2", false, true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			got, err := parseBoolString(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseBoolString(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if got != tt.expected {
				t.Errorf("parseBoolString(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestOrDefault(t *testing.T) {
	t.Parallel()
	tests := []struct {
		value    string
		fallback string
		expected string
	}{
		{"hello", "default", "hello"},
		{"", "default", "default"},
		{"value", "", "value"},
		{"", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.value+"_vs_"+tt.fallback, func(t *testing.T) {
			t.Parallel()
			if got := orDefault(tt.value, tt.fallback); got != tt.expected {
				t.Errorf("orDefault(%q, %q) = %q, want %q", tt.value, tt.fallback, got, tt.expected)
			}
		})
	}
}

func TestValidateIPEntry(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid ipv4", "192.168.1.1", false},
		{"valid ipv6", "::1", false},
		{"valid cidr", "10.0.0.0/8", false},
		{"valid cidr v6", "fd00::/64", false},
		{"invalid ip", "not-an-ip", true},
		{"invalid cidr", "10.0.0.0/33", true},
		{"empty", "", true},
		{"whitespace only", "   ", true},
		{"valid with spaces", "  10.0.0.1  ", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := validateIPEntry(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateIPEntry(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestParseAuditSearchFilters_BasicFields(t *testing.T) {
	t.Parallel()
	query := url.Values{
		"user_id":       {"user-123"},
		"api_key_prefix": {"ck_live_"},
		"route":         {"/api/v1/users"},
		"method":        {"GET"},
		"client_ip":     {"10.0.0.1"},
	}
	filters, err := parseAuditSearchFilters(query)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if filters.UserID != "user-123" {
		t.Errorf("UserID = %q, want user-123", filters.UserID)
	}
	if filters.APIKeyPrefix != "ck_live_" {
		t.Errorf("APIKeyPrefix = %q", filters.APIKeyPrefix)
	}
	if filters.Route != "/api/v1/users" {
		t.Errorf("Route = %q", filters.Route)
	}
	if filters.Method != "GET" {
		t.Errorf("Method = %q", filters.Method)
	}
}

func TestParseAuditSearchFilters_StatusRange(t *testing.T) {
	t.Parallel()
	query := url.Values{
		"status_min": {"400"},
		"status_max": {"599"},
	}
	filters, err := parseAuditSearchFilters(query)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if filters.StatusMin != 400 {
		t.Errorf("StatusMin = %d, want 400", filters.StatusMin)
	}
	if filters.StatusMax != 599 {
		t.Errorf("StatusMax = %d, want 599", filters.StatusMax)
	}
}

func TestParseAuditSearchFilters_InvalidStatusMin(t *testing.T) {
	t.Parallel()
	query := url.Values{"status_min": {"abc"}}
	_, err := parseAuditSearchFilters(query)
	if err == nil {
		t.Fatal("expected error for invalid status_min")
	}
	if !strings.Contains(err.Error(), "numeric") {
		t.Errorf("error = %q, want 'numeric'", err.Error())
	}
}

func TestParseAuditSearchFilters_Latency(t *testing.T) {
	t.Parallel()
	query := url.Values{
		"min_latency_ms": {"100"},
	}
	filters, err := parseAuditSearchFilters(query)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if filters.MinLatencyMS != 100 {
		t.Errorf("MinLatencyMS = %d, want 100", filters.MinLatencyMS)
	}
}

func TestParseAuditSearchFilters_Blocked(t *testing.T) {
	t.Parallel()
	query := url.Values{
		"blocked": {"true"},
	}
	filters, err := parseAuditSearchFilters(query)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if filters.Blocked == nil || !*filters.Blocked {
		t.Error("Blocked should be true")
	}
}

func TestParseAuditSearchFilters_DateRange(t *testing.T) {
	t.Parallel()
	query := url.Values{
		"date_from": {"2026-01-01T00:00:00Z"},
		"date_to":   {"2026-01-31T23:59:59Z"},
	}
	filters, err := parseAuditSearchFilters(query)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if filters.DateFrom.IsZero() {
		t.Error("DateFrom should be set")
	}
	if filters.DateTo.IsZero() {
		t.Error("DateTo should be set")
	}
}

func TestParseAuditSearchFilters_FullTextSearch(t *testing.T) {
	t.Parallel()
	query := url.Values{
		"q": {"error timeout"},
	}
	filters, err := parseAuditSearchFilters(query)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if filters.FullText != "error timeout" {
		t.Errorf("FullText = %q", filters.FullText)
	}
}

func TestParseAuditTime(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"rfc3339", "2026-01-15T10:30:00Z", false},
		{"rfc3339nano", "2026-01-15T10:30:00.123456789Z", false},
		{"empty", "", true},
		{"invalid", "not-a-date", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := parseAuditTime(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseAuditTime(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestAuditExportContentType(t *testing.T) {
	t.Parallel()
	tests := []struct {
		format   string
		expected string
	}{
		{"csv", "text/csv; charset=utf-8"},
		{"json", "application/json; charset=utf-8"},
		{"CSV", "text/csv; charset=utf-8"},
		{"ndjson", "application/x-ndjson; charset=utf-8"},
		{"", "application/x-ndjson; charset=utf-8"},
		{"unknown", "application/x-ndjson; charset=utf-8"},
	}
	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			t.Parallel()
			if got := auditExportContentType(tt.format); got != tt.expected {
				t.Errorf("auditExportContentType(%q) = %q, want %q", tt.format, got, tt.expected)
			}
		})
	}
}

func TestGenerateSecureRandomHex(t *testing.T) {
	t.Parallel()
	result, err := generateSecureRandomHex(16)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 32 { // 16 bytes = 32 hex chars
		t.Errorf("hex length = %d, want 32", len(result))
	}
	for _, c := range result {
		if !strings.Contains("0123456789abcdef", string(c)) {
			t.Errorf("unexpected char %c in hex", c)
		}
	}
}

func TestGenerateSecureRandomHex_Uniqueness(t *testing.T) {
	t.Parallel()
	a, _ := generateSecureRandomHex(32)
	b, _ := generateSecureRandomHex(32)
	if a == b {
		t.Error("two random hex values should not be equal")
	}
}

func TestRandomString(t *testing.T) {
	t.Parallel()
	result, err := randomString(16)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 16 {
		t.Errorf("length = %d, want 16", len(result))
	}
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	for _, c := range result {
		if !strings.ContainsRune(letters, c) {
			t.Errorf("unexpected char %c", c)
		}
	}
}

func TestRandomString_ZeroLength(t *testing.T) {
	t.Parallel()
	result, err := randomString(0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("length = %d, want 0", len(result))
	}
}

func TestGenerateConnID(t *testing.T) {
	t.Parallel()
	id, err := generateConnID()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Format: YYYYMMDDHHmmss-random8chars
	if len(id) != 23 { // 14 (timestamp) + 1 (dash) + 8 (random)
		t.Errorf("conn ID length = %d, want 23, got %q", len(id), id)
	}
	// Should start with a timestamp-like pattern
	if id[14] != '-' {
		t.Errorf("expected dash at position 14, got %c", id[14])
	}
	// Validate timestamp part parses
	tsPart := id[:14]
	if _, err := time.Parse("20060102150405", tsPart); err != nil {
		t.Errorf("timestamp part %q is not valid: %v", tsPart, err)
	}
}

func TestGenerateConnID_Uniqueness(t *testing.T) {
	t.Parallel()
	ids := make(map[string]bool)
	for i := 0; i < 10; i++ {
		id, err := generateConnID()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ids[id] {
			t.Errorf("duplicate conn ID: %s", id)
		}
		ids[id] = true
	}
}
