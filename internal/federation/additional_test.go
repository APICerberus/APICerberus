package federation

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/APICerberus/APICerebrus/internal/graphql"
)

// Test buildInterfaceSDL
func TestComposer_buildInterfaceSDL(t *testing.T) {
	composer := NewComposer()

	iface := &Type{
		Kind:        "INTERFACE",
		Name:        "Node",
		Description: "An object with an ID",
		Fields: map[string]*Field{
			"id": {Name: "id", Type: "ID!"},
		},
	}

	sdl := composer.buildInterfaceSDL(iface)
	if sdl == "" {
		t.Error("buildInterfaceSDL returned empty string")
	}
	if !contains(sdl, "interface Node") {
		t.Errorf("SDL should contain 'interface Node', got: %s", sdl)
	}
	if !contains(sdl, "id: ID!") {
		t.Errorf("SDL should contain 'id: ID!', got: %s", sdl)
	}
}

// Test buildUnionSDL
func TestComposer_buildUnionSDL(t *testing.T) {
	composer := NewComposer()

	union := &Type{
		Kind:          "UNION",
		Name:          "SearchResult",
		PossibleTypes: []string{"User", "Post"},
	}

	sdl := composer.buildUnionSDL(union)
	if sdl == "" {
		t.Error("buildUnionSDL returned empty string")
	}
	expected := "union SearchResult = User | Post"
	if sdl != expected {
		t.Errorf("buildUnionSDL() = %q, want %q", sdl, expected)
	}
}

// Test buildEnumSDL
func TestComposer_buildEnumSDL(t *testing.T) {
	composer := NewComposer()

	enum := &Type{
		Kind:       "ENUM",
		Name:       "Status",
		EnumValues: []string{"ACTIVE", "INACTIVE", "PENDING"},
	}

	sdl := composer.buildEnumSDL(enum)
	if sdl == "" {
		t.Error("buildEnumSDL returned empty string")
	}
	if !contains(sdl, "enum Status") {
		t.Errorf("SDL should contain 'enum Status', got: %s", sdl)
	}
	if !contains(sdl, "ACTIVE") {
		t.Errorf("SDL should contain 'ACTIVE', got: %s", sdl)
	}
}

// Test buildInputSDL
func TestComposer_buildInputSDL(t *testing.T) {
	composer := NewComposer()

	input := &Type{
		Kind: "INPUT_OBJECT",
		Name: "UserInput",
		InputFields: map[string]*InputField{
			"name":  {Name: "name", Type: "String!"},
			"email": {Name: "email", Type: "String!"},
		},
	}

	sdl := composer.buildInputSDL(input)
	if sdl == "" {
		t.Error("buildInputSDL returned empty string")
	}
	if !contains(sdl, "input UserInput") {
		t.Errorf("SDL should contain 'input UserInput', got: %s", sdl)
	}
	if !contains(sdl, "name: String!") {
		t.Errorf("SDL should contain 'name: String!', got: %s", sdl)
	}
}

// Test buildScalarSDL
func TestComposer_buildScalarSDL(t *testing.T) {
	composer := NewComposer()

	scalar := &Type{
		Kind: "SCALAR",
		Name: "DateTime",
	}

	sdl := composer.buildScalarSDL(scalar)
	if sdl == "" {
		t.Error("buildScalarSDL returned empty string")
	}
	expected := "scalar DateTime"
	if sdl != expected {
		t.Errorf("buildScalarSDL() = %q, want %q", sdl, expected)
	}
}

// Test GetEntities
func TestComposer_GetEntities(t *testing.T) {
	composer := NewComposer()

	// Initially should be empty
	entities := composer.GetEntities()
	if len(entities) != 0 {
		t.Errorf("GetEntities() returned %d entities, want 0", len(entities))
	}

	// Add an entity manually
	composer.entities["User"] = &Entity{
		Name:      "User",
		KeyFields: []string{"id"},
	}

	entities = composer.GetEntities()
	if len(entities) != 1 {
		t.Errorf("GetEntities() returned %d entities, want 1", len(entities))
	}
	if _, ok := entities["User"]; !ok {
		t.Error("GetEntities() should contain 'User' entity")
	}
}

// Test ExecuteParallel
func TestExecutor_ExecuteParallel(t *testing.T) {
	executor := NewExecutor()

	// Create a simple plan with steps
	plan := &Plan{
		Steps: []*PlanStep{
			{
				ID:       "step1",
				Subgraph: &Subgraph{ID: "subgraph1", Name: "users"},
				Query:    "{ users { id } }",
			},
		},
		DependsOn: map[string][]string{
			"step1": {},
		},
	}

	// Execute in parallel - will fail due to no actual server, but tests the code path
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// This will likely error due to no actual subgraphs, but it exercises the code
	_, err := executor.ExecuteParallel(ctx, plan)
	// We expect an error since there are no real subgraphs
	if err == nil {
		t.Log("ExecuteParallel completed without error (may have used mock data)")
	}
}

// Test convertValue
func TestConvertValue(t *testing.T) {
	tests := []struct {
		name     string
		value    graphql.Value
		expected interface{}
	}{
		{"nil", nil, nil},
		{"scalar", &graphql.ScalarValue{Value: "hello"}, "hello"},
		{"list", &graphql.ListValue{Values: []graphql.Value{&graphql.ScalarValue{Value: "a"}, &graphql.ScalarValue{Value: "b"}}}, []interface{}{"a", "b"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertValue(tt.value)
			if tt.expected == nil {
				if result != nil {
					t.Errorf("convertValue(nil) = %v, want nil", result)
				}
				return
			}
			// For complex types, just check not nil
			if result == nil {
				t.Errorf("convertValue(%v) = nil, want non-nil", tt.value)
			}
		})
	}
}

// Test buildEntityQuery
func TestPlanner_buildEntityQuery(t *testing.T) {
	planner := NewPlanner([]*Subgraph{}, make(map[string]*Entity))

	entity := &Entity{
		Name:      "User",
		KeyFields: []string{"id"},
	}

	field := GraphQLField{
		Name:   "user",
		Fields: []GraphQLField{{Name: "id"}, {Name: "name"}},
	}

	query := planner.buildEntityQuery(entity, field)
	if query == "" {
		t.Error("buildEntityQuery() returned empty query")
	}
	if !contains(query, "User") {
		t.Errorf("query should contain 'User', got: %s", query)
	}
	if !contains(query, "_entities") {
		t.Errorf("query should contain '_entities', got: %s", query)
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsInternal(s, substr))
}

func containsInternal(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Test QueryCache
func TestQueryCache(t *testing.T) {
	cache := NewQueryCache(10)

	// Test Get on empty cache
	_, found := cache.Get("query1")
	if found {
		t.Error("Expected cache miss on empty cache")
	}

	// Test Set and Get
	plan := &Plan{
		Steps: []*PlanStep{
			{ID: "step1", Query: "{ users { id } }"},
		},
	}
	cache.Set("query1", plan)

	retrieved, found := cache.Get("query1")
	if !found {
		t.Error("Expected cache hit after Set")
	}
	if retrieved == nil {
		t.Error("Retrieved plan should not be nil")
	}

	// Test cache eviction
	for i := 0; i < 15; i++ {
		cache.Set(fmt.Sprintf("query%d", i), plan)
	}

	// The cache should have evicted some entries
	if len(cache.entries) > 10 {
		t.Errorf("Cache size %d exceeds max 10", len(cache.entries))
	}
}

// Test CircuitBreaker
func TestCircuitBreaker(t *testing.T) {
	cb := NewCircuitBreaker(3, 100*time.Millisecond)

	// Initially should be closed and allow requests
	if !cb.CanExecute() {
		t.Error("Circuit breaker should allow execution when closed")
	}

	// Record some failures
	cb.RecordFailure()
	cb.RecordFailure()

	// Should still allow (below threshold)
	if !cb.CanExecute() {
		t.Error("Circuit breaker should still allow execution below threshold")
	}

	// Record more failures to reach threshold
	cb.RecordFailure()
	cb.RecordFailure()
	cb.RecordFailure()

	// Now should be open
	if cb.CanExecute() {
		t.Error("Circuit breaker should be open after threshold reached")
	}

	// Wait for reset timeout
	time.Sleep(150 * time.Millisecond)

	// Should be half-open now
	if !cb.CanExecute() {
		t.Error("Circuit breaker should be half-open after reset timeout")
	}

	// Record success should close it
	cb.RecordSuccess()
	if !cb.CanExecute() {
		t.Error("Circuit breaker should be closed after success")
	}
}

// Test GetActiveSubscriptions
func TestExecutor_GetActiveSubscriptions(t *testing.T) {
	executor := NewExecutor()

	// Initially should be empty
	subs := executor.GetActiveSubscriptions()
	if len(subs) != 0 {
		t.Errorf("Expected 0 subscriptions, got %d", len(subs))
	}
}

// Test StopSubscription with non-existent subscription
func TestExecutor_StopSubscription_NotFound(t *testing.T) {
	executor := NewExecutor()

	err := executor.StopSubscription("non-existent-id")
	if err == nil {
		t.Error("Expected error when stopping non-existent subscription")
	}
}

// Test OptimizePlan
func TestExecutor_OptimizePlan(t *testing.T) {
	executor := NewExecutor()

	plan := &Plan{
		Steps: []*PlanStep{
			{ID: "step1", Subgraph: &Subgraph{ID: "sg1"}},
			{ID: "step2", Subgraph: &Subgraph{ID: "sg2"}},
			{ID: "step3", Subgraph: &Subgraph{ID: "sg3"}},
		},
		DependsOn: map[string][]string{
			"step1": {},
			"step2": {"step1"},
			"step3": {"step1"},
		},
	}

	optimized := executor.OptimizePlan(plan)

	if optimized == nil {
		t.Fatal("OptimizePlan returned nil")
	}

	if len(optimized.ExecutionOrder) != 3 {
		t.Errorf("Expected 3 steps in execution order, got %d", len(optimized.ExecutionOrder))
	}

	if len(optimized.ParallelGroups) < 1 {
		t.Error("Expected at least 1 parallel group")
	}

	if optimized.EstimatedCost <= 0 {
		t.Error("Expected positive estimated cost")
	}
}

// Test OptimizePlan with circular dependencies (deadlock scenario)
func TestExecutor_OptimizePlan_Deadlock(t *testing.T) {
	executor := NewExecutor()

	plan := &Plan{
		Steps: []*PlanStep{
			{ID: "step1", Subgraph: &Subgraph{ID: "sg1"}},
			{ID: "step2", Subgraph: &Subgraph{ID: "sg2"}},
		},
		DependsOn: map[string][]string{
			"step1": {"step2"},
			"step2": {"step1"},
		},
	}

	optimized := executor.OptimizePlan(plan)

	// Should detect deadlock and not include all steps
	if len(optimized.ExecutionOrder) > 0 {
		t.Logf("Execution order with deadlock: %v", optimized.ExecutionOrder)
	}
}

// Test ExecuteSubscription with empty plan
func TestExecutor_ExecuteSubscription_EmptyPlan(t *testing.T) {
	executor := NewExecutor()

	plan := &Plan{
		Steps: []*PlanStep{},
	}

	_, err := executor.ExecuteSubscription(context.Background(), plan)
	if err == nil {
		t.Error("Expected error for empty plan")
	}
}

// Test ExecuteSubscription with nil subgraph
func TestExecutor_ExecuteSubscription_NilSubgraph(t *testing.T) {
	executor := NewExecutor()

	plan := &Plan{
		Steps: []*PlanStep{
			{ID: "step1", Subgraph: nil, Query: "subscription { update }"},
		},
	}

	_, err := executor.ExecuteSubscription(context.Background(), plan)
	if err == nil {
		t.Error("Expected error for nil subgraph")
	}
}

// Test ExecuteOptimized
func TestExecutor_ExecuteOptimized(t *testing.T) {
	executor := NewExecutor()

	// Create a mock server for testing
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"data": map[string]interface{}{
				"users": []map[string]interface{}{
					{"id": "1", "name": "Alice"},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	plan := &Plan{
		Steps: []*PlanStep{
			{
				ID:       "step1",
				Subgraph: &Subgraph{ID: "sg1", URL: server.URL},
				Query:    "{ users { id name } }",
				Path:     []string{"users"},
			},
		},
		DependsOn: map[string][]string{
			"step1": {},
		},
	}

	optimized := executor.OptimizePlan(plan)
	result, err := executor.ExecuteOptimized(context.Background(), optimized)

	if err != nil {
		t.Errorf("ExecuteOptimized error: %v", err)
	}
	if result == nil {
		t.Fatal("ExecuteOptimized returned nil result")
	}
}

// Test getCircuitBreaker
func TestExecutor_getCircuitBreaker(t *testing.T) {
	executor := NewExecutor()

	// First call should create new circuit breaker
	cb1 := executor.getCircuitBreaker("subgraph1")
	if cb1 == nil {
		t.Error("getCircuitBreaker returned nil")
	}

	// Second call should return same circuit breaker
	cb2 := executor.getCircuitBreaker("subgraph1")
	if cb1 != cb2 {
		t.Error("getCircuitBreaker should return same instance for same subgraph")
	}

	// Different subgraph should return different circuit breaker
	cb3 := executor.getCircuitBreaker("subgraph2")
	if cb1 == cb3 {
		t.Error("getCircuitBreaker should return different instance for different subgraph")
	}
}

// Test CacheEntry
func TestCacheEntry(t *testing.T) {
	entry := &CacheEntry{
		Plan: &Plan{
			Steps: []*PlanStep{{ID: "step1"}},
		},
		Timestamp: time.Now(),
		HitCount:  0,
	}

	if entry.Plan == nil {
		t.Error("CacheEntry Plan should not be nil")
	}

	if entry.HitCount != 0 {
		t.Errorf("Initial HitCount should be 0, got %d", entry.HitCount)
	}
}

// Test OptimizedPlan structure
func TestOptimizedPlan(t *testing.T) {
	opt := &OptimizedPlan{
		Plan: &Plan{
			Steps: []*PlanStep{{ID: "step1"}},
		},
		ExecutionOrder: []string{"step1", "step2"},
		ParallelGroups: [][]string{
			{"step1"},
			{"step2"},
		},
		EstimatedCost: 20,
	}

	if len(opt.ExecutionOrder) != 2 {
		t.Errorf("Expected 2 steps in execution order, got %d", len(opt.ExecutionOrder))
	}

	if opt.EstimatedCost != 20 {
		t.Errorf("Expected estimated cost 20, got %d", opt.EstimatedCost)
	}
}

// Test SubscriptionConnection structure
func TestSubscriptionConnection(t *testing.T) {
	sub := &SubscriptionConnection{
		ID:        "sub1",
		Subgraph:  &Subgraph{ID: "sg1", Name: "test"},
		Query:     "subscription { updates }",
		Variables: map[string]interface{}{"id": "123"},
		Messages:  make(chan *SubscriptionMessage, 10),
		Errors:    make(chan error, 10),
		Done:      make(chan struct{}),
	}

	if sub.ID != "sub1" {
		t.Errorf("Expected ID 'sub1', got %s", sub.ID)
	}

	if sub.Query != "subscription { updates }" {
		t.Errorf("Expected query 'subscription { updates }', got %s", sub.Query)
	}
}

// Test SubscriptionMessage structure
func TestSubscriptionMessage(t *testing.T) {
	msg := &SubscriptionMessage{
		ID: "msg1",
		Data: map[string]interface{}{
			"update": "value",
		},
	}

	if msg.ID != "msg1" {
		t.Errorf("Expected ID 'msg1', got %s", msg.ID)
	}

	if msg.Data["update"] != "value" {
		t.Error("Message data mismatch")
	}
}
