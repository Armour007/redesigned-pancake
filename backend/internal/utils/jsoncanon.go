package utils

import (
	"encoding/json"
	"sort"
)

// CanonicalizeJSON takes arbitrary JSON and returns a canonical, stable-ordered JSON encoding.
// Maps are sorted by key recursively; arrays preserve order; numbers/strings/bools/null unchanged.
func CanonicalizeJSON(b []byte) []byte {
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		// if not valid JSON, just return original
		return b
	}
	c := canonicalize(v)
	out, err := json.Marshal(c)
	if err != nil {
		return b
	}
	return out
}

func canonicalize(v any) any {
	switch t := v.(type) {
	case map[string]any:
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		m := make(map[string]any, len(t))
		for _, k := range keys {
			m[k] = canonicalize(t[k])
		}
		return m
	case []any:
		for i := range t {
			t[i] = canonicalize(t[i])
		}
		return t
	default:
		return v
	}
}
