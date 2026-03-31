package federation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Executor executes federated GraphQL queries.
type Executor struct {
	client *http.Client
}

// ExecutionResult represents the result of executing a plan.
type ExecutionResult struct {
	Data   map[string]interface{}   `json:"data,omitempty"`
	Errors []ExecutionError         `json:"errors,omitempty"`
}

// ExecutionError represents an execution error.
type ExecutionError struct {
	Message    string                 `json:"message"`
	Path       []string               `json:"path,omitempty"`
	Extensions map[string]interface{} `json:"extensions,omitempty"`
}

// NewExecutor creates a new executor.
func NewExecutor() *Executor {
	return &Executor{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Execute executes a plan.
func (e *Executor) Execute(ctx context.Context, plan *Plan) (*ExecutionResult, error) {
	result := &ExecutionResult{
		Data:   make(map[string]interface{}),
		Errors: make([]ExecutionError, 0),
	}

	// Execute steps in dependency order
	executedSteps := make(map[string]map[string]interface{})
	stepResults := make(map[string]map[string]interface{})

	for _, step := range plan.Steps {
		// Check if dependencies are met
		deps := plan.DependsOn[step.ID]
		depData := make(map[string]interface{})
		for _, depID := range deps {
			if data, ok := executedSteps[depID]; ok {
				// Merge dependency data
				for k, v := range data {
					depData[k] = v
				}
			}
		}

		// Execute step
		stepData, err := e.executeStep(ctx, step, depData)
		if err != nil {
			result.Errors = append(result.Errors, ExecutionError{
				Message: fmt.Sprintf("step %s failed: %v", step.ID, err),
				Path:    step.Path,
			})
			continue
		}

		executedSteps[step.ID] = stepData
		stepResults[step.ID] = stepData

		// Merge into final result
		e.mergeResult(result.Data, stepData, step.Path)
	}

	return result, nil
}

// executeStep executes a single plan step.
func (e *Executor) executeStep(ctx context.Context, step *PlanStep, depData map[string]interface{}) (map[string]interface{}, error) {
	// Prepare variables
	variables := make(map[string]interface{})
	for k, v := range step.Variables {
		variables[k] = v
	}

	// Add dependency data as variables if this is an entity resolution
	if len(depData) > 0 && step.ResultType != "scalar" {
		variables["representations"] = e.buildRepresentations(depData)
	}

	// Build request
	reqBody := map[string]interface{}{
		"query":     step.Query,
		"variables": variables,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", step.Subgraph.URL, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range step.Subgraph.Headers {
		req.Header.Set(k, v)
	}

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("subgraph returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var subgraphResp struct {
		Data   map[string]interface{} `json:"data"`
		Errors []struct {
			Message string   `json:"message"`
			Path    []string `json:"path"`
		} `json:"errors"`
	}

	if err := json.Unmarshal(body, &subgraphResp); err != nil {
		return nil, err
	}

	// Extract data
	if subgraphResp.Data != nil {
		// Handle _entities response
		if entities, ok := subgraphResp.Data["_entities"].([]interface{}); ok {
			if len(entities) > 0 {
				if entity, ok := entities[0].(map[string]interface{}); ok {
					return entity, nil
				}
			}
		}

		// Return first field result
		for _, v := range subgraphResp.Data {
			if data, ok := v.(map[string]interface{}); ok {
				return data, nil
			}
		}
	}

	return subgraphResp.Data, nil
}

// buildRepresentations builds entity representations for Apollo Federation.
func (e *Executor) buildRepresentations(depData map[string]interface{}) []interface{} {
	representations := make([]interface{}, 0)

	// Build representation with __typename and key fields
	rep := make(map[string]interface{})
	for k, v := range depData {
		rep[k] = v
	}

	representations = append(representations, rep)
	return representations
}

// mergeResult merges step result into the final result.
func (e *Executor) mergeResult(data map[string]interface{}, stepData map[string]interface{}, path []string) {
	if len(path) == 0 {
		for k, v := range stepData {
			data[k] = v
		}
		return
	}

	// Navigate to the correct position in the data tree
	current := data
	for i, key := range path[:len(path)-1] {
		if _, ok := current[key]; !ok {
			current[key] = make(map[string]interface{})
		}
		if next, ok := current[key].(map[string]interface{}); ok {
			current = next
		} else {
			// Cannot navigate further
			return
		}

		_ = i
	}

	// Set the final value
	lastKey := path[len(path)-1]
	if existing, ok := current[lastKey].(map[string]interface{}); ok {
		// Merge with existing data
		for k, v := range stepData {
			existing[k] = v
		}
	} else {
		current[lastKey] = stepData
	}
}

// ExecuteParallel executes steps in parallel where possible.
func (e *Executor) ExecuteParallel(ctx context.Context, plan *Plan) (*ExecutionResult, error) {
	result := &ExecutionResult{
		Data:   make(map[string]interface{}),
		Errors: make([]ExecutionError, 0),
	}

	// Build execution graph
	pendingSteps := make(map[string]*PlanStep)
	completedSteps := make(map[string]map[string]interface{})
	inProgress := make(map[string]bool)

	for _, step := range plan.Steps {
		pendingSteps[step.ID] = step
	}

	// Execute steps
	var wg sync.WaitGroup
	var mu sync.Mutex

	for len(pendingSteps) > 0 {
		// Find steps that can be executed (all dependencies met)
		executable := make([]*PlanStep, 0)
		for _, step := range pendingSteps {
			if inProgress[step.ID] {
				continue
			}

			canExecute := true
			for _, depID := range plan.DependsOn[step.ID] {
				if _, ok := completedSteps[depID]; !ok {
					canExecute = false
					break
				}
			}

			if canExecute {
				executable = append(executable, step)
			}
		}

		if len(executable) == 0 {
			// Check for deadlock
			if len(pendingSteps) > 0 && len(inProgress) == 0 {
				return nil, fmt.Errorf("deadlock detected: unable to execute remaining steps")
			}
			// Wait for some steps to complete
			break
		}

		// Execute steps in parallel
		for _, step := range executable {
			wg.Add(1)
			inProgress[step.ID] = true

			go func(s *PlanStep) {
				defer wg.Done()

				// Gather dependency data
				depData := make(map[string]interface{})
				for _, depID := range plan.DependsOn[s.ID] {
					if data, ok := completedSteps[depID]; ok {
						for k, v := range data {
							depData[k] = v
						}
					}
				}

				// Execute
				stepData, err := e.executeStep(ctx, s, depData)

				mu.Lock()
				defer mu.Unlock()

				if err != nil {
					result.Errors = append(result.Errors, ExecutionError{
						Message: fmt.Sprintf("step %s failed: %v", s.ID, err),
						Path:    s.Path,
					})
				} else {
					completedSteps[s.ID] = stepData
					e.mergeResult(result.Data, stepData, s.Path)
				}

				delete(inProgress, s.ID)
				delete(pendingSteps, s.ID)
			}(step)
		}

		// Wait for batch to complete
		wg.Wait()
	}

	return result, nil
}

// BatchRequest represents a batched GraphQL request.
type BatchRequest struct {
	Queries []string
}

// BatchResponse represents a batched GraphQL response.
type BatchResponse struct {
	Results []map[string]interface{}
	Errors  []ExecutionError
}

// ExecuteBatch executes multiple queries in a batch.
func (e *Executor) ExecuteBatch(ctx context.Context, subgraph *Subgraph, batch *BatchRequest) (*BatchResponse, error) {
	response := &BatchResponse{
		Results: make([]map[string]interface{}, 0, len(batch.Queries)),
		Errors:  make([]ExecutionError, 0),
	}

	// Build batched query
	var sb strings.Builder
	sb.WriteString("{\n")

	for i, query := range batch.Queries {
		sb.WriteString(fmt.Sprintf("  batch_%d: %s\n", i, query))
	}

	sb.WriteString("}")

	reqBody := map[string]interface{}{
		"query": sb.String(),
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", subgraph.URL, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var batchResp map[string]interface{}
	if err := json.Unmarshal(body, &batchResp); err != nil {
		return nil, err
	}

	// Extract results
	for i := range batch.Queries {
		key := fmt.Sprintf("batch_%d", i)
		if data, ok := batchResp[key]; ok {
			if d, ok := data.(map[string]interface{}); ok {
				response.Results = append(response.Results, d)
			}
		}
	}

	return response, nil
}
