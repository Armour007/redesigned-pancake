package api

import (
	"encoding/json"
	"net/http"
	"strings"

	database "github.com/Armour007/aura-backend/internal"
	"github.com/Armour007/aura-backend/internal/policy"
	"github.com/gin-gonic/gin"
)

type NLCompileRequest struct {
	Engine string `json:"engine"` // rego|aurajson
	NL     string `json:"nl"`
}

type NLCompileResponse struct {
	Engine string          `json:"engine"`
	Body   json.RawMessage `json:"body"`
	Notes  []string        `json:"notes,omitempty"`
}

// POST /v2/policy/author/nl-compile
func CompilePolicyFromNL(c *gin.Context) {
	var req NLCompileRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Engine == "" || strings.TrimSpace(req.NL) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "engine and nl required"})
		return
	}
	eng := strings.ToLower(req.Engine)
	nl := strings.ToLower(req.NL)
	var body any
	notes := []string{"heuristic translation; review before use"}
	// Very simple heuristics
	if eng == policy.EngineAuraJSON {
		rules := []map[string]any{}
		if strings.Contains(nl, "never") || strings.Contains(nl, "forbid") || strings.Contains(nl, "deny") {
			rules = append(rules, map[string]any{"id": "deny_1", "effect": "deny", "when": map[string]any{}})
		}
		if strings.Contains(nl, "only if") || strings.Contains(nl, "require") {
			// create a guard that requires approval by default; user can refine conditions
			rules = append(rules, map[string]any{"id": "needs_approval_1", "effect": "require_approval", "hint": "Manual review due to policy"})
		}
		if len(rules) == 0 {
			rules = append(rules, map[string]any{"id": "allow_all", "effect": "allow"})
		}
		body = map[string]any{"rules": rules, "precedence": map[string]any{"deny_overrides": true}}
	} else if eng == policy.EngineRego || eng == "rego" || eng == "opa" {
		// Generate a minimal Rego template. The user can refine input fields.
		tmpl := `package aura.guard

default allow = false

# Set allow to true when conditions are met
allow {
  # TODO: add conditions based on request_context fields
}

# Optional: require approval (out-of-band hint)
require_approval {
  false
}
`
		body = map[string]any{"module": tmpl}
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported engine"})
		return
	}
	b, _ := json.Marshal(body)
	c.JSON(http.StatusOK, NLCompileResponse{Engine: eng, Body: b, Notes: notes})
}

type PolicyTestCase struct {
	Input  json.RawMessage `json:"input"`
	Expect string          `json:"expect"` // allow|deny|needs_approval
}

type PolicyTestRequest struct {
	Engine string           `json:"engine"`
	Body   json.RawMessage  `json:"body"`
	Tests  []PolicyTestCase `json:"tests"`
}

type PolicyTestResult struct {
	Index  int      `json:"index"`
	Status string   `json:"status"`
	Reason string   `json:"reason,omitempty"`
	Hints  []string `json:"hints,omitempty"`
	Pass   bool     `json:"pass"`
}

// POST /v2/policy/tests/run
func RunPolicyTests(c *gin.Context) {
	var req PolicyTestRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Engine == "" || len(req.Body) == 0 || len(req.Tests) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "engine, body, and tests required"})
		return
	}
	engine := evalRegistry[req.Engine]
	if engine == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported engine"})
		return
	}
	cp, err := engine.Compile(req.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	results := make([]PolicyTestResult, 0, len(req.Tests))
	for i, tc := range req.Tests {
		dec, err := engine.Evaluate(cp, tc.Input)
		status := "deny"
		hints := dec.Hints
		reason := dec.Reason
		if err == nil {
			if dec.Allow {
				status = "allow"
			} else if dec.RequireApproval {
				status = "needs_approval"
			}
		} else {
			reason = err.Error()
		}
		pass := strings.EqualFold(status, tc.Expect)
		results = append(results, PolicyTestResult{Index: i, Status: status, Reason: reason, Hints: hints, Pass: pass})
	}
	c.JSON(http.StatusOK, gin.H{"results": results})
}

type PreviewRequest struct {
	Engine        string          `json:"engine"`         // optional when policy_id provided
	Body          json.RawMessage `json:"body"`           // inline policy
	PolicyID      string          `json:"policy_id"`      // optional
	PolicyVersion int             `json:"policy_version"` // optional
	Limit         int             `json:"limit"`          // default 100
}

// POST /v2/policy/preview — evaluate a policy against recent decision traces
func PreviewPolicyAgainstTraces(c *gin.Context) {
	orgID := c.GetString("orgID")
	var req PreviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	limit := req.Limit
	if limit <= 0 || limit > 1000 {
		limit = 100
	}

	var engine policy.Evaluator
	var body json.RawMessage
	if len(req.Body) > 0 && req.Engine != "" {
		engine = evalRegistry[req.Engine]
		body = req.Body
	} else if req.PolicyID != "" {
		// load selected policy version
		var b []byte
		if req.PolicyVersion > 0 {
			_ = database.DB.Get(&b, `SELECT body FROM policy_versions WHERE policy_id=$1 AND version=$2`, req.PolicyID, req.PolicyVersion)
		} else {
			_ = database.DB.Get(&b, `SELECT body FROM policy_versions WHERE policy_id=$1 ORDER BY version DESC LIMIT 1`, req.PolicyID)
		}
		if len(b) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "policy not found"})
			return
		}
		body = json.RawMessage(b)
		var eng string
		_ = database.DB.Get(&eng, `SELECT engine_type FROM policies WHERE id=$1`, req.PolicyID)
		engine = evalRegistry[eng]
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "provide engine+body or policy_id"})
		return
	}
	if engine == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported engine"})
		return
	}
	cp, err := engine.Compile(body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// fetch recent traces
	type row struct {
		Trace json.RawMessage `db:"trace"`
		Allow bool            `db:"allow"`
	}
	rows := []row{}
	_ = database.DB.Select(&rows, `SELECT trace, allow FROM decision_traces WHERE org_id=$1 ORDER BY created_at DESC LIMIT $2`, orgID, limit)

	var allowCnt, denyCnt, needCnt, changed int
	samples := []map[string]any{}
	for _, r := range rows {
		var in json.RawMessage
		// Use stored input if present in trace; fallback to empty
		var tr struct {
			InputContext json.RawMessage `json:"input_context"`
		}
		_ = json.Unmarshal(r.Trace, &tr)
		in = tr.InputContext
		dec, err := engine.Evaluate(cp, in)
		if err != nil {
			continue
		}
		status := "deny"
		if dec.Allow {
			status = "allow"
		} else if dec.RequireApproval {
			status = "needs_approval"
		}
		switch status {
		case "allow":
			allowCnt++
		case "deny":
			denyCnt++
		case "needs_approval":
			needCnt++
		}
		// compare to original allow
		orig := "deny"
		if r.Allow {
			orig = "allow"
		}
		if orig != status {
			changed++
		}
		if len(samples) < 5 {
			samples = append(samples, map[string]any{"original": orig, "now": status, "reason": dec.Reason})
		}
	}
	c.JSON(http.StatusOK, gin.H{"summary": map[string]any{"allow": allowCnt, "deny": denyCnt, "needs_approval": needCnt, "changed": changed}, "samples": samples})
}
