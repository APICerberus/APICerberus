package yaml

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestParseCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want any
	}{
		{
			name: "simple map",
			in:   "name: cerberus\n",
			want: map[string]any{"name": "cerberus"},
		},
		{
			name: "nested map",
			in: "gateway:\n" +
				"  http:\n" +
				"    addr: \":8080\"\n",
			want: map[string]any{
				"gateway": map[string]any{
					"http": map[string]any{
						"addr": ":8080",
					},
				},
			},
		},
		{
			name: "sequence of scalars",
			in: "methods:\n" +
				"  - GET\n" +
				"  - POST\n",
			want: map[string]any{
				"methods": []any{"GET", "POST"},
			},
		},
		{
			name: "sequence of maps",
			in: "targets:\n" +
				"  - id: t1\n" +
				"    address: http://a\n" +
				"  - id: t2\n" +
				"    address: http://b\n",
			want: map[string]any{
				"targets": []any{
					map[string]any{"id": "t1", "address": "http://a"},
					map[string]any{"id": "t2", "address": "http://b"},
				},
			},
		},
		{
			name: "nested sequence under map",
			in: "route:\n" +
				"  paths:\n" +
				"    - /a\n" +
				"    - /b\n",
			want: map[string]any{
				"route": map[string]any{
					"paths": []any{"/a", "/b"},
				},
			},
		},
		{
			name: "quoted strings and comments",
			in: "text: \"value # keep\"\n" +
				"desc: 'it''s fine'\n" +
				"# this is ignored\n" +
				"flag: true # trailing comment\n",
			want: map[string]any{
				"text": "value # keep",
				"desc": "it's fine",
				"flag": "true",
			},
		},
		{
			name: "literal multiline",
			in: "message: |\n" +
				"  line1\n" +
				"  line2\n",
			want: map[string]any{
				"message": "line1\nline2",
			},
		},
		{
			name: "folded multiline",
			in: "message: >\n" +
				"  line1\n" +
				"  line2\n" +
				"\n" +
				"  line4\n",
			want: map[string]any{
				"message": "line1 line2\n\nline4",
			},
		},
		{
			name: "empty scalar when no nested block",
			in: "a:\n" +
				"b: c\n",
			want: map[string]any{
				"a": "",
				"b": "c",
			},
		},
		{
			name: "sequence items as nested maps",
			in: "items:\n" +
				"  -\n" +
				"    name: one\n" +
				"    age: 1\n" +
				"  -\n" +
				"    name: two\n",
			want: map[string]any{
				"items": []any{
					map[string]any{"name": "one", "age": "1"},
					map[string]any{"name": "two"},
				},
			},
		},
		{
			name: "quoted key",
			in:   "\"complex key\": value\n",
			want: map[string]any{
				"complex key": "value",
			},
		},
		{
			name: "top level sequence",
			in:   "- one\n- two\n",
			want: []any{"one", "two"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			node, err := Parse([]byte(tt.in))
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			got := nodeToAny(node)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("unexpected parse result\nwant: %#v\ngot:  %#v", tt.want, got)
			}
		})
	}
}

func TestParseErrors(t *testing.T) {
	t.Parallel()

	_, err := Parse([]byte("a:\n   b: c\n  d: e\n"))
	if err == nil {
		t.Fatalf("expected indentation error")
	}

	_, err = Parse([]byte("a:\n\tb: c\n"))
	if err == nil {
		t.Fatalf("expected tab indentation error")
	}
}

func TestUnmarshalCoercionCases(t *testing.T) {
	t.Parallel()

	type nested struct {
		Limit int `yaml:"limit"`
	}
	type cfg struct {
		Port     int                `yaml:"port"`
		Ratio    float64            `yaml:"ratio"`
		Enabled  bool               `yaml:"enabled"`
		Timeout  time.Duration      `yaml:"timeout"`
		Names    []string           `yaml:"names"`
		Codes    []int              `yaml:"codes"`
		Weights  map[string]float64 `yaml:"weights"`
		Nested   nested             `yaml:"nested"`
		Optional *nested            `yaml:"optional"`
	}

	tests := []struct {
		name string
		in   string
		want func(t *testing.T, got cfg)
	}{
		{
			name: "int coercion",
			in:   "port: \"8080\"\n",
			want: func(t *testing.T, got cfg) {
				t.Helper()
				if got.Port != 8080 {
					t.Fatalf("want port=8080 got %d", got.Port)
				}
			},
		},
		{
			name: "float coercion",
			in:   "ratio: \"0.75\"\n",
			want: func(t *testing.T, got cfg) {
				t.Helper()
				if got.Ratio != 0.75 {
					t.Fatalf("want ratio=0.75 got %v", got.Ratio)
				}
			},
		},
		{
			name: "bool coercion yes",
			in:   "enabled: yes\n",
			want: func(t *testing.T, got cfg) {
				t.Helper()
				if !got.Enabled {
					t.Fatalf("want enabled=true")
				}
			},
		},
		{
			name: "bool coercion off",
			in:   "enabled: off\n",
			want: func(t *testing.T, got cfg) {
				t.Helper()
				if got.Enabled {
					t.Fatalf("want enabled=false")
				}
			},
		},
		{
			name: "duration coercion",
			in:   "timeout: 1m30s\n",
			want: func(t *testing.T, got cfg) {
				t.Helper()
				if got.Timeout != (90 * time.Second) {
					t.Fatalf("want timeout=90s got %v", got.Timeout)
				}
			},
		},
		{
			name: "string slice coercion",
			in: "names:\n" +
				"  - alpha\n" +
				"  - beta\n",
			want: func(t *testing.T, got cfg) {
				t.Helper()
				if !reflect.DeepEqual(got.Names, []string{"alpha", "beta"}) {
					t.Fatalf("unexpected names: %#v", got.Names)
				}
			},
		},
		{
			name: "int slice coercion",
			in: "codes:\n" +
				"  - \"1\"\n" +
				"  - \"2\"\n",
			want: func(t *testing.T, got cfg) {
				t.Helper()
				if !reflect.DeepEqual(got.Codes, []int{1, 2}) {
					t.Fatalf("unexpected codes: %#v", got.Codes)
				}
			},
		},
		{
			name: "map coercion",
			in: "weights:\n" +
				"  a: \"1.5\"\n" +
				"  b: \"2.5\"\n",
			want: func(t *testing.T, got cfg) {
				t.Helper()
				if got.Weights["a"] != 1.5 || got.Weights["b"] != 2.5 {
					t.Fatalf("unexpected weights: %#v", got.Weights)
				}
			},
		},
		{
			name: "nested struct",
			in: "nested:\n" +
				"  limit: \"42\"\n",
			want: func(t *testing.T, got cfg) {
				t.Helper()
				if got.Nested.Limit != 42 {
					t.Fatalf("unexpected nested limit: %d", got.Nested.Limit)
				}
			},
		},
		{
			name: "pointer nested struct",
			in: "optional:\n" +
				"  limit: 7\n",
			want: func(t *testing.T, got cfg) {
				t.Helper()
				if got.Optional == nil || got.Optional.Limit != 7 {
					t.Fatalf("unexpected optional: %#v", got.Optional)
				}
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var out cfg
			if err := Unmarshal([]byte(tt.in), &out); err != nil {
				t.Fatalf("Unmarshal error: %v", err)
			}
			tt.want(t, out)
		})
	}
}

func TestMarshalCases(t *testing.T) {
	t.Parallel()

	type child struct {
		Name string `yaml:"name"`
	}
	type cfg struct {
		Port    int           `yaml:"port"`
		Enabled bool          `yaml:"enabled"`
		Timeout time.Duration `yaml:"timeout"`
		Message string        `yaml:"message"`
		Items   []child       `yaml:"items"`
	}

	input := cfg{
		Port:    8080,
		Enabled: true,
		Timeout: 2 * time.Second,
		Message: "line1\nline2",
		Items: []child{
			{Name: "a"},
			{Name: "b"},
		},
	}

	data, err := Marshal(input)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	yml := string(data)
	mustContainAll(t, yml,
		"port: 8080",
		"enabled: true",
		"timeout: 2s",
		"message: |",
		"-",
		"name: a",
		"name: b",
	)

	var out cfg
	if err := Unmarshal(data, &out); err != nil {
		t.Fatalf("roundtrip Unmarshal error: %v", err)
	}
	if out.Port != input.Port || out.Enabled != input.Enabled || out.Timeout != input.Timeout {
		t.Fatalf("roundtrip mismatch: %#v", out)
	}
	if len(out.Items) != 2 || out.Items[0].Name != "a" || out.Items[1].Name != "b" {
		t.Fatalf("roundtrip items mismatch: %#v", out.Items)
	}
}

func mustContainAll(t *testing.T, text string, parts ...string) {
	t.Helper()
	for _, part := range parts {
		if !strings.Contains(text, part) {
			t.Fatalf("expected output to contain %q\nfull output:\n%s", part, text)
		}
	}
}
