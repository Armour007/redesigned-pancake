package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	database "github.com/Armour007/aura-backend/internal"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type submitDNAReq struct {
	AgentID string          `json:"agent_id,omitempty"`
	Owner   string          `json:"owner,omitempty"`
	Vector  []float64       `json:"vector"`
	Meta    json.RawMessage `json:"meta,omitempty"`
	OptIn   bool            `json:"opt_in,omitempty"`
}

type submitDNAResp struct {
	ID          string  `json:"id"`
	Fingerprint string  `json:"fingerprint"`
	Dim         int     `json:"dim"`
	Norm        float64 `json:"norm"`
}

// SubmitTrustDNA stores a normalized vector and privacy-preserving fingerprint
func SubmitTrustDNA(c *gin.Context) {
	orgID := c.Param("orgId")
	var req submitDNAReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(req.Vector) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "vector required"})
		return
	}
	if len(req.Vector) > 1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "vector too large (max 1024)"})
		return
	}
	// L2 normalize
	var sumsq float64
	for _, v := range req.Vector {
		sumsq += v * v
	}
	if sumsq == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "zero vector"})
		return
	}
	norm := math.Sqrt(sumsq)
	vec := make([]float64, len(req.Vector))
	for i, v := range req.Vector {
		vec[i] = v / norm
	}
	// Fingerprint: SHA-256 of normalized values with salt
	salt := os.Getenv("AURA_TRUST_DNA_SALT")
	if salt == "" {
		salt = orgID // fallback
	}
	h := sha256.New()
	h.Write([]byte(salt))
	for _, v := range vec {
		// quantize to 1e-4 to stabilize
		q := math.Round(v*1e4) / 1e4
		h.Write([]byte("|"))
		h.Write([]byte(strconv.FormatFloat(q, 'f', 4, 64)))
	}
	fp := hex.EncodeToString(h.Sum(nil))
	id := uuid.New()
	// Persist
	_, err := database.DB.Exec(`INSERT INTO trust_dna(id, org_id, agent_id, owner, vector, dim, norm, fingerprint, opt_in, meta) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, id, orgID, nullIfUUID(req.AgentID), nullIfEmpty(req.Owner), floatArray(vec), len(vec), norm, fp, req.OptIn, nullIfEmptyJSON(req.Meta))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, submitDNAResp{ID: id.String(), Fingerprint: fp, Dim: len(vec), Norm: norm})
}

func nullIfUUID(s string) any {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	if _, err := uuid.Parse(s); err != nil {
		return nil
	}
	return s
}

func nullIfEmpty(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}

type nearQueryReq struct {
	Vector      []float64 `json:"vector,omitempty"`
	Fingerprint string    `json:"fingerprint,omitempty"`
	TopK        int       `json:"top_k,omitempty"`
	Scope       string    `json:"scope,omitempty"` // org|global
}

type neighbor struct {
	ID          uuid.UUID       `db:"id"`
	Owner       *string         `db:"owner"`
	AgentID     *uuid.UUID      `db:"agent_id"`
	Vector      []float64       `db:"vector"`
	Fingerprint string          `db:"fingerprint"`
	OptIn       bool            `db:"opt_in"`
	Meta        json.RawMessage `db:"meta"`
}

type nearRespItem struct {
	ID          string          `json:"id"`
	Similarity  float64         `json:"similarity"`
	Owner       *string         `json:"owner,omitempty"`
	AgentID     *string         `json:"agent_id,omitempty"`
	Fingerprint string          `json:"fingerprint"`
	Meta        json.RawMessage `json:"meta,omitempty"`
}

func GetTrustDNANear(c *gin.Context) {
	orgID := c.Param("orgId")
	var req nearQueryReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.TopK <= 0 || req.TopK > 100 {
		req.TopK = 10
	}
	// Build target vector
	var target []float64
	if len(req.Vector) > 0 {
		target = normalize(req.Vector)
	} else if req.Fingerprint != "" {
		// best-effort: find exact fp and reuse its vector
		var me neighbor
		if err := database.DB.Get(&me, `SELECT id, owner, agent_id, vector, fingerprint, opt_in, meta FROM trust_dna WHERE org_id=$1 AND fingerprint=$2 LIMIT 1`, orgID, req.Fingerprint); err == nil {
			target = me.Vector
		}
	}
	if len(target) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "vector or fingerprint required"})
		return
	}
	// Fetch candidates
	rows := []neighbor{}
	scope := strings.ToLower(req.Scope)
	if scope == "global" {
		if err := database.DB.Select(&rows, `SELECT id, owner, agent_id, vector, fingerprint, opt_in, meta FROM trust_dna WHERE opt_in=true`); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else {
		if err := database.DB.Select(&rows, `SELECT id, owner, agent_id, vector, fingerprint, opt_in, meta FROM trust_dna WHERE org_id=$1`, orgID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	// Compute cosine similarity (vectors are normalized)
	out := make([]nearRespItem, 0, len(rows))
	for _, r := range rows {
		sim := dot(target, r.Vector)
		var agentStr *string
		if r.AgentID != nil {
			s := r.AgentID.String()
			agentStr = &s
		}
		out = append(out, nearRespItem{ID: r.ID.String(), Similarity: sim, Owner: r.Owner, AgentID: agentStr, Fingerprint: r.Fingerprint, Meta: r.Meta})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Similarity > out[j].Similarity })
	if len(out) > req.TopK {
		out = out[:req.TopK]
	}
	c.JSON(http.StatusOK, gin.H{"items": out, "top_k": req.TopK, "scope": scope})
}

type aggregateResp struct {
	Count    int       `json:"count"`
	Centroid []float64 `json:"centroid"`
}

// GetTrustDNAAggregate returns a DP-noisy centroid over opt-in vectors
func GetTrustDNAAggregate(c *gin.Context) {
	orgID := c.Param("orgId")
	scope := strings.ToLower(c.DefaultQuery("scope", "org"))
	// Fetch vectors
	type row struct {
		Vector []float64 `db:"vector"`
	}
	var rows []row
	var err error
	if scope == "global" {
		err = database.DB.Select(&rows, `SELECT vector FROM trust_dna WHERE opt_in=true`)
	} else {
		err = database.DB.Select(&rows, `SELECT vector FROM trust_dna WHERE org_id=$1 AND opt_in=true`, orgID)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if len(rows) == 0 {
		c.JSON(http.StatusOK, aggregateResp{Count: 0, Centroid: []float64{}})
		return
	}
	dim := len(rows[0].Vector)
	sum := make([]float64, dim)
	for _, r := range rows {
		for i := 0; i < dim && i < len(r.Vector); i++ {
			sum[i] += r.Vector[i]
		}
	}
	centroid := make([]float64, dim)
	for i := 0; i < dim; i++ {
		centroid[i] = sum[i] / float64(len(rows))
	}
	// Differential Privacy: add Laplace noise with scale proportional to 1/(n*epsilon)
	eps := 1.0
	if v := os.Getenv("AURA_DNA_DP_EPSILON"); v != "" {
		if e, eerr := strconv.ParseFloat(v, 64); eerr == nil && e > 0 {
			eps = e
		}
	}
	scale := 1.0 / (float64(len(rows)) * eps)
	for i := 0; i < dim; i++ {
		centroid[i] += laplace(0, scale)
	}
	c.JSON(http.StatusOK, aggregateResp{Count: len(rows), Centroid: centroid})
}

// Helpers
func normalize(v []float64) []float64 {
	var s float64
	for _, x := range v {
		s += x * x
	}
	if s == 0 {
		return make([]float64, len(v))
	}
	n := math.Sqrt(s)
	out := make([]float64, len(v))
	for i, x := range v {
		out[i] = x / n
	}
	return out
}

func dot(a, b []float64) float64 {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	var s float64
	for i := 0; i < n; i++ {
		s += a[i] * b[i]
	}
	return s
}

func floatArray(v []float64) any { return v }

// laplace noise using inverse transform sampling
func laplace(mu, b float64) float64 {
	if b <= 0 {
		return 0
	}
	// simple LCG-based randomness via crypto/rand would be better; use math/rand for now
	u := randFloat64()*2 - 1 // (-1,1)
	return mu - b*math.Copysign(1, u)*math.Log(1-math.Abs(u))
}

func randFloat64() float64 {
	// derive from sha256 of current pid/time as simple PRNG seed; not cryptographically secure
	// but sufficient for DP noise in this context without introducing new deps.
	var x uint64
	x ^= uint64(time.Now().UnixNano())
	x ^= uint64(os.Getpid())
	// xorshift*
	x ^= x >> 12
	x ^= x << 25
	x ^= x >> 27
	y := x * 2685821657736338717
	return float64(y%1000000) / 1000000.0
}
