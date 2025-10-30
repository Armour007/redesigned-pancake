package api

import (
	"encoding/json"
	"net/http"

	database "github.com/Armour007/aura-backend/internal"
	"github.com/Armour007/aura-backend/internal/policy"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// InlineGuardRequest evaluates either an inline policy body or the currently assigned policy for the org
// and returns allow|deny|needs_approval with mitigation hints. Intended for tool/function-call gating.
type InlineGuardRequest struct {
	AgentID        uuid.UUID       `json:"agent_id"`
	Action         string          `json:"action,omitempty"`
	Resource       string          `json:"resource,omitempty"`
	RequestContext json.RawMessage `json:"request_context"`
	// Optional inline policy: if provided, evaluate this instead of assigned policy
	Policy *struct {
		Engine string          `json:"engine"`
		Body   json.RawMessage `json:"body"`
	} `json:"policy,omitempty"`
}

type InlineGuardResponse struct {
	Status  string   `json:"status"` // allow|deny|needs_approval
	Reason  string   `json:"reason,omitempty"`
	Hints   []string `json:"hints,omitempty"`
	TraceID string   `json:"trace_id,omitempty"`
}

// POST /v2/guard — inline policy guard for agent actions
func InlineGuard(c *gin.Context) {
	orgID := c.GetString("orgID")
	var req InlineGuardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Decide which engine and body to use
	var engine policy.Evaluator
	var body json.RawMessage
	if req.Policy != nil && len(req.Policy.Body) > 0 {
		engine = evalRegistry[req.Policy.Engine]
		body = req.Policy.Body
		if engine == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported engine"})
			return
		}
	} else {
		// Load active assignment and current version similar to VerifyV2
		assigns, err := policy.GetActiveAssignmentsForOrg(c.Request.Context(), uuid.MustParse(orgID))
		if err != nil || len(assigns) == 0 {
			c.JSON(http.StatusOK, InlineGuardResponse{Status: "deny", Reason: "No active policy assignment"})
			return
		}
		p := assigns[0].Policy
		v := assigns[0].Version
		engine = evalRegistry[p.EngineType]
		if engine == nil {
			c.JSON(http.StatusOK, InlineGuardResponse{Status: "deny", Reason: "Unsupported engine"})
			return
		}
		body = v.Body
	}

	// Compile and evaluate
	cp, err := engine.Compile(body)
	if err != nil {
		c.JSON(http.StatusOK, InlineGuardResponse{Status: "deny", Reason: err.Error()})
		return
	}
	dec, err := engine.Evaluate(cp, req.RequestContext)
	if err != nil {
		c.JSON(http.StatusOK, InlineGuardResponse{Status: "deny", Reason: err.Error()})
		return
	}

	// Persist a lightweight decision trace for analytics (optional)
	go func() {
		var traceJSON json.RawMessage
		if dec.Trace != nil {
			if b, err := json.Marshal(dec.Trace); err == nil {
				traceJSON = b
			}
		}
		var agentPtr *uuid.UUID
		if req.AgentID != uuid.Nil {
			agentPtr = &req.AgentID
		}
		_, _ = database.DB.Exec(`INSERT INTO decision_traces (org_id, trace_id, agent_id, allow, reason, trace) VALUES ($1,$2,$3,$4,$5,$6)`,
			orgID, dec.TraceID, agentPtr, dec.Allow, dec.Reason, traceJSON)
	}()

	// Map decision to guard status
	status := "deny"
	if dec.Allow {
		status = "allow"
	} else if dec.RequireApproval {
		status = "needs_approval"
	}
	c.JSON(http.StatusOK, InlineGuardResponse{Status: status, Reason: dec.Reason, Hints: dec.Hints, TraceID: dec.TraceID})
}
