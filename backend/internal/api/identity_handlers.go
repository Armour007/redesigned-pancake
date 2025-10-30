package api

import (
	"context"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	database "github.com/Armour007/aura-backend/internal"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/Armour007/aura-backend/internal/audit"
	kms "github.com/Armour007/aura-backend/internal/crypto"
	ld "github.com/piprate/json-gold/ld"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type createAuraIDReq struct {
	Owner            string          `json:"owner"`
	ModelHash        string          `json:"model_hash"`
	CodeHash         string          `json:"code_hash"`
	BuildAttestation json.RawMessage `json:"build_attestation,omitempty"`
}

type createAuraIDResp struct {
	ID    string `json:"id"`
	VCJWT string `json:"vc_jwt"`
	VCLDP any    `json:"vc_ldp,omitempty"`
}

type publicAuraIDResp struct {
	ID        string    `json:"id"`
	OrgID     string    `json:"org_id"`
	Owner     string    `json:"owner"`
	ModelHash string    `json:"model_hash"`
	CodeHash  string    `json:"code_hash"`
	CreatedAt time.Time `json:"created_at"`
	VCJWT     string    `json:"vc_jwt"`
	VC        any       `json:"vc,omitempty"`
	VCLDP     any       `json:"vc_ldp,omitempty"`
}

// BuildVCJWT creates a W3C VC as a JWT (VC-JWT) signed by the org's trust key when available.
func BuildVCJWT(ctx context.Context, orgID uuid.UUID, id uuid.UUID, owner, modelHash, codeHash string, buildAtt json.RawMessage, iat time.Time) (string, error) {
	tr := otel.Tracer("aura")
	ctx, span := tr.Start(ctx, "aura_id.build_vc_jwt")
	defer span.End()
	span.SetAttributes(attribute.String("org_id", orgID.String()))
	iss := "did:aura:org:" + orgID.String()
	// VC claim body
	vc := map[string]any{
		"@context": []string{"https://www.w3.org/2018/credentials/v1"},
		"type":     []string{"VerifiableCredential", "AuraIdentity"},
		"credentialSubject": map[string]any{
			"id":         "urn:owner:" + owner,
			"owner":      owner,
			"model_hash": modelHash,
			"code_hash":  codeHash,
		},
	}
	if len(buildAtt) != 0 {
		var att any
		_ = json.Unmarshal(buildAtt, &att)
		vc["credentialSubject"].(map[string]any)["build_attestation"] = att
	}
	claims := map[string]any{
		"iss": iss,
		"sub": owner,
		"jti": id.String(),
		"iat": iat.Unix(),
		"nbf": iat.Unix(),
		"vc":  vc,
	}
	payload, _ := json.Marshal(claims)
	// choose signer: prefer org trust key (EdDSA/ES256), then env Ed25519, else HS256 fallback
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
	if err := database.DB.Get(&tk, `SELECT alg, COALESCE(kid,''), provider, key_ref, key_version, COALESCE(provider_config,'{}'::jsonb), ed25519_private_key_base64, COALESCE(jwk_pub,'{}'::jsonb) FROM trust_keys WHERE org_id=$1 AND active=true ORDER BY created_at DESC LIMIT 1`, orgID.String()); err == nil && tk.Alg != "" {
		h := fmt.Sprintf(`{"alg":"%s","typ":"JWT","kid":"%s"}`, tk.Alg, tk.Kid)
		hb := base64.RawURLEncoding.EncodeToString([]byte(h))
		pb := base64.RawURLEncoding.EncodeToString(payload)
		unsigned := hb + "." + pb
		if (tk.Prov == nil || *tk.Prov == "" || strings.ToLower(*tk.Prov) == "local") && tk.Alg == "EdDSA" && tk.Enc != nil && *tk.Enc != "" {
			if priv, _, _ := loadOrgEd25519Key(orgID.String()); priv != nil {
				sig := ed25519.Sign(priv, []byte(unsigned))
				return unsigned + "." + base64.RawURLEncoding.EncodeToString(sig), nil
			}
		}
		rec := kms.TrustKeyRecord{Provider: strPtrVal(tk.Prov), KeyRef: strPtrVal(tk.Ref), KeyVersion: strPtrVal(tk.Ver), Alg: tk.Alg, Kid: tk.Kid, EncPriv: strPtrVal(tk.Enc), ProviderConfig: tk.Cfg, JWKPub: tk.JWK}
		if signer, err := kms.NewSignerFromRecord(rec); err == nil {
			if sig, err := signer.Sign(ctx, []byte(unsigned)); err == nil {
				return unsigned + "." + base64.RawURLEncoding.EncodeToString(sig), nil
			}
		}
	}
	// env Ed25519
	if priv, _, kid := loadEd25519KeyFromEnv(); priv != nil {
		h := fmt.Sprintf(`{"alg":"EdDSA","typ":"JWT","kid":"%s"}`, kid)
		hb := base64.RawURLEncoding.EncodeToString([]byte(h))
		pb := base64.RawURLEncoding.EncodeToString(payload)
		unsigned := hb + "." + pb
		sig := ed25519.Sign(priv, []byte(unsigned))
		return unsigned + "." + base64.RawURLEncoding.EncodeToString(sig), nil
	}
	// HS256 fallback
	secret := os.Getenv("AURA_TRUST_TOKEN_SIGNING_KEY")
	if secret == "" {
		secret = os.Getenv("JWT_SECRET")
	}
	if secret == "" {
		return "", fmt.Errorf("no signing key available")
	}
	h := `{"alg":"HS256","typ":"JWT"}`
	hb := base64.RawURLEncoding.EncodeToString([]byte(h))
	pb := base64.RawURLEncoding.EncodeToString(payload)
	unsigned := hb + "." + pb
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(unsigned))
	sig := mac.Sum(nil)
	return unsigned + "." + base64.RawURLEncoding.EncodeToString(sig), nil
}

// BuildVCLDP creates a JSON-LD VC with a lightweight JWS-based Linked Data Proof (experimental)
// Note: This implementation uses a compact JSON with a JsonWebSignature2020-style proof whose JWS
// payload is the canonical JSON bytes of the VC without the proof. It is intended for interoperability
// with ecosystems that prefer LDP; full URDNA2015 canonicalization is not applied here.
func BuildVCLDP(ctx context.Context, orgID uuid.UUID, id uuid.UUID, owner, modelHash, codeHash string, buildAtt json.RawMessage, iat time.Time) (map[string]any, error) {
	tr := otel.Tracer("aura")
	ctx, span := tr.Start(ctx, "aura_id.build_vc_ldp")
	defer span.End()
	span.SetAttributes(attribute.String("org_id", orgID.String()))

	did := "did:aura:org:" + orgID.String()
	// Build base VC document
	vc := map[string]any{
		"@context":     []string{"https://www.w3.org/2018/credentials/v1"},
		"id":           "urn:uuid:" + id.String(),
		"type":         []string{"VerifiableCredential", "AuraIdentity"},
		"issuer":       did,
		"issuanceDate": iat.UTC().Format(time.RFC3339Nano),
		"credentialSubject": map[string]any{
			"id":         "urn:owner:" + owner,
			"owner":      owner,
			"model_hash": modelHash,
			"code_hash":  codeHash,
		},
	}
	if len(buildAtt) != 0 {
		var att any
		_ = json.Unmarshal(buildAtt, &att)
		vc["credentialSubject"].(map[string]any)["build_attestation"] = att
	}

	// JSON-LD Normalize (URDNA2015) to N-Quads as detached payload
	delete(vc, "proof")
	proc := ld.NewJsonLdProcessor()
	opts := ld.NewJsonLdOptions("")
	opts.ProcessingMode = ld.JsonLd_1_1
	opts.Format = "application/n-quads"
	norm, nerr := proc.Normalize(vc, opts)
	var nquads string
	if nerr == nil {
		if s, ok := norm.(string); ok {
			nquads = s
		}
	}
	if nquads == "" {
		// Fallback: compact JSON
		b, _ := json.Marshal(vc)
		nquads = string(b)
	}

	// Select signer similar to BuildVCJWT (prefer org trust key; fallback to env; last-resort HS256)
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
	var kid string
	var sigB64 string
	if err := database.DB.Get(&tk, `SELECT alg, COALESCE(kid,''), provider, key_ref, key_version, COALESCE(provider_config,'{}'::jsonb), ed25519_private_key_base64, COALESCE(jwk_pub,'{}'::jsonb) FROM trust_keys WHERE org_id=$1 AND active=true ORDER BY created_at DESC LIMIT 1`, orgID.String()); err == nil && tk.Alg != "" {
		kid = tk.Kid
		// RFC 7797: detached JWS with unencoded payload
		ph := map[string]any{"alg": tk.Alg, "kid": tk.Kid, "b64": false, "crit": []string{"b64"}}
		phb, _ := json.Marshal(ph)
		hb := base64.RawURLEncoding.EncodeToString(phb)
		unsigned := hb + "." + nquads
		if (tk.Prov == nil || *tk.Prov == "" || strings.ToLower(*tk.Prov) == "local") && tk.Alg == "EdDSA" && tk.Enc != nil && *tk.Enc != "" {
			if priv, _, _ := loadOrgEd25519Key(orgID.String()); priv != nil {
				sig := ed25519.Sign(priv, []byte(unsigned))
				sigB64 = base64.RawURLEncoding.EncodeToString(sig)
			}
		}
		if sigB64 == "" {
			rec := kms.TrustKeyRecord{Provider: strPtrVal(tk.Prov), KeyRef: strPtrVal(tk.Ref), KeyVersion: strPtrVal(tk.Ver), Alg: tk.Alg, Kid: tk.Kid, EncPriv: strPtrVal(tk.Enc), ProviderConfig: tk.Cfg, JWKPub: tk.JWK}
			if signer, e := kms.NewSignerFromRecord(rec); e == nil {
				if sig, e := signer.Sign(ctx, []byte(unsigned)); e == nil {
					sigB64 = base64.RawURLEncoding.EncodeToString(sig)
				}
			}
		}
		if sigB64 != "" {
			// Attach proof
			proof := map[string]any{
				"type":               "JsonWebSignature2020",
				"created":            iat.UTC().Format(time.RFC3339Nano),
				"proofPurpose":       "assertionMethod",
				"verificationMethod": fmt.Sprintf("%s#%s", did, kid),
				// Detached payload compact serialization
				"jws": hb + ".." + sigB64,
			}
			vc["proof"] = proof
			return vc, nil
		}
	}
	// Env Ed25519 fallback
	if priv, _, kidEnv := loadEd25519KeyFromEnv(); priv != nil {
		kid = kidEnv
		ph := map[string]any{"alg": "EdDSA", "kid": kidEnv, "b64": false, "crit": []string{"b64"}}
		phb, _ := json.Marshal(ph)
		hb := base64.RawURLEncoding.EncodeToString(phb)
		unsigned := hb + "." + nquads
		sig := ed25519.Sign(priv, []byte(unsigned))
		sigB64 = base64.RawURLEncoding.EncodeToString(sig)
		proof := map[string]any{
			"type":               "JsonWebSignature2020",
			"created":            iat.UTC().Format(time.RFC3339Nano),
			"proofPurpose":       "assertionMethod",
			"verificationMethod": fmt.Sprintf("%s#%s", did, kid),
			"jws":                hb + ".." + sigB64,
		}
		vc["proof"] = proof
		return vc, nil
	}
	// HS256 fallback: embed MAC as non-standard proof (for dev/testing only)
	secret := os.Getenv("AURA_TRUST_TOKEN_SIGNING_KEY")
	if secret == "" {
		secret = os.Getenv("JWT_SECRET")
	}
	if secret == "" {
		return nil, fmt.Errorf("no signing key available for vc_ldp")
	}
	ph := map[string]any{"alg": "HS256", "b64": false, "crit": []string{"b64"}}
	phb, _ := json.Marshal(ph)
	hb := base64.RawURLEncoding.EncodeToString(phb)
	unsigned := hb + "." + nquads
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(unsigned))
	sig := mac.Sum(nil)
	sigB64 = base64.RawURLEncoding.EncodeToString(sig)
	proof := map[string]any{
		"type":               "JsonWebSignature2020",
		"created":            iat.UTC().Format(time.RFC3339Nano),
		"proofPurpose":       "assertionMethod",
		"verificationMethod": did + "#hs256",
		"jws":                hb + ".." + sigB64,
	}
	vc["proof"] = proof
	return vc, nil
}

// CreateAuraID issues a verifiable identity VC-JWT for an AI/agent owned by the org
func CreateAuraID(c *gin.Context) {
	orgID := c.Param("orgId")
	var req createAuraIDReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Owner == "" || req.ModelHash == "" || req.CodeHash == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "owner, model_hash, code_hash are required"})
		return
	}
	id := uuid.New()
	now := time.Now().UTC()
	vc, err := BuildVCJWT(c.Request.Context(), uuid.MustParse(orgID), id, req.Owner, req.ModelHash, req.CodeHash, req.BuildAttestation, now)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// Also build JSON-LD LDP variant (best-effort)
	vcldp, _ := BuildVCLDP(c.Request.Context(), uuid.MustParse(orgID), id, req.Owner, req.ModelHash, req.CodeHash, req.BuildAttestation, now)
	// Persist
	_, err = database.DB.Exec(`INSERT INTO aura_identities(id, org_id, owner, model_hash, code_hash, build_attestation, vc_jwt, vc_ldp) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`, id, orgID, req.Owner, req.ModelHash, req.CodeHash, nullIfEmptyJSON(req.BuildAttestation), vc, toJSONB(vcldp))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	_ = audit.Append(c.Request.Context(), uuid.MustParse(orgID), "aura_id_issued", gin.H{"aura_id": id.String(), "owner": req.Owner}, nil, nil)
	c.JSON(http.StatusOK, createAuraIDResp{ID: id.String(), VCJWT: vc, VCLDP: vcldp})
}

func nullIfEmptyJSON(b json.RawMessage) any {
	if len(b) == 0 || string(b) == "null" || string(b) == "{}" {
		return nil
	}
	return b
}

func toJSONB(v any) any {
	if v == nil {
		return nil
	}
	b, err := json.Marshal(v)
	if err != nil || string(b) == "null" || string(b) == "{}" {
		return nil
	}
	return json.RawMessage(b)
}

// GetAuraID returns the public registry entry for a given id
func GetAuraID(c *gin.Context) {
	id := c.Param("id")
	var row struct {
		ID        uuid.UUID        `db:"id"`
		OrgID     uuid.UUID        `db:"org_id"`
		Owner     string           `db:"owner"`
		ModelHash string           `db:"model_hash"`
		CodeHash  string           `db:"code_hash"`
		VCJWT     string           `db:"vc_jwt"`
		VCLDP     *json.RawMessage `db:"vc_ldp"`
		CreatedAt time.Time        `db:"created_at"`
	}
	if err := database.DB.Get(&row, `SELECT id, org_id, owner, model_hash, code_hash, vc_jwt, vc_ldp, created_at FROM aura_identities WHERE id=$1`, id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	// Decode VC from VC-JWT for convenience
	var prettyVC any
	parts := strings.Split(row.VCJWT, ".")
	if len(parts) == 3 {
		if dec, err := base64.RawURLEncoding.DecodeString(parts[1]); err == nil {
			var claims map[string]any
			if json.Unmarshal(dec, &claims) == nil {
				if v, ok := claims["vc"]; ok {
					prettyVC = v
				}
			}
		}
	}
	var vcldp any
	if row.VCLDP != nil {
		_ = json.Unmarshal(*row.VCLDP, &vcldp)
	}
	c.JSON(http.StatusOK, publicAuraIDResp{ID: row.ID.String(), OrgID: row.OrgID.String(), Owner: row.Owner, ModelHash: row.ModelHash, CodeHash: row.CodeHash, CreatedAt: row.CreatedAt, VCJWT: row.VCJWT, VC: prettyVC, VCLDP: vcldp})
}

// ListAuraIDs lists issued identities for an org (admin-only)
func ListAuraIDs(c *gin.Context) {
	orgID := c.Param("orgId")
	// Basic pagination
	limit := 50
	if v := c.Query("limit"); v != "" {
		var l int
		if _, err := fmt.Sscanf(v, "%d", &l); err == nil && l > 0 && l <= 200 {
			limit = l
		}
	}
	offset := 0
	if v := c.Query("offset"); v != "" {
		var o int
		if _, err := fmt.Sscanf(v, "%d", &o); err == nil && o >= 0 {
			offset = o
		}
	}
	rows := []struct {
		ID        uuid.UUID        `db:"id"`
		Owner     string           `db:"owner"`
		ModelHash string           `db:"model_hash"`
		CodeHash  string           `db:"code_hash"`
		CreatedAt time.Time        `db:"created_at"`
		VCJWT     string           `db:"vc_jwt"`
		VCLDP     *json.RawMessage `db:"vc_ldp"`
	}{}
	if err := database.DB.Select(&rows, `SELECT id, owner, model_hash, code_hash, created_at, vc_jwt, vc_ldp FROM aura_identities WHERE org_id=$1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`, orgID, limit, offset); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := make([]any, 0, len(rows))
	for _, r := range rows {
		var vcldp any
		if r.VCLDP != nil {
			_ = json.Unmarshal(*r.VCLDP, &vcldp)
		}
		out = append(out, gin.H{
			"id":         r.ID.String(),
			"owner":      r.Owner,
			"model_hash": r.ModelHash,
			"code_hash":  r.CodeHash,
			"created_at": r.CreatedAt,
			"vc_jwt":     r.VCJWT,
			"vc_ldp":     vcldp,
		})
	}
	// total count for pagination metadata
	var total int
	if err := database.DB.Get(&total, `SELECT COUNT(*) FROM aura_identities WHERE org_id=$1`, orgID); err != nil {
		total = -1
	}
	c.JSON(http.StatusOK, gin.H{"items": out, "limit": limit, "offset": offset, "total": total})
}

// ResolveDID resolves did:aura:org:<orgId> into a DID Document using active org JWKS
func ResolveDID(c *gin.Context) {
	did := c.Query("did")
	if did == "" {
		// Support path style: /resolve/:did
		did = c.Param("did")
	}
	if did == "" || !strings.HasPrefix(did, "did:aura:org:") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing or invalid did"})
		return
	}
	orgID := strings.TrimPrefix(did, "did:aura:org:")
	// Fetch active trust keys
	var keys []struct {
		Kid string          `db:"kid"`
		Alg string          `db:"alg"`
		JWK json.RawMessage `db:"jwk_pub"`
	}
	if err := database.DB.Select(&keys, `SELECT kid, alg, COALESCE(jwk_pub,'{}'::jsonb) FROM trust_keys WHERE org_id=$1 AND active=true ORDER BY created_at DESC`, orgID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	vm := make([]any, 0, len(keys))
	am := make([]string, 0, len(keys))
	for _, k := range keys {
		vm = append(vm, gin.H{
			"id":           did + "#" + k.Kid,
			"type":         "JsonWebKey2020",
			"controller":   did,
			"publicKeyJwk": json.RawMessage(k.JWK),
		})
		am = append(am, did+"#"+k.Kid)
	}
	doc := gin.H{
		"@context":           []string{"https://www.w3.org/ns/did/v1"},
		"id":                 did,
		"verificationMethod": vm,
		"assertionMethod":    am,
	}
	c.Header("Content-Type", "application/did+ld+json")
	c.JSON(http.StatusOK, doc)
}
