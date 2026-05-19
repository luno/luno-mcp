package tools

import (
	"encoding/json"
	"reflect"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/luno/luno-go"
	"github.com/luno/luno-go/decimal"
	"github.com/mark3labs/mcp-go/mcp"
)

// outputSchemaTypes patches the reflection-based schema generator for Luno SDK
// types whose custom MarshalJSON emits a primitive. Without these overrides,
// google/jsonschema-go (used by mcp-go's WithOutputSchema) sees structs with
// no exported fields and emits {type: "object"}, so the runtime value -- a
// string like "100.50" -- fails validation on strict clients.
var outputSchemaTypes = map[reflect.Type]*jsonschema.Schema{
	reflect.TypeFor[decimal.Decimal](): {Type: "string"},
	reflect.TypeFor[luno.Time]():       {Types: []string{"null", "string"}},
}

// withOutputSchema is a drop-in for mcp.WithOutputSchema[T] that applies the
// outputSchemaTypes overrides. Falls back to the stock helper if generation
// or marshalling fails, so we never silently drop the schema.
func withOutputSchema[T any]() mcp.ToolOption {
	schema, err := jsonschema.For[T](&jsonschema.ForOptions{
		IgnoreInvalidTypes: true,
		TypeSchemas:        outputSchemaTypes,
	})
	if err != nil {
		return mcp.WithOutputSchema[T]()
	}
	raw, err := json.Marshal(schema)
	if err != nil {
		return mcp.WithOutputSchema[T]()
	}
	return mcp.WithRawOutputSchema(raw)
}
