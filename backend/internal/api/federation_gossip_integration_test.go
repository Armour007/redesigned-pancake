package api

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	database "github.com/Armour007/aura-backend/internal"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Deterministic ingest→pull integration test using a live Postgres if AURA_TEST_LIVE_DB=1.
func TestGossipIngestPullDeterministic(t *testing.T) {
	if os.Getenv("AURA_TEST_LIVE_DB") != "1" {
		t.Skip("integration test requires AURA_TEST_LIVE_DB=1 and a running Postgres with migrations applied")
	}
	// Connect to DB via app helper (expects DB_* env set)
	if database.DB == nil {
		database.Connect()
	}
	orgID := uuid.New().String()
	// Generate local Ed25519 key and publish public JWK into trust_keys for signature verification
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("ed25519 keygen: %v", err)
	}
	jwk := map[string]any{"kty": "OKP", "crv": "Ed25519", "x": base64.RawURLEncoding.EncodeToString(pub)}
	jwkB, _ := json.Marshal(jwk)
	sum := sha256.Sum256(pub)
	kid := base64.RawURLEncoding.EncodeToString(sum[:8])
	// Clean topic state and insert trust key
	_, _ = database.DB.Exec(`DELETE FROM federation_gossip WHERE topic='revocation'`)
	_, err = database.DB.Exec(`INSERT INTO trust_keys(org_id, alg, kid, provider, ed25519_private_key_base64, jwk_pub, active, created_at) VALUES ($1,'EdDSA',$2,'local',$3,$4,true,NOW())`, orgID, kid, base64.RawURLEncoding.EncodeToString(priv), jwkB)
	if err != nil {
		t.Fatalf("insert trust_key: %v", err)
	}
	// Build two canonical messages with fixed nonces and close timestamps
	now := time.Now().UTC()
	m1 := GossipMessage{OrgID: orgID, Topic: "revocation", Ts: now.Add(-70 * time.Second), Nonce: "n-1", Payload: json.RawMessage(`{"x":1}`)}
	m2 := GossipMessage{OrgID: orgID, Topic: "revocation", Ts: now.Add(-10 * time.Second), Nonce: "n-2", Payload: json.RawMessage(`{"x":2}`)}
	// Sign canonical bytes: header(alg,kid).payload(can)
	hdr := map[string]any{"alg": "EdDSA", "kid": kid}
	hb, _ := json.Marshal(hdr)
	hB64 := base64.RawURLEncoding.EncodeToString(hb)
	can1 := canonicalizeGossipForHash(m1)
	can2 := canonicalizeGossipForHash(m2)
	p1 := base64.RawURLEncoding.EncodeToString(can1)
	p2 := base64.RawURLEncoding.EncodeToString(can2)
	s1 := ed25519.Sign(priv, []byte(hB64+"."+p1))
	s2 := ed25519.Sign(priv, []byte(hB64+"."+p2))
	m1.JWS = hB64 + "." + p1 + "." + base64.RawURLEncoding.EncodeToString(s1)
	m2.JWS = hB64 + "." + p2 + "." + base64.RawURLEncoding.EncodeToString(s2)
	// Prepare gin engine and call IngestGossip
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/v2/federation/gossip", IngestGossip)
	body := map[string]any{"messages": []GossipMessage{m1, m2}}
	bb, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/v2/federation/gossip", bytes.NewReader(bb))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("ingest status=%d body=%s", w.Code, w.Body.String())
	}
	// Compute expected head: sha256(prev||can), with prev empty then can1 then can2
	h1 := sha256.Sum256(append([]byte(""), can1...))
	h2 := sha256.Sum256(append([]byte(hex.EncodeToString(h1[:])), can2...))
	expected := hex.EncodeToString(h2[:])
	// Pull and assert head
	r.GET("/v2/federation/gossip", PullGossip)
	req2 := httptest.NewRequest(http.MethodGet, "/v2/federation/gossip?topic=revocation", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("pull status=%d body=%s", w2.Code, w2.Body.String())
	}
	var resp struct {
		Head string `json:"head"`
	}
	if err := json.Unmarshal(w2.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal pull: %v", err)
	}
	if resp.Head != expected {
		t.Fatalf("head mismatch: got=%s want=%s", resp.Head, expected)
	}
}

// Mixed orgs in a single batch: chain should converge globally per topic based on timestamp order
func TestGossipMixedOrgsConverge(t *testing.T) {
	if os.Getenv("AURA_TEST_LIVE_DB") != "1" {
		t.Skip("integration test requires AURA_TEST_LIVE_DB=1")
	}
	if database.DB == nil {
		database.Connect()
	}
	// Two orgs and keys
	org1 := uuid.New().String()
	org2 := uuid.New().String()
	pub1, priv1, _ := ed25519.GenerateKey(rand.Reader)
	pub2, priv2, _ := ed25519.GenerateKey(rand.Reader)
	sum1 := sha256.Sum256(pub1)
	sum2 := sha256.Sum256(pub2)
	kid1 := base64.RawURLEncoding.EncodeToString(sum1[:8])
	kid2 := base64.RawURLEncoding.EncodeToString(sum2[:8])
	jwk1 := map[string]any{"kty": "OKP", "crv": "Ed25519", "x": base64.RawURLEncoding.EncodeToString(pub1)}
	jwk2 := map[string]any{"kty": "OKP", "crv": "Ed25519", "x": base64.RawURLEncoding.EncodeToString(pub2)}
	jwk1b, _ := json.Marshal(jwk1)
	jwk2b, _ := json.Marshal(jwk2)
	_, _ = database.DB.Exec(`DELETE FROM federation_gossip WHERE topic='revocation'`)
	_, _ = database.DB.Exec(`DELETE FROM trust_keys WHERE org_id IN ($1,$2)`, org1, org2)
	_, _ = database.DB.Exec(`INSERT INTO trust_keys(org_id, alg, kid, provider, ed25519_private_key_base64, jwk_pub, active, created_at) VALUES ($1,'EdDSA',$2,'local',$3,$4,true,NOW())`, org1, kid1, base64.RawURLEncoding.EncodeToString(priv1), jwk1b)
	_, _ = database.DB.Exec(`INSERT INTO trust_keys(org_id, alg, kid, provider, ed25519_private_key_base64, jwk_pub, active, created_at) VALUES ($1,'EdDSA',$2,'local',$3,$4,true,NOW())`, org2, kid2, base64.RawURLEncoding.EncodeToString(priv2), jwk2b)
	now := time.Now().UTC()
	// Interleave ts across orgs
	m1 := GossipMessage{OrgID: org1, Topic: "revocation", Ts: now.Add(-90 * time.Second), Nonce: "a1", Payload: json.RawMessage(`{"i":1}`)}
	m2 := GossipMessage{OrgID: org2, Topic: "revocation", Ts: now.Add(-60 * time.Second), Nonce: "b1", Payload: json.RawMessage(`{"i":2}`)}
	m3 := GossipMessage{OrgID: org1, Topic: "revocation", Ts: now.Add(-30 * time.Second), Nonce: "a2", Payload: json.RawMessage(`{"i":3}`)}
	m4 := GossipMessage{OrgID: org2, Topic: "revocation", Ts: now.Add(-10 * time.Second), Nonce: "b2", Payload: json.RawMessage(`{"i":4}`)}
	// Sign per org
	sign := func(kid string, priv ed25519.PrivateKey, m GossipMessage) string {
		hdr := map[string]any{"alg": "EdDSA", "kid": kid}
		hb, _ := json.Marshal(hdr)
		hB64 := base64.RawURLEncoding.EncodeToString(hb)
		can := canonicalizeGossipForHash(m)
		pB64 := base64.RawURLEncoding.EncodeToString(can)
		sig := ed25519.Sign(priv, []byte(hB64+"."+pB64))
		return hB64 + "." + pB64 + "." + base64.RawURLEncoding.EncodeToString(sig)
	}
	m1.JWS = sign(kid1, priv1, m1)
	m2.JWS = sign(kid2, priv2, m2)
	m3.JWS = sign(kid1, priv1, m3)
	m4.JWS = sign(kid2, priv2, m4)
	// Batch ingest
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/v2/federation/gossip", IngestGossip)
	r.GET("/v2/federation/gossip", PullGossip)
	body := map[string]any{"messages": []GossipMessage{m1, m2, m3, m4}}
	bb, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/v2/federation/gossip", bytes.NewReader(bb))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("ingest status=%d", w.Code)
	}
	// Compute expected head by sorted ts
	cans := [][]byte{canonicalizeGossipForHash(m1), canonicalizeGossipForHash(m2), canonicalizeGossipForHash(m3), canonicalizeGossipForHash(m4)}
	h := [32]byte{}
	for i, c := range cans {
		if i == 0 {
			h = sha256.Sum256(append([]byte(""), c...))
		} else {
			h = sha256.Sum256(append([]byte(hex.EncodeToString(h[:])), c...))
		}
	}
	expected := hex.EncodeToString(h[:])
	// Pull
	req2 := httptest.NewRequest(http.MethodGet, "/v2/federation/gossip?topic=revocation", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("pull status=%d", w2.Code)
	}
	var resp struct {
		Head string `json:"head"`
	}
	_ = json.Unmarshal(w2.Body.Bytes(), &resp)
	if resp.Head != expected {
		t.Fatalf("head mismatch: got=%s want=%s", resp.Head, expected)
	}
}

// Reconciliation: two simulated nodes with disjoint subsets converge after exchanging messages
func TestGossipReconciliationConverge(t *testing.T) {
	if os.Getenv("AURA_TEST_LIVE_DB") != "1" {
		t.Skip("requires live DB")
	}
	if database.DB == nil {
		database.Connect()
	}
	org := uuid.New().String()
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	sum := sha256.Sum256(pub)
	kid := base64.RawURLEncoding.EncodeToString(sum[:8])
	jwk := map[string]any{"kty": "OKP", "crv": "Ed25519", "x": base64.RawURLEncoding.EncodeToString(pub)}
	jwkb, _ := json.Marshal(jwk)
	_, _ = database.DB.Exec(`DELETE FROM federation_gossip WHERE topic='revocation'`)
	_, _ = database.DB.Exec(`DELETE FROM trust_keys WHERE org_id=$1`, org)
	_, _ = database.DB.Exec(`INSERT INTO trust_keys(org_id, alg, kid, provider, ed25519_private_key_base64, jwk_pub, active, created_at) VALUES ($1,'EdDSA',$2,'local',$3,$4,true,NOW())`, org, kid, base64.RawURLEncoding.EncodeToString(priv), jwkb)
	now := time.Now().UTC()
	// Node A subset
	a1 := GossipMessage{OrgID: org, Topic: "revocation", Ts: now.Add(-100 * time.Second), Nonce: "a1", Payload: json.RawMessage(`{"a":1}`)}
	a2 := GossipMessage{OrgID: org, Topic: "revocation", Ts: now.Add(-80 * time.Second), Nonce: "a2", Payload: json.RawMessage(`{"a":2}`)}
	// Node B subset
	b1 := GossipMessage{OrgID: org, Topic: "revocation", Ts: now.Add(-60 * time.Second), Nonce: "b1", Payload: json.RawMessage(`{"b":1}`)}
	b2 := GossipMessage{OrgID: org, Topic: "revocation", Ts: now.Add(-40 * time.Second), Nonce: "b2", Payload: json.RawMessage(`{"b":2}`)}
	sign := func(m GossipMessage) string {
		hdr := map[string]any{"alg": "EdDSA", "kid": kid}
		hb, _ := json.Marshal(hdr)
		hB64 := base64.RawURLEncoding.EncodeToString(hb)
		can := canonicalizeGossipForHash(m)
		pB64 := base64.RawURLEncoding.EncodeToString(can)
		sig := ed25519.Sign(priv, []byte(hB64+"."+pB64))
		return hB64 + "." + pB64 + "." + base64.RawURLEncoding.EncodeToString(sig)
	}
	a1.JWS = sign(a1)
	a2.JWS = sign(a2)
	b1.JWS = sign(b1)
	b2.JWS = sign(b2)
	// In-memory heads for Node A and Node B
	hA1 := sha256.Sum256(append([]byte(""), canonicalizeGossipForHash(a1)...))
	headA := hex.EncodeToString(hA1[:])
	htmp := sha256.Sum256(append([]byte(headA), canonicalizeGossipForHash(a2)...))
	headA = hex.EncodeToString(htmp[:])
	hB1 := sha256.Sum256(append([]byte(""), canonicalizeGossipForHash(b1)...))
	headB := hex.EncodeToString(hB1[:])
	htmp = sha256.Sum256(append([]byte(headB), canonicalizeGossipForHash(b2)...))
	headB = hex.EncodeToString(htmp[:])
	// Ingest only Node A subset
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/v2/federation/gossip", IngestGossip)
	r.GET("/v2/federation/gossip", PullGossip)
	bodyA := map[string]any{"messages": []GossipMessage{a1, a2}}
	bbA, _ := json.Marshal(bodyA)
	reqA := httptest.NewRequest(http.MethodPost, "/v2/federation/gossip", bytes.NewReader(bbA))
	reqA.Header.Set("Content-Type", "application/json")
	wA := httptest.NewRecorder()
	r.ServeHTTP(wA, reqA)
	if wA.Code != http.StatusNoContent {
		t.Fatalf("ingest A status=%d", wA.Code)
	}
	// Pull head: should equal headA
	reqP := httptest.NewRequest(http.MethodGet, "/v2/federation/gossip?topic=revocation", nil)
	wP := httptest.NewRecorder()
	r.ServeHTTP(wP, reqP)
	var respA struct {
		Head string `json:"head"`
	}
	_ = json.Unmarshal(wP.Body.Bytes(), &respA)
	if respA.Head != headA {
		t.Fatalf("pre-merge head mismatch: got=%s want=%s", respA.Head, headA)
	}
	// Ingest Node B subset (reconciliation)
	bodyB := map[string]any{"messages": []GossipMessage{b1, b2}}
	bbB, _ := json.Marshal(bodyB)
	reqB := httptest.NewRequest(http.MethodPost, "/v2/federation/gossip", bytes.NewReader(bbB))
	reqB.Header.Set("Content-Type", "application/json")
	wB := httptest.NewRecorder()
	r.ServeHTTP(wB, reqB)
	if wB.Code != http.StatusNoContent {
		t.Fatalf("ingest B status=%d", wB.Code)
	}
	// Combined expected head
	cans := [][]byte{canonicalizeGossipForHash(a1), canonicalizeGossipForHash(a2), canonicalizeGossipForHash(b1), canonicalizeGossipForHash(b2)}
	hh := [32]byte{}
	for i, c := range cans {
		if i == 0 {
			hh = sha256.Sum256(append([]byte(""), c...))
		} else {
			hh = sha256.Sum256(append([]byte(hex.EncodeToString(hh[:])), c...))
		}
	}
	exp := hex.EncodeToString(hh[:])
	// Pull and assert convergence
	reqC := httptest.NewRequest(http.MethodGet, "/v2/federation/gossip?topic=revocation", nil)
	wC := httptest.NewRecorder()
	r.ServeHTTP(wC, reqC)
	var respC struct {
		Head string `json:"head"`
	}
	_ = json.Unmarshal(wC.Body.Bytes(), &respC)
	if respC.Head != exp {
		t.Fatalf("converged head mismatch: got=%s want=%s", respC.Head, exp)
	}
}
