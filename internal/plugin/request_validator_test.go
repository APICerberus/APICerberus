package plugin

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequestValidatorValidPayload(t *testing.T) {
	t.Parallel()

	validator, err := NewRequestValidator(RequestValidatorConfig{
		Schema: map[string]any{
			"type":     "object",
			"required": []any{"name", "email"},
			"properties": map[string]any{
				"name":  map[string]any{"type": "string"},
				"email": map[string]any{"type": "string", "format": "email"},
				"age":   map[string]any{"type": "integer"},
			},
		},
	})
	if err != nil {
		t.Fatalf("NewRequestValidator error: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "http://gateway.local/validate", bytes.NewBufferString(`{"name":"Alice","email":"alice@example.com","age":30}`))
	ctx := &PipelineContext{Request: req}
	if err := validator.Validate(ctx); err != nil {
		if verr, ok := err.(*RequestValidatorError); ok {
			t.Fatalf("expected valid payload, got error %v details=%v", err, verr.Details)
		}
		t.Fatalf("expected valid payload, got error %v", err)
	}
}

func TestRequestValidatorMissingRequiredField(t *testing.T) {
	t.Parallel()

	validator, err := NewRequestValidator(RequestValidatorConfig{
		Schema: map[string]any{
			"type":     "object",
			"required": []any{"name"},
			"properties": map[string]any{
				"name": map[string]any{"type": "string"},
			},
		},
	})
	if err != nil {
		t.Fatalf("NewRequestValidator error: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "http://gateway.local/validate", bytes.NewBufferString(`{"email":"x@example.com"}`))
	ctx := &PipelineContext{Request: req}
	err = validator.Validate(ctx)
	if err == nil {
		t.Fatalf("expected validation error")
	}
	verr, ok := err.(*RequestValidatorError)
	if !ok {
		t.Fatalf("expected RequestValidatorError got %T", err)
	}
	if len(verr.Details) == 0 || !strings.Contains(verr.Details[0], "required field") {
		t.Fatalf("expected missing required detail, got %#v", verr.Details)
	}
}

func TestRequestValidatorTypeAndFormatValidation(t *testing.T) {
	t.Parallel()

	validator, err := NewRequestValidator(RequestValidatorConfig{
		Schema: map[string]any{
			"type":     "object",
			"required": []any{"name", "email"},
			"properties": map[string]any{
				"name":  map[string]any{"type": "string"},
				"email": map[string]any{"type": "string", "format": "email"},
				"age":   map[string]any{"type": "integer"},
			},
		},
	})
	if err != nil {
		t.Fatalf("NewRequestValidator error: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "http://gateway.local/validate", bytes.NewBufferString(`{"name":123,"email":"not-email","age":"old"}`))
	ctx := &PipelineContext{Request: req}
	err = validator.Validate(ctx)
	if err == nil {
		t.Fatalf("expected validation error")
	}
	verr := err.(*RequestValidatorError)
	if len(verr.Details) < 2 {
		t.Fatalf("expected multiple validation details, got %#v", verr.Details)
	}
}
