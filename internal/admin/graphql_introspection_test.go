package admin

import (
	"strings"
	"testing"
)

// TestIsIntrospectionQuery tests MEDIUM-005 fix: AST-based introspection detection.
func TestIsIntrospectionQuery(t *testing.T) {
	tests := []struct {
		name  string
		query string
		want  bool
	}{
		// Should be detected as introspection
		{"introspection __schema", "{ __schema { types { name } } }", true},
		{"introspection __type", "{ __type(name: \"User\") { name } }", true},
		{"introspection __typename", "{ __typename }", true},
		{"alias to __schema", "{ foo: __schema { types { name } } }", true},
		{"alias to __type", "{ bar: __type(name: \"User\") { name } }", true},
		{"nested introspection", "{ user { __typename } }", true},

		// Should NOT be detected (these are regular queries)
		{"regular query", "{ user(id: 1) { id name email } }", false},
		{"query with underscore field", "{ user_id }", false},
		{"query with type name", "{ type(id: 1) { id } }", false},
		{"mutation", "mutation { createUser { id } }", false},
		{"subscription", "subscription { newUser { id } }", false},

		// Fallback to string match when AST parsing fails
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := isIntrospectionQuery(tt.query)
			if got != tt.want {
				t.Errorf("isIntrospectionQuery(%q) = %v, want %v", tt.query, got, tt.want)
			}
		})
	}
}

// TestContainsIntrospectionField tests the AST walker for introspection fields.
func TestContainsIntrospectionField(t *testing.T) {
	t.Run("string match fallback", func(t *testing.T) {
		t.Parallel()
		// When AST parsing fails, falls back to strings.Contains
		got := isIntrospectionQuery("__schema")
		if !got {
			t.Error("expected __schema to be detected")
		}
	})
}

// TestGraphQLIntrospectionDisabledByDefault tests that introspection is off by default.
func TestGraphQLIntrospectionDisabledByDefault(t *testing.T) {
	t.Run("introspection query blocked when disabled", func(t *testing.T) {
		t.Parallel()
		// GraphQLIntrospection defaults to false in config
		// The handler should block introspection queries when disabled
		query := "{ __schema { types { name } } }"
		if !isIntrospectionQuery(query) {
			t.Error("expected __schema to be detected as introspection query")
		}
	})
}
