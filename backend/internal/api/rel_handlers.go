package api

import (
	"context"
	"net/http"

	"github.com/Armour007/aura-backend/internal/audit"
	"github.com/Armour007/aura-backend/internal/rel"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var relStore = rel.NewStore() // in-memory mirror
var relDB = rel.TupleDB{}

type tupleReq struct {
	Tuples []rel.Tuple `json:"tuples"`
}

func UpsertTuples(c *gin.Context) {
	var req tupleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// write-through DB and mirror to memory
	if err := relDB.Upsert(c.Request.Context(), req.Tuples); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	relStore.Upsert(req.Tuples)
	// invalidate cached checks
	ClearGraphCache()
	// publish graph invalidation across mesh
	PublishGraphInvalidate(c.Request.Context())
	// audit
	orgID := c.Param("orgId")
	_ = audit.Append(c.Request.Context(), uuid.MustParse(orgID), "rel_upsert", gin.H{"tuples": req.Tuples}, nil, nil)
	c.Status(http.StatusNoContent)
}

type checkReq struct {
	SubjectType string `json:"subject_type"`
	SubjectID   string `json:"subject_id"`
	Relation    string `json:"relation"`
	ObjectType  string `json:"object_type"`
	ObjectID    string `json:"object_id"`
}

func CheckRelation(c *gin.Context) {
	var req checkReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if ok, err := relDB.Check(context.Background(), req.SubjectType, req.SubjectID, req.Relation, req.ObjectType, req.ObjectID); err == nil {
		c.JSON(http.StatusOK, gin.H{"allowed": ok})
		return
	}
	ok := relStore.Check(req.SubjectType, req.SubjectID, req.Relation, req.ObjectType, req.ObjectID)
	c.JSON(http.StatusOK, gin.H{"allowed": ok})
}
