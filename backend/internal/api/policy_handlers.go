package api

import (
	"context"
	"encoding/json"
	"net/http"

	db "github.com/Armour007/aura-backend/internal"
	"github.com/Armour007/aura-backend/internal/audit"
	"github.com/Armour007/aura-backend/internal/policy"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Evaluator registry (process-wide)
var evalRegistry = func() map[string]policy.Evaluator {
	m := map[string]policy.Evaluator{}
	m[policy.EngineAuraJSON] = &policy.AuraJSONEvaluator{}
	return m
}()

// CreatePolicy - prototype in-memory policy
type createPolicyReq struct {
	Name       string `json:"name" binding:"required"`
	EngineType string `json:"engine_type" binding:"required"`
}

func CreatePolicy(c *gin.Context) {
	orgID := c.Param("orgId")
	var req createPolicyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	o := uuid.MustParse(orgID)
	createdBy := c.GetString("userID")
	var uid *uuid.UUID
	if createdBy != "" {
		u := uuid.MustParse(createdBy)
		uid = &u
	}
	p, err := policy.CreatePolicy(context.Background(), o, req.Name, req.EngineType, uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, p)
}

type addVersionReq struct {
	Body         any    `json:"body" binding:"required"`
	ChangeTicket string `json:"change_ticket"`
}

func AddPolicyVersion(c *gin.Context) {
	pid, err := uuid.Parse(c.Param("policyId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad id"})
		return
	}
	var req addVersionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	b, _ := json.Marshal(req.Body)
	createdBy := c.GetString("userID")
	var uid *uuid.UUID
	if createdBy != "" {
		u := uuid.MustParse(createdBy)
		uid = &u
	}
	pv, err := policy.AddVersion(context.Background(), pid, b, uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if req.ChangeTicket != "" {
		_, _ = db.DB.Exec(`UPDATE policy_versions SET change_ticket=$1 WHERE policy_id=$2 AND version=$3`, req.ChangeTicket, pid, pv.Version)
	}
	// optional: pre-compile into cache to speed up first request
	// pick evaluator
	var pol db.Policy
	if err := db.DB.Get(&pol, `SELECT id, org_id, name, engine_type, created_by_user_id, created_at FROM policies WHERE id=$1`, pid); err == nil {
		if e := evalRegistry[pol.EngineType]; e != nil {
			if cp, err := e.Compile(b); err == nil {
				policy.PutCompiled(pid, pv.Version, cp)
			}
		}
	}
	// Invalidate other compiled versions to avoid stale behavior after new version is introduced (still draft)
	policy.DeleteCompiled(pid, 0)
	// publish policy invalidation across mesh
	PublishPolicyInvalidate(c.Request.Context(), pid.String())
	c.JSON(http.StatusCreated, pv)
}

type assignReq struct {
	ScopeType string `json:"scope_type"`
	ScopeID   string `json:"scope_id"`
}

func AssignPolicy(c *gin.Context) {
	pid, err := uuid.Parse(c.Param("policyId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad id"})
		return
	}
	var req assignReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := policy.Assign(context.Background(), pid, req.ScopeType, req.ScopeID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// Invalidate cache to ensure next verify fetches active version fresh
	policy.DeleteCompiled(pid, 0)
	// publish policy invalidation across mesh
	PublishPolicyInvalidate(c.Request.Context(), pid.String())
	// audit assignment
	_ = audit.Append(c.Request.Context(), uuid.MustParse(c.Param("orgId")), "policy_assigned", map[string]any{"policy_id": pid, "scope_type": req.ScopeType, "scope_id": req.ScopeID}, nil, nil)
	c.Status(http.StatusNoContent)
}
