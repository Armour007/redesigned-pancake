package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/Armour007/aura-backend/internal/audit"
	polrepo "github.com/Armour007/aura-backend/internal/policy"
)

type activateReq struct {
	// Optional simulation context that must allow=true to proceed
	RequestContext json.RawMessage `json:"request_context"`
}

// POST /organizations/:orgId/policies/:policyId/versions/:version/activate
// Enforces safety gate: version must pass simulation (Allow==true) with provided context before activation
func ActivatePolicyVersion(c *gin.Context) {
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
	var req activateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Load policy and version
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
	comp, err := e.Compile(v.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Require simulation to succeed before activation
	dec, err := e.Evaluate(comp, req.RequestContext)
	if err != nil || !dec.Allow {
		reason := "evaluation error"
		if err == nil {
			reason = dec.Reason
		} else {
			reason = err.Error()
		}
		c.JSON(http.StatusUnprocessableEntity, gin.H{"allow": false, "reason": reason})
		return
	}

	if err := polrepo.ActivateVersion(c.Request.Context(), pid, version); err != nil {
		// If not approved or other conflict, return 409 for clarity
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	// Warm cache after activation
	polrepo.PutCompiled(pid, version, comp)
	// Invalidate other compiled versions to avoid stale active lookup
	polrepo.DeleteCompiled(pid, 0)
	// publish policy invalidation across mesh
	PublishPolicyInvalidate(c.Request.Context(), pid.String())
	_ = audit.Append(c.Request.Context(), uuid.Nil, "policy_version_activate", map[string]any{"policy_id": pid, "version": version}, nil, nil)
	c.Status(http.StatusNoContent)
}
