package federation

import (
	"strings"
	"testing"
)

func TestCompose_NoSubgraphs(t *testing.T) {
	t.Parallel()
	c := NewComposer()
	_, err := c.Compose(nil)
	if err == nil {
		t.Fatal("expected error for no subgraphs")
	}
	if !strings.Contains(err.Error(), "no subgraphs") {
		t.Errorf("error = %q, want 'no subgraphs'", err.Error())
	}
}

func TestCompose_NilSchema(t *testing.T) {
	t.Parallel()
	c := NewComposer()
	sg := &Subgraph{ID: "s1", Schema: nil}
	schema, err := c.Compose([]*Subgraph{sg})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if schema == nil {
		t.Fatal("expected non-nil schema")
	}
}

func TestCompose_IntrospectionTypeSkipped(t *testing.T) {
	t.Parallel()
	c := NewComposer()
	sg := &Subgraph{
		ID: "s1",
		Schema: &Schema{
			Types: map[string]*Type{
				"__Type": {Kind: "OBJECT", Name: "__Type", Fields: map[string]*Field{"name": {Name: "name", Type: "String"}}},
				"User":   {Kind: "OBJECT", Name: "User", Fields: map[string]*Field{"id": {Name: "id", Type: "ID!"}}},
			},
		},
	}
	schema, err := c.Compose([]*Subgraph{sg})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := schema.Types["__Type"]; ok {
		t.Error("__Type should be excluded from supergraph types")
	}
	if _, ok := schema.Types["User"]; !ok {
		t.Error("User should be in supergraph types")
	}
}

func TestMergeTypes_Fields(t *testing.T) {
	t.Parallel()
	c := NewComposer()
	existing := &Type{
		Kind:   "OBJECT",
		Name:   "User",
		Fields: map[string]*Field{"id": {Name: "id", Type: "ID!"}},
	}
	new := &Type{
		Kind:   "OBJECT",
		Name:   "User",
		Fields: map[string]*Field{"email": {Name: "email", Type: "String!"}, "id": {Name: "id", Type: "ID!"}},
	}
	err := c.mergeTypes(existing, new, &Subgraph{ID: "s1"})
	if err != nil {
		t.Fatalf("merge failed: %v", err)
	}
	if _, ok := existing.Fields["email"]; !ok {
		t.Error("email field should be merged")
	}
	if _, ok := existing.Fields["id"]; !ok {
		t.Error("id field should still exist")
	}
}

func TestMergeTypes_Interfaces(t *testing.T) {
	t.Parallel()
	c := NewComposer()
	existing := &Type{
		Kind:       "OBJECT",
		Name:       "User",
		Fields:     map[string]*Field{"id": {Name: "id", Type: "ID!"}},
		Interfaces: []string{"Node"},
	}
	new := &Type{
		Kind:       "OBJECT",
		Name:       "User",
		Fields:     map[string]*Field{},
		Interfaces: []string{"Entity", "Node"},
	}
	err := c.mergeTypes(existing, new, &Subgraph{ID: "s1"})
	if err != nil {
		t.Fatalf("merge failed: %v", err)
	}
	hasEntity := false
	for _, iface := range existing.Interfaces {
		if iface == "Entity" {
			hasEntity = true
		}
	}
	if !hasEntity {
		t.Error("Entity interface should be merged")
	}
	// Node should not be duplicated
	count := 0
	for _, iface := range existing.Interfaces {
		if iface == "Node" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("Node interface count = %d, want 1", count)
	}
}

func TestMergeTypes_PossibleTypes(t *testing.T) {
	t.Parallel()
	c := NewComposer()
	existing := &Type{
		Kind:          "UNION",
		Name:          "SearchResult",
		Fields:        map[string]*Field{},
		PossibleTypes: []string{"User"},
	}
	new := &Type{
		Kind:          "UNION",
		Name:          "SearchResult",
		Fields:        map[string]*Field{},
		PossibleTypes: []string{"Post", "User"},
	}
	err := c.mergeTypes(existing, new, &Subgraph{ID: "s1"})
	if err != nil {
		t.Fatalf("merge failed: %v", err)
	}
	hasPost := false
	for _, pt := range existing.PossibleTypes {
		if pt == "Post" {
			hasPost = true
		}
	}
	if !hasPost {
		t.Error("Post possible type should be merged")
	}
}

func TestIsEntity(t *testing.T) {
	t.Parallel()
	c := NewComposer()
	tests := []struct {
		name      string
		typeDef   *Type
		expected  bool
	}{
		{
			"with @key directive",
			&Type{Name: "User", Directives: []TypeDirective{{Name: "key", Args: map[string]string{"fields": "id"}}}},
			true,
		},
		{
			"without @key directive",
			&Type{Name: "User", Directives: []TypeDirective{{Name: "deprecated"}}},
			false,
		},
		{
			"no directives",
			&Type{Name: "User"},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := c.isEntity(tt.typeDef); got != tt.expected {
				t.Errorf("isEntity() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestAddEntity(t *testing.T) {
	t.Parallel()
	c := NewComposer()
	sg := &Subgraph{
		ID: "users",
		Schema: &Schema{
			Types: map[string]*Type{
				"User": {
					Name: "User",
					Directives: []TypeDirective{{Name: "key", Args: map[string]string{"fields": "id email"}}},
				},
			},
		},
	}
	c.addEntity("User", sg)
	entity, ok := c.entities["User"]
	if !ok {
		t.Fatal("expected User entity")
	}
	if len(entity.KeyFields) != 2 {
		t.Errorf("KeyFields = %v, want 2 fields", entity.KeyFields)
	}
	if _, ok := entity.Subgraphs["users"]; !ok {
		t.Error("users subgraph should be registered")
	}
}

func TestAddEntity_DefaultKeyField(t *testing.T) {
	t.Parallel()
	c := NewComposer()
	sg := &Subgraph{
		ID: "products",
		Schema: &Schema{
			Types: map[string]*Type{
				"Product": {Name: "Product"},
			},
		},
	}
	c.addEntity("Product", sg)
	entity := c.entities["Product"]
	if len(entity.KeyFields) != 1 || entity.KeyFields[0] != "id" {
		t.Errorf("KeyFields = %v, want [id]", entity.KeyFields)
	}
}

func TestGetAuthorizedFields_TypeLevel(t *testing.T) {
	t.Parallel()
	c := NewComposer()
	c.supergraph.Types["User"] = &Type{
		Name:   "User",
		Fields: map[string]*Field{"id": {Name: "id", Type: "ID!"}},
		Directives: []TypeDirective{
			{Name: "authorized", Args: map[string]string{"roles": "admin super"}},
		},
	}
	auth := c.GetAuthorizedFields()
	if roles, ok := auth["User.*"]; !ok {
		t.Error("expected User.* in authorized fields")
	} else if len(roles) != 2 {
		t.Errorf("User.* roles = %v, want 2", roles)
	}
}

func TestGetAuthorizedFields_FieldLevel(t *testing.T) {
	t.Parallel()
	c := NewComposer()
	c.supergraph.Types["User"] = &Type{
		Name: "User",
		Fields: map[string]*Field{
			"email": {
				Name: "email",
				Type: "String!",
				Directives: []TypeDirective{
					{Name: "authorized", Args: map[string]string{"roles": "admin"}},
				},
			},
		},
	}
	auth := c.GetAuthorizedFields()
	if roles, ok := auth["User.email"]; !ok {
		t.Error("expected User.email in authorized fields")
	} else if len(roles) != 1 || roles[0] != "admin" {
		t.Errorf("User.email roles = %v, want [admin]", roles)
	}
}

func TestGetAuthorizedFields_IntrospectionSkipped(t *testing.T) {
	t.Parallel()
	c := NewComposer()
	c.supergraph.Types["__Type"] = &Type{
		Name: "__Type",
		Fields: map[string]*Field{
			"name": {Name: "name", Type: "String"},
		},
		Directives: []TypeDirective{
			{Name: "authorized", Args: map[string]string{"roles": "admin"}},
		},
	}
	auth := c.GetAuthorizedFields()
	if _, ok := auth["__Type.*"]; ok {
		t.Error("__Type should be skipped")
	}
}

func TestBuildObjectSDL_Basic(t *testing.T) {
	t.Parallel()
	c := NewComposer()
	typ := &Type{
		Kind:   "OBJECT",
		Name:   "User",
		Fields: map[string]*Field{"id": {Name: "id", Type: "ID!"}, "name": {Name: "name", Type: "String!"}},
	}
	sdl := c.buildObjectSDL(typ)
	if !strings.Contains(sdl, "type User") {
		t.Error("expected 'type User' in SDL")
	}
	if !strings.Contains(sdl, "id: ID!") {
		t.Error("expected 'id: ID!' in SDL")
	}
}

func TestBuildObjectSDL_WithDescription(t *testing.T) {
	t.Parallel()
	c := NewComposer()
	typ := &Type{
		Kind:        "OBJECT",
		Name:        "User",
		Description: "A user account",
		Fields:      map[string]*Field{"id": {Name: "id", Type: "ID!"}},
	}
	sdl := c.buildObjectSDL(typ)
	if !strings.Contains(sdl, `"A user account"`) {
		t.Error("expected description in SDL")
	}
}

func TestBuildObjectSDL_WithInterfaces(t *testing.T) {
	t.Parallel()
	c := NewComposer()
	typ := &Type{
		Kind:       "OBJECT",
		Name:       "User",
		Interfaces: []string{"Node", "Entity"},
		Fields:     map[string]*Field{"id": {Name: "id", Type: "ID!"}},
	}
	sdl := c.buildObjectSDL(typ)
	if !strings.Contains(sdl, "implements Node & Entity") {
		t.Error("expected 'implements Node & Entity' in SDL")
	}
}

func TestBuildObjectSDL_WithFieldArgs(t *testing.T) {
	t.Parallel()
	c := NewComposer()
	typ := &Type{
		Kind: "OBJECT",
		Name: "Query",
		Fields: map[string]*Field{
			"user": {
				Name: "user",
				Type: "User",
				Args: map[string]*Argument{
					"id":   {Name: "id", Type: "ID!"},
					"lang": {Name: "lang", Type: "String"},
				},
			},
		},
	}
	sdl := c.buildObjectSDL(typ)
	if !strings.Contains(sdl, "user(") {
		t.Error("expected field args in SDL")
	}
	if !strings.Contains(sdl, "id: ID!") {
		t.Error("expected id arg in SDL")
	}
}

func TestBuildObjectSDL_DeprecatedField(t *testing.T) {
	t.Parallel()
	c := NewComposer()
	typ := &Type{
		Kind: "OBJECT",
		Name: "User",
		Fields: map[string]*Field{
			"oldField": {
				Name:              "oldField",
				Type:              "String",
				IsDeprecated:      true,
				DeprecationReason: "Use newField instead",
			},
		},
	}
	sdl := c.buildObjectSDL(typ)
	if !strings.Contains(sdl, "@deprecated") {
		t.Error("expected @deprecated in SDL")
	}
	if !strings.Contains(sdl, "Use newField instead") {
		t.Error("expected deprecation reason in SDL")
	}
}

func TestBuildInterfaceSDL(t *testing.T) {
	t.Parallel()
	c := NewComposer()
	typ := &Type{
		Kind:        "INTERFACE",
		Name:        "Node",
		Description: "Base interface",
		Fields: map[string]*Field{
			"id": {Name: "id", Type: "ID!"},
		},
	}
	sdl := c.buildInterfaceSDL(typ)
	if !strings.Contains(sdl, "interface Node") {
		t.Error("expected 'interface Node' in SDL")
	}
	if !strings.Contains(sdl, `"Base interface"`) {
		t.Error("expected description in SDL")
	}
	if !strings.Contains(sdl, "id: ID!") {
		t.Error("expected field in SDL")
	}
}

func TestBuildUnionSDL(t *testing.T) {
	t.Parallel()
	c := NewComposer()
	typ := &Type{
		Kind:          "UNION",
		Name:          "SearchResult",
		PossibleTypes: []string{"User", "Post"},
	}
	sdl := c.buildUnionSDL(typ)
	if !strings.Contains(sdl, "union SearchResult = User | Post") {
		t.Errorf("SDL = %q", sdl)
	}
}

func TestBuildEnumSDL(t *testing.T) {
	t.Parallel()
	c := NewComposer()
	typ := &Type{
		Kind:       "ENUM",
		Name:       "Role",
		EnumValues: []string{"ADMIN", "USER", "GUEST"},
	}
	sdl := c.buildEnumSDL(typ)
	if !strings.Contains(sdl, "enum Role") {
		t.Error("expected 'enum Role' in SDL")
	}
	if !strings.Contains(sdl, "ADMIN") {
		t.Error("expected ADMIN in SDL")
	}
}

func TestBuildInputSDL(t *testing.T) {
	t.Parallel()
	c := NewComposer()
	typ := &Type{
		Kind: "INPUT_OBJECT",
		Name: "UserInput",
		InputFields: map[string]*InputField{
			"name":  {Name: "name", Type: "String!"},
			"email": {Name: "email", Type: "String!"},
		},
	}
	sdl := c.buildInputSDL(typ)
	if !strings.Contains(sdl, "input UserInput") {
		t.Error("expected 'input UserInput' in SDL")
	}
	if !strings.Contains(sdl, "name: String!") {
		t.Error("expected name field in SDL")
	}
}

func TestBuildScalarSDL(t *testing.T) {
	t.Parallel()
	c := NewComposer()
	typ := &Type{Kind: "SCALAR", Name: "DateTime"}
	sdl := c.buildScalarSDL(typ)
	if sdl != "scalar DateTime" {
		t.Errorf("SDL = %q, want 'scalar DateTime'", sdl)
	}
}

func TestGetEntities(t *testing.T) {
	t.Parallel()
	c := NewComposer()
	c.entities["User"] = &Entity{Name: "User", KeyFields: []string{"id"}}
	entities := c.GetEntities()
	if len(entities) != 1 {
		t.Errorf("entities count = %d, want 1", len(entities))
	}
	if entities["User"].Name != "User" {
		t.Error("User entity should exist")
	}
}
