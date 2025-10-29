package api

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"os"
	"strings"
	"time"

	database "github.com/Armour007/aura-backend/internal"
	"github.com/Armour007/aura-backend/internal/audit"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type createTrustKeyReq struct {
	Kid    string `json:"kid,omitempty"`
	Active bool   `json:"active,omitempty"`
}

type rotateTrustKeyReq struct {
	Kid          string `json:"kid,omitempty"`
	Grace        string `json:"grace,omitempty"`         // duration string, e.g., "15m", "2h"
	GraceSeconds int    `json:"grace_seconds,omitempty"` // alternative numeric seconds
}

// POST /organizations/:orgId/trust-keys
func CreateTrustKey(c *gin.Context) {
	orgID := c.Param("orgId")
	if _, err := uuid.Parse(orgID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid org id"})
		return
	}
	var req createTrustKeyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// generate ed25519 keypair
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		c.JSON(500, gin.H{"error": "keygen failed"})
		return
	}
	enc := base64.RawURLEncoding.EncodeToString(priv)
	kid := strings.TrimSpace(req.Kid)
	if kid == "" {
		sum := sha256.Sum256(pub)
		kid = base64.RawURLEncoding.EncodeToString(sum[:8])
	}
	active := req.Active
	row := database.DB.QueryRowx(`INSERT INTO trust_keys(org_id, alg, ed25519_private_key_base64, kid, active) VALUES ($1,'EdDSA',$2,$3,$4) RETURNING id::text, created_at::text`, orgID, enc, kid, active)
	var id, created string
	if err := row.Scan(&id, &created); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	_ = audit.Append(c.Request.Context(), uuid.MustParse(orgID), "trust_key_created", gin.H{"id": id, "kid": kid, "active": active}, nil, nil)
	c.JSON(http.StatusCreated, gin.H{"id": id, "org_id": orgID, "kid": kid, "active": active, "alg": "EdDSA", "created_at": created})
}

// GET /organizations/:orgId/trust-keys
func ListTrustKeys(c *gin.Context) {
	orgID := c.Param("orgId")
	if _, err := uuid.Parse(orgID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid org id"})
		return
	}
	rows, err := database.DB.Queryx(`SELECT id::text, kid, alg, active, created_at::text, COALESCE(deactivate_after::text,'') FROM trust_keys WHERE org_id=$1 ORDER BY created_at DESC LIMIT 100`, orgID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id, kid, alg, created, deact string
		var active bool
		if err := rows.Scan(&id, &kid, &alg, &active, &created, &deact); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		m := gin.H{"id": id, "kid": kid, "alg": alg, "active": active, "created_at": created}
		if deact != "" {
			m["deactivate_after"] = deact
		}
		items = append(items, m)
	}
	c.JSON(200, gin.H{"keys": items})
}

// POST /organizations/:orgId/trust-keys/:keyId/activate?exclusive=1
func ActivateTrustKey(c *gin.Context) {
	orgID := c.Param("orgId")
	keyID := c.Param("keyId")
	if _, err := uuid.Parse(orgID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid org id"})
		return
	}
	kid := ""
	exclusive := c.Query("exclusive") == "1"
	// ensure key exists and belongs to org
	if err := database.DB.Get(&kid, `SELECT COALESCE(kid,'') FROM trust_keys WHERE id=$1 AND org_id=$2`, keyID, orgID); err != nil {
		c.JSON(404, gin.H{"error": "key not found"})
		return
	}
	if exclusive {
		// deactivate others with optional grace; allow request override via query params
		var d time.Duration
		if gs := strings.TrimSpace(c.Query("grace")); gs != "" {
			if dd, err := time.ParseDuration(gs); err == nil && dd > 0 {
				d = dd
			}
		} else if gsec := strings.TrimSpace(c.Query("grace_seconds")); gsec != "" {
			if n, err := time.ParseDuration(gsec + "s"); err == nil && n > 0 {
				d = n
			}
		} else if gs := strings.TrimSpace(os.Getenv("AURA_TRUST_KEY_GRACE")); gs != "" {
			if dd, err := time.ParseDuration(gs); err == nil && dd > 0 {
				d = dd
			}
		}
		if d > 0 {
			deadline := time.Now().Add(d)
			_, _ = database.DB.Exec(`UPDATE trust_keys SET deactivate_after=$1 WHERE org_id=$2 AND id<>$3 AND active=true`, deadline, orgID, keyID)
		} else {
			_, _ = database.DB.Exec(`UPDATE trust_keys SET active=false WHERE org_id=$1 AND id<>$2`, orgID, keyID)
		}
	}
	res, err := database.DB.Exec(`UPDATE trust_keys SET active=true WHERE id=$1 AND org_id=$2`, keyID, orgID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		c.JSON(404, gin.H{"error": "key not found"})
		return
	}
	_ = audit.Append(c.Request.Context(), uuid.MustParse(orgID), "trust_key_activated", gin.H{"id": keyID, "kid": kid, "exclusive": exclusive}, nil, nil)
	c.JSON(200, gin.H{"id": keyID, "kid": kid, "active": true, "exclusive": exclusive})
}

// POST /organizations/:orgId/trust-keys/:keyId/deactivate
func DeactivateTrustKey(c *gin.Context) {
	orgID := c.Param("orgId")
	keyID := c.Param("keyId")
	if _, err := uuid.Parse(orgID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid org id"})
		return
	}
	kid := ""
	if err := database.DB.Get(&kid, `SELECT COALESCE(kid,'') FROM trust_keys WHERE id=$1 AND org_id=$2`, keyID, orgID); err != nil {
		c.JSON(404, gin.H{"error": "key not found"})
		return
	}
	res, err := database.DB.Exec(`UPDATE trust_keys SET active=false WHERE id=$1 AND org_id=$2`, keyID, orgID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		c.JSON(404, gin.H{"error": "key not found"})
		return
	}
	_ = audit.Append(c.Request.Context(), uuid.MustParse(orgID), "trust_key_deactivated", gin.H{"id": keyID, "kid": kid}, nil, nil)
	c.JSON(200, gin.H{"id": keyID, "kid": kid, "active": false})
}

// optional: simple rotate helper
// POST /organizations/:orgId/trust-keys/rotate  (creates a new active key)
func RotateTrustKey(c *gin.Context) {
	// Parse rotate request
	var req rotateTrustKeyReq
	_ = c.ShouldBindJSON(&req)
	c.Request.Body.Close()
	// hack: rebind by setting context? simpler: call CreateTrustKey after setting JSON? we'll just implement inline
	orgID := c.Param("orgId")
	if _, err := uuid.Parse(orgID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid org id"})
		return
	}
	// generate ed25519 keypair
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		c.JSON(500, gin.H{"error": "keygen failed"})
		return
	}
	enc := base64.RawURLEncoding.EncodeToString(priv)
	kid := strings.TrimSpace(req.Kid)
	if kid == "" {
		sum := sha256.Sum256(pub)
		kid = base64.RawURLEncoding.EncodeToString(sum[:8])
	}
	row := database.DB.QueryRowx(`INSERT INTO trust_keys(org_id, alg, ed25519_private_key_base64, kid, active) VALUES ($1,'EdDSA',$2,$3,true) RETURNING id::text, created_at::text`, orgID, enc, kid)
	var id, created string
	if err := row.Scan(&id, &created); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	// mark previous active keys for deactivation after grace; prefer request override, else env
	var d time.Duration
	if gs := strings.TrimSpace(req.Grace); gs != "" {
		if dd, err := time.ParseDuration(gs); err == nil && dd > 0 {
			d = dd
		}
	} else if req.GraceSeconds > 0 {
		d = time.Duration(req.GraceSeconds) * time.Second
	} else if gs := strings.TrimSpace(os.Getenv("AURA_TRUST_KEY_GRACE")); gs != "" {
		if dd, err := time.ParseDuration(gs); err == nil && dd > 0 {
			d = dd
		}
	}
	if d > 0 {
		deadline := time.Now().Add(d)
		_, _ = database.DB.Exec(`UPDATE trust_keys SET deactivate_after=$1 WHERE org_id=$2 AND id<>$3 AND active=true`, deadline, orgID, id)
	}
	_ = audit.Append(c.Request.Context(), uuid.MustParse(orgID), "trust_key_rotated", gin.H{"id": id, "kid": kid}, nil, nil)
	c.JSON(http.StatusCreated, gin.H{"id": id, "org_id": orgID, "kid": kid, "active": true, "alg": "EdDSA", "created_at": created})
}
