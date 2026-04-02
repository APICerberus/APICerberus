package federation

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewExecutor(t *testing.T) {
	executor := NewExecutor()
	if executor == nil {
		t.Fatal("NewExecutor() returned nil")
	}
	if executor.client == nil {
		t.Error("Executor.client not initialized")
	}
	if executor.client.Timeout != 30*time.Second {
		t.Errorf("Timeout = %v, want 30s", executor.client.Timeout)
	}
}

func TestExecutor_Execute(t *testing.T) {
	// Create a mock subgraph server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody struct {
			Query     string                 `json:"query"`
			Variables map[string]interface{} `json:"variables"`
		}
		json.NewDecoder(r.Body).Decode(&reqBody)

		// Return mock response
		response := map[string]interface{}{
			"data": map[string]interface{}{
				"users": []map[string]interface{}{
					{"id": "1", "name": "Alice"},
					{"id": "2", "name": "Bob"},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	executor := NewExecutor()
	plan := &Plan{
		Steps: []*PlanStep{
			{
				ID:        "step1",
				Subgraph:  &Subgraph{URL: server.URL},
				Query:     "{ users { id name } }",
				Path:      []string{"users"},
				Variables: map[string]interface{}{},
			},
		},
		DependsOn: map[string][]string{},
	}

	result, err := executor.Execute(context.Background(), plan)
	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}
	if result == nil {
		t.Fatal("Execute() returned nil result")
	}
}

func TestExecutor_Execute_WithDependencies(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody struct {
			Query     string                 `json:"query"`
			Variables map[string]interface{} `json:"variables"`
		}
		json.NewDecoder(r.Body).Decode(&reqBody)

		callCount++

		var response map[string]interface{}
		if callCount == 1 {
			response = map[string]interface{}{
				"data": map[string]interface{}{
					"user": map[string]interface{}{
						"id":   "1",
						"name": "Alice",
					},
				},
			}
		} else {
			response = map[string]interface{}{
				"data": map[string]interface{}{
					"orders": []map[string]interface{}{
						{"id": "o1", "total": 100},
					},
				},
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	executor := NewExecutor()
	plan := &Plan{
		Steps: []*PlanStep{
			{
				ID:        "step1",
				Subgraph:  &Subgraph{URL: server.URL},
				Query:     "{ user(id: 1) { id name } }",
				Path:      []string{"user"},
				Variables: map[string]interface{}{},
			},
			{
				ID:         "step2",
				Subgraph:   &Subgraph{URL: server.URL},
				Query:      "{ orders(userId: $userId) { id total } }",
				Path:       []string{"orders"},
				Variables:  map[string]interface{}{"userId": "1"},
				ResultType: "Order",
			},
		},
		DependsOn: map[string][]string{
			"step2": {"step1"},
		},
	}

	result, err := executor.Execute(context.Background(), plan)
	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}
	if result == nil {
		t.Fatal("Execute() returned nil result")
	}
}

func TestExecutor_Execute_SubgraphError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"errors": []map[string]interface{}{
				{"message": "Internal server error"},
			},
		})
	}))
	defer server.Close()

	executor := NewExecutor()
	plan := &Plan{
		Steps: []*PlanStep{
			{
				ID:       "step1",
				Subgraph: &Subgraph{URL: server.URL},
				Query:    "{ users { id } }",
				Path:     []string{"users"},
			},
		},
		DependsOn: map[string][]string{},
	}

	result, err := executor.Execute(context.Background(), plan)
	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}
	if len(result.Errors) == 0 {
		t.Error("Execute() should return errors when subgraph fails")
	}
}

func TestExecutor_Execute_Entities(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return _entities response for federation
		response := map[string]interface{}{
			"data": map[string]interface{}{
				"_entities": []map[string]interface{}{
					{
						"__typename": "User",
						"id":         "1",
						"name":       "Alice",
						"email":      "alice@example.com",
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	executor := NewExecutor()
	plan := &Plan{
		Steps: []*PlanStep{
			{
				ID:         "step1",
				Subgraph:   &Subgraph{URL: server.URL},
				Query:      "query($representations: [_Any!]!) { _entities(representations: $representations) { ... on User { email } } }",
				Path:       []string{"user"},
				Variables:  map[string]interface{}{},
				ResultType: "User",
			},
		},
		DependsOn: map[string][]string{},
	}

	result, err := executor.Execute(context.Background(), plan)
	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}
	if result == nil {
		t.Fatal("Execute() returned nil result")
	}
}

func TestExecutor_mergeResult(t *testing.T) {
	executor := NewExecutor()

	t.Run("Empty path", func(t *testing.T) {
		data := make(map[string]interface{})
		stepData := map[string]interface{}{"id": "1", "name": "Test"}

		executor.mergeResult(data, stepData, []string{})

		if data["id"] != "1" {
			t.Errorf("id = %v, want 1", data["id"])
		}
		if data["name"] != "Test" {
			t.Errorf("name = %v, want Test", data["name"])
		}
	})

	t.Run("Nested path", func(t *testing.T) {
		data := make(map[string]interface{})
		stepData := map[string]interface{}{"id": "1", "title": "Post"}

		executor.mergeResult(data, stepData, []string{"user", "posts"})

		user, ok := data["user"].(map[string]interface{})
		if !ok {
			t.Fatal("user not found in data")
		}
		posts, ok := user["posts"].(map[string]interface{})
		if !ok {
			t.Fatal("posts not found in user")
		}
		if posts["title"] != "Post" {
			t.Errorf("title = %v, want Post", posts["title"])
		}
	})

	t.Run("Merge with existing", func(t *testing.T) {
		data := map[string]interface{}{
			"user": map[string]interface{}{
				"id":   "1",
				"name": "Alice",
			},
		}
		stepData := map[string]interface{}{"email": "alice@example.com"}

		executor.mergeResult(data, stepData, []string{"user"})

		user := data["user"].(map[string]interface{})
		if user["id"] != "1" {
			t.Errorf("id = %v, want 1", user["id"])
		}
		if user["email"] != "alice@example.com" {
			t.Errorf("email = %v, want alice@example.com", user["email"])
		}
	})
}

func TestExecutor_buildRepresentations(t *testing.T) {
	executor := NewExecutor()
	depData := map[string]interface{}{
		"id":   "1",
		"name": "Alice",
	}

	representations := executor.buildRepresentations(depData, "User")

	if len(representations) != 1 {
		t.Errorf("len(representations) = %v, want 1", len(representations))
	}

	rep := representations[0].(map[string]interface{})
	if rep["__typename"] != "User" {
		t.Errorf("__typename = %v, want User", rep["__typename"])
	}
	if rep["id"] != "1" {
		t.Errorf("id = %v, want 1", rep["id"])
	}
}

func TestExecutionResult(t *testing.T) {
	result := &ExecutionResult{
		Data: map[string]interface{}{
			"users": []map[string]interface{}{
				{"id": "1", "name": "Alice"},
			},
		},
		Errors: []ExecutionError{
			{Message: "Partial error", Path: []string{"users", "0", "email"}},
		},
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Errorf("Failed to marshal ExecutionResult: %v", err)
	}

	var decoded ExecutionResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Errorf("Failed to unmarshal ExecutionResult: %v", err)
	}

	if len(decoded.Errors) != 1 {
		t.Errorf("Errors count = %v, want 1", len(decoded.Errors))
	}
}

func TestExecutionError(t *testing.T) {
	execErr := ExecutionError{
		Message: "Test error",
		Path:    []string{"user", "name"},
		Extensions: map[string]interface{}{
			"code": "VALIDATION_ERROR",
		},
	}

	data, err := json.Marshal(execErr)
	if err != nil {
		t.Errorf("Failed to marshal ExecutionError: %v", err)
	}

	var decoded ExecutionError
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Errorf("Failed to unmarshal ExecutionError: %v", err)
	}

	if decoded.Message != execErr.Message {
		t.Errorf("Message = %v, want %v", decoded.Message, execErr.Message)
	}
}

func TestBatchRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"data": map[string]interface{}{
				"batch_0": map[string]interface{}{"id": "1"},
				"batch_1": map[string]interface{}{"id": "2"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	executor := NewExecutor()
	batch := &BatchRequest{
		Queries: []string{"user(id: 1) { id }", "user(id: 2) { id }"},
	}

	subgraph := &Subgraph{URL: server.URL}
	result, err := executor.ExecuteBatch(context.Background(), subgraph, batch)
	if err != nil {
		t.Errorf("ExecuteBatch() error = %v", err)
	}
	if result == nil {
		t.Fatal("ExecuteBatch() returned nil result")
	}
}
