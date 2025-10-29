package attest

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"net/http"
)

// Principal represents the caller identity determined from request
type Principal struct {
	OrgID           string
	AgentID         string
	SPIFFEID        string
	AuthnKind       string // apikey|spiffe|oidc
	CertFingerprint string
}

// FromRequest extracts a minimal principal from headers and TLS peer cert (if present)
func FromRequest(r *http.Request, fallbackOrg string, fallbackAgent string) Principal {
	p := Principal{OrgID: fallbackOrg, AgentID: fallbackAgent, AuthnKind: "apikey"}
	// Prefer SPIFFE ID from TLS peer cert URI SAN, else X-SPIFFE-ID header
	if r.TLS != nil && len(r.TLS.PeerCertificates) > 0 {
		leaf := r.TLS.PeerCertificates[0]
		if id := spiffeFromCert(leaf); id != "" {
			p.SPIFFEID = id
			p.AuthnKind = "spiffe"
		}
		p.CertFingerprint = sha256Hex(leaf.Raw)
	}
	if p.SPIFFEID == "" {
		if s := r.Header.Get("X-SPIFFE-ID"); s != "" {
			p.SPIFFEID = s
			p.AuthnKind = "spiffe"
		}
	}
	return p
}

func sha256Hex(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func spiffeFromCert(cert *x509.Certificate) string {
	for _, uri := range cert.URIs {
		if uri.Scheme == "spiffe" {
			return uri.String()
		}
	}
	return ""
}
