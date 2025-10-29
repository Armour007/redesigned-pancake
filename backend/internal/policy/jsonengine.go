package policy

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// AuraJSONEvaluator implements a simple JSON DSL with and/or/not/in, arrays, and precedence
type AuraJSONEvaluator struct{}

func (e *AuraJSONEvaluator) Name() string { return EngineAuraJSON }

type compiledJSON struct {
	Body   map[string]any
	Schema map[string]any
}

func (e *AuraJSONEvaluator) Compile(policyBody json.RawMessage) (CompiledPolicy, error) {
	var m map[string]any
	if err := json.Unmarshal(policyBody, &m); err != nil {
		return nil, fmt.Errorf("invalid policy body: %w", err)
	}
	var schema map[string]any
	if s, ok := m["schema"].(map[string]any); ok {
		schema = s
	}
	return &compiledJSON{Body: m, Schema: schema}, nil
}

func (e *AuraJSONEvaluator) Evaluate(compiled CompiledPolicy, input json.RawMessage) (Decision, error) {
	cj, ok := compiled.(*compiledJSON)
	if !ok {
		return Decision{}, fmt.Errorf("bad compiled policy type")
	}
	start := time.Now()
	var in map[string]any
	_ = json.Unmarshal(input, &in) // best-effort

	// optional schema validation
	trace := &Trace{EvaluatedRules: []RuleTrace{}, At: time.Now(), Engine: e.Name(), InputContext: input}
	if len(cj.Schema) > 0 {
		if errs := validateAgainstSchema(input, cj.Schema); len(errs) > 0 {
			trace.Validations = errs
			d := Decision{Allow: false, Reason: "Schema validation failed", Trace: trace}
			d.TraceID = hashDecision(input, cj.Body)
			d.Trace.DurationMS = time.Since(start).Milliseconds()
			return d, nil
		}
	}

	rules, _ := cj.Body["rules"].([]any)
	precedence := map[string]bool{"deny_overrides": true}
	if p, ok := cj.Body["precedence"].(map[string]any); ok {
		if v, ok := p["deny_overrides"].(bool); ok {
			precedence["deny_overrides"] = v
		}
	}

	var allow bool
	reason := "No matching allow rule"

	for _, r := range rules {
		rm, ok := r.(map[string]any)
		if !ok {
			continue
		}
		effect := strings.ToLower(fmt.Sprintf("%v", rm["effect"]))
		ruleID := fmt.Sprintf("%v", rm["id"])
		when, _ := rm["when"].(map[string]any)
		matched := evalExpr(in, when)
		trace.EvaluatedRules = append(trace.EvaluatedRules, RuleTrace{RuleID: ruleID, Matched: matched, Effect: effect})
		if matched {
			if effect == "deny" {
				allow = false
				reason = "Matched deny rule"
				if precedence["deny_overrides"] {
					break
				}
			}
			if effect == "allow" {
				allow = true
				reason = "Matched allow rule"
				// continue to see if a deny appears later and overrides when configured
			}
		}
	}

	d := Decision{Allow: allow, Reason: reason, Trace: trace}
	d.TraceID = hashDecision(input, cj.Body)
	d.Trace.DurationMS = time.Since(start).Milliseconds()
	return d, nil
}

func hashDecision(input any, body any) string {
	b1, _ := json.Marshal(input)
	b2, _ := json.Marshal(body)
	h := sha256.Sum256(append(b1, b2...))
	return hex.EncodeToString(h[:8])
}

// evalExpr supports {"and": [...]}, {"or": [...]}, {"not": expr}, operators on fields and array membership
func evalExpr(input map[string]any, expr map[string]any) bool {
	if expr == nil {
		return true
	}
	if v, ok := expr["and"].([]any); ok {
		for _, e := range v {
			if !evalExpr(input, asMap(e)) {
				return false
			}
		}
		return true
	}
	if v, ok := expr["or"].([]any); ok {
		for _, e := range v {
			if evalExpr(input, asMap(e)) {
				return true
			}
		}
		return false
	}
	if v, ok := expr["not"].(map[string]any); ok {
		return !evalExpr(input, v)
	}
	// field ops: {"field": {"eq": 1, "in": [..]}}
	for k, vv := range expr {
		ops, ok := vv.(map[string]any)
		if !ok {
			continue
		}
		val, has := pluck(input, k)
		if !has {
			return false
		}
		for op, rhs := range ops {
			switch strings.ToLower(op) {
			case "eq":
				if fmt.Sprintf("%v", val) != fmt.Sprintf("%v", rhs) {
					return false
				}
			case "neq":
				if fmt.Sprintf("%v", val) == fmt.Sprintf("%v", rhs) {
					return false
				}
			case "gt":
				if !cmpNumber(val, rhs, ">") {
					return false
				}
			case "gte":
				if !cmpNumber(val, rhs, ">=") {
					return false
				}
			case "lt":
				if !cmpNumber(val, rhs, "<") {
					return false
				}
			case "lte":
				if !cmpNumber(val, rhs, "<=") {
					return false
				}
			case "in":
				if !inArray(val, rhs) {
					return false
				}
			case "contains":
				if !contains(val, rhs) {
					return false
				}
			}
		}
	}
	return true
}

func asMap(v any) map[string]any { m, _ := v.(map[string]any); return m }

func pluck(m map[string]any, path string) (any, bool) {
	parts := strings.Split(path, ".")
	var cur any = m
	for _, token := range parts {
		name, idx, hasIdx := parseIndex(token)
		switch c := cur.(type) {
		case map[string]any:
			if name == "" {
				return nil, false
			}
			v, ok := c[name]
			if !ok {
				return nil, false
			}
			cur = v
		case []any:
			// if current is array, we require an index in this token
			if !hasIdx {
				return nil, false
			}
			if idx < 0 || idx >= len(c) {
				return nil, false
			}
			cur = c[idx]
			// we've consumed index; no further array step for this token
			hasIdx = false
		default:
			return nil, false
		}
		if hasIdx {
			if arr, ok := cur.([]any); ok {
				if idx < 0 || idx >= len(arr) {
					return nil, false
				}
				cur = arr[idx]
			} else {
				return nil, false
			}
		}
	}
	return cur, true
}

func parseIndex(s string) (field string, idx int, has bool) {
	// allow either field[3] or numeric segment "3" to index
	if i := strings.Index(s, "["); i >= 0 && strings.HasSuffix(s, "]") {
		field = s[:i]
		n := s[i+1 : len(s)-1]
		var j int
		_, err := fmt.Sscanf(n, "%d", &j)
		if err == nil {
			return field, j, true
		}
		return field, -1, false
	}
	// numeric segment
	var j int
	if _, err := fmt.Sscanf(s, "%d", &j); err == nil {
		return "", j, true
	}
	return s, -1, false
}

func cmpNumber(a any, b any, op string) bool {
	af, aok := toFloat(a)
	bf, bok := toFloat(b)
	if !aok || !bok {
		return false
	}
	switch op {
	case ">=":
		return af >= bf
	case "<=":
		return af <= bf
	case ">":
		return af > bf
	case "<":
		return af < bf
	}
	return false
}

func toFloat(v any) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case int:
		return float64(t), true
	case int64:
		return float64(t), true
	case json.Number:
		f, err := t.Float64()
		if err == nil {
			return f, true
		}
		return 0, false
	case string:
		var f float64
		_, err := fmt.Sscanf(t, "%f", &f)
		return f, err == nil
	}
	return 0, false
}

func inArray(val any, rhs any) bool {
	switch a := rhs.(type) {
	case []any:
		for _, x := range a {
			if fmt.Sprintf("%v", x) == fmt.Sprintf("%v", val) {
				return true
			}
		}
	case []string:
		for _, x := range a {
			if x == fmt.Sprintf("%v", val) {
				return true
			}
		}
	}
	return false
}

func contains(val any, rhs any) bool {
	return strings.Contains(fmt.Sprintf("%v", val), fmt.Sprintf("%v", rhs))
}
