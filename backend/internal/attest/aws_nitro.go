package attest

import (
	"bytes"
	"crypto"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	cbor "github.com/fxamacker/cbor/v2"
	pkcs7 "github.com/smallstep/pkcs7"
	"golang.org/x/crypto/ocsp"
)

// AWSNitroVerifier verifies AWS Nitro Enclave attestation documents signed via CMS/PKCS#7.
// Requirements:
// - Payload must include one of: { pkcs7_der_b64 } or { pkcs7_pem }
// - Verification uses a pinned root(s) from env AURA_NITRO_ROOT_PEM (PEM; may contain multiple certs)
// - Enforces leaf EKU: CodeSigning or any OID in AURA_NITRO_ALLOWED_EKU_OIDS (comma-separated) or payload.allowed_eku_oids
// - Enforces signing time freshness: default max_age=24h; override with payload.max_age_seconds
// - Optional revocation checks can be enabled via AURA_NITRO_CHECK_REVOCATION=1 (best-effort; currently stubbed)
type AWSNitroVerifier struct{}

func (AWSNitroVerifier) Type() string { return "aws_nitro" }

func (AWSNitroVerifier) Verify(payload map[string]any) (*VerifierResult, error) {
	// 1) Decode PKCS7 container
	p7Bytes, err := extractPKCS7(payload)
	if err != nil {
		return nil, err
	}

	// 2) Build root pool from pinned env
	roots, err := loadPinnedRoots()
	if err != nil {
		return nil, err
	}

	// 3) Parse PKCS7
	p7, err := pkcs7.Parse(p7Bytes)
	if err != nil {
		return nil, fmt.Errorf("pkcs7 parse: %w", err)
	}
	if len(p7.Signers) == 0 {
		return nil, errors.New("no signers")
	}

	// 4) Verify signature
	// Configure trust store; smallstep/pkcs7 supports setting verification options
	if err := p7.VerifyWithChain(roots); err != nil {
		return nil, fmt.Errorf("pkcs7 verify: %w", err)
	}

	// 5) Get signer certificate (leaf)
	leaf := signerLeaf(p7)
	if leaf == nil {
		return nil, errors.New("signer leaf not found")
	}

	// 6) EKU checks
	if err := checkEKU(leaf, payload); err != nil {
		return nil, err
	}
	// 7) Optional revocation (stubbed best-effort)
	if os.Getenv("AURA_NITRO_CHECK_REVOCATION") == "1" {
		if err := bestEffortRevocationCheck(leaf, p7.Certificates); err != nil {
			return nil, err
		}
	}

	// 8) Extract content
	content := p7.Content
	if len(content) == 0 {
		return nil, errors.New("empty content")
	}
	// Parse attestation content (CBOR or JSON) to enforce freshness and nonce
	claims, _ := parseNitroContent(content)
	// Enforce max age (seconds)
	maxAge := time.Duration(envIntDefault("AURA_NITRO_MAX_AGE_SECONDS", 86400)) * time.Second
	if v, ok := payload["max_age_seconds"].(float64); ok && v > 0 {
		maxAge = time.Duration(int64(v)) * time.Second
	}
	if ts, ok := extractTimestamp(claims); ok {
		if time.Since(ts) > maxAge {
			return nil, fmt.Errorf("attestation too old: %s > %s", time.Since(ts), maxAge)
		}
	}
	// Enforce schema (module_id, pcrs, public_key, nonce) unless relaxed
	if os.Getenv("AURA_NITRO_RELAXED") != "1" {
		if err := validateNitroClaims(claims); err != nil {
			return nil, err
		}
	}
	// Enforce expected nonce if provided
	if expNonce, ok := getString(payload, "expected_nonce"); ok {
		if got, ok2 := getString(claims, "nonce"); !ok2 || got != expNonce {
			return nil, errors.New("nonce mismatch")
		}
	}

	// 9) Fingerprint/measurement
	sum := sha256.Sum256(content)
	meas := hex.EncodeToString(sum[:])

	posture := map[string]any{
		"provider":       "aws_nitro",
		"content_sha256": meas,
		"signer_subject": leaf.Subject.String(),
		"claims":         claims,
	}

	return &VerifierResult{
		Fingerprint: "aws-nitro:" + meas,
		Measurement: meas,
		Posture:     posture,
		PostureOK:   true,
	}, nil
}

// parseNitroContent attempts CBOR first, then JSON; returns claims as map[string]any.
func parseNitroContent(b []byte) (map[string]any, error) {
	var out map[string]any
	// Try CBOR
	if err := cbor.Unmarshal(b, &out); err == nil && len(out) > 0 {
		return out, nil
	}
	// Try JSON
	out = map[string]any{}
	if err := json.Unmarshal(b, &out); err == nil && len(out) > 0 {
		return out, nil
	}
	return nil, errors.New("unsupported content format")
}

// extractTimestamp searches known keys and interprets ints as seconds or milliseconds.
func extractTimestamp(claims map[string]any) (time.Time, bool) {
	if claims == nil {
		return time.Time{}, false
	}
	// known keys: timestamp, iat, issued_at
	for _, k := range []string{"timestamp", "iat", "issued_at"} {
		if v, ok := claims[k]; ok {
			switch t := v.(type) {
			case float64:
				return tsFromNumber(t)
			case int64:
				return tsFromNumber(float64(t))
			case int:
				return tsFromNumber(float64(t))
			case string:
				if n, err := strconv.ParseInt(t, 10, 64); err == nil {
					return tsFromNumber(float64(n))
				}
			}
		}
	}
	return time.Time{}, false
}

func tsFromNumber(n float64) (time.Time, bool) {
	// Heuristic: if > 1e12 treat as ms, else seconds
	if n > 1_000_000_000_000 { // ms
		sec := int64(n) / 1000
		return time.Unix(sec, 0), true
	}
	return time.Unix(int64(n), 0), true
}

func envIntDefault(key string, def int) int {
	if s := strings.TrimSpace(os.Getenv(key)); s != "" {
		if v, err := strconv.Atoi(s); err == nil {
			return v
		}
	}
	return def
}

func getString(m map[string]any, key string) (string, bool) {
	if m == nil {
		return "", false
	}
	if v, ok := m[key]; ok {
		if s, ok2 := v.(string); ok2 {
			return s, true
		}
	}
	return "", false
}

// validateNitroClaims enforces presence and basic shapes of known Nitro attestation fields.
func validateNitroClaims(claims map[string]any) error {
	// module_id: string
	if s, ok := getString(claims, "module_id"); !ok || strings.TrimSpace(s) == "" {
		return errors.New("missing module_id")
	}
	// nonce: string
	if s, ok := getString(claims, "nonce"); !ok || strings.TrimSpace(s) == "" {
		return errors.New("missing nonce")
	}
	// public_key: string (PEM/JWK)
	if s, ok := getString(claims, "public_key"); !ok || strings.TrimSpace(s) == "" {
		return errors.New("missing public_key")
	}
	// pcrs: map present under one of these keys
	if _, ok := claims["pcrs"]; !ok {
		if _, ok2 := claims["pcrs_sha256"]; !ok2 {
			if _, ok3 := claims["pcr_values"]; !ok3 {
				return errors.New("missing pcrs")
			}
		}
	}
	// Optional: enforce hash type if provided
	return nil
}

// extractPKCS7 reads pkcs7 from payload["pkcs7_der_b64"] or payload["pkcs7_pem"].
func extractPKCS7(payload map[string]any) ([]byte, error) {
	if s, _ := payload["pkcs7_der_b64"].(string); s != "" {
		b, err := base64.StdEncoding.DecodeString(s)
		if err != nil {
			return nil, fmt.Errorf("bad pkcs7_der_b64: %w", err)
		}
		return b, nil
	}
	if p, _ := payload["pkcs7_pem"].(string); p != "" {
		blk, _ := pem.Decode([]byte(p))
		if blk == nil {
			return nil, errors.New("bad pkcs7_pem")
		}
		return blk.Bytes, nil
	}
	return nil, errors.New("missing pkcs7 data")
}

// loadPinnedRoots loads PEM roots from AURA_NITRO_ROOT_PEM; require at least one.
func loadPinnedRoots() (*x509.CertPool, error) {
	pemAll := os.Getenv("AURA_NITRO_ROOT_PEM")
	if strings.TrimSpace(pemAll) == "" {
		return nil, errors.New("nitro root not configured (AURA_NITRO_ROOT_PEM)")
	}
	pool := x509.NewCertPool()
	var any bool
	rest := []byte(pemAll)
	for {
		var blk *pem.Block
		blk, rest = pem.Decode(rest)
		if blk == nil {
			break
		}
		if blk.Type != "CERTIFICATE" {
			continue
		}
		if cert, err := x509.ParseCertificate(blk.Bytes); err == nil {
			pool.AddCert(cert)
			any = true
		}
		if len(rest) == 0 {
			break
		}
	}
	if !any {
		return nil, errors.New("no valid root certs in AURA_NITRO_ROOT_PEM")
	}
	return pool, nil
}

// signerLeaf attempts to find the signer's leaf certificate.
func signerLeaf(p7 *pkcs7.PKCS7) *x509.Certificate {
	if len(p7.Certificates) == 0 {
		return nil
	}
	for _, c := range p7.Certificates {
		if !c.IsCA {
			return c
		}
	}
	return p7.Certificates[0]
}

func checkEKU(leaf *x509.Certificate, payload map[string]any) error {
	// Allow CodeSigning by default
	allowCodeSigning := true
	if allowCodeSigning {
		for _, eku := range leaf.ExtKeyUsage {
			if eku == x509.ExtKeyUsageCodeSigning {
				return nil
			}
		}
	}
	// Allow specific OIDs via env or payload
	allowed := map[string]struct{}{}
	if s := os.Getenv("AURA_NITRO_ALLOWED_EKU_OIDS"); strings.TrimSpace(s) != "" {
		for _, p := range strings.Split(s, ",") {
			if t := strings.TrimSpace(p); t != "" {
				allowed[t] = struct{}{}
			}
		}
	}
	if any, ok := payload["allowed_eku_oids"].([]any); ok {
		for _, v := range any {
			if str, ok2 := v.(string); ok2 && strings.TrimSpace(str) != "" {
				allowed[strings.TrimSpace(str)] = struct{}{}
			}
		}
	}
	if len(allowed) == 0 {
		// if nothing explicitly allowed and CodeSigning not present, reject
		return errors.New("leaf certificate lacks required EKU")
	}
	for _, oid := range leaf.UnknownExtKeyUsage {
		if _, ok := allowed[oid.String()]; ok {
			return nil
		}
	}
	return errors.New("leaf certificate lacks allowed EKU OIDs")
}

// Simple caches for revocation results
var ocspCache sync.Map // key: url|serial -> ocspEntry
type ocspEntry struct {
	status  int
	err     string
	expires time.Time
}

var crlCache sync.Map // key: url -> crlEntry
type crlEntry struct {
	revoked map[string]struct{}
	expires time.Time
}

func bestEffortRevocationCheck(leaf *x509.Certificate, chain []*x509.Certificate) error {
	issuer := findIssuer(leaf, chain)
	timeout := time.Duration(envIntDefault("AURA_NITRO_REVOCATION_TIMEOUT_MS", 1500)) * time.Millisecond
	// Prefer OCSP if available
	if issuer != nil && len(leaf.OCSPServer) > 0 {
		if err := checkOCSP(leaf, issuer, timeout); err != nil {
			return err
		}
	}
	// Fallback CRL if configured on cert
	if issuer != nil && len(leaf.CRLDistributionPoints) > 0 {
		if err := checkCRL(leaf, issuer, timeout); err != nil {
			return err
		}
	}
	return nil
}

func findIssuer(leaf *x509.Certificate, certs []*x509.Certificate) *x509.Certificate {
	if leaf == nil {
		return nil
	}
	for _, c := range certs {
		if c == nil || c.Equal(leaf) {
			continue
		}
		if c.IsCA && leaf.Issuer.String() == c.Subject.String() {
			return c
		}
		// match by AuthorityKeyId/SubjectKeyId when present
		if len(leaf.AuthorityKeyId) > 0 && len(c.SubjectKeyId) > 0 &&
			string(leaf.AuthorityKeyId) == string(c.SubjectKeyId) {
			return c
		}
	}
	return nil
}

func checkOCSP(leaf, issuer *x509.Certificate, timeout time.Duration) error {
	for _, url := range leaf.OCSPServer {
		key := url + "|" + leaf.SerialNumber.Text(16)
		if v, ok := ocspCache.Load(key); ok {
			e := v.(ocspEntry)
			if time.Now().Before(e.expires) {
				if e.err != "" {
					return errors.New(e.err)
				}
				if e.status == ocsp.Revoked {
					return errors.New("certificate revoked (OCSP)")
				}
				return nil
			}
		}
		// Build request
		req, err := ocsp.CreateRequest(leaf, issuer, &ocsp.RequestOptions{Hash: crypto.SHA1})
		if err != nil {
			continue
		}
		client := &http.Client{Timeout: timeout}
		start := time.Now()
		resp, err := client.Post(url, "application/ocsp-request", bytes.NewReader(req))
		dur := time.Since(start)
		if err != nil {
			log.Printf("nitro.revocation.ocsp url=%s error=%v ms=%d", url, err, dur.Milliseconds())
			continue
		}
		b, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		ocspResp, err := ocsp.ParseResponse(b, issuer)
		if err != nil {
			log.Printf("nitro.revocation.ocsp url=%s parse_error=%v ms=%d", url, err, dur.Milliseconds())
			continue
		}
		// Cache based on NextUpdate with a floor TTL
		ttl := 10 * time.Minute
		if !ocspResp.NextUpdate.IsZero() {
			if d := time.Until(ocspResp.NextUpdate); d > time.Minute {
				ttl = d
			}
		}
		entry := ocspEntry{status: ocspResp.Status, expires: time.Now().Add(ttl)}
		switch ocspResp.Status {
		case ocsp.Good:
			log.Printf("nitro.revocation.ocsp url=%s status=good ms=%d ttl_s=%d", url, dur.Milliseconds(), int64(ttl.Seconds()))
			ocspCache.Store(key, entry)
			return nil
		case ocsp.Revoked:
			entry.err = "certificate revoked (OCSP)"
			ocspCache.Store(key, entry)
			return errors.New(entry.err)
		default:
			entry.err = "ocsp status unknown"
			ocspCache.Store(key, entry)
			return errors.New(entry.err)
		}
	}
	return nil
}

func checkCRL(leaf, issuer *x509.Certificate, timeout time.Duration) error {
	serialHex := leaf.SerialNumber.Text(16)
	for _, url := range leaf.CRLDistributionPoints {
		if v, ok := crlCache.Load(url); ok {
			e := v.(crlEntry)
			if time.Now().Before(e.expires) {
				if _, revoked := e.revoked[serialHex]; revoked {
					return errors.New("certificate revoked (CRL)")
				}
				return nil
			}
		}
		client := &http.Client{Timeout: timeout}
		start := time.Now()
		resp, err := client.Get(url)
		dur := time.Since(start)
		if err != nil {
			log.Printf("nitro.revocation.crl url=%s error=%v ms=%d", url, err, dur.Milliseconds())
			continue
		}
		b, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		crl, err := x509.ParseCRL(b)
		if err != nil {
			log.Printf("nitro.revocation.crl url=%s parse_error=%v ms=%d", url, err, dur.Milliseconds())
			continue
		}
		if err := issuer.CheckCRLSignature(crl); err != nil {
			log.Printf("nitro.revocation.crl url=%s bad_signature ms=%d", url, dur.Milliseconds())
			continue
		}
		revoked := make(map[string]struct{})
		for _, rc := range crl.TBSCertList.RevokedCertificates {
			revoked[rc.SerialNumber.Text(16)] = struct{}{}
		}
		ttl := 10 * time.Minute
		if !crl.TBSCertList.NextUpdate.IsZero() {
			if d := time.Until(crl.TBSCertList.NextUpdate); d > time.Minute {
				ttl = d
			}
		}
		crlCache.Store(url, crlEntry{revoked: revoked, expires: time.Now().Add(ttl)})
		if _, ok := revoked[serialHex]; ok {
			return errors.New("certificate revoked (CRL)")
		}
		log.Printf("nitro.revocation.crl url=%s status=good ms=%d ttl_s=%d", url, dur.Milliseconds(), int64(ttl.Seconds()))
		return nil
	}
	return nil
}
