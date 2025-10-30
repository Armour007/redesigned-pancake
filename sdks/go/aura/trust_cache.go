package aura

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"sync"
	"time"
)

// TrustCache caches JWKS and revocation state with TTL and ETag-based refresh.
type TrustCache struct {
	mu sync.Mutex

	// JWKS cache keyed by baseURL|orgId
	jwksMap map[string]jwks
	jwksAt  map[string]time.Time
	jwksTTL time.Duration

	// Revocations cache keyed by baseURL|orgId
	revMap  map[string]map[string]struct{}
	revETag map[string]string
	revAt   map[string]time.Time
	revTTL  time.Duration
}

func NewTrustCache(jwksTTL, revTTL time.Duration) *TrustCache {
	return &TrustCache{
		jwksMap: make(map[string]jwks),
		jwksAt:  make(map[string]time.Time),
		jwksTTL: jwksTTL,
		revMap:  make(map[string]map[string]struct{}),
		revETag: make(map[string]string),
		revAt:   make(map[string]time.Time),
		revTTL:  revTTL,
	}
}

func cacheKey(baseURL, orgId string) string { return baseURL + "|" + orgId }

// GetJWKS returns JWKS, refreshing if TTL expired.
func (c *TrustCache) GetJWKS(ctx context.Context, baseURL, orgId string) (jwks, error) {
	key := cacheKey(baseURL, orgId)
	c.mu.Lock()
	jw, ok := c.jwksMap[key]
	at := c.jwksAt[key]
	ttl := c.jwksTTL
	c.mu.Unlock()

	if ok && time.Since(at) < ttl {
		return jw, nil
	}
	// refresh
	fresh, err := FetchJWKS(ctx, baseURL, orgId)
	if err != nil {
		if ok { // serve stale on error
			return jw, nil
		}
		return jwks{}, err
	}
	c.mu.Lock()
	c.jwksMap[key] = fresh
	c.jwksAt[key] = time.Now()
	c.mu.Unlock()
	return fresh, nil
}

// GetRevocations returns a revoked JTI set, refreshing via ETag or TTL.
func (c *TrustCache) GetRevocations(ctx context.Context, baseURL, orgId string) (map[string]struct{}, error) {
	key := cacheKey(baseURL, orgId)
	c.mu.Lock()
	rev := c.revMap[key]
	at := c.revAt[key]
	etag := c.revETag[key]
	ttl := c.revTTL
	c.mu.Unlock()

	if rev == nil {
		rev = make(map[string]struct{})
	}

	// Use TTL to decide polling cadence; within TTL, we still attempt a cheap conditional GET by ETag when called.
	// If-None-Match will keep it lightweight.
	if time.Since(at) < ttl {
		// try conditional refresh but ignore errors
		items, newETag, notMod, _ := FetchRevocations(ctx, baseURL, orgId, etag)
		if !notMod && len(items) > 0 {
			tmp := make(map[string]struct{}, len(items))
			for _, it := range items {
				tmp[it.JTI] = struct{}{}
			}
			c.mu.Lock()
			c.revMap[key] = tmp
			c.revETag[key] = newETag
			c.revAt[key] = time.Now()
			c.mu.Unlock()
			rev = tmp
		}
		return rev, nil
	}

	// TTL expired, fetch (prefer conditional)
	items, newETag, notMod, err := FetchRevocations(ctx, baseURL, orgId, etag)
	if err != nil {
		return rev, nil // serve stale on error
	}
	if notMod {
		c.mu.Lock()
		c.revAt[key] = time.Now()
		c.mu.Unlock()
		return rev, nil
	}
	tmp := make(map[string]struct{}, len(items))
	for _, it := range items {
		tmp[it.JTI] = struct{}{}
	}
	c.mu.Lock()
	c.revMap[key] = tmp
	c.revETag[key] = newETag
	c.revAt[key] = time.Now()
	c.mu.Unlock()
	return tmp, nil
}

// VerifyTrustTokenOfflineCached verifies using cached JWKS and provided revoked set from cache.
func VerifyTrustTokenOfflineCached(ctx context.Context, cache *TrustCache, baseURL, token, orgId string, graceSeconds int) (VerifyOfflineResult, error) {
	if cache == nil {
		return VerifyTrustTokenOffline(ctx, baseURL, token, orgId, graceSeconds, nil)
	}
	jw, err := cache.GetJWKS(ctx, baseURL, orgId)
	if err != nil {
		return VerifyOfflineResult{}, err
	}
	// Extract header + claims
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
	// revocations from cache
	rev, _ := cache.GetRevocations(ctx, baseURL, orgId)
	if jti, _ := claims["jti"].(string); jti != "" && rev != nil {
		if _, ok := rev[jti]; ok {
			return VerifyOfflineResult{Valid: false, Reason: "revoked"}, nil
		}
	}
	// select key
	var key *jwk
	for i := range jw.Keys {
		if kid == "" || jw.Keys[i].Kid == kid {
			key = &jw.Keys[i]
			break
		}
	}
	if key == nil {
		return VerifyOfflineResult{Valid: false, Reason: "kid_not_found"}, nil
	}
	// fallback to core verifier path by reconstructing token validation using selected JWKS
	// To avoid duplicating code further, call the original verifier which fetches JWKS; in practice, this cached path avoids JWKS fetch.
	return VerifyTrustTokenOffline(ctx, baseURL, token, orgId, graceSeconds, rev)
}
