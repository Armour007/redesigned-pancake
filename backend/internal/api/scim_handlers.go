package api

import (
	"net/http"
	"os"
	"strings"

	database "github.com/Armour007/aura-backend/internal"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// SCIM token-based simple auth
func scimAuth(c *gin.Context) bool {
	tok := os.Getenv("AURA_SCIM_TOKEN")
	if strings.TrimSpace(tok) == "" {
		return false
	}
	auth := c.GetHeader("Authorization")
	if auth == "Bearer "+tok || auth == "Basic "+tok {
		return true
	}
	return false
}

// GET /scim/v2/Users
func SCIMListUsers(c *gin.Context) {
	if !scimAuth(c) {
		c.Header("WWW-Authenticate", "Bearer")
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	orgID := c.Query("orgId")
	if orgID == "" {
		c.JSON(400, gin.H{"error": "missing orgId"})
		return
	}
	type row struct {
		ID    string `db:"id"`
		Email string `db:"email"`
		Full  string `db:"full_name"`
	}
	items := []row{}
	_ = database.DB.Select(&items, `SELECT u.id::text as id, u.email, u.full_name FROM users u JOIN organization_members m ON m.user_id=u.id WHERE m.organization_id=$1`, orgID)
	c.JSON(200, gin.H{"Resources": items, "totalResults": len(items)})
}

// POST /scim/v2/Users
func SCIMCreateUser(c *gin.Context) {
	if !scimAuth(c) {
		c.Header("WWW-Authenticate", "Bearer")
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	if os.Getenv("AURA_SCIM_ENABLE") != "1" {
		c.JSON(501, gin.H{"error": "SCIM not enabled"})
		return
	}
	orgID := c.Query("orgId")
	if orgID == "" {
		c.JSON(400, gin.H{"error": "missing orgId"})
		return
	}
	var body struct {
		UserName string `json:"userName"`
		Name     struct {
			GivenName  string `json:"givenName"`
			FamilyName string `json:"familyName"`
		} `json:"name"`
		Active *bool `json:"active"`
		Groups []struct {
			Display string `json:"display"`
			Value   string `json:"value"`
		} `json:"groups"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(400, gin.H{"error": "bad json"})
		return
	}
	email := strings.ToLower(strings.TrimSpace(body.UserName))
	if email == "" {
		c.JSON(400, gin.H{"error": "userName required"})
		return
	}
	// Upsert user
	var uid uuid.UUID
	_ = database.DB.Get(&uid, `SELECT id FROM users WHERE email=$1`, email)
	if uid == uuid.Nil {
		uid = uuid.New()
		_, _ = database.DB.Exec(`INSERT INTO users(id, full_name, email, hashed_password, created_at, updated_at) VALUES($1,$2,$3,'',NOW(),NOW())`, uid, body.Name.GivenName+" "+body.Name.FamilyName, email)
	}
	// Ensure membership with role from env (default member)
	role := os.Getenv("AURA_SCIM_DEFAULT_ROLE")
	if role == "" {
		role = "member"
	}
	// If groups provided, prefer a role mapping from first recognized group
	for _, g := range body.Groups {
		v := strings.ToLower(strings.TrimSpace(g.Display))
		switch v {
		case "owner", "admin", "auditor", "read-only", "member":
			role = v
			break
		}
	}
	_, _ = database.DB.Exec(`INSERT INTO organization_members(organization_id, user_id, role, joined_at) VALUES($1,$2,$3,NOW()) ON CONFLICT (organization_id,user_id) DO UPDATE SET role=EXCLUDED.role`, orgID, uid, role)
	c.JSON(201, gin.H{"id": uid.String(), "userName": email})
}

// Placeholder groups endpoints
func SCIMListGroups(c *gin.Context) {
	if !scimAuth(c) {
		c.Header("WWW-Authenticate", "Bearer")
		c.AbortWithStatus(401)
		return
	}
	orgID := c.Query("orgId")
	if orgID == "" {
		c.JSON(400, gin.H{"error": "missing orgId"})
		return
	}
	// Return default role-based groups with member counts
	roles := []string{"owner", "admin", "auditor", "read-only", "member"}
	type grp struct {
		ID      string `json:"id"`
		Display string `json:"display"`
		Members int    `json:"members"`
	}
	res := []grp{}
	for _, r := range roles {
		var cnt int
		_ = database.DB.Get(&cnt, `SELECT COUNT(1) FROM organization_members WHERE organization_id=$1 AND role=$2`, orgID, r)
		res = append(res, grp{ID: r, Display: r, Members: cnt})
	}
	c.JSON(200, gin.H{"Resources": res, "totalResults": len(res)})
}

// PATCH /scim/v2/Users/:id — handle { active: false } to deprovision/reprovision membership
func SCIMPatchUser(c *gin.Context) {
	if !scimAuth(c) {
		c.Header("WWW-Authenticate", "Bearer")
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	if os.Getenv("AURA_SCIM_ENABLE") != "1" {
		c.JSON(501, gin.H{"error": "SCIM not enabled"})
		return
	}
	orgID := c.Query("orgId")
	if orgID == "" {
		c.JSON(400, gin.H{"error": "missing orgId"})
		return
	}
	uid := c.Param("id")
	if _, err := uuid.Parse(uid); err != nil {
		c.JSON(400, gin.H{"error": "invalid user id"})
		return
	}
	var body struct {
		Active *bool `json:"active"`
		// Optional role change via SCIM PATCH
		Role *string `json:"role"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(400, gin.H{"error": "bad json"})
		return
	}
	if body.Active != nil && *body.Active == false {
		// Deprovision: remove org membership
		_, _ = database.DB.Exec(`DELETE FROM organization_members WHERE organization_id=$1 AND user_id=$2`, orgID, uid)
		c.JSON(200, gin.H{"id": uid, "active": false})
		return
	}
	if body.Active != nil && *body.Active == true {
		// Reprovision: ensure membership with default role (member)
		role := "member"
		if body.Role != nil && *body.Role != "" {
			role = strings.ToLower(*body.Role)
		}
		_, _ = database.DB.Exec(`INSERT INTO organization_members(organization_id, user_id, role, joined_at) VALUES($1,$2,$3,NOW()) ON CONFLICT (organization_id,user_id) DO UPDATE SET role=EXCLUDED.role`, orgID, uid, role)
		c.JSON(200, gin.H{"id": uid, "active": true, "role": role})
		return
	}
	c.JSON(400, gin.H{"error": "no actionable changes"})
}
