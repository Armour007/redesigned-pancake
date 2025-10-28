package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"regexp"

	database "github.com/Armour007/aura-backend/internal"
	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Test that accessing an agent's permissions with a mismatched org is forbidden
func TestGetPermissionRules_TenantGuard(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Prepare sqlmock
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("error creating sqlmock: %v", err)
	}
	defer db.Close()
	database.DB = sqlx.NewDb(db, "sqlmock")

	// IDs
	orgID := uuid.New()
	agentID := uuid.New()

	// Expect tenancy check query to return count=0 (agent not in org)
	rows := sqlmock.NewRows([]string{"count"}).AddRow(0)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(1) FROM agents WHERE id=$1 AND organization_id=$2")).
		WithArgs(agentID, orgID).
		WillReturnRows(rows)

	// Router and route
	r := gin.New()
	r.GET("/organizations/:orgId/agents/:agentId/permissions", func(c *gin.Context) {
		// Inject a dummy userID as expected by handler
		c.Set("userID", uuid.New().String())
		GetPermissionRules(c)
	})

	req := httptest.NewRequest("GET", "/organizations/"+orgID.String()+"/agents/"+agentID.String()+"/permissions", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden, got %d, body=%s", w.Code, w.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}
