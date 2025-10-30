package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	database "github.com/Armour007/aura-backend/internal"
	"github.com/Armour007/aura-backend/internal/attest"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// POST /v2/attest — verify/store TPM/TEE claims and update device posture
// Dev MVP: supports type="tpm" with DevTPMVerifier; trusts provided fields and sets posture_ok=true
// Body: { type: "tpm"|..., payload: {...} }
func HandleAttest(c *gin.Context) {
	orgIDStr := c.GetString("orgID")
	if orgIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing org context"})
		return
	}
	orgID := uuid.MustParse(orgIDStr)

	var req struct {
		Type    string          `json:"type"`
		Payload json.RawMessage `json:"payload"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Type == "" || len(req.Payload) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "type and payload required"})
		return
	}

	// Select verifier
	var v attest.Verifier
	switch req.Type {
	case "tpm":
		v = attest.DevTPMVerifier{}
	case "azure_snp":
		v = attest.AzureSNPVerifier{}
	case "aws_nitro":
		v = attest.AWSNitroVerifier{}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported attestation type"})
		return
	}
	var payload map[string]any
	if err := json.Unmarshal(req.Payload, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad payload"})
		return
	}
	res, err := v.Verify(payload)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Nonce replay protection if verifier surfaced claims.nonce
	if claimsAny, ok := res.Posture["claims"]; ok {
		if claims, ok2 := claimsAny.(map[string]any); ok2 {
			if nAny, ok3 := claims["nonce"]; ok3 {
				if nonce, ok4 := nAny.(string); ok4 && nonce != "" {
					ttl := parseEnvInt("AURA_ATTEST_NONCE_TTL_SECONDS", 600)
					if ttl <= 0 {
						ttl = 600
					}
					if replayed := checkNonceReplayOnce(c, nonce, time.Duration(ttl)*time.Second); replayed {
						c.JSON(http.StatusConflict, gin.H{"error": "nonce replay detected"})
						return
					}
				}
			}
		}
	}

	// Upsert device by fingerprint
	var deviceID uuid.UUID
	err = database.DB.Get(&deviceID, `SELECT id FROM devices WHERE device_fingerprint=$1 AND org_id=$2`, res.Fingerprint, orgID)
	if err != nil || deviceID == uuid.Nil {
		deviceID = uuid.New()
		_, _ = database.DB.Exec(`INSERT INTO devices(id, org_id, device_fingerprint, tpm_ek_pub, tpm_ak_pub, tee_provider, last_attested_at, posture, posture_ok) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
			deviceID, orgID, res.Fingerprint, payload["ek_pub"], payload["ak_pub"], req.Type, time.Now(), mapToJSON(res.Posture), res.PostureOK)
	} else {
		_, _ = database.DB.Exec(`UPDATE devices SET last_attested_at=$1, posture=$2, posture_ok=$3 WHERE id=$4`, time.Now(), mapToJSON(res.Posture), res.PostureOK, deviceID)
	}

	// Store attestation record
	_, _ = database.DB.Exec(`INSERT INTO device_attestations(org_id, device_id, type, raw, verified, verified_at) VALUES($1,$2,$3,$4,$5,$6)`, orgID, deviceID, req.Type, req.Payload, true, time.Now())

	c.JSON(http.StatusOK, gin.H{"device_id": deviceID, "posture_ok": res.PostureOK})
}

// Helper: convert map to json.RawMessage
func mapToJSON(m map[string]any) json.RawMessage {
	if m == nil {
		return nil
	}
	b, _ := json.Marshal(m)
	return b
}

// checkNonceReplayOnce returns true if nonce was already seen within TTL; stores if new.
func checkNonceReplayOnce(c *gin.Context, nonce string, ttl time.Duration) bool {
	if nonce == "" {
		return false
	}
	rc := getRedisFromEnv()
	key := "aura:attest:nonce:" + nonce
	if rc != nil {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 200*time.Millisecond)
		defer cancel()
		ok, _ := rc.SetNX(ctx, key, "1", ttl).Result()
		return !ok
	}
	// fallback to in-memory store
	if _, found := idemStore.LoadOrStore(key, struct{}{}); found {
		return true
	}
	go func(k string) { time.Sleep(ttl); idemStore.Delete(k) }(key)
	return false
}
