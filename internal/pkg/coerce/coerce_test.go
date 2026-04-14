package coerce

import (
	"fmt"
	"testing"
	"time"
)

func TestAsString(t *testing.T) {
	tests := []struct {
		input    any
		expected string
	}{
		{nil, ""},
		{"hello", "hello"},
		{"  hello  ", "hello"},
		{"hello\nworld", "hello world"},
		{"hello\r\nworld", "hello  world"},
		{123, "123"},
		{int64(456), "456"},
		{3.14, "3.14"},
		{true, "true"},
		{false, "false"},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%T(%v)", tt.input, tt.input), func(t *testing.T) {
			t.Parallel()
			got := AsString(tt.input)
			if got != tt.expected {
				t.Errorf("AsString(%v) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestAsStringPtr(t *testing.T) {
	t.Run("nil returns nil", func(t *testing.T) {
		t.Parallel()
		got := AsStringPtr(nil)
		if got != nil {
			t.Error("expected nil for nil input")
		}
	})
	t.Run("string returns pointer", func(t *testing.T) {
		t.Parallel()
		got := AsStringPtr("hello")
		if got == nil || *got != "hello" {
			t.Errorf("expected *\"hello\", got %v", got)
		}
	})
}

func TestAsInt(t *testing.T) {
	tests := []struct {
		input    any
		fallback int
		expected int
	}{
		{nil, -1, -1},
		{42, -1, 42},
		{int64(100), -1, 100},
		{int32(50), -1, 50},
		{float64(3.7), -1, 3},
		{float32(2.9), -1, 2},
		{"123", -1, 123},
		{"-5", -1, -5},
		{"not a number", -1, -1},
		{true, -1, -1},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v", tt.input), func(t *testing.T) {
			t.Parallel()
			got := AsInt(tt.input, tt.fallback)
			if got != tt.expected {
				t.Errorf("AsInt(%v, %d) = %d, want %d", tt.input, tt.fallback, got, tt.expected)
			}
		})
	}
}

func TestAsInt64(t *testing.T) {
	tests := []struct {
		input    any
		fallback int64
		expected int64
	}{
		{nil, -1, -1},
		{42, -1, 42},
		{int64(100), -1, 100},
		{int32(50), -1, 50},
		{float64(3.7), -1, 3},
		{"123", -1, 123},
		{"not a number", -1, -1},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v", tt.input), func(t *testing.T) {
			t.Parallel()
			got := AsInt64(tt.input, tt.fallback)
			if got != tt.expected {
				t.Errorf("AsInt64(%v, %d) = %d, want %d", tt.input, tt.fallback, got, tt.expected)
			}
		})
	}
}

func TestAsBool(t *testing.T) {
	tests := []struct {
		input    any
		fallback bool
		expected bool
	}{
		{nil, false, false},
		{nil, true, true},
		{true, false, true},
		{false, true, false},
		{"1", false, true},
		{"true", false, true},
		{"TRUE", false, true},
		{"yes", false, true},
		{"on", false, true},
		{"0", false, false},
		{"false", false, false},
		{"no", false, false},
		{"off", false, false},
		{"", false, false},
		{"random", false, false},
		{42, false, false},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v/%v", tt.input, tt.fallback), func(t *testing.T) {
			t.Parallel()
			got := AsBool(tt.input, tt.fallback)
			if got != tt.expected {
				t.Errorf("AsBool(%v, %v) = %v, want %v", tt.input, tt.fallback, got, tt.expected)
			}
		})
	}
}

func TestAsFloat64(t *testing.T) {
	tests := []struct {
		input    any
		expected float64
		ok       bool
	}{
		{nil, 0, false},
		{3.14, 3.14, true},
		{float32(2.5), 2.5, true},
		{42, 42.0, true},
		{int64(100), 100.0, true},
		{"3.14", 3.14, true},
		{"not a float", 0, false},
		{true, 0, false},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v", tt.input), func(t *testing.T) {
			t.Parallel()
			got, ok := AsFloat64(tt.input, 0)
			if ok != tt.ok || (ok && got != tt.expected) {
				t.Errorf("AsFloat64(%v) = (%f, %v), want (%f, %v)", tt.input, got, ok, tt.expected, tt.ok)
			}
		})
	}
}

func TestAsFloat(t *testing.T) {
	tests := []struct {
		input    any
		fallback float64
		expected float64
	}{
		{nil, -1.0, -1.0},
		{3.14, -1.0, 3.14},
		{"2.5", -1.0, 2.5},
		{"bad", -1.0, -1.0},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v", tt.input), func(t *testing.T) {
			t.Parallel()
			got := AsFloat(tt.input, tt.fallback)
			if got != tt.expected {
				t.Errorf("AsFloat(%v, %v) = %v, want %v", tt.input, tt.fallback, got, tt.expected)
			}
		})
	}
}

func TestAsStringSlice(t *testing.T) {
	tests := []struct {
		input    any
		expected []string
	}{
		{nil, nil},
		{[]string{"a", "b"}, []string{"a", "b"}},
		{[]any{"a", "b", ""}, []string{"a", "b"}},
		{[]any{1, "b"}, []string{"b"}},
		{"a,b,c", []string{"a", "b", "c"}},
		{"a, b , c", []string{"a", "b", "c"}},
		{"", nil},
		{42, nil},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v", tt.input), func(t *testing.T) {
			t.Parallel()
			got := AsStringSlice(tt.input)
			if len(got) != len(tt.expected) {
				t.Errorf("AsStringSlice(%v) = %v, want %v", tt.input, got, tt.expected)
				return
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("AsStringSlice(%v)[%d] = %q, want %q", tt.input, i, got[i], tt.expected[i])
				}
			}
		})
	}
}

func TestAsIntSlice(t *testing.T) {
	fallback := []int{-1}
	tests := []struct {
		input    any
		expected []int
	}{
		{nil, fallback},
		{[]int{1, 2, 3}, []int{1, 2, 3}},
		{[]any{1, 2, 3}, []int{1, 2, 3}},
		{42, fallback},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v", tt.input), func(t *testing.T) {
			t.Parallel()
			got := AsIntSlice(tt.input, fallback)
			if len(got) != len(tt.expected) {
				t.Errorf("AsIntSlice(%v) = %v, want %v", tt.input, got, tt.expected)
				return
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("AsIntSlice(%v)[%d] = %d, want %d", tt.input, i, got[i], tt.expected[i])
				}
			}
		})
	}
}

func TestAsAnyMap(t *testing.T) {
	t.Run("nil returns empty map", func(t *testing.T) {
		t.Parallel()
		got := AsAnyMap(nil)
		if got == nil || len(got) != 0 {
			t.Error("expected non-nil empty map")
		}
	})
	t.Run("map[string]any trims keys", func(t *testing.T) {
		t.Parallel()
		got := AsAnyMap(map[string]any{"  key  ": "value", "": "skip"})
		if len(got) != 1 || got["key"] != "value" {
			t.Errorf("unexpected result: %v", got)
		}
	})
	t.Run("map[any]any", func(t *testing.T) {
		t.Parallel()
		got := AsAnyMap(map[any]any{"key": "value", 123: "skip"})
		if len(got) != 1 || got["key"] != "value" {
			t.Errorf("unexpected result: %v", got)
		}
	})
	t.Run("unsupported type returns empty", func(t *testing.T) {
		t.Parallel()
		got := AsAnyMap("not a map")
		if len(got) != 0 {
			t.Errorf("expected empty map, got %v", got)
		}
	})
}

func TestAsStringMap(t *testing.T) {
	t.Run("nil returns nil", func(t *testing.T) {
		t.Parallel()
		got := AsStringMap(nil)
		if got != nil {
			t.Error("expected nil")
		}
	})
	t.Run("map[string]any converts values", func(t *testing.T) {
		t.Parallel()
		got := AsStringMap(map[string]any{"key": "value", "num": 42})
		if got["key"] != "value" {
			t.Errorf("expected 'value', got %q", got["key"])
		}
		if got["num"] != "42" {
			t.Errorf("expected '42', got %q", got["num"])
		}
	})
	t.Run("map[string]string passthrough", func(t *testing.T) {
		t.Parallel()
		got := AsStringMap(map[string]string{"key": "value"})
		if got["key"] != "value" {
			t.Error("expected passthrough")
		}
	})
}

func TestAsDuration(t *testing.T) {
	tests := []struct {
		input    any
		fallback time.Duration
		expected time.Duration
	}{
		{nil, time.Second, time.Second},
		{5 * time.Second, time.Second, 5 * time.Second},
		{10, time.Second, 10 * time.Second},
		{int64(30), time.Second, 30 * time.Second},
		{float64(1.5), time.Second, 1500 * time.Millisecond},
		{"500ms", time.Second, 500 * time.Millisecond},
		{"10s", time.Second, 10 * time.Second},
		{"60", time.Second, 60 * time.Second},
		{"invalid", time.Second, time.Second},
		{true, time.Second, time.Second},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v", tt.input), func(t *testing.T) {
			t.Parallel()
			got := AsDuration(tt.input, tt.fallback)
			if got != tt.expected {
				t.Errorf("AsDuration(%v) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestGet(t *testing.T) {
	m := map[string]any{
		"primary":   "value1",
		"fallback1": "value2",
	}
	t.Run("exact key match", func(t *testing.T) {
		t.Parallel()
		got := Get(m, "primary")
		if got != "value1" {
			t.Errorf("expected 'value1', got %v", got)
		}
	})
	t.Run("fallback key match", func(t *testing.T) {
		t.Parallel()
		got := Get(m, "missing", "fallback1")
		if got != "value2" {
			t.Errorf("expected 'value2', got %v", got)
		}
	})
	t.Run("no match returns nil", func(t *testing.T) {
		t.Parallel()
		got := Get(m, "missing", "also_missing")
		if got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})
	t.Run("nil map returns nil", func(t *testing.T) {
		t.Parallel()
		got := Get(nil, "key")
		if got != nil {
			t.Error("expected nil for nil map")
		}
	})
}

func TestGetHelpers(t *testing.T) {
	m := map[string]any{
		"str":   "hello",
		"num":   42,
		"flag":  true,
		"slice": "a,b,c",
	}
	t.Run("GetString", func(t *testing.T) {
		t.Parallel()
		if got := GetString(m, "str"); got != "hello" {
			t.Errorf("got %q, want 'hello'", got)
		}
	})
	t.Run("GetInt", func(t *testing.T) {
		t.Parallel()
		if got := GetInt(m, "num", -1); got != 42 {
			t.Errorf("got %d, want 42", got)
		}
		if got := GetInt(m, "missing", -1); got != -1 {
			t.Errorf("got %d, want -1", got)
		}
	})
	t.Run("GetBool", func(t *testing.T) {
		t.Parallel()
		if got := GetBool(m, "flag", false); !got {
			t.Error("expected true")
		}
		if got := GetBool(m, "missing", false); got {
			t.Error("expected false")
		}
	})
	t.Run("GetStringSlice", func(t *testing.T) {
		t.Parallel()
		got := GetStringSlice(m, "slice")
		if len(got) != 3 || got[0] != "a" {
			t.Errorf("expected [a b c], got %v", got)
		}
	})
}
