package api

import (
	"crypto/ed25519"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"net/http"
	"time"

	"math/big"

	database "github.com/Armour007/aura-backend/internal"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// GET /v2/certs — list client certs for org
func ListClientCerts(c *gin.Context) {
	orgIDStr := c.GetString("orgID")
	if orgIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing org"})
		return
	}
	orgID := uuid.MustParse(orgIDStr)
	type row struct {
		Serial, Subject     string
		DeviceID            *uuid.UUID
		Revoked             bool
		NotBefore, NotAfter time.Time
	}
	rows := []row{}
	_ = database.DB.Select(&rows, `SELECT serial, subject, device_id, revoked, not_before, not_after FROM client_certs WHERE org_id=$1 ORDER BY created_at DESC`, orgID)
	items := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		items = append(items, map[string]any{
			"serial":     r.Serial,
			"subject":    r.Subject,
			"device_id":  r.DeviceID,
			"revoked":    r.Revoked,
			"not_before": r.NotBefore,
			"not_after":  r.NotAfter,
		})
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

// POST /v2/certs/:serial/revoke — mark cert revoked
func RevokeClientCert(c *gin.Context) {
	orgIDStr := c.GetString("orgID")
	if orgIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing org"})
		return
	}
	orgID := uuid.MustParse(orgIDStr)
	serial := c.Param("serial")
	res, err := database.DB.Exec(`UPDATE client_certs SET revoked=true WHERE serial=$1 AND org_id=$2 AND revoked=false`, serial, orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.Status(http.StatusNoContent)
}

// GET /v2/certs/crl.pem — optional CRL in dev if CA key is present
func GetCRL(c *gin.Context) {
	orgIDStr := c.GetString("orgID")
	if orgIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing org"})
		return
	}
	orgID := uuid.MustParse(orgIDStr)
	var certPEM, keyPEM string
	if err := database.DB.QueryRowx(`SELECT cert_pem, key_pem FROM org_client_ca WHERE org_id=$1 AND active=true`, orgID).Scan(&certPEM, &keyPEM); err != nil || certPEM == "" || keyPEM == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "crl unavailable"})
		return
	}
	cblk, _ := pem.Decode([]byte(certPEM))
	if cblk == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "bad ca cert"})
		return
	}
	caCert, err := x509.ParseCertificate(cblk.Bytes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	kblk, _ := pem.Decode([]byte(keyPEM))
	if kblk == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "bad ca key"})
		return
	}
	keyAny, err := x509.ParsePKCS8PrivateKey(kblk.Bytes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	caKey, ok := keyAny.(ed25519.PrivateKey)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unsupported ca key"})
		return
	}

	// Gather revoked serials
	type rrow struct {
		Serial   string
		NotAfter time.Time
	}
	rlist := []rrow{}
	_ = database.DB.Select(&rlist, `SELECT serial, not_after FROM client_certs WHERE org_id=$1 AND revoked=true`, orgID)

	// Create a minimal CRL
	now := time.Now()
	revoked := make([]pkix.RevokedCertificate, 0, len(rlist))
	for _, rr := range rlist {
		sn := new(big.Int)
		sn.SetString(rr.Serial, 16)
		revoked = append(revoked, pkix.RevokedCertificate{SerialNumber: sn, RevocationTime: now})
	}
	crlBytes, err := x509.CreateRevocationList(nil, &x509.RevocationList{SignatureAlgorithm: x509.PureEd25519, Number: new(big.Int).SetInt64(now.Unix()), ThisUpdate: now, NextUpdate: now.Add(24 * time.Hour), RevokedCertificates: revoked}, caCert, caKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	crlPEM := pem.EncodeToMemory(&pem.Block{Type: "X509 CRL", Bytes: crlBytes})
	c.Data(http.StatusOK, "application/pkix-crl", crlPEM)
}
