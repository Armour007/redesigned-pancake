package api

import (
	"encoding/json"
	"net/http"
	"time"

	database "github.com/Armour007/aura-backend/internal"
	"github.com/gin-gonic/gin"
)

// GetEventLogs handles requests to retrieve event logs
func GetEventLogs(c *gin.Context) {
	orgId := c.Param("orgId")
	agentId := c.Query("agentId")
	limit := 100

	type EventLogResponse struct {
		ID         int64     `json:"id"`
		Timestamp  time.Time `json:"timestamp"`
		EventType  string    `json:"event_type"`
		Decision   string    `json:"decision"`
		Reason     *string   `json:"decision_reason,omitempty"`
		Details    any       `json:"request_details"`
		AgentID    *string   `json:"agent_id,omitempty"`
		IP         *string   `json:"client_ip_address,omitempty"`
		ApiKeyPref *string   `json:"api_key_prefix_used,omitempty"`
	}

	// Build query
	query := `SELECT id, timestamp, event_type, decision, decision_reason, request_details, agent_id, client_ip_address, api_key_prefix_used
			  FROM event_logs WHERE organization_id = $1`
	args := []any{orgId}
	if agentId != "" {
		query += " AND agent_id = $2"
		args = append(args, agentId)
	}
	query += " ORDER BY timestamp DESC LIMIT $3"
	args = append(args, limit)

	// Since request_details is jsonb, scan into raw bytes
	rows, err := database.DB.Queryx(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch event logs"})
		return
	}
	defer rows.Close()

	logs := []EventLogResponse{}
	for rows.Next() {
		var (
			id      int64
			ts      time.Time
			et, dec string
			reason  *string
			details []byte
			agentID *string
			ip      *string
			pref    *string
		)
		if err := rows.Scan(&id, &ts, &et, &dec, &reason, &details, &agentID, &ip, &pref); err != nil {
			continue
		}
		var anyJSON any
		_ = json.Unmarshal(details, &anyJSON)
		logs = append(logs, EventLogResponse{ID: id, Timestamp: ts, EventType: et, Decision: dec, Reason: reason, Details: anyJSON, AgentID: agentID, IP: ip, ApiKeyPref: pref})
	}
	c.JSON(http.StatusOK, logs)
}
