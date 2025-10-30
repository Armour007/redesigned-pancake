package attest

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"time"
)

// VerifierResult captures normalized attestation outcomes
// Measurement can be a canonical string for device posture (e.g., PCR digest or enclave measurement)
// Fingerprint is a stable device identifier we bind cert issuance to.
// PostureOK indicates this attestation satisfies the configured policy.
// TODO: Integrate policy engine for posture decision.

type VerifierResult struct {
	Fingerprint string
	Measurement string
	Posture     map[string]any
	PostureOK   bool
}

type Verifier interface {
	Type() string
	Verify(payload map[string]any) (*VerifierResult, error)
}

// DevTPMVerifier is a placeholder that trusts provided fields in dev mode.
// Expected payload: { "ek_pub": string, "ak_pub": string, "pcr": { ... }, "quote": "..." }

type DevTPMVerifier struct{}

func (DevTPMVerifier) Type() string { return "tpm" }

func (DevTPMVerifier) Verify(payload map[string]any) (*VerifierResult, error) {
	ek, _ := payload["ek_pub"].(string)
	ak, _ := payload["ak_pub"].(string)
	if ek == "" && ak == "" {
		return nil, errors.New("missing ek_pub/ak_pub")
	}
	fp := ek + "|" + ak
	return &VerifierResult{
		Fingerprint: fp,
		Measurement: "tpm-dev",
		Posture:     map[string]any{"source": "tpm", "dev": true},
		PostureOK:   true,
	}, nil
}

// Simple helpers for local CA generation and cert issuance (Ed25519)

// GenerateDevCA returns a PEM-encoded self-signed CA cert/key pair using Ed25519.
func GenerateDevCA(commonName string, years int) (certPEM, keyPEM []byte, err error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	tmpl := &x509.Certificate{
		SerialNumber:          newSerial(),
		Subject:               pkixName(commonName),
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().AddDate(years, 0, 0),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		SignatureAlgorithm:    x509.PureEd25519,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, pub, priv)
	if err != nil {
		return nil, nil, err
	}
	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return nil, nil, err
	}
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyBytes})
	return
}

// CreateClientCert signs a client cert using an Ed25519 CA key.
func CreateClientCert(caCert *x509.Certificate, caKey ed25519.PrivateKey, cn string, days int) (certPEM []byte, nb, na time.Time, serial string, err error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, time.Time{}, time.Time{}, "", err
	}
	serialNum := newSerial()
	now := time.Now()
	na = now.AddDate(0, 0, days)
	nb = now.Add(-5 * time.Minute)
	tmpl := &x509.Certificate{
		SerialNumber:       serialNum,
		Subject:            pkixName(cn),
		NotBefore:          nb,
		NotAfter:           na,
		KeyUsage:           x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:        []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		SignatureAlgorithm: x509.PureEd25519,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, caCert, pub, caKey)
	if err != nil {
		return nil, time.Time{}, time.Time{}, "", err
	}
	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	// Return serial as hex
	serial = serialNum.Text(16)
	_ = priv // client key generation omitted; we only return cert here
	return
}

// private helpers

func newSerial() *big.Int {
	// 128-bit random serial
	max := new(big.Int).Lsh(big.NewInt(1), 128)
	n, _ := rand.Int(rand.Reader, max)
	return n
}

func pkixName(cn string) pkix.Name {
	return pkix.Name{CommonName: cn}
}
