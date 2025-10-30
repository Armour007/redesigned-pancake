package crypto

import (
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"math/big"
	"net/http"
	"os"
	"strings"
	"time"

	aws "github.com/aws/aws-sdk-go-v2/aws"
	awscfg "github.com/aws/aws-sdk-go-v2/config"
	kms "github.com/aws/aws-sdk-go-v2/service/kms"
	kmstypes "github.com/aws/aws-sdk-go-v2/service/kms/types"

	// GCP KMS
	gcpkms "cloud.google.com/go/kms/apiv1"
	kmspb "cloud.google.com/go/kms/apiv1/kmspb"
	"google.golang.org/api/option"

	// Azure Key Vault
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azkeys"
)

// SignAlgorithm identifiers for JWT headers
const (
	AlgEdDSA = "EdDSA"
	AlgES256 = "ES256"
)

// Signer signs JWT-like payloads (header.payload) and exposes public JWK material
type Signer interface {
	Algorithm() string
	KeyID() string
	PublicJWK(ctx context.Context) (map[string]any, error)
	// Sign returns the raw signature bytes appropriate for the algorithm (Ed25519: 64 bytes; ES256: JOSE r||s 64 bytes)
	Sign(ctx context.Context, unsigned []byte) ([]byte, error)
}

// TrustKeyRecord describes key material row for constructing signers
type TrustKeyRecord struct {
	Provider       string          // e.g., "local","aws","gcp","azure","vault"
	KeyRef         string          // e.g., ARN, resource name, KV key identifier, vault key name
	KeyVersion     string          // optional version (annotated in kid)
	Alg            string          // AlgEdDSA or AlgES256
	Kid            string          // preferred kid
	EncPriv        string          // ed25519 private key (base64url/base64) for local
	ProviderConfig json.RawMessage // provider-specific settings
	JWKPub         json.RawMessage // cached/public JWK
}

// NewSignerFromRecord creates a signer for the given trust key record
func NewSignerFromRecord(rec TrustKeyRecord) (Signer, error) {
	switch strings.ToLower(rec.Provider) {
	case "", "local":
		// local Ed25519 only
		if rec.Alg == "" || rec.Alg == AlgEdDSA {
			priv, kid, err := parseLocalEd25519(rec.EncPriv, rec.Kid)
			if err != nil {
				return nil, err
			}
			return &LocalEd25519Signer{priv: priv, kid: kid}, nil
		}
		return nil, errors.New("local provider supports only EdDSA")
	case "vault":
		return NewVaultSigner(rec)
	case "aws":
		return NewAWSSigner(rec)
	case "gcp":
		return NewGCPSigner(rec)
	case "azure":
		return NewAzureSigner(rec)
	default:
		return nil, errors.New("unknown provider")
	}
}

// ----- Local Ed25519 signer -----
type LocalEd25519Signer struct {
	priv ed25519.PrivateKey
	kid  string
}

func (s *LocalEd25519Signer) Algorithm() string { return AlgEdDSA }
func (s *LocalEd25519Signer) KeyID() string     { return s.kid }

func (s *LocalEd25519Signer) PublicJWK(ctx context.Context) (map[string]any, error) {
	pub := ed25519.PrivateKey(s.priv).Public().(ed25519.PublicKey)
	xb := base64.RawURLEncoding.EncodeToString(pub)
	return map[string]any{"kty": "OKP", "crv": "Ed25519", "alg": AlgEdDSA, "use": "sig", "kid": s.kid, "x": xb}, nil
}

func (s *LocalEd25519Signer) Sign(ctx context.Context, unsigned []byte) ([]byte, error) {
	sig := ed25519.Sign(s.priv, unsigned)
	return sig, nil
}

func parseLocalEd25519(encPriv, kid string) (ed25519.PrivateKey, string, error) {
	if encPriv == "" {
		return nil, "", errors.New("missing private key")
	}
	// Try base64url then base64
	var raw []byte
	var err error
	if raw, err = base64.RawURLEncoding.DecodeString(encPriv); err != nil {
		raw, err = base64.StdEncoding.DecodeString(encPriv)
		if err != nil {
			return nil, "", err
		}
	}
	if l := len(raw); l != ed25519.PrivateKeySize {
		return nil, "", errors.New("bad ed25519 private key length")
	}
	priv := ed25519.PrivateKey(raw)
	if strings.TrimSpace(kid) == "" {
		// derive a short kid from pub
		pub := priv.Public().(ed25519.PublicKey)
		sum := sha256.Sum256(pub)
		kid = base64.RawURLEncoding.EncodeToString(sum[:8])
	}
	return priv, kid, nil
}

// ----- Vault Transit signer (basic) -----
type VaultSigner struct {
	keyName string
	mount   string
	addr    string
	token   string
	alg     string
	kid     string
	jwk     map[string]any // optional cached JWK
	http    *http.Client
}

func NewVaultSigner(rec TrustKeyRecord) (Signer, error) {
	cfg := map[string]any{}
	if len(rec.ProviderConfig) > 0 {
		_ = json.Unmarshal(rec.ProviderConfig, &cfg)
	}
	mount, _ := cfg["mount"].(string)
	if mount == "" {
		mount = "transit"
	}
	addr := os.Getenv("VAULT_ADDR")
	if v, ok := cfg["addr"].(string); ok && v != "" {
		addr = v
	}
	tok := os.Getenv("VAULT_TOKEN")
	if v, ok := cfg["token"].(string); ok && v != "" {
		tok = v
	}
	if addr == "" || tok == "" {
		return nil, errors.New("vault addr/token not configured")
	}
	alg := rec.Alg
	if alg == "" {
		alg = AlgEdDSA
	}
	kid := rec.Kid
	if strings.TrimSpace(kid) == "" {
		kid = "vault:" + rec.KeyRef
		if rec.KeyVersion != "" {
			kid += "@" + rec.KeyVersion
		}
	}
	var jwk map[string]any
	if len(rec.JWKPub) > 0 {
		_ = json.Unmarshal(rec.JWKPub, &jwk)
	}
	return &VaultSigner{keyName: rec.KeyRef, mount: mount, addr: addr, token: tok, alg: alg, kid: kid, jwk: jwk, http: &http.Client{Timeout: 6 * time.Second}}, nil
}

func (s *VaultSigner) Algorithm() string { return s.alg }
func (s *VaultSigner) KeyID() string     { return s.kid }

func (s *VaultSigner) PublicJWK(ctx context.Context) (map[string]any, error) {
	if s.jwk != nil {
		return s.jwk, nil
	}
	// best-effort fetch
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(s.addr, "/")+"/v1/"+s.mount+"/keys/"+s.keyName, nil)
	req.Header.Set("X-Vault-Token", s.token)
	resp, err := s.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out struct {
		Data struct {
			Keys map[string]struct {
				PublicKey string `json:"public_key"`
			} `json:"keys"`
		} `json:"data"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&out)
	// Not all Vault responses include PEM; prefer configured jwk
	return nil, errors.New("vault public jwk not available; provide jwk_pub at creation")
}

func (s *VaultSigner) Sign(ctx context.Context, unsigned []byte) ([]byte, error) {
	// Vault expects base64 input; for ES256 we let Vault hash; for EdDSA it's pure EdDSA over input
	in := map[string]any{
		"input": base64.StdEncoding.EncodeToString(unsigned),
	}
	if s.alg == AlgES256 {
		in["hash_algorithm"] = "sha2-256"
		in["marshaling_algorithm"] = "jws"
		// Note: For ECDSA, Vault returns ASN.1 by default; marshaling_algorithm=jws requests JOSE format if supported
	}
	b, _ := json.Marshal(in)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(s.addr, "/")+"/v1/"+s.mount+"/sign/"+s.keyName, strings.NewReader(string(b)))
	req.Header.Set("X-Vault-Token", s.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out struct {
		Data struct {
			Signature string `json:"signature"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	sig := out.Data.Signature
	// Format: vault:vX:base64
	parts := strings.Split(sig, ":")
	rawB64 := parts[len(parts)-1]
	der, err := base64.StdEncoding.DecodeString(rawB64)
	if err != nil {
		return nil, err
	}
	if s.alg == AlgEdDSA {
		return der, nil // vault returns raw ed25519 sig bytes
	}
	// ES256: Vault may return ASN.1 DER; convert to JOSE r||s if needed
	// Try to parse ASN.1; if length is 64 already, assume JOSE
	if len(der) == 64 {
		return der, nil
	}
	r, sBig, ok := parseECDSADER(der)
	if !ok {
		return nil, errors.New("invalid ECDSA signature format")
	}
	// left-pad to 32 bytes
	rb := r.Bytes()
	sb := sBig.Bytes()
	sigOut := make([]byte, 64)
	copy(sigOut[32-len(rb):32], rb)
	copy(sigOut[64-len(sb):], sb)
	return sigOut, nil
}

// ----- AWS KMS signer (ES256) -----
type AWSSigner struct {
	keyID  string
	alg    string
	kid    string
	jwk    map[string]any
	client *kms.Client
}

func NewAWSSigner(rec TrustKeyRecord) (Signer, error) {
	cfgMap := map[string]any{}
	if len(rec.ProviderConfig) > 0 {
		_ = json.Unmarshal(rec.ProviderConfig, &cfgMap)
	}
	region := os.Getenv("AWS_REGION")
	if v, ok := cfgMap["region"].(string); ok && v != "" {
		region = v
	}
	if region == "" {
		return nil, errors.New("aws region not configured")
	}
	// Optional endpoint override (e.g., localstack)
	endpointURL, _ := cfgMap["endpoint_url"].(string)
	// Load AWS config
	cfg, err := awscfg.LoadDefaultConfig(context.Background(), awscfg.WithRegion(region))
	if err != nil {
		return nil, err
	}
	var client *kms.Client
	if endpointURL != "" {
		client = kms.NewFromConfig(cfg, func(o *kms.Options) { o.BaseEndpoint = aws.String(endpointURL) })
	} else {
		client = kms.NewFromConfig(cfg)
	}
	alg := rec.Alg
	if alg == "" {
		alg = AlgES256
	}
	kid := rec.Kid
	if strings.TrimSpace(kid) == "" {
		kid = "aws:" + rec.KeyRef
		if rec.KeyVersion != "" {
			kid += "@" + rec.KeyVersion
		}
	}
	var jwk map[string]any
	if len(rec.JWKPub) > 0 {
		_ = json.Unmarshal(rec.JWKPub, &jwk)
	}
	return &AWSSigner{keyID: rec.KeyRef, alg: alg, kid: kid, jwk: jwk, client: client}, nil
}

func (s *AWSSigner) Algorithm() string { return s.alg }
func (s *AWSSigner) KeyID() string     { return s.kid }

func (s *AWSSigner) PublicJWK(ctx context.Context) (map[string]any, error) {
	if s.jwk != nil {
		return s.jwk, nil
	}
	// Fetch from AWS KMS
	out, err := s.client.GetPublicKey(ctx, &kms.GetPublicKeyInput{KeyId: &s.keyID})
	if err != nil {
		return nil, err
	}
	// Parse DER SubjectPublicKeyInfo
	pk, err := x509.ParsePKIXPublicKey(out.PublicKey)
	if err != nil {
		return nil, err
	}
	ec, ok := pk.(*ecdsa.PublicKey)
	if !ok {
		return nil, errors.New("kms public key is not ecdsa")
	}
	jwk := ECP256PublicJWK(ec, s.kid)
	s.jwk = jwk
	return jwk, nil
}

func (s *AWSSigner) Sign(ctx context.Context, unsigned []byte) ([]byte, error) {
	if s.alg != AlgES256 {
		return nil, errors.New("aws signer only supports ES256")
	}
	algo := kmstypes.SigningAlgorithmSpecEcdsaSha256
	mt := kmstypes.MessageTypeRaw
	out, err := s.client.Sign(ctx, &kms.SignInput{
		KeyId:            &s.keyID,
		Message:          unsigned,
		SigningAlgorithm: algo,
		MessageType:      mt,
	})
	if err != nil {
		return nil, err
	}
	// KMS returns ASN.1 DER for ECDSA signatures
	r, sBig, ok := parseECDSADER(out.Signature)
	if !ok {
		return nil, errors.New("invalid ECDSA signature from KMS")
	}
	rb := r.Bytes()
	sb := sBig.Bytes()
	sig := make([]byte, 64)
	copy(sig[32-len(rb):32], rb)
	copy(sig[64-len(sb):], sb)
	return sig, nil
}

// ----- GCP KMS signer (ES256) -----
type GCPSigner struct {
	versionName string
	alg         string
	kid         string
	jwk         map[string]any
	client      *gcpkms.KeyManagementClient
}

func NewGCPSigner(rec TrustKeyRecord) (Signer, error) {
	cfg := map[string]any{}
	if len(rec.ProviderConfig) > 0 {
		_ = json.Unmarshal(rec.ProviderConfig, &cfg)
	}
	// Key reference can be either a cryptoKeyVersion name or a cryptoKey; prefer full version
	name := rec.KeyRef
	if !strings.Contains(name, "/cryptoKeyVersions/") {
		ver := rec.KeyVersion
		if ver == "" {
			ver = "1"
		}
		if strings.Contains(name, "/cryptoKeys/") {
			name = strings.TrimSuffix(name, "/")
			name = name + "/cryptoKeyVersions/" + ver
		}
	}
	if name == "" {
		return nil, errors.New("gcp kms key name not configured")
	}
	// Optional: endpoint or credentials file
	var opts []option.ClientOption
	if ep, _ := cfg["endpoint"].(string); ep != "" {
		opts = append(opts, option.WithEndpoint(ep))
	}
	if credFile, _ := cfg["credentials_file"].(string); credFile != "" {
		opts = append(opts, option.WithCredentialsFile(credFile))
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := gcpkms.NewKeyManagementClient(ctx, opts...)
	if err != nil {
		return nil, err
	}
	alg := rec.Alg
	if alg == "" {
		alg = AlgES256
	}
	kid := rec.Kid
	if strings.TrimSpace(kid) == "" {
		kid = "gcp:" + name
	}
	var jwk map[string]any
	if len(rec.JWKPub) > 0 {
		_ = json.Unmarshal(rec.JWKPub, &jwk)
	}
	return &GCPSigner{versionName: name, alg: alg, kid: kid, jwk: jwk, client: client}, nil
}

func (s *GCPSigner) Algorithm() string { return s.alg }
func (s *GCPSigner) KeyID() string     { return s.kid }

func (s *GCPSigner) PublicJWK(ctx context.Context) (map[string]any, error) {
	if s.jwk != nil {
		return s.jwk, nil
	}
	// Fetch public key (PEM) from KMS
	out, err := s.client.GetPublicKey(ctx, &kmspb.GetPublicKeyRequest{Name: s.versionName})
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode([]byte(out.Pem))
	if block == nil {
		return nil, errors.New("gcp kms returned no pem block")
	}
	pk, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	ec, ok := pk.(*ecdsa.PublicKey)
	if !ok {
		return nil, errors.New("gcp kms public key is not ecdsa")
	}
	jwk := ECP256PublicJWK(ec, s.kid)
	s.jwk = jwk
	return jwk, nil
}

func (s *GCPSigner) Sign(ctx context.Context, unsigned []byte) ([]byte, error) {
	if s.alg != AlgES256 {
		return nil, errors.New("gcp signer only supports ES256")
	}
	h := sha256.Sum256(unsigned)
	req := &kmspb.AsymmetricSignRequest{
		Name: s.versionName,
		Digest: &kmspb.Digest{
			Digest: &kmspb.Digest_Sha256{Sha256: h[:]},
		},
	}
	out, err := s.client.AsymmetricSign(ctx, req)
	if err != nil {
		return nil, err
	}
	// DER to JOSE r||s
	r, sbig, ok := parseECDSADER(out.Signature)
	if !ok {
		return nil, errors.New("invalid ECDSA signature from GCP KMS")
	}
	rb := r.Bytes()
	sb := sbig.Bytes()
	sig := make([]byte, 64)
	copy(sig[32-len(rb):32], rb)
	copy(sig[64-len(sb):], sb)
	return sig, nil
}

// ----- Azure Key Vault signer (ES256) -----
type AzureSigner struct {
	keyID   string // full key identifier URL including version
	vault   string // vault URL
	keyName string
	version string
	alg     string
	kid     string
	jwk     map[string]any
	kclient *azkeys.Client
}

func NewAzureSigner(rec TrustKeyRecord) (Signer, error) {
	cfg := map[string]any{}
	if len(rec.ProviderConfig) > 0 {
		_ = json.Unmarshal(rec.ProviderConfig, &cfg)
	}
	vaultURL, _ := cfg["vault_url"].(string)
	keyID := rec.KeyRef
	keyName := ""
	version := rec.KeyVersion
	if strings.HasPrefix(strings.ToLower(keyID), "https://") {
		// full identifier; extract vault URL and possibly version
		parts := strings.Split(strings.TrimPrefix(keyID, "https://"), "/")
		if len(parts) >= 3 && parts[1] == "keys" {
			vaultURL = "https://" + parts[0]
			keyName = parts[2]
			if len(parts) >= 4 {
				version = parts[3]
			}
		}
	} else {
		// build from vault_url + key name
		keyName = keyID
	}
	if vaultURL == "" || keyName == "" {
		return nil, errors.New("azure vault_url and key name required")
	}
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, err
	}
	kclient, err := azkeys.NewClient(vaultURL, cred, nil)
	if err != nil {
		return nil, err
	}
	// Build full key identifier URL with version if available
	fullID := vaultURL + "/keys/" + keyName
	if version != "" {
		fullID += "/" + version
	}
	alg := rec.Alg
	if alg == "" {
		alg = AlgES256
	}
	kid := rec.Kid
	if strings.TrimSpace(kid) == "" {
		kid = "azure:" + fullID
	}
	var jwk map[string]any
	if len(rec.JWKPub) > 0 {
		_ = json.Unmarshal(rec.JWKPub, &jwk)
	}
	return &AzureSigner{keyID: fullID, vault: vaultURL, keyName: keyName, version: version, alg: alg, kid: kid, jwk: jwk, kclient: kclient}, nil
}

func (s *AzureSigner) Algorithm() string { return s.alg }
func (s *AzureSigner) KeyID() string     { return s.kid }

func (s *AzureSigner) PublicJWK(ctx context.Context) (map[string]any, error) {
	if s.jwk != nil {
		return s.jwk, nil
	}
	// Fetch key to extract public parameters
	var version string
	if s.version != "" {
		version = s.version
	}
	resp, err := s.kclient.GetKey(ctx, s.keyName, version, nil)
	if err != nil {
		return nil, err
	}
	if resp.Key == nil || resp.Key.X == nil || resp.Key.Y == nil {
		return nil, errors.New("azure key missing EC coordinates")
	}
	// Build JWK from coordinates
	x := base64.RawURLEncoding.EncodeToString(resp.Key.X)
	y := base64.RawURLEncoding.EncodeToString(resp.Key.Y)
	jwk := map[string]any{"kty": "EC", "crv": "P-256", "alg": AlgES256, "use": "sig", "kid": s.kid, "x": x, "y": y}
	s.jwk = jwk
	return jwk, nil
}

func (s *AzureSigner) Sign(ctx context.Context, unsigned []byte) ([]byte, error) {
	if s.alg != AlgES256 {
		return nil, errors.New("azure signer only supports ES256")
	}
	h := sha256.Sum256(unsigned)
	algo := azkeys.SignatureAlgorithmES256
	res, err := s.kclient.Sign(ctx, s.keyName, s.version, azkeys.SignParameters{Algorithm: &algo, Value: h[:]}, nil)
	if err != nil {
		return nil, err
	}
	// Azure returns DER-encoded ECDSA signature
	r, sbig, ok := parseECDSADER(res.KeyOperationResult.Result)
	if !ok {
		return nil, errors.New("invalid ECDSA signature from Azure KV")
	}
	rb := r.Bytes()
	sb := sbig.Bytes()
	sig := make([]byte, 64)
	copy(sig[32-len(rb):32], rb)
	copy(sig[64-len(sb):], sb)
	return sig, nil
}

// parseECDSADER parses a minimal ASN.1 ECDSA signature (r,s)
func parseECDSADER(b []byte) (*big.Int, *big.Int, bool) {
	// Very small DER parser for ECDSA signatures: 0x30 len 0x02 lenR R 0x02 lenS S
	if len(b) < 8 || b[0] != 0x30 {
		return nil, nil, false
	}
	i := 2 // skip SEQUENCE and length
	if i >= len(b) || b[i] != 0x02 {
		return nil, nil, false
	}
	i++
	if i >= len(b) {
		return nil, nil, false
	}
	lr := int(b[i])
	i++
	if i+lr > len(b) {
		return nil, nil, false
	}
	r := new(big.Int).SetBytes(b[i : i+lr])
	i += lr
	if i >= len(b) || b[i] != 0x02 {
		return nil, nil, false
	}
	i++
	if i >= len(b) {
		return nil, nil, false
	}
	ls := int(b[i])
	i++
	if i+ls > len(b) {
		return nil, nil, false
	}
	s := new(big.Int).SetBytes(b[i : i+ls])
	return r, s, true
}

// Helper to build EC P-256 public JWK from key
func ECP256PublicJWK(k *ecdsa.PublicKey, kid string) map[string]any {
	xb := k.X.Bytes()
	yb := k.Y.Bytes()
	// left pad to 32 bytes
	x := make([]byte, 32)
	y := make([]byte, 32)
	copy(x[32-len(xb):], xb)
	copy(y[32-len(yb):], yb)
	return map[string]any{"kty": "EC", "crv": "P-256", "alg": AlgES256, "use": "sig", "kid": kid, "x": base64.RawURLEncoding.EncodeToString(x), "y": base64.RawURLEncoding.EncodeToString(y)}
}
