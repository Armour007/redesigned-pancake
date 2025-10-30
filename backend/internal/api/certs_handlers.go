package api

import (
	"crypto/ed25519"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"time"

	database "github.com/Armour007/aura-backend/internal"
	"github.com/Armour007/aura-backend/internal/attest"
	policyrepo "github.com/Armour007/aura-backend/internal/policy"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// POST /v2/certs/issue — issue a client cert bound to a device posture
// Body: { device_id: uuid, subject_cn: string, days: int }
func IssueClientCert(c *gin.Context) {
	orgIDStr := c.GetString("orgID")
	if orgIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing org"})
		return
	}
	orgID := uuid.MustParse(orgIDStr)
	var req struct {
		DeviceID  string `json:"device_id"`
		SubjectCN string `json:"subject_cn"`
		Days      int    `json:"days"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.DeviceID == "" || req.SubjectCN == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device_id and subject_cn required"})
		return
	}
	devID := uuid.MustParse(req.DeviceID)

	// Load posture for policy decision
	var postureOK bool
	var lastAtt time.Time
	var postureJSON []byte
	err := database.DB.QueryRowx(`SELECT posture_ok, COALESCE(last_attested_at, to_timestamp(0)), COALESCE(posture,'{}'::jsonb) FROM devices WHERE id=$1 AND org_id=$2`, devID, orgID).Scan(&postureOK, &lastAtt, &postureJSON)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "device not found"})
		return
	}
	// Require recent attestation (default 24h)
	ttl := 24 * time.Hour
	if !postureOK || time.Since(lastAtt) > ttl {
		c.JSON(http.StatusForbidden, gin.H{"error": "posture not ok or stale"})
		return
	}

	// Policy evaluation: require allow for issuance when a policy is assigned
	if ok, reason, perr := evaluateAllowPolicy(c, orgID, map[string]any{
		"action": "issue_cert",
		"device": map[string]any{"id": devID.String(), "posture": jsonRaw(postureJSON), "last_attested_at": lastAtt},
	}); perr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": perr.Error()})
		return
	} else if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": reason})
		return
	}

	// Load or create CA for org
	var certPEM, keyPEM string
	err = database.DB.QueryRowx(`SELECT cert_pem, key_pem FROM org_client_ca WHERE org_id=$1 AND active=true LIMIT 1`, orgID).Scan(&certPEM, &keyPEM)
	if err != nil || certPEM == "" || keyPEM == "" {
		// Create dev CA
		cPEM, kPEM, err2 := attest.GenerateDevCA("AURA Dev CA", 5)
		if err2 != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err2.Error()})
			return
		}
		certPEM = string(cPEM)
		keyPEM = string(kPEM)
		_, _ = database.DB.Exec(`INSERT INTO org_client_ca(org_id, cert_pem, key_pem, active) VALUES($1,$2,$3,true) ON CONFLICT (org_id) WHERE active=true DO NOTHING`, orgID, certPEM, keyPEM)
	}

	// Parse CA
	block, _ := pem.Decode([]byte(certPEM))
	if block == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "bad ca cert"})
		return
	}
	caCert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	kblk, _ := pem.Decode([]byte(keyPEM))
	if kblk == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "bad ca key"})
		return
	}
	keyAny, err := x509.ParsePKCS8PrivateKey(kblk.Bytes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	caKey, ok := keyAny.(ed25519.PrivateKey)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unsupported ca key type"})
		return
	}

	days := req.Days
	if days <= 0 || days > 365 {
		days = 30
	}
	certOut, nb, na, serial, err := attest.CreateClientCert(caCert, caKey, req.SubjectCN, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	_, _ = database.DB.Exec(`INSERT INTO client_certs(serial, org_id, device_id, subject, cert_pem, not_before, not_after) VALUES($1,$2,$3,$4,$5,$6,$7)`, serial, orgID, devID, req.SubjectCN, string(certOut), nb, na)

	c.JSON(http.StatusOK, gin.H{"serial": serial, "cert_pem": string(certOut), "not_before": nb, "not_after": na})
}

// evaluateAllowPolicy compiles the active policy (if any) and returns true if allow.
func evaluateAllowPolicy(c *gin.Context, orgID uuid.UUID, input map[string]any) (bool, string, error) {
	assigns, err := policyrepo.GetActiveAssignmentsForOrg(c.Request.Context(), orgID)
	if err != nil || len(assigns) == 0 {
		return true, "", nil // no policy: allow
	}
	p := assigns[0].Policy
	v := assigns[0].Version
	e := evalRegistry[p.EngineType]
	if e == nil {
		return false, "unsupported engine", nil
	}
	cp, err := e.Compile(v.Body)
	if err != nil {
		return false, err.Error(), nil
	}
	b, _ := json.Marshal(input)
	dec, err := e.Evaluate(cp, b)
	if err != nil {
		return false, err.Error(), nil
	}
	if dec.Allow {
		return true, dec.Reason, nil
	}
	if dec.RequireApproval {
		return false, "needs_approval", nil
	}
	return false, dec.Reason, nil
}

// helper to embed existing JSON as raw
func jsonRaw(b []byte) any {
	var v any
	_ = json.Unmarshal(b, &v)
	return v
}
