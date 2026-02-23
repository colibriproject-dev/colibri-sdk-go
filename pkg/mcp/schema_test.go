package mcp

import (
	"encoding/json"
	"testing"

	"github.com/invopop/jsonschema"
)

type ExampleArgs struct {
	ID    int    `json:"id" jsonschema:"description=O identificador único,example=123"`
	Name  string `json:"name" jsonschema:"description=O nome do usuário"`
	Email string `json:"email,omitempty"`
}

func TestReflectStructToSchema(t *testing.T) {
	schema := reflectStructToSchema(ExampleArgs{})

	if schema.Type != "object" {
		t.Errorf("expected type 'object', got %s", schema.Type)
	}

	if _, ok := schema.Properties["id"]; !ok {
		t.Error("expected property 'id' to exist")
	}

	if _, ok := schema.Properties["name"]; !ok {
		t.Error("expected property 'name' to exist")
	}

	// Verifica se a descrição foi capturada (usando map[string]interface{})
	idProp, ok := schema.Properties["id"].(*jsonschema.Schema)
	if !ok {
		// Se não for *jsonschema.Schema diretamente, vamos tentar ver o que é
		t.Logf("id property type: %T", schema.Properties["id"])
	} else if idProp.Description != "O identificador único" {
		t.Errorf("expected description 'O identificador único', got %s", idProp.Description)
	}

	// Verifica campos obrigatórios
	// Por padrão, jsonschema marca campos como obrigatórios a menos que tenham omitempty ou sejam ponteiros
	requiredFields := make(map[string]bool)
	for _, req := range schema.Required {
		requiredFields[req] = true
	}

	if !requiredFields["id"] {
		t.Error("expected 'id' to be required")
	}
	if !requiredFields["name"] {
		t.Error("expected 'name' to be required")
	}
	if requiredFields["email"] {
		t.Error("expected 'email' to not be required")
	}

	// Log do JSON gerado para inspeção manual se necessário
	data, _ := json.MarshalIndent(schema, "", "  ")
	t.Logf("Generated Schema:\n%s", string(data))
}
