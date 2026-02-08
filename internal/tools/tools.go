// tools.go - Base tool definitions and schema generation
package tools

import (
	"encoding/json"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/invopop/jsonschema"
)

// ToolDefinition defines a function that Claude can call
// Each tool has:
// - Name: The identifier Claude uses to call it
// - Description: Tells Claude when and how to use this tool
// - InputSchema: JSON schema describing the parameters
// - Function: The actual Go function to execute
type ToolDefinition struct {
	Name        string                                      `json:"name"`
	Description string                                      `json:"description"`
	InputSchema anthropic.ToolInputSchemaParam              `json:"input_schema"`
	Function    func(input json.RawMessage) (string, error) // Takes JSON input, returns string result
}

// GenerateSchema generates a JSON schema from a Go struct using generics
// This is a helper function that automatically creates the schema Claude needs
// to understand what parameters a tool accepts
//
// Example: GenerateSchema[ReadFileInput]() produces a schema like:
//
//	{
//	  "type": "object",
//	  "properties": {
//	    "path": {
//	      "type": "string",
//	      "description": "The relative path of a file in the working directory."
//	    }
//	  },
//	  "required": ["path"]
//	}
func GenerateSchema[T any]() anthropic.ToolInputSchemaParam {
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false, // Strict: only allow defined properties
		DoNotReference:            true,  // Inline all definitions instead of using $ref
	}
	var v T

	// Use reflection to generate the schema from the struct
	schema := reflector.Reflect(v)

	return anthropic.ToolInputSchemaParam{
		Properties: schema.Properties, // The field definitions from the struct
	}
}
