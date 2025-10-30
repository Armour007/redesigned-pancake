package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	db "github.com/Armour007/aura-backend/internal"
	"github.com/gin-gonic/gin"
)

type revokeReq struct {
	JTI    string `json:"jti" binding:"required"`
	Reason string `json:"reason,omitempty"`
}

type revocationsResp struct {
	Items []struct {
		JTI       string    `json:"jti"`
		RevokedAt time.Time `json:"revoked_at"`
		Reason    string    `json:"reason,omitempty"`
	} `json:"items"`
}

// POST /organizations/:orgId/trust-tokens/revocations
func RevokeTrustToken(c *gin.Context) {
	orgID := c.Param("orgId")
	var req revokeReq
	if err := c.ShouldBindJSON(&req); err != nil || req.JTI == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "jti required"})
		return
	}
	_, err := db.DB.Exec(`INSERT INTO trust_token_revocations (org_id, jti, reason, revoked_at) VALUES ($1,$2,$3,NOW()) ON CONFLICT (org_id, jti) DO UPDATE SET reason=excluded.reason, revoked_at=excluded.revoked_at`, orgID, req.JTI, req.Reason)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// GET /organizations/:orgId/trust-tokens/revocations
// Supports ETag via If-None-Match on latest revoked_at snapshot
func GetRevocations(c *gin.Context) {
	orgID := c.Param("orgId")
	var latest time.Time
	_ = db.DB.Get(&latest, `SELECT COALESCE(MAX(revoked_at), to_timestamp(0)) FROM trust_token_revocations WHERE org_id=$1`, orgID)
	etag := makeETag(orgID, latest)
	if inm := c.GetHeader("If-None-Match"); inm != "" && inm == etag {
		c.Status(http.StatusNotModified)
		return
	}
	rows, err := db.DB.Queryx(`SELECT jti, revoked_at, COALESCE(reason,'') FROM trust_token_revocations WHERE org_id=$1 ORDER BY revoked_at DESC LIMIT 1000`, orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	out := revocationsResp{Items: []struct {
		JTI       string    `json:"jti"`
		RevokedAt time.Time `json:"revoked_at"`
		Reason    string    `json:"reason,omitempty"`
	}{}}
	for rows.Next() {
		var jti, reason string
		var ts time.Time
		if err := rows.Scan(&jti, &ts, &reason); err == nil {
			out.Items = append(out.Items, struct {
				JTI       string    `json:"jti"`
				RevokedAt time.Time `json:"revoked_at"`
				Reason    string    `json:"reason,omitempty"`
			}{JTI: jti, RevokedAt: ts, Reason: reason})
		}
	}
	b, _ := json.Marshal(out)
	c.Header("ETag", etag)
	c.Data(http.StatusOK, "application/json", b)
}

func makeETag(orgID string, t time.Time) string {
	h := sha256.Sum256([]byte(orgID + "|" + t.UTC().Format(time.RFC3339Nano)))
	return `W/"` + hex.EncodeToString(h[:8]) + `"`
}
