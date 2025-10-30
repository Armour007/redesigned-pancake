package api

import (
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	database "github.com/Armour007/aura-backend/internal"
	kms "github.com/Armour007/aura-backend/internal/crypto"
	"github.com/gin-gonic/gin"
)

// GossipMessage is the canonical payload that is signed (protected header via JWS compact) by the sender's org trust key.
// The detached JWS is compact form with base64 payload (normal JWS, not b64=false): we verify against Org JWKS via existing resolver.
type GossipMessage struct {
	OrgID   string          `json:"org_id"`
	Topic   string          `json:"topic"` // revocation|key_rotation|org_registry
	Ts      time.Time       `json:"ts"`
	Nonce   string          `json:"nonce"`
	Parents []string        `json:"parents,omitempty"`
	Payload json.RawMessage `json:"payload"`
	JWS     string          `json:"jws"`
}

type gossipIngestReq struct {
	Messages []GossipMessage `json:"messages"`
}

type gossipPullResp struct {
	Messages []map[string]any `json:"messages"`
	Head     string           `json:"head"`
	Anchor   map[string]any   `json:"anchor,omitempty"`
}

// POST /v2/federation/gossip — ingest externally signed messages
func IngestGossip(c *gin.Context) {
	var req gossipIngestReq
	if err := c.ShouldBindJSON(&req); err != nil || len(req.Messages) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "messages required"})
		return
	}
	// sort messages by ts to maintain deterministic chaining when inserting a batch
	sort.Slice(req.Messages, func(i, j int) bool { return req.Messages[i].Ts.Before(req.Messages[j].Ts) })
	for _, m := range req.Messages {
		// basic validation
		topic := strings.ToLower(strings.TrimSpace(m.Topic))
		if topic != "revocation" && topic != "key_rotation" && topic != "org_registry" {
			continue
		}
		if m.OrgID == "" || m.Nonce == "" || m.Ts.IsZero() || len(m.Payload) == 0 || m.JWS == "" {
			continue
		}
		// Reject timestamps too far in the future (>5 minutes)
		if m.Ts.After(time.Now().UTC().Add(5 * time.Minute)) {
			continue
		}
		// Simple per-topic/org rate check within last minute (DB-based)
		limit := 200
		if v := strings.TrimSpace(os.Getenv("AURA_GOSSIP_INGEST_RATE_PER_MIN")); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				limit = n
			}
		}
		var cnt int
		_ = database.DB.Get(&cnt, `SELECT COUNT(1) FROM federation_gossip WHERE topic=$1 AND org_id=$2 AND ts > NOW() - INTERVAL '1 minute'`, topic, m.OrgID)
		if cnt >= limit {
			continue
		}
		// verify JWS using org-scoped JWKS
		if !verifyJWSWithOrgJWKS(c.Request.Context(), m.JWS, m.OrgID) {
			continue
		}
		// compute hash. If parents provided, use DAG hash over sorted parents; else use linear prev chain
		var prev string
		parents := m.Parents
		if len(parents) == 0 {
			_ = database.DB.Get(&prev, `SELECT hash FROM federation_gossip WHERE topic=$1 ORDER BY ts DESC, created_at DESC LIMIT 1`, topic)
		}
		can := canonicalizeGossipForHash(m)
		cur := computeGossipHash(topic, parents, prev, can)
		// insert if not exists
		// set prev_hash for backwards compat (first parent if provided)
		prevCompat := prev
		if prevCompat == "" && len(parents) > 0 {
			prevCompat = parents[0]
		}
		_, _ = database.DB.Exec(`INSERT INTO federation_gossip(org_id, topic, payload, ts, nonce, parents, kid, jws, prev_hash, hash) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) ON CONFLICT DO NOTHING`, m.OrgID, topic, m.Payload, m.Ts, m.Nonce, pqStringArray(parents), extractKid(m.JWS), m.JWS, prevCompat, cur)
	}
	c.Status(http.StatusNoContent)
}

// GET /v2/federation/gossip?topic=revocation&since=2025-10-29T00:00:00Z
func PullGossip(c *gin.Context) {
	topic := strings.ToLower(c.DefaultQuery("topic", "revocation"))
	since := c.Query("since")
	var rows []struct {
		OrgID   string    `db:"org_id"`
		Payload []byte    `db:"payload"`
		Ts      time.Time `db:"ts"`
		Nonce   string    `db:"nonce"`
		Kid     string    `db:"kid"`
		JWS     string    `db:"jws"`
		Hash    string    `db:"hash"`
	}
	q := `SELECT org_id::text, payload, ts, nonce, kid, jws, hash FROM federation_gossip WHERE topic=$1`
	args := []any{topic}
	if since != "" {
		q += ` AND ts > $2`
		args = append(args, since)
	}
	q += ` ORDER BY ts ASC LIMIT 500`
	if err := database.DB.Select(&rows, q, args...); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := make([]map[string]any, 0, len(rows))
	var head string
	for _, r := range rows {
		out = append(out, gin.H{"org_id": r.OrgID, "payload": json.RawMessage(r.Payload), "ts": r.Ts.Format(time.RFC3339Nano), "nonce": r.Nonce, "kid": r.Kid, "jws": r.JWS, "hash": r.Hash})
		head = r.Hash
	}
	// Optionally include latest anchor reference for this topic if configured
	anchor := map[string]any(nil)
	if anchorOrg := strings.TrimSpace(os.Getenv("AURA_FEDERATION_ANCHOR_ORG_ID")); anchorOrg != "" {
		var date, root, ext string
		// Get most recent anchor for org (we encode topic in external_ref to disambiguate)
		if err := database.DB.QueryRow(`SELECT anchor_date::text, root_hash, COALESCE(external_ref,'') FROM audit_anchors WHERE org_id=$1 ORDER BY anchor_date DESC LIMIT 1`, anchorOrg).Scan(&date, &root, &ext); err == nil {
			anchor = gin.H{"org_id": anchorOrg, "date": date, "root_hash": root, "external_ref": ext}
		}
	}
	c.JSON(http.StatusOK, gossipPullResp{Messages: out, Head: head, Anchor: anchor})
}

// POST /v2/federation/gossip/publish — sign and store a local message (server holds org trust key)
func PublishGossip(c *gin.Context) {
	orgID := c.GetString("orgID")
	var body struct {
		Topic   string          `json:"topic"`
		Payload json.RawMessage `json:"payload"`
		Nonce   string          `json:"nonce"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Topic == "" || len(body.Payload) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}
	topic := strings.ToLower(strings.TrimSpace(body.Topic))
	if topic != "revocation" && topic != "key_rotation" && topic != "org_registry" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid topic"})
		return
	}
	ts := time.Now().UTC()
	nonce := body.Nonce
	if nonce == "" {
		nonce = randomNonce()
	}
	// build canonical message for signing (without JWS)
	canMsg := map[string]any{"org_id": orgID, "topic": topic, "ts": ts.Format(time.RFC3339Nano), "nonce": nonce, "payload": json.RawMessage(body.Payload)}
	canBytes, _ := json.Marshal(canMsg)
	jws, kid, err := signJWSCompact(canBytes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	var prev string
	_ = database.DB.Get(&prev, `SELECT hash FROM federation_gossip WHERE topic=$1 ORDER BY ts DESC, created_at DESC LIMIT 1`, topic)
	cur := computeGossipHash(topic, nil, prev, canBytes)
	_, err = database.DB.Exec(`INSERT INTO federation_gossip(org_id, topic, payload, ts, nonce, parents, kid, jws, prev_hash, hash) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, orgID, topic, body.Payload, ts, nonce, nil, kid, jws, prev, cur)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"hash": cur, "kid": kid, "jws": jws, "ts": ts})
}

// Helpers
func canonicalizeGossipForHash(m GossipMessage) []byte {
	// Stable key order via struct marshaling
	type canon struct {
		OrgID   string          `json:"org_id"`
		Topic   string          `json:"topic"`
		Ts      string          `json:"ts"`
		Nonce   string          `json:"nonce"`
		Payload json.RawMessage `json:"payload"`
	}
	x := canon{
		OrgID:   m.OrgID,
		Topic:   strings.ToLower(m.Topic),
		Ts:      m.Ts.UTC().Format(time.RFC3339Nano),
		Nonce:   m.Nonce,
		Payload: json.RawMessage(m.Payload),
	}
	b, _ := json.Marshal(x)
	return b
}

// computeGossipHash computes the DAG-aware hash for a message.
// If parents is non-empty, hash = sha256_hex(sort(parents)||canonical);
// else, hash = sha256_hex(prev_hash||canonical).
func computeGossipHash(topic string, parents []string, prev string, canonical []byte) string {
	if len(parents) > 0 {
		ps := append([]string(nil), parents...)
		sort.Strings(ps)
		var buf strings.Builder
		for _, p := range ps {
			buf.WriteString(p)
		}
		h := sha256.Sum256(append([]byte(buf.String()), canonical...))
		return hex.EncodeToString(h[:])
	}
	h := sha256.Sum256(append([]byte(prev), canonical...))
	return hex.EncodeToString(h[:])
}

// helper to pass []string as text[] parameter
func pqStringArray(a []string) interface{} {
	if a == nil {
		return nil
	}
	return a
}

// CreateMergeGossip creates and inserts a merge message signed by the given org.
func CreateMergeGossip(ctx context.Context, orgID, topic string, parents []string, payload json.RawMessage, nonce string) (string, string, string, time.Time, error) {
	if len(parents) < 2 {
		return "", "", "", time.Time{}, errors.New("at least two parents required")
	}
	ts := time.Now().UTC()
	if nonce == "" {
		nonce = randomNonce()
	}
	// canonical for signing includes parents to bind DAG
	canMsg := map[string]any{"org_id": orgID, "topic": topic, "ts": ts.Format(time.RFC3339Nano), "nonce": nonce, "parents": parents, "payload": json.RawMessage(payload)}
	canBytes, _ := json.Marshal(canMsg)
	jws, kid, err := signJWSCompact(canBytes)
	if err != nil {
		return "", "", "", time.Time{}, err
	}
	cur := computeGossipHash(topic, parents, "", canBytes)
	prevCompat := parents[0]
	if _, err := database.DB.Exec(`INSERT INTO federation_gossip(org_id, topic, payload, ts, nonce, parents, kid, jws, prev_hash, hash) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) ON CONFLICT DO NOTHING`, orgID, topic, payload, ts, nonce, pqStringArray(parents), kid, jws, prevCompat, cur); err != nil {
		return "", "", "", time.Time{}, err
	}
	return cur, kid, jws, ts, nil
}

// POST /v2/federation/gossip/merge — create a merge message referencing multiple parents
func MergeGossip(c *gin.Context) {
	orgID := c.GetString("orgID")
	var body struct {
		Topic   string          `json:"topic"`
		Parents []string        `json:"parents"`
		Payload json.RawMessage `json:"payload"`
		Nonce   string          `json:"nonce"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Topic == "" || len(body.Parents) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "topic and at least two parents required"})
		return
	}
	topic := strings.ToLower(strings.TrimSpace(body.Topic))
	if topic != "revocation" && topic != "key_rotation" && topic != "org_registry" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid topic"})
		return
	}
	hash, kid, jws, ts, err := CreateMergeGossip(c.Request.Context(), orgID, topic, body.Parents, body.Payload, body.Nonce)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"hash": hash, "kid": kid, "jws": jws, "ts": ts})
}

func extractKid(jws string) string {
	parts := strings.Split(jws, ".")
	if len(parts) != 3 {
		return ""
	}
	var hdr map[string]any
	if hb, err := b64urlDecode(parts[0]); err == nil {
		_ = json.Unmarshal(hb, &hdr)
	}
	if k, _ := hdr["kid"].(string); k != "" {
		return k
	}
	return ""
}

// --- Helpers: base64url, crypto verify/sign, nonce ---
func b64urlDecode(s string) ([]byte, error) {
	if m := len(s) % 4; m != 0 {
		s += strings.Repeat("=", 4-m)
	}
	return base64.URLEncoding.DecodeString(s)
}

// verifyJWSWithOrgJWKS verifies a compact JWS using the org's active public keys in trust_keys.jwk_pub
func verifyJWSWithOrgJWKS(ctx context.Context, jws, orgID string) bool {
	parts := strings.Split(jws, ".")
	if len(parts) != 3 {
		return false
	}
	hb, err := b64urlDecode(parts[0])
	if err != nil {
		return false
	}
	var hdr map[string]any
	if err := json.Unmarshal(hb, &hdr); err != nil {
		return false
	}
	alg, _ := hdr["alg"].(string)
	kid, _ := hdr["kid"].(string)
	unsigned := []byte(parts[0] + "." + parts[1])
	sig, err := b64urlDecode(parts[2])
	if err != nil {
		return false
	}
	// query a few active keys; filter by kid if provided
	rows, err := database.DB.Queryx(`SELECT COALESCE(jwk_pub,'{}'::jsonb), COALESCE(alg,''), COALESCE(kid,'') FROM trust_keys WHERE org_id=$1 AND active=true ORDER BY created_at DESC LIMIT 10`, orgID)
	if err != nil {
		return false
	}
	defer rows.Close()
	for rows.Next() {
		var jwk json.RawMessage
		var a, k string
		if err := rows.Scan(&jwk, &a, &k); err != nil {
			continue
		}
		if kid != "" && k != kid {
			continue
		}
		var key map[string]any
		_ = json.Unmarshal(jwk, &key)
		if verifyWithJWK(alg, key, unsigned, sig) {
			return true
		}
	}
	return false
}

func verifyWithJWK(alg string, jwk map[string]any, unsigned, sig []byte) bool {
	kty, _ := jwk["kty"].(string)
	switch {
	case alg == kms.AlgEdDSA && kty == "OKP" && jwk["crv"] == "Ed25519":
		x, _ := jwk["x"].(string)
		xb, err := b64urlDecode(x)
		if err != nil || len(xb) != 32 || len(sig) != 64 {
			return false
		}
		return ed25519.Verify(ed25519.PublicKey(xb), unsigned, sig)
	case alg == kms.AlgES256 && kty == "EC" && jwk["crv"] == "P-256":
		xs, _ := jwk["x"].(string)
		ys, _ := jwk["y"].(string)
		xb, err1 := b64urlDecode(xs)
		yb, err2 := b64urlDecode(ys)
		if err1 != nil || err2 != nil || len(sig) != 64 {
			return false
		}
		h := sha256.Sum256(unsigned)
		pub := ecdsa.PublicKey{Curve: elliptic.P256(), X: new(big.Int).SetBytes(xb), Y: new(big.Int).SetBytes(yb)}
		r := new(big.Int).SetBytes(sig[:32])
		s := new(big.Int).SetBytes(sig[32:])
		return ecdsa.Verify(&pub, h[:], r, s)
	default:
		return false
	}
}

// signJWSCompact signs the given canonical payload using the org's active trust key
func signJWSCompact(unsignedPayload []byte) (string, string, error) {
	var temp struct {
		OrgID string `json:"org_id"`
	}
	if err := json.Unmarshal(unsignedPayload, &temp); err != nil {
		return "", "", err
	}
	if temp.OrgID == "" {
		return "", "", errors.New("org_id missing in payload")
	}
	// load active key for org
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
	if err := database.DB.Get(&tk, `SELECT alg, COALESCE(kid,''), provider, key_ref, key_version, COALESCE(provider_config,'{}'::jsonb), ed25519_private_key_base64, COALESCE(jwk_pub,'{}'::jsonb) FROM trust_keys WHERE org_id=$1 AND active=true ORDER BY created_at DESC LIMIT 1`, temp.OrgID); err != nil {
		return "", "", err
	}
	rec := kms.TrustKeyRecord{Provider: strPtr(tk.Prov), KeyRef: strPtr(tk.Ref), KeyVersion: strPtr(tk.Ver), Alg: tk.Alg, Kid: tk.Kid, EncPriv: strPtr(tk.Enc), ProviderConfig: tk.Cfg, JWKPub: tk.JWK}
	signer, err := kms.NewSignerFromRecord(rec)
	if err != nil {
		return "", "", err
	}
	hdr := map[string]any{"alg": signer.Algorithm(), "kid": signer.KeyID()}
	hb, _ := json.Marshal(hdr)
	hB64 := base64.RawURLEncoding.EncodeToString(hb)
	pB64 := base64.RawURLEncoding.EncodeToString(unsignedPayload)
	toSign := []byte(hB64 + "." + pB64)
	sig, err := signer.Sign(context.Background(), toSign)
	if err != nil {
		return "", "", err
	}
	sB64 := base64.RawURLEncoding.EncodeToString(sig)
	return hB64 + "." + pB64 + "." + sB64, signer.KeyID(), nil
}

func strPtr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func randomNonce() string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
