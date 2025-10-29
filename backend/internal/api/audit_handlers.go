package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	database "github.com/Armour007/aura-backend/internal"
	"github.com/Armour007/aura-backend/internal/audit"
)

// GET /v2/audit/ledger?limit=200
func GetAuditLedger(c *gin.Context) {
	orgID := c.GetString("orgID")
	limit := 200
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 5000 {
			limit = n
		}
	}
	type row struct {
		Seq        int64      `db:"seq" json:"seq"`
		Time       string     `db:"event_time" json:"event_time"`
		Type       string     `db:"event_type" json:"event_type"`
		ActorUser  *uuid.UUID `db:"actor_user_id" json:"actor_user_id"`
		ActorAgent *uuid.UUID `db:"actor_agent_id" json:"actor_agent_id"`
		Payload    any        `db:"payload" json:"payload"`
		ThisHash   string     `db:"this_hash" json:"this_hash"`
	}
	rows := []row{}
	_ = database.DB.Select(&rows, `SELECT seq, event_time::text, event_type, actor_user_id, actor_agent_id, payload, this_hash FROM audit_ledger WHERE org_id=$1 ORDER BY seq DESC LIMIT $2`, orgID, limit)
	c.JSON(http.StatusOK, gin.H{"org_id": orgID, "items": rows, "count": len(rows)})
}

// GET /v2/audit/verify?limit=10000
func VerifyAuditChain(c *gin.Context) {
	orgID := c.GetString("orgID")
	limit := 10000
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	oid := uuid.MustParse(orgID)
	breakAt, err := audit.Verify(c.Request.Context(), oid, limit)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"ok": false, "break_at": breakAt, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "break_at": 0})
}

type setAnchorReq struct {
	Date        string `json:"date"`         // YYYY-MM-DD (optional: defaults to today UTC)
	RootHash    string `json:"root_hash"`    // hex string
	ExternalRef string `json:"external_ref"` // optional URI or note
}

// POST /v2/audit/anchor
func SetAuditAnchor(c *gin.Context) {
	orgID := c.GetString("orgID")
	var req setAnchorReq
	if err := c.ShouldBindJSON(&req); err != nil || req.RootHash == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "root_hash required"})
		return
	}
	d := req.Date
	if d == "" {
		d = time.Now().UTC().Format("2006-01-02")
	}
	if _, err := database.DB.Exec(`INSERT INTO audit_anchors(org_id, anchor_date, root_hash, external_ref) VALUES ($1,$2,$3,$4)
		ON CONFLICT (org_id, anchor_date) DO UPDATE SET root_hash=EXCLUDED.root_hash, external_ref=EXCLUDED.external_ref`, orgID, d, req.RootHash, req.ExternalRef); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// ledger record
	_ = audit.Append(c.Request.Context(), uuid.MustParse(orgID), "audit_anchor_set", gin.H{"date": d, "root_hash": req.RootHash, "external_ref": req.ExternalRef}, nil, nil)
	c.JSON(http.StatusOK, gin.H{"org_id": orgID, "date": d, "root_hash": req.RootHash, "external_ref": req.ExternalRef})
}

// GET /v2/audit/anchor?date=YYYY-MM-DD
// If date param is omitted, returns the latest anchor for the org
func GetAuditAnchor(c *gin.Context) {
	orgID := c.GetString("orgID")
	date := c.Query("date")
	type anchorRow struct {
		Date        string `db:"anchor_date" json:"date"`
		RootHash    string `db:"root_hash" json:"root_hash"`
		ExternalRef string `db:"external_ref" json:"external_ref"`
	}
	var row anchorRow
	var err error
	if date == "" {
		err = database.DB.Get(&row, `SELECT anchor_date::text, root_hash, external_ref FROM audit_anchors WHERE org_id=$1 ORDER BY anchor_date DESC LIMIT 1`, orgID)
	} else {
		err = database.DB.Get(&row, `SELECT anchor_date::text, root_hash, external_ref FROM audit_anchors WHERE org_id=$1 AND anchor_date=$2`, orgID, date)
	}
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "anchor not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"org_id": orgID, "date": row.Date, "root_hash": row.RootHash, "external_ref": row.ExternalRef})
}
