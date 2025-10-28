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
