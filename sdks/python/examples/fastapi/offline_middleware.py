from fastapi import FastAPI, Request, Response
from aura_sdk import verify_trust_token_offline, TrustCaches
import os

app = FastAPI()
base = os.environ.get('AURA_API_BASE_URL', 'http://localhost:8081')
org_id = os.environ.get('ORG_ID', '')
cache = TrustCaches(jwks_ttl=300, rev_ttl=60)

def bearer_token(auth: str | None) -> str | None:
    if not auth:
        return None
    if auth.lower().startswith('bearer '):
        return auth[7:].strip()
    return auth

@app.middleware('http')
async def trust_token_offline(request: Request, call_next):
    token = bearer_token(request.headers.get('authorization'))
    if not token:
        return Response(status_code=401, content='{"error":"missing_token"}', media_type='application/json')
    try:
        # ensure revocations are reasonably fresh
        _ = cache.get_revocations(base, org_id)
        res = verify_trust_token_offline(token, base_url=base, org_id=org_id, grace_seconds=30, cache=cache)
        if not res.get('valid'):
            return Response(status_code=401, content='{"error":"invalid_token","reason":"%s"}' % res.get('reason'), media_type='application/json')
    except Exception:
        return Response(status_code=401, content='{"error":"verify_error"}', media_type='application/json')
    return await call_next(request)

@app.get('/secure')
async def secure():
    return { 'ok': True }
