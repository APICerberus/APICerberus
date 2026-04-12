package graphql

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestIsGraphQLRequest_New(t *testing.T) {
	tests := []struct {
		name        string
		method      string
		contentType string
		query       string
		want        bool
	}{
		{"POST with application/json", "POST", "application/json", "", true},
		{"POST with application/graphql", "POST", "application/graphql", "", true},
		{"GET with query param", "GET", "", "{ users { id } }", true},
		{"GET without query param", "GET", "", "", false},
		{"POST with text/plain", "POST", "text/plain", "", false},
		{"PUT with application/json", "PUT", "application/json", "", false},
		{"DELETE with query param", "DELETE", "", "query", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.query != "" {
				req = httptest.NewRequest(tt.method, "/graphql?query="+url.QueryEscape(tt.query), nil)
			} else {
				req = httptest.NewRequest(tt.method, "/graphql", nil)
			}
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}

			got := IsGraphQLRequest(req)
			if got != tt.want {
				t.Errorf("IsGraphQLRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseRequest_GET(t *testing.T) {
	t.Run("Valid GET request", func(t *testing.T) {
		queryStr := url.QueryEscape("{users{id}}")
		varsStr := url.QueryEscape(`{"id":"123"}`)
		req := httptest.NewRequest("GET", "/graphql?query="+queryStr+"&variables="+varsStr+"&operationName=GetUser", nil)

		got, err := ParseRequest(req)
		if err != nil {
			t.Errorf("ParseRequest() error = %v", err)
		}
		if got.Query != "{users{id}}" {
			t.Errorf("Query = %v, want {users{id}}", got.Query)
		}
		if got.OperationName != "GetUser" {
			t.Errorf("OperationName = %v, want GetUser", got.OperationName)
		}
	})

	t.Run("GET without query", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/graphql", nil)

		_, err := ParseRequest(req)
		if err == nil {
			t.Error("ParseRequest() should return error for GET without query")
		}
	})

	t.Run("GET with invalid variables", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/graphql?query={users}&variables=invalid", nil)

		_, err := ParseRequest(req)
		if err == nil {
			t.Error("ParseRequest() should return error for invalid variables")
		}
	})
}

func TestParseRequest_POST(t *testing.T) {
	t.Run("POST with application/graphql", func(t *testing.T) {
		body := "{ users { id name } }"
		req := httptest.NewRequest("POST", "/graphql", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/graphql")

		got, err := ParseRequest(req)
		if err != nil {
			t.Errorf("ParseRequest() error = %v", err)
		}
		if got.Query != body {
			t.Errorf("Query = %v, want %v", got.Query, body)
		}
	})

	t.Run("POST with application/json", func(t *testing.T) {
		requestBody := Request{
			Query:         "query GetUser($id: ID!) { user(id: $id) { name } }",
			Variables:     map[string]interface{}{"id": "123"},
			OperationName: "GetUser",
		}
		jsonBody, _ := json.Marshal(requestBody)
		req := httptest.NewRequest("POST", "/graphql", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		got, err := ParseRequest(req)
		if err != nil {
			t.Errorf("ParseRequest() error = %v", err)
		}
		if got.Query != requestBody.Query {
			t.Errorf("Query = %v, want %v", got.Query, requestBody.Query)
		}
		if got.OperationName != requestBody.OperationName {
			t.Errorf("OperationName = %v, want %v", got.OperationName, requestBody.OperationName)
		}
	})

	t.Run("POST with invalid JSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/graphql", strings.NewReader("not json"))
		req.Header.Set("Content-Type", "application/json")

		_, err := ParseRequest(req)
		if err == nil {
			t.Error("ParseRequest() should return error for invalid JSON")
		}
	})

	t.Run("POST with missing query field", func(t *testing.T) {
		body := `{"variables": {}}`
		req := httptest.NewRequest("POST", "/graphql", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		_, err := ParseRequest(req)
		if err == nil {
			t.Error("ParseRequest() should return error for missing query field")
		}
	})

	t.Run("POST with unsupported content type", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/graphql", strings.NewReader("data"))
		req.Header.Set("Content-Type", "text/plain")

		_, err := ParseRequest(req)
		if err == nil {
			t.Error("ParseRequest() should return error for unsupported content type")
		}
	})
}

func TestParseRequest_UnsupportedMethod(t *testing.T) {
	req := httptest.NewRequest("PUT", "/graphql", strings.NewReader("query"))

	_, err := ParseRequest(req)
	if err == nil {
		t.Error("ParseRequest() should return error for unsupported method")
	}
}

func TestWriteResponse(t *testing.T) {
	rec := httptest.NewRecorder()
	resp := &Response{
		Data: json.RawMessage(`{"user":{"name":"Alice"}}`),
	}

	WriteResponse(rec, resp, http.StatusOK)

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", rec.Code, http.StatusOK)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %v, want application/json", ct)
	}

	var result Response
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Errorf("Response is not valid JSON: %v", err)
	}
}

func TestWriteError(t *testing.T) {
	rec := httptest.NewRecorder()

	WriteError(rec, "something went wrong", http.StatusBadRequest)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Status = %v, want %v", rec.Code, http.StatusBadRequest)
	}

	var result Response
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Errorf("Response is not valid JSON: %v", err)
	}

	if len(result.Errors) != 1 {
		t.Errorf("Errors length = %v, want 1", len(result.Errors))
	}

	if result.Errors[0].Message != "something went wrong" {
		t.Errorf("Error message = %v, want 'something went wrong'", result.Errors[0].Message)
	}
}

func TestGraphQLError_Marshal(t *testing.T) {
	err := GraphQLError{
		Message: "test error",
		Path:    []interface{}{"user", "name"},
		Extensions: map[string]interface{}{
			"code": "INVALID_INPUT",
		},
	}

	data, err2 := json.Marshal(err)
	if err2 != nil {
		t.Errorf("Failed to marshal GraphQLError: %v", err2)
	}

	var decoded GraphQLError
	if err3 := json.Unmarshal(data, &decoded); err3 != nil {
		t.Errorf("Failed to unmarshal GraphQLError: %v", err3)
	}

	if decoded.Message != err.Message {
		t.Errorf("Message = %v, want %v", decoded.Message, err.Message)
	}
}

func TestRequest_JSONMarshal(t *testing.T) {
	req := Request{
		Query:         "{ users { id } }",
		Variables:     map[string]interface{}{"limit": 10},
		OperationName: "GetUsers",
		Extensions:    map[string]interface{}{"persistedQuery": map[string]string{"sha256Hash": "abc123"}},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Errorf("Failed to marshal Request: %v", err)
	}

	var decoded Request
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Errorf("Failed to unmarshal Request: %v", err)
	}

	if decoded.Query != req.Query {
		t.Errorf("Query = %v, want %v", decoded.Query, req.Query)
	}
}

func TestResponse_JSONMarshal(t *testing.T) {
	resp := Response{
		Data:   json.RawMessage(`{"users":[{"id":"1"}]}`),
		Errors: []GraphQLError{{Message: "partial error"}},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Errorf("Failed to marshal Response: %v", err)
	}

	var decoded Response
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Errorf("Failed to unmarshal Response: %v", err)
	}

	if len(decoded.Errors) != 1 {
		t.Errorf("Errors length = %v, want 1", len(decoded.Errors))
	}
}
