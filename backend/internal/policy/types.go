package policy

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// EngineType enumerates supported engines
const (
	EngineAuraJSON = "aurajson"
	EngineRego     = "rego"
)

// Decision is the outcome of evaluation
type Decision struct {
	Allow           bool     `json:"allow"`
	Reason          string   `json:"reason,omitempty"`
	TraceID         string   `json:"trace_id,omitempty"`
	Trace           *Trace   `json:"trace,omitempty"`
	RequireApproval bool     `json:"require_approval,omitempty"`
	Hints           []string `json:"hints,omitempty"`
}

// Trace captures explainability details
type Trace struct {
	PolicyID      uuid.UUID `json:"policy_id"`
	PolicyVersion int       `json:"policy_version"`
	// Principal identity at evaluation time (optional enrichment)
	Principal      *PrincipalTrace `json:"principal,omitempty"`
	EvaluatedRules []RuleTrace     `json:"evaluated_rules"`
	InputContext   json.RawMessage `json:"input_context,omitempty"`
	DurationMS     int64           `json:"duration_ms"`
	At             time.Time       `json:"at"`
	Engine         string          `json:"engine"`
	Validations    []string        `json:"validations,omitempty"`
}

// PrincipalTrace captures caller identity included in traces
type PrincipalTrace struct {
	OrgID           string `json:"org_id,omitempty"`
	AgentID         string `json:"agent_id,omitempty"`
	SPIFFEID        string `json:"spiffe_id,omitempty"`
	AuthnKind       string `json:"authn_kind,omitempty"` // apikey|spiffe|oidc
	CertFingerprint string `json:"cert_fingerprint,omitempty"`
}

type RuleTrace struct {
	RuleID  string `json:"rule_id"`
	Matched bool   `json:"matched"`
	Effect  string `json:"effect"`
	Reason  string `json:"reason,omitempty"`
}

// Evaluator is the pluggable policy evaluator interface
type Evaluator interface {
	Compile(policyBody json.RawMessage) (CompiledPolicy, error)
	Evaluate(compiled CompiledPolicy, input json.RawMessage) (Decision, error)
	Name() string
}

// CompiledPolicy is an opaque compiled artifact
type CompiledPolicy interface{}

// Policy entity (prototype; DB-backed later)
type Policy struct {
	ID         uuid.UUID `json:"id"`
	OrgID      uuid.UUID `json:"org_id"`
	Name       string    `json:"name"`
	EngineType string    `json:"engine_type"`
	CreatedAt  time.Time `json:"created_at"`
}

type PolicyVersion struct {
	PolicyID    uuid.UUID       `json:"policy_id"`
	Version     int             `json:"version"`
	Body        json.RawMessage `json:"body"`
	Compiled    CompiledPolicy  `json:"-"`
	Checksum    string          `json:"checksum"`
	Status      string          `json:"status"` // draft|active
	CreatedBy   uuid.UUID       `json:"created_by"`
	CreatedAt   time.Time       `json:"created_at"`
	ActivatedAt *time.Time      `json:"activated_at,omitempty"`
}

type Assignment struct {
	PolicyID  uuid.UUID `json:"policy_id"`
	ScopeType string    `json:"scope_type"` // org|team|agent
	ScopeID   string    `json:"scope_id"`
	CreatedAt time.Time `json:"created_at"`
}

// In-memory prototype store
type Store struct {
	Policies    map[uuid.UUID]Policy
	Versions    map[uuid.UUID][]PolicyVersion // by policy id
	Assignments []Assignment
	Evaluators  map[string]Evaluator
}

func NewStore() *Store {
	return &Store{
		Policies:    map[uuid.UUID]Policy{},
		Versions:    map[uuid.UUID][]PolicyVersion{},
		Assignments: []Assignment{},
		Evaluators:  map[string]Evaluator{},
	}
}

func (s *Store) RegisterEvaluator(e Evaluator) {
	s.Evaluators[e.Name()] = e
}

// Factory to construct OPA evaluator without importing OPA in API layer
// (OPA evaluator registered in API layer when enabled by env)
