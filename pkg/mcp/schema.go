package mcp

import (
	"github.com/invopop/jsonschema"
)

// reflectStructToSchema converte uma struct Go em um InputSchema compatível com MCP usando reflection.
func reflectStructToSchema(input any) InputSchema {
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            true,
	}

	schema := reflector.Reflect(input)

	inputSchema := InputSchema{
		Type:       "object",
		Properties: make(map[string]interface{}),
		Required:   schema.Required,
	}

	// Mapeia as propriedades geradas para o formato esperado pelo MCP.
	if schema.Properties != nil {
		for pair := schema.Properties.Oldest(); pair != nil; pair = pair.Next() {
			inputSchema.Properties[pair.Key] = pair.Value
		}
	}

	return inputSchema
}
