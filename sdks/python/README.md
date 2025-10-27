# AURA Python SDK

Minimal client and webhook verifier for AURA.

## Install

Local usage from this monorepo:

```bash
pip install -e ./sdks/python
```

## Usage

```python
from aura_sdk import AuraClient
import os

client = AuraClient(
    api_key=os.environ['AURA_API_KEY'],
    base_url=os.environ.get('AURA_API_BASE'),
    version=os.environ.get('AURA_VERSION', '2025-10-01'),
)

resp = client.verify(os.environ['AURA_AGENT_ID'], { 'action': 'deploy:prod', 'branch': 'main' })
print(resp)
```

### Webhooks (FastAPI)

```python
from fastapi import FastAPI, Request, Header, HTTPException
from aura_sdk import verify_signature

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
- AURA_API_BASE: Backend base URL
- AURA_VERSION: Optional API version header (default `2025-10-01`)
- AURA_WEBHOOK_SECRET: For verifying webhook signatures
