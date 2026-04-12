package yaml

import (
	"strings"
	"testing"
)

func TestUnmarshalBasic(t *testing.T) {
	t.Parallel()

	type Config struct {
		Name  string `yaml:"name"`
		Value int    `yaml:"value"`
	}

	var cfg Config
	if err := Unmarshal([]byte("name: test\nvalue: 42\n"), &cfg); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if cfg.Name != "test" || cfg.Value != 42 {
		t.Fatalf("expected name=test, value=42, got name=%q, value=%d", cfg.Name, cfg.Value)
	}
}

func TestUnmarshalDuration(t *testing.T) {
	t.Parallel()

	type Config struct {
		Timeout string `yaml:"timeout"`
	}

	var cfg Config
	if err := Unmarshal([]byte("timeout: 30s\n"), &cfg); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if cfg.Timeout != "30s" {
		t.Fatalf("expected timeout=30s, got %q", cfg.Timeout)
	}
}

func TestUnmarshalNested(t *testing.T) {
	t.Parallel()

	type Inner struct {
		Key string `yaml:"key"`
	}
	type Outer struct {
		Inner Inner `yaml:"inner"`
	}

	var out Outer
	if err := Unmarshal([]byte("inner:\n  key: value\n"), &out); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if out.Inner.Key != "value" {
		t.Fatalf("expected inner.key=value, got %q", out.Inner.Key)
	}
}

func TestUnmarshalDepthLimit(t *testing.T) {
	t.Parallel()

	// Build deeply nested YAML that exceeds maxYAMLDepth=100
	var sb strings.Builder
	for i := 0; i < 110; i++ {
		sb.WriteString(strings.Repeat(" ", i*2) + "a:\n")
	}
	sb.WriteString(strings.Repeat(" ", 220) + "leaf: true\n")

	var out map[string]any
	err := Unmarshal([]byte(sb.String()), &out)
	if err == nil {
		t.Fatal("expected depth error, got nil")
	}
}

func TestUnmarshalNodeLimit(t *testing.T) {
	t.Parallel()

	// Build YAML with many nodes exceeding maxYAMLNodes=100000
	var sb strings.Builder
	for i := 0; i < 110000; i++ {
		sb.WriteString("k" + string(rune('a'+i%26)) + string(rune(i%10+'0')) + ": v\n")
	}

	var out map[string]any
	err := Unmarshal([]byte(sb.String()), &out)
	if err == nil {
		t.Fatal("expected node limit error, got nil")
	}
}

func TestMarshal(t *testing.T) {
	t.Parallel()

	type Config struct {
		Name  string `yaml:"name"`
		Value int    `yaml:"value"`
	}

	cfg := Config{Name: "test", Value: 42}
	data, err := Marshal(cfg)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var parsed Config
	if err := Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Round-trip Unmarshal error: %v", err)
	}
	if parsed.Name != "test" || parsed.Value != 42 {
		t.Fatalf("round-trip: expected name=test, value=42, got name=%q, value=%d", parsed.Name, parsed.Value)
	}
}

func TestUnmarshalNil(t *testing.T) {
	t.Parallel()

	err := Unmarshal([]byte("foo: bar"), nil)
	if err == nil {
		t.Fatal("expected error for nil target, got nil")
	}
}

func TestUnmarshalNonPointer(t *testing.T) {
	t.Parallel()

	var cfg struct{ Name string }
	err := Unmarshal([]byte("name: test"), cfg)
	if err == nil {
		t.Fatal("expected error for non-pointer target, got nil")
	}
}
