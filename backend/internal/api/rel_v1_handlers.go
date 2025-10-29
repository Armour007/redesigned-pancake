package api

import (
	"net/http"

	"github.com/Armour007/aura-backend/internal/rel"
	"github.com/gin-gonic/gin"
)

type upsertTuplesReq struct {
	Tuples []rel.Tuple `json:"tuples" binding:"required"`
}

// POST /v1/tuples
func UpsertTuplesV1(c *gin.Context) {
	var req upsertTuplesReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(req.Tuples) == 0 {
		c.Status(http.StatusNoContent)
		return
	}
	if err := getGraph().UpsertBatch(c.Request.Context(), req.Tuples); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// invalidate cache on write
	ClearGraphCache()
	// publish graph invalidation across mesh
	PublishGraphInvalidate(c.Request.Context())
	c.Status(http.StatusNoContent)
}

type checkReqV1 struct {
	Subject  rel.RelationRef `json:"subject" binding:"required"`
	Relation string          `json:"relation" binding:"required"`
	Object   rel.RelationRef `json:"object" binding:"required"`
}
type checkRespV1 struct {
	Allowed bool   `json:"allowed"`
	Source  string `json:"source"`
}

// POST /v1/check
func CheckRelationV1(c *gin.Context) {
	var req checkReqV1
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	allowed, source, err := getGraph().Check(c.Request.Context(), req.Subject, req.Relation, req.Object)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, checkRespV1{Allowed: allowed, Source: source})
}

// GET /v1/trust/graph/expand?object=team:devs&relation=member
func ExpandTrustGraphV1(c *gin.Context) {
	object := c.Query("object")
	relation := c.Query("relation")
	if object == "" || relation == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "object and relation required"})
		return
	}
	// parse object in ns:id format
	var ns, id string
	for i := 0; i < len(object); i++ {
		if object[i] == ':' {
			ns = object[:i]
			id = object[i+1:]
			break
		}
	}
	if ns == "" || id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "object must be ns:id"})
		return
	}
	exp, err := getGraph().Expand(c.Request.Context(), relation, rel.RelationRef{Namespace: ns, ObjectID: id}, 1)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, exp)
}
