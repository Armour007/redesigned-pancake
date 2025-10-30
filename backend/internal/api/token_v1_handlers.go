package api

import (
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	database "github.com/Armour007/aura-backend/internal"
	kms "github.com/Armour007/aura-backend/internal/crypto"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// IssueTrustTokenV1 issues a compact JWT trust token signed with the org's active trust key
// POST /v1/token/issue
// Body: { "org_id":"...", "sub":"agent|user", "aud":"svc", "action":"...", "resource":"...", "ttl_sec": 600, "nbf": <unix> }
// Requires API key or attestation middleware on the group
func IssueTrustTokenV1(c *gin.Context) {
	type reqT struct {
		OrgID    string `json:"org_id"`
		Sub      string `json:"sub"`
		Aud      string `json:"aud"`
		Action   string `json:"action"`
		Resource string `json:"resource"`
		TTL      int    `json:"ttl_sec"`
		Nbf      int64  `json:"nbf,omitempty"`
	}
	var req reqT
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if strings.TrimSpace(req.OrgID) == "" || req.TTL <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "org_id and ttl_sec required"})
		return
	}
	orgID := req.OrgID
	jti := uuid.New().String()
	iat := time.Now().UTC()
	exp := iat.Add(time.Duration(req.TTL) * time.Second)
	if req.Nbf == 0 {
		req.Nbf = iat.Unix()
	}
	claims := map[string]any{
		"iss":      "did:aura:org:" + orgID,
		"sub":      req.Sub,
		"aud":      req.Aud,
		"org_id":   orgID,
		"action":   req.Action,
		"resource": req.Resource,
		"jti":      jti,
		"iat":      iat.Unix(),
		"nbf":      req.Nbf,
		"exp":      exp.Unix(),
	}
	// Try org trust key first
	var tk struct {
		Alg  string          `db:"alg"`
		Kid  string          `db:"kid"`
		Prov *string         `db:"provider"`
		Ref  *string         `db:"key_ref"`
		Ver  *string         `db:"key_version"`
		Cfg  json.RawMessage `db:"provider_config"`
		Enc  *string         `db:"ed25519_private_key_base64"`
		JWK  json.RawMessage `db:"jwk_pub"`
	}
	if err := database.DB.Get(&tk, `SELECT alg, COALESCE(kid,''), provider, key_ref, key_version, COALESCE(provider_config,'{}'::jsonb), ed25519_private_key_base64, COALESCE(jwk_pub,'{}'::jsonb) FROM trust_keys WHERE org_id=$1 AND active=true ORDER BY created_at DESC LIMIT 1`, orgID); err == nil && tk.Alg != "" {
		header := map[string]any{"alg": tk.Alg, "typ": "JWT", "kid": tk.Kid}
		hb, _ := json.Marshal(header)
		pb, _ := json.Marshal(claims)
		unsigned := base64.RawURLEncoding.EncodeToString(hb) + "." + base64.RawURLEncoding.EncodeToString(pb)
		// local ed25519 fast path
		if (tk.Prov == nil || *tk.Prov == "" || strings.ToLower(*tk.Prov) == "local") && tk.Alg == kms.AlgEdDSA && tk.Enc != nil && *tk.Enc != "" {
			if priv, _, _ := loadOrgEd25519Key(orgID); priv != nil {
				sig := ed25519.Sign(priv, []byte(unsigned))
				tok := unsigned + "." + base64.RawURLEncoding.EncodeToString(sig)
				c.JSON(http.StatusOK, gin.H{"token": tok, "kid": tk.Kid, "alg": tk.Alg, "exp": exp.Unix(), "jti": jti})
				return
			}
		}
		rec := kms.TrustKeyRecord{Provider: strPtrVal(tk.Prov), KeyRef: strPtrVal(tk.Ref), KeyVersion: strPtrVal(tk.Ver), Alg: tk.Alg, Kid: tk.Kid, EncPriv: strPtrVal(tk.Enc), ProviderConfig: tk.Cfg, JWKPub: tk.JWK}
		if signer, err := kms.NewSignerFromRecord(rec); err == nil {
			if sig, err := signer.Sign(c.Request.Context(), []byte(unsigned)); err == nil {
				tok := unsigned + "." + base64.RawURLEncoding.EncodeToString(sig)
				c.JSON(http.StatusOK, gin.H{"token": tok, "kid": signer.KeyID(), "alg": signer.Algorithm(), "exp": exp.Unix(), "jti": jti})
				return
			}
		}
	}
	// Env Ed25519 fallback
	if priv, _, kid := loadEd25519KeyFromEnv(); priv != nil {
		header := map[string]any{"alg": "EdDSA", "typ": "JWT", "kid": kid}
		hb, _ := json.Marshal(header)
		pb, _ := json.Marshal(claims)
		unsigned := base64.RawURLEncoding.EncodeToString(hb) + "." + base64.RawURLEncoding.EncodeToString(pb)
		sig := ed25519.Sign(priv, []byte(unsigned))
		tok := unsigned + "." + base64.RawURLEncoding.EncodeToString(sig)
		c.JSON(http.StatusOK, gin.H{"token": tok, "kid": kid, "alg": "EdDSA", "exp": exp.Unix(), "jti": jti})
		return
	}
	// HS256 fallback using JWT_SECRET/AURA_TRUST_TOKEN_SIGNING_KEY
	secret := os.Getenv("AURA_TRUST_TOKEN_SIGNING_KEY")
	if secret == "" {
		secret = os.Getenv("JWT_SECRET")
	}
	if secret == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "signing key not configured"})
		return
	}
	header := map[string]any{"alg": "HS256", "typ": "JWT"}
	hb, _ := json.Marshal(header)
	pb, _ := json.Marshal(claims)
	unsigned := base64.RawURLEncoding.EncodeToString(hb) + "." + base64.RawURLEncoding.EncodeToString(pb)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(unsigned))
	tok := unsigned + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	c.JSON(http.StatusOK, gin.H{"token": tok, "alg": "HS256", "exp": exp.Unix(), "jti": jti})
}

// VerifyTrustTokenV1 verifies a token (HS256 or EdDSA) and returns validity/claims
func VerifyTrustTokenV1(c *gin.Context) {
	var in struct {
		Token string `json:"token"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || strings.TrimSpace(in.Token) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token required"})
		return
	}
	valid, claims, reason := validateJWT(in.Token)
	c.JSON(http.StatusOK, gin.H{"valid": valid, "reason": reason, "claims": claims})
}

// RevokeTrustTokenV1 inserts a JTI into the revocation table; clients should check replay via introspection or DB
func RevokeTrustTokenV1(c *gin.Context) {
	var in struct {
		OrgID string `json:"org_id"`
		JTI   string `json:"jti"`
		Exp   int64  `json:"exp"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.OrgID == "" || in.JTI == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "org_id and jti required"})
		return
	}
	if in.Exp == 0 {
		in.Exp = time.Now().Add(24 * time.Hour).Unix()
	}
	if _, err := database.DB.Exec(`INSERT INTO trust_token_jti(org_id, jti, exp_at) VALUES ($1,$2,to_timestamp($3)) ON CONFLICT DO NOTHING`, in.OrgID, in.JTI, in.Exp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"revoked": true})
}

//
