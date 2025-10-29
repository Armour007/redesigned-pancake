package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// GET /v2/signals/risk?org_id=&agent_id=
func GetRiskSignals(c *gin.Context) {
	orgID := c.Query("org_id")
	if orgID == "" {
		orgID = c.GetString("orgID")
	}
	agentID := c.Query("agent_id")
	if agentID == "" {
		agentID = c.GetString("agentID")
	}
	s := getRiskTracker().Get(orgID, agentID, time.Now())
	c.JSON(http.StatusOK, gin.H{"org_id": orgID, "agent_id": agentID, "risk": gin.H{"score": s.Score, "flags": s.Flags}})
}

type riskAlertRequest struct {
	Flag       string `json:"flag"`
	TTLSeconds int    `json:"ttl_seconds"`
	AgentID    string `json:"agent_id"`
}

// POST /v2/signals/risk/alerts
// Body: {"flag":"external_alert","ttl_seconds":120,"agent_id":"<optional>"}
func RaiseRiskAlert(c *gin.Context) {
	orgID := c.GetString("orgID")
	var req riskAlertRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Flag == "" || req.TTLSeconds <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}
	agentID := req.AgentID
	if agentID == "" {
		agentID = c.GetString("agentID")
	}
	exp := time.Now().Add(time.Duration(req.TTLSeconds) * time.Second)
	getRiskTracker().RaiseAlert(orgID, agentID, req.Flag, exp)
	c.JSON(http.StatusOK, gin.H{"status": "ok", "org_id": orgID, "agent_id": agentID, "flag": req.Flag, "expires_at": exp.UTC().Format(time.RFC3339)})
}

// DELETE /v2/signals/risk/alerts
// Body: {"flag":"external_alert","agent_id":"<optional>"}
func ClearRiskAlert(c *gin.Context) {
	orgID := c.GetString("orgID")
	var req riskAlertRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Flag == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}
	agentID := req.AgentID
	if agentID == "" {
		agentID = c.GetString("agentID")
	}
	getRiskTracker().ClearAlert(orgID, agentID, req.Flag)
	c.Status(http.StatusNoContent)
}
