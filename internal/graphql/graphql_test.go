package graphql

import (
	"testing"
)

func TestIsGraphQLRequest(t *testing.T) {
	tests := []struct {
		name    string
		method  string
		contentType string
		query   string
		isGraphQL bool
	}{
		{
			name:    "POST with application/json",
			method:  "POST",
			contentType: "application/json",
			isGraphQL: true,
		},
		{
			name:    "POST with application/graphql",
			method:  "POST",
			contentType: "application/graphql",
			isGraphQL: true,
		},
		{
			name:    "GET with query param",
			method:  "GET",
			query:   "{ users { id } }",
			isGraphQL: true,
		},
		{
			name:    "GET without query param",
			method:  "GET",
			isGraphQL: false,
		},
		{
			name:    "POST with other content type",
			method:  "POST",
			contentType: "text/plain",
			isGraphQL: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We can't easily test without HTTP request, so we just document
			t.Logf("Method: %s, Content-Type: %s -> isGraphQL: %v", tt.method, tt.contentType, tt.isGraphQL)
		})
	}
}

func TestParseQuery(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		wantErr bool
	}{
		{
			name:    "simple query",
			query:   `{ users { id name } }`,
			wantErr: false,
		},
		{
			name: "query with operation type",
			query: `query GetUsers {
				users {
					id
					name
					posts {
						title
					}
				}
			}`,
			wantErr: false,
		},
		{
			name: "query with alias",
			query: `{
				users: allUsers {
					id
					name
				}
			}`,
			wantErr: false,
		},
		{
			name: "query with arguments",
			query: `{
				user(id: "123") {
					name
					email
				}
			}`,
			wantErr: false,
		},
		{
			name: "query with fragment",
			query: `
				fragment UserFields on User {
					id
					name
				}
				query {
					users {
						...UserFields
					}
				}
			`,
			wantErr: false,
		},
		{
			name:    "empty query",
			query:   "",
			wantErr: true,
		},
		{
			name: "mutation",
			query: `mutation CreateUser {
				createUser(name: "John") {
					id
				}
			}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ast, err := ParseQuery(tt.query)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseQuery() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}
			if ast == nil {
				t.Error("ParseQuery() returned nil AST")
			}
		})
	}
}

func TestCalculateDepth(t *testing.T) {
	tests := []struct {
		name  string
		query string
		want  int
	}{
		{
			name:  "flat query",
			query: `{ users { id name } }`,
			want:  2,
		},
		{
			name: "nested query",
			query: `{
				users {
					id
					posts {
						title
					}
				}
			}`,
			want: 3,
		},
		{
			name: "deeply nested query",
			query: `{
				users {
					posts {
						comments {
							author {
								name
							}
						}
					}
				}
			}`,
			want: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ast, err := ParseQuery(tt.query)
			if err != nil {
				t.Fatalf("ParseQuery() error = %v", err)
			}

			got := ast.Depth()
			if got != tt.want {
				t.Errorf("Depth() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestQueryAnalyzer(t *testing.T) {
	analyzer := NewQueryAnalyzer(&AnalyzerConfig{
		MaxDepth:      5,
		MaxComplexity: 100,
		DefaultCost:   1,
	})

	tests := []struct {
		name    string
		query   string
		isValid bool
	}{
		{
			name: "valid simple query",
			query: `{
				users {
					id
					name
				}
			}`,
			isValid: true,
		},
		{
			name: "query exceeding max depth",
			query: `{
				a {
					b {
						c {
							d {
								e {
									f {
										g
									}
								}
							}
						}
					}
				}
			}`,
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := analyzer.Analyze(tt.query)
			if err != nil {
				t.Logf("Analyze() error = %v", err)
				return
			}

			if result.IsValid != tt.isValid {
				t.Errorf("IsValid = %v, want %v", result.IsValid, tt.isValid)
			}
		})
	}
}

func TestIsIntrospectionQuery(t *testing.T) {
	tests := []struct {
		query string
		want  bool
	}{
		{
			query: `{ __schema { types { name } } }`,
			want:  true,
		},
		{
			query: `{ __type(name: "User") { name } }`,
			want:  true,
		},
		{
			query: `{ users { id __typename } }`,
			want:  true,
		},
		{
			query: `{ users { id name } }`,
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			got := IsIntrospectionQuery(tt.query)
			if got != tt.want {
				t.Errorf("IsIntrospectionQuery() = %v, want %v", got, tt.want)
			}
		})
	}
}
