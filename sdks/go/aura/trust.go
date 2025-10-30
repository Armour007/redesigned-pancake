package aura

import (
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/asn1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"time"
)

type jwk struct {
	Kty string `json:"kty"`
	Crv string `json:"crv,omitempty"`
	Alg string `json:"alg,omitempty"`
	Use string `json:"use,omitempty"`
	Kid string `json:"kid,omitempty"`
	X   string `json:"x,omitempty"`
	Y   string `json:"y,omitempty"`
}

type jwks struct {
	Keys []jwk `json:"keys"`
}

type RevocationItem struct {
	JTI       string    `json:"jti"`
	RevokedAt time.Time `json:"revoked_at"`
	Reason    string    `json:"reason,omitempty"`
}

type RevocationsResp struct {
	Items []RevocationItem `json:"items"`
}

func FetchJWKS(ctx context.Context, baseURL, orgId string) (jwks, error) {
	url := strings.TrimRight(baseURL, "/")
	if orgId != "" {
		url += "/.well-known/aura/" + orgId + "/jwks.json"
	} else {
		url += "/.well-known/aura-jwks.json"
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return jwks{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return jwks{}, fmt.Errorf("jwks status %d", resp.StatusCode)
	}
	b, _ := io.ReadAll(resp.Body)
	var out jwks
	if err := json.Unmarshal(b, &out); err != nil {
		return jwks{}, err
	}
	return out, nil
}

func FetchRevocations(ctx context.Context, baseURL, orgId, etag string) (items []RevocationItem, newETag string, notModified bool, err error) {
	url := strings.TrimRight(baseURL, "/") + "/organizations/" + orgId + "/trust-tokens/revocations"
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 304 {
		return nil, etag, true, nil
	}
	if resp.StatusCode != 200 {
		return nil, "", false, fmt.Errorf("revocations status %d", resp.StatusCode)
	}
	var body RevocationsResp
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, "", false, err
	}
	return body.Items, resp.Header.Get("ETag"), false, nil
}

type VerifyOfflineResult struct {
	Valid  bool
	Reason string
	Claims map[string]any
}

func VerifyTrustTokenOffline(ctx context.Context, baseURL, token, orgId string, graceSeconds int, revoked map[string]struct{}) (VerifyOfflineResult, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return VerifyOfflineResult{Valid: false, Reason: "bad_format"}, nil
	}
	var header map[string]any
	var claims map[string]any
	if hb, err := base64.RawURLEncoding.DecodeString(parts[0]); err != nil {
		return VerifyOfflineResult{Valid: false, Reason: "bad_json"}, nil
	} else {
		_ = json.Unmarshal(hb, &header)
	}
	if pb, err := base64.RawURLEncoding.DecodeString(parts[1]); err != nil {
		return VerifyOfflineResult{Valid: false, Reason: "bad_json"}, nil
	} else {
		_ = json.Unmarshal(pb, &claims)
	}
	alg, _ := header["alg"].(string)
	kid, _ := header["kid"].(string)
	// exp check with grace
	exp := int64(0)
	switch v := claims["exp"].(type) {
	case float64:
		exp = int64(v)
	case json.Number:
		if n, err := v.Int64(); err == nil {
			exp = n
		}
	case nil:
		return VerifyOfflineResult{Valid: false, Reason: "missing_exp"}, nil
	}
	if exp == 0 {
		return VerifyOfflineResult{Valid: false, Reason: "missing_exp"}, nil
	}
	if time.Now().Unix() > exp+int64(graceSeconds) {
		return VerifyOfflineResult{Valid: false, Reason: "expired"}, nil
	}
	if jti, _ := claims["jti"].(string); jti != "" && revoked != nil {
		if _, ok := revoked[jti]; ok {
			return VerifyOfflineResult{Valid: false, Reason: "revoked"}, nil
		}
	}
	unsigned := parts[0] + "." + parts[1]
	if alg == "HS256" {
		return VerifyOfflineResult{Valid: false, Reason: "hs256_offline_unsupported"}, nil
	}
	j, err := FetchJWKS(ctx, baseURL, orgId)
	if err != nil {
		return VerifyOfflineResult{}, err
	}
	var key *jwk
	for i := range j.Keys {
		if kid == "" || j.Keys[i].Kid == kid {
			key = &j.Keys[i]
			break
		}
	}
	if key == nil {
		return VerifyOfflineResult{Valid: false, Reason: "kid_not_found"}, nil
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return VerifyOfflineResult{Valid: false, Reason: "bad_sig"}, nil
	}
	switch {
	case alg == "EdDSA" && key.Kty == "OKP" && key.Crv == "Ed25519":
		x, err := base64.RawURLEncoding.DecodeString(key.X)
		if err != nil {
			return VerifyOfflineResult{Valid: false, Reason: "bad_key"}, nil
		}
		pub := ed25519.PublicKey(x)
		if !ed25519.Verify(pub, []byte(unsigned), sig) {
			return VerifyOfflineResult{Valid: false, Reason: "bad_sig"}, nil
		}
		return VerifyOfflineResult{Valid: true, Claims: claims}, nil
	case alg == "ES256" && key.Kty == "EC" && key.Crv == "P-256":
		xb, err1 := base64.RawURLEncoding.DecodeString(key.X)
		yb, err2 := base64.RawURLEncoding.DecodeString(key.Y)
		if err1 != nil || err2 != nil {
			return VerifyOfflineResult{Valid: false, Reason: "bad_key"}, nil
		}
		x, y := elliptic.P256().ScalarBaseMult([]byte{0}) // dummy to satisfy types
		_ = x
		_ = y // not used
		pub := ecdsa.PublicKey{Curve: elliptic.P256(), X: new(big.Int).SetBytes(xb), Y: new(big.Int).SetBytes(yb)}
		var rs struct{ R, S *big.Int }
		// sig is JOSE r||s; need to convert to DER
		if len(sig) != 64 {
			return VerifyOfflineResult{Valid: false, Reason: "bad_sig"}, nil
		}
		rs.R = new(big.Int).SetBytes(sig[:32])
		rs.S = new(big.Int).SetBytes(sig[32:])
		der, _ := asn1.Marshal(rs)
		h := sha256.Sum256([]byte(unsigned))
		if !ecdsa.VerifyASN1(&pub, h[:], der) {
			return VerifyOfflineResult{Valid: false, Reason: "bad_sig"}, nil
		}
		return VerifyOfflineResult{Valid: true, Claims: claims}, nil
	default:
		return VerifyOfflineResult{Valid: false, Reason: "unsupported_alg"}, nil
	}
}
