package federation

import (
	"fmt"
	"strings"
)

// Planner plans federated GraphQL queries.
type Planner struct {
	subgraphs []*Subgraph
	entities  map[string]*Entity
}

// Plan represents an execution plan for a federated query.
type Plan struct {
	Steps     []*PlanStep
	DependsOn map[string][]string // step ID -> dependencies
}

// PlanStep represents a single step in the execution plan.
type PlanStep struct {
	ID          string
	Subgraph    *Subgraph
	Query       string
	Variables   map[string]interface{}
	DependsOn   []string
	ResultType  string
	Path        []string
}

// NewPlanner creates a new query planner.
func NewPlanner(subgraphs []*Subgraph, entities map[string]*Entity) *Planner {
	return &Planner{
		subgraphs: subgraphs,
		entities:  entities,
	}
}

// Plan plans the execution of a GraphQL query.
func (p *Planner) Plan(query string, variables map[string]interface{}) (*Plan, error) {
	// Parse the query
	doc, err := ParseGraphQLQuery(query)
	if err != nil {
		return nil, fmt.Errorf("failed to parse query: %w", err)
	}

	plan := &Plan{
		Steps:     make([]*PlanStep, 0),
		DependsOn: make(map[string][]string),
	}

	// Analyze query and create steps
	for _, op := range doc.Operations {
		steps, err := p.planOperation(op, variables)
		if err != nil {
			return nil, err
		}
		plan.Steps = append(plan.Steps, steps...)
	}

	// Build dependency graph
	for _, step := range plan.Steps {
		plan.DependsOn[step.ID] = step.DependsOn
	}

	return plan, nil
}

// planOperation plans a single operation.
func (p *Planner) planOperation(op GraphQLOperation, variables map[string]interface{}) ([]*PlanStep, error) {
	steps := make([]*PlanStep, 0)

	// For each field in the operation, determine which subgraph can resolve it
	for _, field := range op.Fields {
		fieldSteps, err := p.planField(field, variables, []string{})
		if err != nil {
			return nil, err
		}
		steps = append(steps, fieldSteps...)
	}

	return steps, nil
}

// planField plans a single field.
func (p *Planner) planField(field GraphQLField, variables map[string]interface{}, path []string) ([]*PlanStep, error) {
	steps := make([]*PlanStep, 0)
	currentPath := append(path, field.Name)

	// Find which subgraph can resolve this field
	subgraph := p.findSubgraphForField(field.Name)
	if subgraph == nil {
		return nil, fmt.Errorf("no subgraph can resolve field: %s", field.Name)
	}

	// Check if this is an entity field that requires resolution
	if entity, ok := p.entities[field.Name]; ok {
		// Create entity resolution step
		step := &PlanStep{
			ID:         fmt.Sprintf("step_%s", strings.Join(currentPath, "_")),
			Subgraph:   subgraph,
			Query:      p.buildEntityQuery(entity, field),
			Variables:  variables,
			ResultType: field.Name,
			Path:       currentPath,
		}
		steps = append(steps, step)

		// Plan nested fields
		for _, nestedField := range field.Fields {
			nestedSteps, err := p.planField(nestedField, variables, currentPath)
			if err != nil {
				return nil, err
			}
			// Mark dependency
			for _, nestedStep := range nestedSteps {
				nestedStep.DependsOn = append(nestedStep.DependsOn, step.ID)
			}
			steps = append(steps, nestedSteps...)
		}
	} else {
		// Regular field query
		step := &PlanStep{
			ID:         fmt.Sprintf("step_%s", strings.Join(currentPath, "_")),
			Subgraph:   subgraph,
			Query:      p.buildFieldQuery(field),
			Variables:  variables,
			ResultType: "scalar",
			Path:       currentPath,
		}
		steps = append(steps, step)

		// Plan nested fields on the same subgraph if possible
		for _, nestedField := range field.Fields {
			nestedSteps, err := p.planField(nestedField, variables, currentPath)
			if err != nil {
				return nil, err
			}
			steps = append(steps, nestedSteps...)
		}
	}

	return steps, nil
}

// findSubgraphForField finds a subgraph that can resolve the given field.
func (p *Planner) findSubgraphForField(fieldName string) *Subgraph {
	// Check if any entity has this field
	for _, entity := range p.entities {
		for _, sg := range entity.Subgraphs {
			if sg.Schema != nil {
				if _, ok := sg.Schema.Types[fieldName]; ok {
					return sg
				}
			}
		}
	}

	// Otherwise, find any subgraph that has this field in its Query type
	for _, sg := range p.subgraphs {
		if sg.Schema != nil && sg.Schema.QueryType != "" {
			if queryType, ok := sg.Schema.Types[sg.Schema.QueryType]; ok {
				if _, ok := queryType.Fields[fieldName]; ok {
					return sg
				}
			}
		}
	}

	return nil
}

// buildEntityQuery builds a query for entity resolution.
func (p *Planner) buildEntityQuery(entity *Entity, field GraphQLField) string {
	var sb strings.Builder

	// Build the entity representation query
	sb.WriteString("query ($representations: [_Any!]!) {\n")
	sb.WriteString(fmt.Sprintf("  _entities(representations: $representations) {\n"))
	sb.WriteString(fmt.Sprintf("    ... on %s {\n", entity.Name))

	// Add fields
	for _, f := range field.Fields {
		sb.WriteString(fmt.Sprintf("      %s\n", f.Name))
	}

	sb.WriteString("    }\n")
	sb.WriteString("  }\n")
	sb.WriteString("}")

	return sb.String()
}

// buildFieldQuery builds a query for a regular field.
func (p *Planner) buildFieldQuery(field GraphQLField) string {
	var sb strings.Builder

	sb.WriteString("{\n")
	sb.WriteString(p.buildFieldSelection(field, 1))
	sb.WriteString("}")

	return sb.String()
}

// buildFieldSelection builds a selection for a field.
func (p *Planner) buildFieldSelection(field GraphQLField, indent int) string {
	var sb strings.Builder
	prefix := strings.Repeat("  ", indent)

	sb.WriteString(fmt.Sprintf("%s%s", prefix, field.Name))

	// Add arguments if any
	if len(field.Args) > 0 {
		sb.WriteString("(")
		first := true
		for name, value := range field.Args {
			if !first {
				sb.WriteString(", ")
			}
			sb.WriteString(fmt.Sprintf("%s: %v", name, value))
			first = false
		}
		sb.WriteString(")")
	}

	// Add subfields if any
	if len(field.Fields) > 0 {
		sb.WriteString(" {\n")
		for _, f := range field.Fields {
			sb.WriteString(p.buildFieldSelection(f, indent+1))
		}
		sb.WriteString(fmt.Sprintf("%s}", prefix))
	}

	sb.WriteString("\n")

	return sb.String()
}

// GraphQLDocument represents a parsed GraphQL document.
type GraphQLDocument struct {
	Operations []GraphQLOperation
}

// GraphQLOperation represents a GraphQL operation.
type GraphQLOperation struct {
	Type       string // query, mutation, subscription
	Name       string
	Fields     []GraphQLField
	Variables  map[string]string
}

// GraphQLField represents a GraphQL field.
type GraphQLField struct {
	Name  string
	Alias string
	Args  map[string]interface{}
	Fields []GraphQLField
}

// ParseGraphQLQuery parses a GraphQL query string.
func ParseGraphQLQuery(query string) (*GraphQLDocument, error) {
	// This is a simplified parser
	// In production, use a proper GraphQL parser like graphql-go/graphql
	doc := &GraphQLDocument{
		Operations: make([]GraphQLOperation, 0),
	}

	// Parse operation type
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("empty query")
	}

	if strings.HasPrefix(query, "query") || strings.HasPrefix(query, "{") {
		op := GraphQLOperation{
			Type:   "query",
			Fields: make([]GraphQLField, 0),
		}

		// Extract fields (simplified)
		if idx := strings.Index(query, "{"); idx != -1 {
			fields := extractFields(query[idx:])
			op.Fields = fields
		}

		doc.Operations = append(doc.Operations, op)
	}

	return doc, nil
}

// extractFields extracts fields from a GraphQL selection set.
func extractFields(selection string) []GraphQLField {
	fields := make([]GraphQLField, 0)

	// Very simplified field extraction
	// In production, use a proper parser
	selection = strings.Trim(selection, "{}")
	selection = strings.TrimSpace(selection)

	// Split by newlines
	lines := strings.Split(selection, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Remove commas
		line = strings.TrimSuffix(line, ",")

		// Simple field name extraction
		fieldName := strings.Fields(line)[0]
		fieldName = strings.TrimSpace(fieldName)

		if fieldName != "" {
			fields = append(fields, GraphQLField{
				Name: fieldName,
			})
		}
	}

	return fields
}
