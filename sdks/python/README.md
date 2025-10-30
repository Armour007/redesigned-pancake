# AURA Python SDK

Minimal client, decorator adapter, and webhook verifier for AURA.

## Install

Local usage from this monorepo:

```bash
pip install -e ./sdks/python
```

## 1‑minute plug‑in: decorator

Annotate sensitive functions with a single decorator. The adapter sends your context to AURA and only runs the function when the decision is ALLOWED.

```python
import os
from aura_sdk import protect

os.environ.setdefault('AURA_API_BASE_URL', 'http://localhost:8081')
os.environ['AURA_API_KEY'] = 'aura_sk_...'        # set in your env
os.environ['AURA_AGENT_ID'] = '<uuid>'            # set in your env

@protect()  # reads AURA_API_KEY/AURA_AGENT_ID/AURA_API_BASE_URL
def dangerous_function(user_id: str):
    # ... code that accesses sensitive resources ...
    return f"deleted user {user_id}"

print(dangerous_function('123'))
```

Customize:
- @protect(agent_id="...") to pass explicit agent
- @protect(on_deny="return_none") or on_deny=lambda reason, f, a, kw: ...
- @protect(context_builder=lambda f,a,kw: {"function": f.__name__, "args": a})

## Client usage

```python
from aura_sdk import AuraClient
import os

client = AuraClient(
    api_key=os.environ['AURA_API_KEY'],
    base_url=os.environ.get('AURA_API_BASE_URL') or os.environ.get('AURA_API_BASE'),
    version=os.environ.get('AURA_VERSION', '2025-10-01'),
)

resp = client.verify(os.environ['AURA_AGENT_ID'], { 'action': 'deploy:prod', 'branch': 'main' })
print(resp)
```

### Webhooks (FastAPI)

```python
from fastapi import FastAPI, Request, Header, HTTPException
from aura_sdk import verify_signature
import os

app = FastAPI()

@app.post('/webhooks/aura')
async def webhook(request: Request, aura_signature: str = Header(None)):
    body = await request.body()
    if not verify_signature(os.environ['AURA_WEBHOOK_SECRET'], aura_signature, body):
        raise HTTPException(status_code=401, detail='invalid signature')
    return { 'ok': True }
```

## Env
- AURA_API_KEY: aura_sk_...
- AURA_API_BASE_URL (or AURA_API_BASE): Backend base URL (default http://localhost:8081)
- AURA_VERSION: Optional API version header (default `2025-10-01`)
- AURA_WEBHOOK_SECRET: For verifying webhook signatures

## Offline Trust Tokens

Verify short-lived trust tokens locally using JWKS and optional revocation sync. Supported algorithms: EdDSA (Ed25519) and ES256. HS256 is not supported offline.

- Exp enforced with optional grace window
- Revocations fetched from `/organizations/{orgId}/trust-tokens/revocations` with ETag for cheap refresh
- Use `TrustCaches` to cache JWKS and revocations with TTLs

Example:

```python
from aura_sdk import verify_trust_token_offline, TrustCaches

cache = TrustCaches(jwks_ttl=300, rev_ttl=60)
res = verify_trust_token_offline(token, base_url=base_url, org_id=org_id, grace_seconds=10, cache=cache)
if not res.get('valid'):
    print('invalid', res.get('reason'))
else:
    print('claims', res.get('claims'))
```

See `examples/offline_verify.py` for a runnable snippet.

## Verify VC (JSON-LD, JsonWebSignature2020)

Verify a Verifiable Credential with a JSON-LD linked data proof (URDNA2015 + detached JWS with b64=false):

```python
from aura_sdk import verify_vc_ldp

# vc is the VC JSON object (including the `proof` field with `jws` detached signature)
res = verify_vc_ldp(vc, base_url=os.environ.get('AURA_API_BASE_URL'), expected_org_id="<org-uuid>", expected_owner="alice")
if not res.get('valid'):
    print('invalid', res.get('reason'))
```

The helper normalizes the VC using URDNA2015, resolves the DID Document via `/resolve`, and verifies the detached JWS for EdDSA (Ed25519) or ES256 against the referenced verification method.

## Local evaluator (FastAPI middleware)

Enforce trust tokens offline at the edge using a simple middleware and the built-in caches:

```python
from fastapi import FastAPI
from aura_sdk import verify_trust_token_offline, TrustCaches

app = FastAPI()
cache = TrustCaches(jwks_ttl=300, rev_ttl=60)

@app.middleware('http')
async def trust_token_offline(request, call_next):
    auth = request.headers.get('authorization')
    token = auth.split(' ', 1)[1] if auth and auth.lower().startswith('bearer ') else auth
    if not token:
        from fastapi import Response
        return Response(status_code=401, content='{"error":"missing_token"}', media_type='application/json')
    _ = cache.get_revocations('http://localhost:8081', '<org-id>')
    res = verify_trust_token_offline(token, base_url='http://localhost:8081', org_id='<org-id>', grace_seconds=30, cache=cache)
    if not res.get('valid'):
        from fastapi import Response
        return Response(status_code=401, content='{"error":"invalid_token"}', media_type='application/json')
    return await call_next(request)
```
