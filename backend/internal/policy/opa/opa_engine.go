package opa

import (
	"context"
	"encoding/json"

	"github.com/Armour007/aura-backend/internal/policy"
	"github.com/open-policy-agent/opa/rego"
)

// Evaluator implements policy.Evaluator using OPA/Rego.
type Evaluator struct{}

func New() policy.Evaluator { return &Evaluator{} }

// exported as a bridge for policy.NewOPAEvaluator()
func OpaNew() policy.Evaluator { return &Evaluator{} }

func (e *Evaluator) Name() string { return policy.EngineRego }

type compiled struct {
	query rego.PreparedEvalQuery
}

// Compile expects policyBody JSON with field "module" containing a Rego module string.
// The module should define a package named 'aura' and a rule 'allow' boolean.
func (e *Evaluator) Compile(policyBody json.RawMessage) (policy.CompiledPolicy, error) {
	var m map[string]any
	if err := json.Unmarshal(policyBody, &m); err != nil {
		return nil, err
	}
	mod, _ := m["module"].(string)
	if mod == "" {
		return nil, ErrBadRegoModule
	}
	r := rego.New(
		rego.Module("policy.rego", mod),
		rego.Query("data.aura.allow"),
	)
	pq, err := r.PrepareForEval(context.Background())
	if err != nil {
		return nil, err
	}
	return &compiled{query: pq}, nil
}

// Evaluate returns Allow true when the 'allow' rule evaluates truthy; Reason best-effort.
func (e *Evaluator) Evaluate(comp policy.CompiledPolicy, input json.RawMessage) (policy.Decision, error) {
	c, ok := comp.(*compiled)
	if !ok {
		return policy.Decision{}, ErrBadCompiled
	}
	var in any
	_ = json.Unmarshal(input, &in)
	res, err := c.query.Eval(context.Background(), rego.EvalInput(in))
	if err != nil {
		return policy.Decision{}, err
	}
	allow := false
	if len(res) > 0 && len(res[0].Expressions) > 0 {
		if b, ok := res[0].Expressions[0].Value.(bool); ok {
			allow = b
		}
	}
	reason := "OPA allow"
	if !allow {
		reason = "OPA deny"
	}
	return policy.Decision{Allow: allow, Reason: reason}, nil
}

var (
	ErrBadRegoModule = &evalError{"policy body must include a 'module' string with package aura and allow rule"}
	ErrBadCompiled   = &evalError{"invalid compiled policy type"}
)

type evalError struct{ s string }

func (e *evalError) Error() string { return e.s }
