package api

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Armour007/aura-backend/internal/audit"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AgentCSRRequest struct {
	CSRPem   string      `json:"csr_pem"`
	Evidence interface{} `json:"evidence,omitempty"` // TPM/TEE evidence (pass-through MVP)
}

type AgentCSRResponse struct {
	CertPEM string `json:"cert_pem"`
	CaPEM   string `json:"ca_pem"`
	Expires int64  `json:"expires_at"`
}

func loadCAFromEnv() (caCert *x509.Certificate, caKey interface{}, caPEM string, err error) {
	certPath := os.Getenv("AURA_AGENT_CA_CERT_FILE")
	keyPath := os.Getenv("AURA_AGENT_CA_KEY_FILE")
	if certPath == "" || keyPath == "" {
		return nil, nil, "", errors.New("CA cert/key not configured")
	}
	certBytes, err := os.ReadFile(certPath)
	if err != nil {
		return nil, nil, "", err
	}
	keyBytes, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, nil, "", err
	}
	var blk *pem.Block
	blk, _ = pem.Decode(certBytes)
	if blk == nil {
		return nil, nil, "", errors.New("failed to decode CA cert")
	}
	crt, err := x509.ParseCertificate(blk.Bytes)
	if err != nil {
		return nil, nil, "", err
	}

	// Try PKCS8, then EC, then RSA
	var key interface{}
	if kblk, _ := pem.Decode(keyBytes); kblk != nil {
		if k, e := x509.ParsePKCS8PrivateKey(kblk.Bytes); e == nil {
			key = k
		} else if k, e := x509.ParseECPrivateKey(kblk.Bytes); e == nil {
			key = k
		} else if k, e := x509.ParsePKCS1PrivateKey(kblk.Bytes); e == nil {
			key = k
		} else {
			return nil, nil, "", errors.New("failed to parse CA private key")
		}
	} else {
		return nil, nil, "", errors.New("failed to decode CA key")
	}
	return crt, key, string(certBytes), nil
}

func signCSR(csr *x509.CertificateRequest, ca *x509.Certificate, caKey interface{}, agentID string) ([]byte, time.Time, error) {
	// Build leaf cert template for client auth
	now := time.Now().UTC()
	ttlHours := 24
	if v := os.Getenv("AURA_AGENT_CERT_TTL_HOURS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			ttlHours = n
		}
	}
	tmpl := &x509.Certificate{
		SerialNumber:   bigRand(),
		Subject:        csr.Subject,
		NotBefore:      now.Add(-time.Minute),
		NotAfter:       now.Add(time.Duration(ttlHours) * time.Hour),
		KeyUsage:       x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:    []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		DNSNames:       csr.DNSNames,
		EmailAddresses: csr.EmailAddresses,
		IPAddresses:    csr.IPAddresses,
		URIs:           csr.URIs,
	}
	// Ensure CN is agentID (MVP binding)
	tmpl.Subject.CommonName = agentID
	var der []byte
	var err error
	switch k := caKey.(type) {
	case *rsa.PrivateKey:
		der, err = x509.CreateCertificate(rand.Reader, tmpl, ca, csr.PublicKey, k)
	case *ecdsa.PrivateKey:
		der, err = x509.CreateCertificate(rand.Reader, tmpl, ca, csr.PublicKey, k)
	default:
		err = errors.New("unsupported CA key type")
	}
	return der, tmpl.NotAfter, err
}

// small helper to make a random big.Int serial
func bigRand() *big.Int {
	// 128-bit serial
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return new(big.Int).SetBytes(b)
}

// POST /v1/agents/:agentId/csr
// Accepts a CSR PEM and optional TEE/TPM evidence (stored as audit only). Signs CSR using CA from env.
func AcceptAgentCSR(c *gin.Context) {
	agentID := c.Param("agentId")
	if _, err := uuid.Parse(agentID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent id"})
		return
	}
	var req AgentCSRRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if strings.TrimSpace(req.CSRPem) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "csr_pem required"})
		return
	}
	// Parse CSR
	blk, _ := pem.Decode([]byte(req.CSRPem))
	if blk == nil || blk.Type != "CERTIFICATE REQUEST" && blk.Type != "NEW CERTIFICATE REQUEST" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid CSR"})
		return
	}
	csr, err := x509.ParseCertificateRequest(blk.Bytes)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "parse csr failed"})
		return
	}
	if err := csr.CheckSignature(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "csr signature invalid"})
		return
	}
	// Minimal check: subject CN or SAN should match provided agentID if present
	if csr.Subject.CommonName != "" && csr.Subject.CommonName != agentID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "csr subject mismatch"})
		return
	}
	// Load CA
	caCert, caKey, caPem, err := loadCAFromEnv()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	der, exp, err := signCSR(csr, caCert, caKey, agentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	// Audit store of evidence (MVP: write to audit/event_logs)
	_ = audit.Append(c.Request.Context(), uuid.Nil, "agent_csr_signed", gin.H{"agent_id": agentID, "evidence": req.Evidence}, nil, nil)
	c.JSON(http.StatusOK, AgentCSRResponse{CertPEM: string(certPEM), CaPEM: caPem, Expires: exp.Unix()})
}
