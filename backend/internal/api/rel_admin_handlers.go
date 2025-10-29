package api

import (
	"fmt"
	"net/http"
	"strconv"

	database "github.com/Armour007/aura-backend/internal"
	"github.com/Armour007/aura-backend/internal/audit"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// GET /admin/rel/tuples?object_ns=&object_id=&relation=&subject_ns=&subject_id=&limit=100
func AdminListTuples(c *gin.Context) {
	q := `SELECT object_type, object_id, relation, subject_type, subject_id FROM trust_tuples`
	where := []string{}
	args := []any{}
	add := func(col, val string) {
		if val != "" {
			where = append(where, col+"=$"+itoa(len(args)+1))
			args = append(args, val)
		}
	}
	add("object_type", c.Query("object_ns"))
	add("object_id", c.Query("object_id"))
	add("relation", c.Query("relation"))
	add("subject_type", c.Query("subject_ns"))
	add("subject_id", c.Query("subject_id"))
	if len(where) > 0 {
		q += " WHERE " + join(where, " AND ")
	}
	q += " ORDER BY object_type, object_id, relation LIMIT $" + itoa(len(args)+1)
	limit := 100
	if v := c.Query("limit"); v != "" {
		if n, err := atoi(v); err == nil {
			limit = n
		}
	}
	args = append(args, limit)
	rows, err := database.DB.Queryx(q, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	type row struct{ ObjectType, ObjectID, Relation, SubjectType, SubjectID string }
	out := []row{}
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.ObjectType, &r.ObjectID, &r.Relation, &r.SubjectType, &r.SubjectID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		out = append(out, r)
	}
	c.JSON(http.StatusOK, out)
}

// DELETE /admin/rel/tuples?confirm=true&object_ns=&object_id=&relation=&subject_ns=&subject_id=
func AdminDeleteTuples(c *gin.Context) {
	confirm := c.Query("confirm") == "true"
	where := []string{}
	args := []any{}
	add := func(col, val string) {
		if val != "" {
			where = append(where, col+"=$"+itoa(len(args)+1))
			args = append(args, val)
		}
	}
	add("object_type", c.Query("object_ns"))
	add("object_id", c.Query("object_id"))
	add("relation", c.Query("relation"))
	add("subject_type", c.Query("subject_ns"))
	add("subject_id", c.Query("subject_id"))
	if len(where) == 0 && !confirm {
		c.JSON(http.StatusBadRequest, gin.H{"error": "confirm=true required to delete all"})
		return
	}
	q := "DELETE FROM trust_tuples"
	if len(where) > 0 {
		q += " WHERE " + join(where, " AND ")
	}
	if _, err := database.DB.Exec(q, args...); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// Invalidate caches and broadcast graph invalidation
	ClearGraphCache()
	PublishGraphInvalidate(c.Request.Context())
	// Audit deletion with filter context
	_ = audit.Append(c.Request.Context(), uuid.Nil, "rel_delete", gin.H{
		"object_ns":  c.Query("object_ns"),
		"object_id":  c.Query("object_id"),
		"relation":   c.Query("relation"),
		"subject_ns": c.Query("subject_ns"),
		"subject_id": c.Query("subject_id"),
	}, nil, nil)
	c.Status(http.StatusNoContent)
}

// tiny helpers to avoid extra imports
func itoa(i int) string          { return fmt.Sprintf("%d", i) }
func atoi(s string) (int, error) { return strconv.Atoi(s) }
func join(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	out := parts[0]
	for i := 1; i < len(parts); i++ {
		out += sep + parts[i]
	}
	return out
}
