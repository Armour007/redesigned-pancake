package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	database "github.com/Armour007/aura-backend/internal"
	"github.com/gin-gonic/gin"
)

// Minimal Slack interactive payload shape we care about
type slackAction struct {
	ActionID string `json:"action_id"`
	Value    string `json:"value"`
}
type slackUser struct {
	ID string `json:"id"`
}
type slackPayload struct {
	Type            string        `json:"type"`
	Actions         []slackAction `json:"actions"`
	PrivateMetadata string        `json:"private_metadata"`
	User            slackUser     `json:"user"`
	Token           string        `json:"token"`
}

// POST /v2/approvals/slack (public) — stub handler for Slack interactive actions
// Expect either application/x-www-form-urlencoded with a `payload` JSON param
// or raw JSON body. Value convention: "approve:{traceId}" or "deny:{traceId}".
func SlackApprovalWebhook(c *gin.Context) {
	var pl slackPayload
	payload := c.PostForm("payload")
	if payload == "" {
		// try raw body
		if err := c.ShouldBindJSON(&pl); err != nil {
			c.Status(http.StatusOK) // ignore; stub endpoint
			return
		}
	} else {
		_ = json.Unmarshal([]byte(payload), &pl)
	}
	// Extract intent and traceId
	var val string
	if len(pl.Actions) > 0 {
		val = pl.Actions[0].Value
	}
	if val == "" && pl.PrivateMetadata != "" {
		// allow formats like "approve:trace:abc" or direct trace id with prefix
		val = pl.PrivateMetadata
	}
	if val == "" {
		c.Status(http.StatusOK)
		return
	}
	v := strings.ToLower(val)
	status := ""
	traceID := ""
	if strings.HasPrefix(v, "approve:") {
		status = "approved"
		traceID = strings.TrimPrefix(val, "approve:")
	} else if strings.HasPrefix(v, "deny:") || strings.HasPrefix(v, "reject:") {
		status = "denied"
		if strings.HasPrefix(v, "deny:") {
			traceID = strings.TrimPrefix(val, "deny:")
		} else {
			traceID = strings.TrimPrefix(val, "reject:")
		}
	} else if strings.HasPrefix(v, "trace:") {
		// default approve if no explicit verb
		status = "approved"
		traceID = strings.TrimPrefix(val, "trace:")
	}
	traceID = strings.TrimSpace(traceID)
	if status == "" || traceID == "" {
		c.Status(http.StatusOK)
		return
	}

	// Upsert into runtime_approvals
	_, _ = database.DB.Exec(`
        INSERT INTO runtime_approvals(trace_id, status, decided_by, decided_at)
        VALUES ($1, $2, $3, $4)
        ON CONFLICT (trace_id) DO UPDATE SET status=EXCLUDED.status, decided_by=EXCLUDED.decided_by, decided_at=EXCLUDED.decided_at
    `, traceID, status, pl.User.ID, time.Now())

	// Respond with a simple message (Slack will show it as response to interaction)
	c.JSON(http.StatusOK, gin.H{"text": "Recorded " + status + " for trace " + traceID})
}

// GET /v2/approvals/:traceId — check runtime approval status (pending|approved|denied)
func GetApprovalStatus(c *gin.Context) {
	traceID := strings.TrimSpace(c.Param("traceId"))
	if traceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing traceId"})
		return
	}
	var status string
	err := database.DB.Get(&status, `SELECT status FROM runtime_approvals WHERE trace_id=$1`, traceID)
	if err != nil || status == "" {
		// Insert pending if not exists to make idempotent
		_, _ = database.DB.Exec(`INSERT INTO runtime_approvals(trace_id, status) VALUES ($1,'pending') ON CONFLICT (trace_id) DO NOTHING`, traceID)
		status = "pending"
	}
	c.JSON(http.StatusOK, gin.H{"trace_id": traceID, "status": status})
}
