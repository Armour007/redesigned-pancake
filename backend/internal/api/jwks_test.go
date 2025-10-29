package api

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http/httptest"
	"os"
	"regexp"
	"testing"

	database "github.com/Armour007/aura-backend/internal"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// helper: base64url
func b64url(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

func TestOrgJWKS_MultiKey_ReturnsBothKeys(t *testing.T) {
	// ensure env key path doesn't interfere
	os.Unsetenv("AURA_TRUST_ED25519_PRIVATE_KEY")

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()
	database.DB = sqlx.NewDb(db, "sqlmock")

	// two deterministic seeds
	seed1 := make([]byte, 32)
	seed2 := make([]byte, 32)
	for i := range seed2 {
		seed2[i] = 1
	}
	priv1 := ed25519.NewKeyFromSeed(seed1)
	pub1 := priv1.Public().(ed25519.PublicKey)
	priv2 := ed25519.NewKeyFromSeed(seed2)
	pub2 := priv2.Public().(ed25519.PublicKey)
	orgID := "11111111-1111-1111-1111-111111111111"

	// expected computed kid for second when empty
	sum2 := sha256.Sum256(pub2)
	expectedKid2 := b64url(sum2[:8])

	// mock rows for OrgJWKS query
	query := regexp.QuoteMeta(`SELECT ed25519_private_key_base64, COALESCE(kid,'') FROM trust_keys WHERE org_id=$1 AND active=true ORDER BY created_at DESC LIMIT 10`)
	rows := sqlmock.NewRows([]string{"ed25519_private_key_base64", "kid"}).
		AddRow(b64url(seed1), "kid-abc").
		AddRow(b64url(seed2), "")
	mock.ExpectQuery(query).WithArgs(orgID).WillReturnRows(rows)

	// gin test context
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "orgId", Value: orgID}}

	OrgJWKS(c)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var out struct{ Keys []struct{ Kid, X string } }
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if len(out.Keys) != 2 {
		t.Fatalf("expected 2 keys, got %d: %s", len(out.Keys), w.Body.String())
	}
	if out.Keys[0].Kid != "kid-abc" {
		t.Fatalf("expected kid-abc, got %q", out.Keys[0].Kid)
	}
	if out.Keys[0].X != b64url(pub1) {
		t.Fatalf("unexpected X for key0")
	}
	if out.Keys[1].Kid != expectedKid2 {
		t.Fatalf("expected computed kid %q, got %q", expectedKid2, out.Keys[1].Kid)
	}
	if out.Keys[1].X != b64url(pub2) {
		t.Fatalf("unexpected X for key1")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestVerifyEdDSA_KidSelection_FromDB(t *testing.T) {
	// ensure env key path doesn't interfere
	os.Unsetenv("AURA_TRUST_ED25519_PRIVATE_KEY")

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()
	database.DB = sqlx.NewDb(db, "sqlmock")

	// deterministic key and kid
	seed := make([]byte, 32)
	for i := range seed {
		seed[i] = 7
	}
	priv := ed25519.NewKeyFromSeed(seed)
	pub := priv.Public().(ed25519.PublicKey)
	_ = pub // not used directly, but here if needed
	kid := "kid-db"

	// Expect query to fetch by kid and return the seed
	getQ := regexp.QuoteMeta(`SELECT ed25519_private_key_base64 FROM trust_keys WHERE kid=$1 AND active=true ORDER BY created_at DESC LIMIT 1`)
	mock.ExpectQuery(getQ).WithArgs(kid).WillReturnRows(sqlmock.NewRows([]string{"ed25519_private_key_base64"}).AddRow(b64url(seed)))

	// Build a compact JWS with kid and valid signature
	header := b64url([]byte(`{"alg":"EdDSA","typ":"JWT","kid":"` + kid + `"}`))
	payload := b64url([]byte(`{"exp":4102444800}`)) // year 2100
	unsigned := header + "." + payload
	sig := ed25519.Sign(priv, []byte(unsigned))
	sigB64 := b64url(sig)

	if ok := verifyEdDSA(unsigned, sigB64, kid); !ok {
		t.Fatalf("expected verify true with correct kid")
	}

	// Wrong kid should fail; mock query returns no rows
	mock.ExpectQuery(getQ).WithArgs("kid-other").WillReturnError(os.ErrNotExist)
	if ok := verifyEdDSA(unsigned, sigB64, "kid-other"); ok {
		t.Fatalf("expected verify false with wrong kid")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
