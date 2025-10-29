package policy

import (
	"encoding/json"
	"fmt"
)

// validateAgainstSchema implements a very small subset of JSON Schema:
// { "required": ["field"], "properties": {"field": {"type": "string"|"number"|"boolean"|"object"|"array"} } }
// Returns a slice of validation error messages (empty if valid)
func validateAgainstSchema(input json.RawMessage, schema map[string]any) []string {
	var errs []string
	var m map[string]any
	if err := json.Unmarshal(input, &m); err != nil {
		return []string{"input is not valid JSON object"}
	}
	// required fields
	if req, ok := schema["required"].([]any); ok {
		for _, r := range req {
			name, _ := r.(string)
			if name == "" {
				continue
			}
			if _, ok := m[name]; !ok {
				errs = append(errs, fmt.Sprintf("missing required field '%s'", name))
			}
		}
	}
	// properties types
	if props, ok := schema["properties"].(map[string]any); ok {
		for name, specRaw := range props {
			spec, _ := specRaw.(map[string]any)
			expected, _ := spec["type"].(string)
			if expected == "" {
				continue
			}
			v, ok := m[name]
			if !ok {
				continue
			} // handled by required above if needed
			if !typeMatches(v, expected) {
				errs = append(errs, fmt.Sprintf("field '%s' expected type %s", name, expected))
			}
		}
	}
	return errs
}

func typeMatches(v any, t string) bool {
	switch t {
	case "string":
		_, ok := v.(string)
		return ok
	case "number":
		switch v.(type) {
		case float64, int, int64, json.Number:
			return true
		}
		return false
	case "boolean":
		_, ok := v.(bool)
		return ok
	case "object":
		_, ok := v.(map[string]any)
		return ok
	case "array":
		_, ok := v.([]any)
		return ok
	default:
		return true
	}
}
