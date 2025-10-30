# Trust Tokens: Offline Validation (First-Class)

This guide covers how to issue, rotate, and verify AURA trust tokens at the edge without a live round-trip, while staying safe using JWKS, short expirations, and revocation feeds.

## Design

- Short-lived JWTs signed by organization trust keys
  - Algorithms: EdDSA (Ed25519) or ES256 (P-256)
  - Include `exp` and `jti` claims (and any additional app claims)
- Public keys are exposed via JWKS:
  - Global: `/.well-known/aura-jwks.json`
  - Org-scoped: `/.well-known/aura/{orgId}/jwks.json`
- Revocation feed per org with cheap conditional fetches:
  - `GET /organizations/{orgId}/trust-tokens/revocations`
  - Supports `If-None-Match` ETag; returns `{ items: [{ jti, revoked_at, reason? }] }`
  - `POST /organizations/{orgId}/trust-tokens/revocations` to revoke by JTI (admin)

## Rotation Strategy

- Prefer short expirations (e.g., 2–5 minutes) and rotate keys as needed via admin endpoints
- Clients verify `exp` and optionally add a small `graceSeconds` window for clock skew
- Maintain a local in-memory set of revoked JTIs, refresh via ETag periodically

## Offline Validators

Use the SDKs to verify tokens with JWKS and revocation feeds:

- Go: `sdks/go/aura`
  - `VerifyTrustTokenOffline` and `VerifyTrustTokenOfflineCached`
  - `TrustCache` for JWKS and revocations (TTL + ETag)
  - Optional net/http `TrustTokenMiddleware` for edge enforcement
- Node: `sdks/node`
  - `verifyTrustTokenOffline`, `fetchRevocations`, `fetchJWKS`
  - `TrustCache` for TTL + ETag
  - Express `auraTrustMiddleware` for local enforcement
- Python: `sdks/python`
  - `verify_trust_token_offline`, `TrustCaches`

## Example (Node)

```js
import { verifyTrustTokenOffline, TrustCache } from '@auraai/sdk-node'
const cache = new TrustCache({ jwksTTL: 300_000, revTTL: 60_000 })
const revoked = await cache.getRevocations(baseURL, orgId)
const res = await verifyTrustTokenOffline({ baseURL, token, orgId, graceSeconds: 30, revokedJTIs: revoked })
if (!res.valid) throw new Error(res.reason)
```

## Example (Go)

```go
cache := aura.NewTrustCache(5*time.Minute, 1*time.Minute)
res, err := aura.VerifyTrustTokenOfflineCached(ctx, cache, baseURL, token, orgId, 10)
if err != nil || !res.Valid { /* handle */ }
```

## Example (Python)

```python
from aura_sdk import verify_trust_token_offline, TrustCaches
cache = TrustCaches(jwks_ttl=300, rev_ttl=60)
rev = cache.get_revocations(base_url, org_id)
res = verify_trust_token_offline(token, base_url=base_url, org_id=org_id, grace_seconds=10, cache=cache)
```

## Security Notes

- Always enforce `exp`; avoid long-lived tokens
- Treat HS256 as unsupported for offline scenarios
- Prefer Ed25519 (EdDSA) where possible for compact keys
- Protect JWKS URLs behind CDN caching if appropriate; respect ETag on revocations
- Consider mutual TLS or request signing for higher-assurance channels
