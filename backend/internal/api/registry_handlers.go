package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	database "github.com/Armour007/aura-backend/internal"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

type createRegistryOrgReq struct {
	Name         string      `json:"name"`
	Domain       string      `json:"domain"`
	JWKSURL      string      `json:"jwks_url"`
	Status       string      `json:"status"`
	Attestations interface{} `json:"attestations"`
}

// POST /v2/registry/orgs
func CreateRegistryOrg(c *gin.Context) {
	orgID := c.GetString("orgID")
	var req createRegistryOrgReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Status == "" {
		req.Status = "pending"
	}
	var atj json.RawMessage
	if req.Attestations != nil {
		atb, _ := json.Marshal(req.Attestations)
		atj = atb
	}
	row := database.DB.QueryRowx(`INSERT INTO registry_orgs(org_id, name, domain, jwks_url, attestations, status) VALUES ($1,$2,$3,$4,$5,$6) RETURNING id::text, created_at::text`, orgID, req.Name, req.Domain, req.JWKSURL, atj, req.Status)
	var id, created string
	if err := row.Scan(&id, &created); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "org_id": orgID, "name": req.Name, "domain": req.Domain, "jwks_url": req.JWKSURL, "status": req.Status, "created_at": created})
}

// GET /v2/registry/orgs
func ListRegistryOrgs(c *gin.Context) {
	rows, err := database.DB.Queryx(`SELECT id::text, org_id::text, name, domain, jwks_url, attestations, status, created_at::text FROM registry_orgs ORDER BY created_at DESC LIMIT 1000`)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id, org, name, domain, jwks, status, created string
		var at any
		if err := rows.Scan(&id, &org, &name, &domain, &jwks, &at, &status, &created); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		items = append(items, gin.H{"id": id, "org_id": org, "name": name, "domain": domain, "jwks_url": jwks, "attestations": at, "status": status, "created_at": created})
	}
	c.JSON(200, gin.H{"orgs": items})
}

type createRegistryAgentReq struct {
	AgentID      string      `json:"agent_id"`
	Name         string      `json:"name"`
	Status       string      `json:"status"`
	Attestations interface{} `json:"attestations"`
}

// POST /v2/registry/agents
func CreateRegistryAgent(c *gin.Context) {
	orgID := c.GetString("orgID")
	var req createRegistryAgentReq
	if err := c.ShouldBindJSON(&req); err != nil || req.AgentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "agent_id required"})
		return
	}
	if req.Status == "" {
		req.Status = "pending"
	}
	var atj json.RawMessage
	if req.Attestations != nil {
		b, _ := json.Marshal(req.Attestations)
		atj = b
	}
	row := database.DB.QueryRowx(`INSERT INTO registry_agents(org_id, agent_id, name, attestations, status) VALUES ($1,$2,$3,$4,$5) RETURNING id::text, created_at::text`, orgID, req.AgentID, req.Name, atj, req.Status)
	var id, created string
	if err := row.Scan(&id, &created); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(201, gin.H{"id": id, "org_id": orgID, "agent_id": req.AgentID, "name": req.Name, "status": req.Status, "created_at": created})
}

// GET /v2/registry/agents
func ListRegistryAgents(c *gin.Context) {
	orgFilter := c.Query("org_id")
	var rows *sqlx.Rows
	var err error
	if orgFilter != "" {
		rows, err = database.DB.Queryx(`SELECT id::text, org_id::text, agent_id::text, name, attestations, status, created_at::text, COALESCE(last_seen_at::text,'') FROM registry_agents WHERE org_id=$1 ORDER BY created_at DESC LIMIT 1000`, orgFilter)
	} else {
		rows, err = database.DB.Queryx(`SELECT id::text, org_id::text, agent_id::text, name, attestations, status, created_at::text, COALESCE(last_seen_at::text,'') FROM registry_agents ORDER BY created_at DESC LIMIT 1000`)
	}
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id, org, agent, name, status, created, lastSeen string
		var at any
		if err := rows.Scan(&id, &org, &agent, &name, &at, &status, &created, &lastSeen); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		items = append(items, gin.H{"id": id, "org_id": org, "agent_id": agent, "name": name, "attestations": at, "status": status, "created_at": created, "last_seen_at": lastSeen})
	}
	c.JSON(200, gin.H{"agents": items})
}

// --- Public, read-only, paginated listing for discovery ---

func clampLimit(n, def, max int) int {
	if n <= 0 {
		return def
	}
	if n > max {
		return max
	}
	return n
}

// GET /registry/public/orgs?limit=50&page=1
func PublicListRegistryOrgs(c *gin.Context) {
	limit := clampLimit(atoiDefault(c.Query("limit"), 50), 50, 200)
	page := atoiDefault(c.Query("page"), 1)
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * limit
	rows, err := database.DB.Queryx(`SELECT id::text, org_id::text, name, domain, jwks_url, status, created_at::text FROM registry_orgs ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id, org, name, domain, jwks, status, created string
		if err := rows.Scan(&id, &org, &name, &domain, &jwks, &status, &created); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		items = append(items, gin.H{"id": id, "org_id": org, "name": name, "domain": domain, "jwks_url": jwks, "status": status, "created_at": created})
	}
	c.JSON(200, gin.H{"items": items, "limit": limit, "page": page})
}

// GET /registry/public/agents?limit=50&page=1&org_id=...
func PublicListRegistryAgents(c *gin.Context) {
	limit := clampLimit(atoiDefault(c.Query("limit"), 50), 50, 200)
	page := atoiDefault(c.Query("page"), 1)
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * limit
	orgFilter := c.Query("org_id")
	var rows *sqlx.Rows
	var err error
	if orgFilter != "" {
		rows, err = database.DB.Queryx(`SELECT id::text, org_id::text, agent_id::text, name, status, created_at::text, COALESCE(last_seen_at::text,'') FROM registry_agents WHERE org_id=$1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`, orgFilter, limit, offset)
	} else {
		rows, err = database.DB.Queryx(`SELECT id::text, org_id::text, agent_id::text, name, status, created_at::text, COALESCE(last_seen_at::text,'') FROM registry_agents ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	}
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id, org, agent, name, status, created, lastSeen string
		if err := rows.Scan(&id, &org, &agent, &name, &status, &created, &lastSeen); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		items = append(items, gin.H{"id": id, "org_id": org, "agent_id": agent, "name": name, "status": status, "created_at": created, "last_seen_at": lastSeen})
	}
	c.JSON(200, gin.H{"items": items, "limit": limit, "page": page})
}

func atoiDefault(s string, def int) int {
	var n int
	if _, err := fmt.Sscanf(s, "%d", &n); err != nil {
		return def
	}
	return n
}
