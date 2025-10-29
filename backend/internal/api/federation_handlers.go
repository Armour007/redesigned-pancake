package api

import (
	"encoding/json"
	"net/http"

	database "github.com/Armour007/aura-backend/internal"
	"github.com/Armour007/aura-backend/internal/audit"
	"github.com/Armour007/aura-backend/internal/rel"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type createContractReq struct {
	CounterpartyOrgID string `json:"counterparty_org_id"`
	Scope             any    `json:"scope"`
}

// POST /v2/federation/contracts
func CreateFederationContract(c *gin.Context) {
	orgID := c.GetString("orgID")
	var req createContractReq
	if err := c.ShouldBindJSON(&req); err != nil || req.CounterpartyOrgID == "" || req.Scope == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}
	b, _ := json.Marshal(req.Scope)
	row := database.DB.QueryRowx(`INSERT INTO federation_contracts(org_id, counterparty_org_id, scope) VALUES ($1,$2,$3) RETURNING id, created_at::text`, orgID, req.CounterpartyOrgID, b)
	var id uuid.UUID
	var createdAt string
	if err := row.Scan(&id, &createdAt); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	_ = audit.Append(c.Request.Context(), uuid.MustParse(orgID), "federation_contract_created", gin.H{"id": id, "counterparty_org_id": req.CounterpartyOrgID, "scope": req.Scope}, nil, nil)
	c.JSON(http.StatusCreated, gin.H{"id": id, "org_id": orgID, "counterparty_org_id": req.CounterpartyOrgID, "scope": req.Scope, "created_at": createdAt})
}

// GET /v2/federation/contracts
func ListFederationContracts(c *gin.Context) {
	orgID := c.GetString("orgID")
	rows, err := database.DB.Queryx(`SELECT id::text, org_id::text, counterparty_org_id::text, scope, active, created_at::text FROM federation_contracts WHERE org_id=$1 ORDER BY created_at DESC`, orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id, org, cp, created string
		var active bool
		var scope any
		if err := rows.Scan(&id, &org, &cp, &scope, &active, &created); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		items = append(items, gin.H{"id": id, "org_id": org, "counterparty_org_id": cp, "scope": scope, "active": active, "created_at": created})
	}
	c.JSON(http.StatusOK, gin.H{"org_id": orgID, "contracts": items})
}

type boundaryEventReq struct {
	CounterpartyOrgID string `json:"counterparty_org_id"`
	Event             any    `json:"event"`
}

// POST /v2/federation/events
func RecordFederationBoundaryEvent(c *gin.Context) {
	orgID := c.GetString("orgID")
	var req boundaryEventReq
	if err := c.ShouldBindJSON(&req); err != nil || req.CounterpartyOrgID == "" || req.Event == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}
	_ = audit.Append(c.Request.Context(), uuid.MustParse(orgID), "federation_boundary_crossing", gin.H{"counterparty_org_id": req.CounterpartyOrgID, "event": req.Event}, nil, nil)
	c.Status(http.StatusNoContent)
}

type createDelegationReq struct {
	CounterpartyOrgID string `json:"counterparty_org_id"`
	AgentID           string `json:"agent_id"`
	Relation          string `json:"relation"`
	TargetOrgID       string `json:"target_org_id"`
}

// POST /v2/federation/delegations
// Creates a cross-org delegation tuple: agent (remote) can_act_for org (target)
func CreateFederationDelegation(c *gin.Context) {
	callerOrg := c.GetString("orgID")
	var req createDelegationReq
	if err := c.ShouldBindJSON(&req); err != nil || req.CounterpartyOrgID == "" || req.AgentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "counterparty_org_id and agent_id required"})
		return
	}
	relation := req.Relation
	if relation == "" {
		relation = "can_act_for"
	}
	targetOrg := req.TargetOrgID
	if targetOrg == "" {
		targetOrg = callerOrg
	}
	t := rel.Tuple{ObjectType: "org", ObjectID: targetOrg, Relation: relation, SubjectType: "agent", SubjectID: req.AgentID}
	if err := relDB.Upsert(c.Request.Context(), []rel.Tuple{t}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	relStore.Upsert([]rel.Tuple{t})
	ClearGraphCache()
	PublishGraphInvalidate(c.Request.Context())
	_ = audit.Append(c.Request.Context(), uuid.MustParse(targetOrg), "federation_delegation_created", gin.H{"agent_id": req.AgentID, "from_org_id": req.CounterpartyOrgID, "relation": relation}, nil, nil)
	c.Status(http.StatusNoContent)
}
