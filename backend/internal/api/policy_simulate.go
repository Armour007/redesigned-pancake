package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	polrepo "github.com/Armour007/aura-backend/internal/policy"
)

type simulateReq struct {
	RequestContext json.RawMessage `json:"request_context"`
}

type simulateResp struct {
	Allow  bool   `json:"allow"`
	Reason string `json:"reason"`
}

// POST /organizations/:orgId/policies/:policyId/versions/:version/simulate
func SimulatePolicyVersion(c *gin.Context) {
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
	var req simulateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

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
	dec, err := e.Evaluate(comp, req.RequestContext)
	if err != nil {
		c.JSON(http.StatusOK, simulateResp{Allow: false, Reason: err.Error()})
		return
	}
	c.JSON(http.StatusOK, simulateResp{Allow: dec.Allow, Reason: dec.Reason})
}
