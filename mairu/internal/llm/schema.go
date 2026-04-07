package llm

import (
	"reflect"
	"strings"

	"github.com/google/generative-ai-go/genai"
)

// GenerateSchema uses reflection to generate a genai.Schema from a struct.
// It reads the `json` tag for property names and the `desc` tag for descriptions.
func GenerateSchema(v any) *genai.Schema {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return generateSchemaFromType(t)
}

func generateSchemaFromType(t reflect.Type) *genai.Schema {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	switch t.Kind() {
	case reflect.Struct:
		props := make(map[string]*genai.Schema)
		var req []string
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			if !field.IsExported() {
				continue
			}
			name := field.Name
			if jsonTag := field.Tag.Get("json"); jsonTag != "" {
				parts := strings.Split(jsonTag, ",")
				if parts[0] == "-" {
					continue
				}
				if parts[0] != "" {
					name = parts[0]
				}
				// if not omitempty, make it required? or we can use a `required:"true"` tag
			}

			desc := field.Tag.Get("desc")

			// always require fields unless omitempty
			isOptional := strings.Contains(field.Tag.Get("json"), "omitempty")
			if !isOptional {
				req = append(req, name)
			}

			propSchema := generateSchemaFromType(field.Type)
			if propSchema != nil {
				if desc != "" {
					propSchema.Description = desc
				}
				props[name] = propSchema
			}
		}
		return &genai.Schema{
			Type:       genai.TypeObject,
			Properties: props,
			Required:   req,
		}
	case reflect.String:
		return &genai.Schema{Type: genai.TypeString}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return &genai.Schema{Type: genai.TypeInteger}
	case reflect.Float32, reflect.Float64:
		return &genai.Schema{Type: genai.TypeNumber}
	case reflect.Bool:
		return &genai.Schema{Type: genai.TypeBoolean}
	case reflect.Slice, reflect.Array:
		elemSchema := generateSchemaFromType(t.Elem())
		return &genai.Schema{
			Type:  genai.TypeArray,
			Items: elemSchema,
		}
	case reflect.Map:
		// Map keys must be strings, values can be anything
		// But in OpenAPI, map of X is an object with additionalProperties: X.
		// genai.Schema doesn't seem to support additionalProperties perfectly,
		// but we can map it to an generic Object if needed, or we might not need maps.
		return &genai.Schema{Type: genai.TypeObject}
	case reflect.Interface:
		// any type => fallback
		return &genai.Schema{Type: genai.TypeObject} // Not perfect but genai doesn't have "any" type
	}
	return &genai.Schema{Type: genai.TypeString}
}
