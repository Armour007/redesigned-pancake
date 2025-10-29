package api

import (
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/Armour007/aura-backend/internal/audit"
	polrepo "github.com/Armour007/aura-backend/internal/policy"
)

// POST /organizations/:orgId/policies/:policyId/versions/:version/approve
func ApprovePolicyVersion(c *gin.Context) {
	// orgID reserved for future scoping checks
	_ = c.Param("orgId")
	pid, err := uuid.Parse(c.Param("policyId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad policy id"})
		return
	}
	var version int
	if _, err := fmt.Sscanf(c.Param("version"), "%d", &version); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad version"})
		return
	}
	// Optional: get approver from auth context
	var approver *uuid.UUID
	if uid := c.GetString("userID"); uid != "" {
		u := uuid.MustParse(uid)
		approver = &u
	}
	// Basic compile validation before approval to ensure policy body is valid
	p, err := polrepo.GetPolicy(c.Request.Context(), pid)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "policy not found"})
		return
	}
	v, err := polrepo.GetVersion(c.Request.Context(), pid, version)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "policy version not found"})
		return
	}
	e := evalRegistry[p.EngineType]
	if e == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported engine"})
		return
	}
	if _, err := e.Compile(v.Body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Multi-approver: record approval and only mark approved after threshold reached
	approvalsRequired := 2
	if v := os.Getenv("AURA_POLICY_APPROVALS_REQUIRED"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			approvalsRequired = n
		}
	}
	if err := polrepo.RecordApproval(c.Request.Context(), pid, version, approver); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	count, err := polrepo.CountApprovals(c.Request.Context(), pid, version)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if count >= approvalsRequired {
		if err := polrepo.ApproveVersion(c.Request.Context(), pid, version, approver); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	// Invalidate compiled policy cache for this policy
	polrepo.DeleteCompiled(pid, 0)
	// publish policy invalidation across mesh
	PublishPolicyInvalidate(c.Request.Context(), pid.String())
	// Audit ledger entry
	_ = audit.Append(c.Request.Context(), uuid.Nil, "policy_version_approval", map[string]any{"policy_id": pid, "version": version, "approvals": count}, approver, nil)
	c.Status(http.StatusNoContent)
}
