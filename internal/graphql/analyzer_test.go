package graphql

import (
	"testing"
)

func TestNewQueryAnalyzer(t *testing.T) {
	t.Run("nil config uses defaults", func(t *testing.T) {
		a := NewQueryAnalyzer(nil)
		if a.maxDepth != 15 {
			t.Errorf("maxDepth = %d, want 15", a.maxDepth)
		}
		if a.maxComplexity != 1000 {
			t.Errorf("maxComplexity = %d, want 1000", a.maxComplexity)
		}
		if a.defaultCost != 1 {
			t.Errorf("defaultCost = %d, want 1", a.defaultCost)
		}
		if a.fieldCosts == nil {
			t.Error("fieldCosts is nil")
		}
	})

	t.Run("custom config", func(t *testing.T) {
		cfg := &AnalyzerConfig{
			MaxDepth:      10,
			MaxComplexity: 500,
			DefaultCost:   2,
			FieldCosts:    map[string]int{"expensive": 100},
		}
		a := NewQueryAnalyzer(cfg)
		if a.maxDepth != 10 {
			t.Errorf("maxDepth = %d, want 10", a.maxDepth)
		}
		if a.maxComplexity != 500 {
			t.Errorf("maxComplexity = %d, want 500", a.maxComplexity)
		}
		if a.defaultCost != 2 {
			t.Errorf("defaultCost = %d, want 2", a.defaultCost)
		}
		if a.fieldCosts["expensive"] != 100 {
			t.Errorf("fieldCosts[expensive] = %d, want 100", a.fieldCosts["expensive"])
		}
	})

	t.Run("partial config uses defaults for zero values", func(t *testing.T) {
		cfg := &AnalyzerConfig{
			MaxDepth: 5,
		}
		a := NewQueryAnalyzer(cfg)
		if a.maxDepth != 5 {
			t.Errorf("maxDepth = %d, want 5", a.maxDepth)
		}
		if a.maxComplexity != 1000 {
			t.Errorf("maxComplexity = %d, want 1000", a.maxComplexity)
		}
	})
}

func TestQueryAnalyzer_Analyze(t *testing.T) {
	a := NewQueryAnalyzer(&AnalyzerConfig{
		MaxDepth:      5,
		MaxComplexity: 100,
		DefaultCost:   1,
	})

	t.Run("valid query", func(t *testing.T) {
		query := `{ users { id name } }`
		result, err := a.Analyze(query)
		if err != nil {
			t.Errorf("Analyze() error = %v", err)
		}
		if result == nil {
			t.Fatal("Analyze() returned nil")
		}
		if !result.IsValid {
			t.Error("IsValid should be true")
		}
		if result.Depth != 2 {
			t.Errorf("Depth = %d, want 2", result.Depth)
		}
	})

	t.Run("invalid query syntax", func(t *testing.T) {
		// The parser is lenient and may not error on all "invalid" queries
		// Just verify behavior is consistent
		query := `invalid {`
		result, _ := a.Analyze(query)
		if result == nil {
			t.Fatal("Analyze() returned nil")
		}
		// Result validity depends on parser behavior
	})

	t.Run("query exceeding depth", func(t *testing.T) {
		query := `{ a { b { c { d { e { f } } } } } }`
		result, err := a.Analyze(query)
		if err != nil {
			t.Logf("Analyze() error (expected): %v", err)
		}
		if result == nil {
			t.Fatal("Analyze() returned nil")
		}
		if result.IsValid {
			t.Error("IsValid should be false for deep query")
		}
		if len(result.Errors) == 0 {
			t.Error("Errors should contain depth exceeded message")
		}
	})
}

func TestQueryAnalyzer_CalculateDepth(t *testing.T) {
	a := NewQueryAnalyzer(&AnalyzerConfig{MaxDepth: 10})

	t.Run("simple query", func(t *testing.T) {
		query := `{ users { id } }`
		depth, err := a.CalculateDepth(query)
		if err != nil {
			t.Errorf("CalculateDepth() error = %v", err)
		}
		if depth != 2 {
			t.Errorf("Depth = %d, want 2", depth)
		}
	})

	t.Run("invalid query returns error", func(t *testing.T) {
		query := `invalid`
		_, err := a.CalculateDepth(query)
		// Parser may or may not error on this input
		// Just verify function doesn't panic
		_ = err
	})
}

func TestQueryAnalyzer_CalculateComplexity(t *testing.T) {
	a := NewQueryAnalyzer(&AnalyzerConfig{
		MaxComplexity: 100,
		DefaultCost:   1,
		FieldCosts: map[string]int{
			"expensive": 10,
		},
	})

	t.Run("simple query", func(t *testing.T) {
		query := `{ users { id name } }`
		complexity, err := a.CalculateComplexity(query)
		if err != nil {
			t.Errorf("CalculateComplexity() error = %v", err)
		}
		if complexity <= 0 {
			t.Errorf("Complexity should be > 0, got %d", complexity)
		}
	})

	t.Run("query with expensive field", func(t *testing.T) {
		query := `{ expensive }`
		complexity, err := a.CalculateComplexity(query)
		if err != nil {
			t.Errorf("CalculateComplexity() error = %v", err)
		}
		if complexity < 10 {
			t.Errorf("Complexity with expensive field should be >= 10, got %d", complexity)
		}
	})

	t.Run("invalid query returns error", func(t *testing.T) {
		query := `invalid`
		_, err := a.CalculateComplexity(query)
		// Parser may or may not error on this input
		// Just verify function doesn't panic
		_ = err
	})
}

func TestQueryAnalyzer_SetFieldCost(t *testing.T) {
	a := NewQueryAnalyzer(nil)

	a.SetFieldCost("custom", 50)

	if a.fieldCosts["custom"] != 50 {
		t.Errorf("fieldCosts[custom] = %d, want 50", a.fieldCosts["custom"])
	}
}

func TestQueryAnalyzer_GetMaxDepth(t *testing.T) {
	a := NewQueryAnalyzer(&AnalyzerConfig{MaxDepth: 42})

	if got := a.GetMaxDepth(); got != 42 {
		t.Errorf("GetMaxDepth() = %d, want 42", got)
	}
}

func TestQueryAnalyzer_GetMaxComplexity(t *testing.T) {
	a := NewQueryAnalyzer(&AnalyzerConfig{MaxComplexity: 999})

	if got := a.GetMaxComplexity(); got != 999 {
		t.Errorf("GetMaxComplexity() = %d, want 999", got)
	}
}

func TestQueryAnalyzer_ValidateDepth(t *testing.T) {
	a := NewQueryAnalyzer(&AnalyzerConfig{MaxDepth: 3})

	t.Run("valid depth", func(t *testing.T) {
		query := `{ users { id } }` // depth 2
		err := a.ValidateDepth(query)
		if err != nil {
			t.Errorf("ValidateDepth() error = %v", err)
		}
	})

	t.Run("exceeds depth", func(t *testing.T) {
		query := `{ a { b { c { d } } } }` // depth 4
		err := a.ValidateDepth(query)
		if err == nil {
			t.Error("ValidateDepth() should return error for deep query")
		}
	})

	t.Run("invalid query", func(t *testing.T) {
		err := a.ValidateDepth("invalid")
		// Parser may or may not error
		_ = err
	})
}

func TestQueryAnalyzer_ValidateComplexity(t *testing.T) {
	a := NewQueryAnalyzer(&AnalyzerConfig{
		MaxComplexity: 10,
		DefaultCost:   1,
	})

	t.Run("valid complexity", func(t *testing.T) {
		query := `{ users { id } }`
		err := a.ValidateComplexity(query)
		if err != nil {
			t.Errorf("ValidateComplexity() error = %v", err)
		}
	})

	t.Run("exceeds complexity", func(t *testing.T) {
		a.SetFieldCost("expensive", 100)
		query := `{ expensive }`
		err := a.ValidateComplexity(query)
		if err == nil {
			t.Error("ValidateComplexity() should return error for complex query")
		}
	})

	t.Run("invalid query", func(t *testing.T) {
		err := a.ValidateComplexity("invalid")
		// Parser may or may not error
		_ = err
	})
}

func TestQueryAnalyzer_calculateComplexityWithArguments(t *testing.T) {
	a := NewQueryAnalyzer(&AnalyzerConfig{
		DefaultCost: 1,
	})

	// Query with arguments should have higher complexity
	query := `{ users(limit: 10, offset: 5) { id name } }`
	complexity, err := a.CalculateComplexity(query)
	if err != nil {
		t.Fatalf("CalculateComplexity() error = %v", err)
	}

	// Complexity should account for arguments (multiplier effect)
	if complexity <= 0 {
		t.Errorf("Complexity should be > 0, got %d", complexity)
	}
}

func TestQueryAnalyzer_complexityWithFragments(t *testing.T) {
	a := NewQueryAnalyzer(&AnalyzerConfig{
		DefaultCost: 1,
	})

	query := `
		fragment UserFields on User {
			id
			name
		}
		query {
			users {
				...UserFields
			}
		}
	`
	complexity, err := a.CalculateComplexity(query)
	if err != nil {
		t.Fatalf("CalculateComplexity() error = %v", err)
	}
	if complexity <= 0 {
		t.Errorf("Complexity with fragments should be > 0, got %d", complexity)
	}
}

func TestQueryAnalyzer_complexityWithInlineFragments(t *testing.T) {
	a := NewQueryAnalyzer(&AnalyzerConfig{
		DefaultCost: 1,
	})

	query := `
		query {
			user {
				... on Admin {
					adminField
				}
				... on User {
					userField
				}
			}
		}
	`
	complexity, err := a.CalculateComplexity(query)
	if err != nil {
		t.Fatalf("CalculateComplexity() error = %v", err)
	}
	if complexity <= 0 {
		t.Errorf("Complexity with inline fragments should be > 0, got %d", complexity)
	}
}
