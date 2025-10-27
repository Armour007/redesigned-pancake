# AURA Client Guide

This guide helps your engineering team integrate AURA via API or SDKs and operate the SaaS with observability.

## 1) Concepts
- Organization: your tenant. Users belong to organizations.
- Agents: identities you authorize (bots, services, LLM apps).
- Permissions (rules): JSON policies attached to agents.
- API keys: aura_sk_... credentials for machine calls to /v1/verify.
- Events & Webhooks: signed decision events your systems can react to.

## 2) Provisioning flow
1. Sign up and log in (Dashboard or API).
2. Create an Agent in your Organization.
3. Add Permission rules (allow/deny with optional conditions/time windows).
4. Create an API key for machine access.
5. (Optional) Create a Webhook endpoint to receive decision events.

## 3) Verify decisions (SDKs)

Headers: you may add `AURA-Version: 2025-10-01` to pin behavior.

- Node
```js
import { AuraClient } from '@auraai/sdk-node';
const client = new AuraClient({ apiKey: process.env.AURA_API_KEY, baseURL: process.env.AURA_API_BASE, version: '2025-10-01' });
const res = await client.verify(process.env.AURA_AGENT_ID, { action: 'deploy:prod', branch: 'main' });
```

- Python
```python
from aura_sdk import AuraClient
client = AuraClient(api_key=os.environ['AURA_API_KEY'], base_url=os.environ['AURA_API_BASE'], version='2025-10-01')
res = client.verify(os.environ['AURA_AGENT_ID'], { 'action': 'deploy:prod', 'branch': 'main' })
```

- Go
```go
c := aura.NewClient(os.Getenv("AURA_API_KEY"), os.Getenv("AURA_API_BASE"), "2025-10-01")
res, err := c.Verify(os.Getenv("AURA_AGENT_ID"), map[string]any{"action":"deploy:prod","branch":"main"})
```

## 4) Webhooks
- Configure a webhook endpoint URL in the dashboard.
- We sign events with `AURA-Signature: t=<unix>, v1=<hex>` computed over "{ts}.{body}" using your endpoint secret.
- Verify on your side using SDK helpers.

- Node (Express)
```js
app.post('/webhooks/aura', express.raw({ type: 'application/json' }), expressVerifier(process.env.AURA_WEBHOOK_SECRET), (req, res) => {
  const event = JSON.parse(req.body.toString());
  res.sendStatus(200);
});
```

- Python (FastAPI)
```python
@app.post('/webhooks/aura')
async def webhook(request: Request, aura_signature: str = Header(None)):
    body = await request.body()
    if not verify_signature(os.environ['AURA_WEBHOOK_SECRET'], aura_signature, body):
        raise HTTPException(status_code=401, detail='invalid signature')
    return { 'ok': True }
```

- Go (net/http)
```go
ok, _ := aura.VerifySignature(os.Getenv("AURA_WEBHOOK_SECRET"), r.Header.Get("AURA-Signature"), body, 0)
if !ok { w.WriteHeader(401); return }
```

## 5) API versioning and idempotency
- AURA-Version (request): optional header to pin behavior. Response includes X-AURA-Version.
- Idempotency-Key (request): use on POST mutations (create agent/key/rule/webhook). First 2xx result is cached 24h; retries return same body with header `X-Idempotent-Replay: true`.

## 6) Observability
- Metrics: Prometheus at /metrics. Use Grafana to see latency and decisions.
- Tracing: enable with AURA_OTEL_ENABLE=true; traces → OTEL Collector → Tempo → Grafana Explore.

Docker Compose stack (prod-ish):
- Backend: http://localhost:8080
- Frontend: http://localhost:3000
- Prometheus: http://localhost:9090
- Grafana: http://localhost:3001
- Tempo: http://localhost:3200 (Grafana uses this datasource)

In Grafana:
- Prometheus datasource pre-provisioned.
- Tempo datasource pre-provisioned.
- Dashboard "AURA Overview" is available.
- Use Explore → Tempo to see traces; filter by service=aura-backend.

## 7) Security & headers
- Always send API calls over HTTPS in production.
- Restrict API keys; rotate regularly.
- CORS: configure AURA_CORS_ORIGINS.
- Trusted proxies: if behind a load balancer, set AURA_TRUSTED_PROXIES.

## 8) Quick PowerShell snippets

Verify with curl:
```powershell
curl -Method POST -Uri "$env:AURA_API_BASE/v1/verify" -Headers @{
  'Content-Type' = 'application/json';
  'X-API-Key'    = $env:AURA_API_KEY;
  'AURA-Version' = '2025-10-01'
} -Body (@{ agent_id = $env:AURA_AGENT_ID; request_context = @{ action = 'deploy:prod'; branch = 'main' } } | ConvertTo-Json)
```

Bring up stack:
```powershell
$env:POSTGRES_PASSWORD='supersecret'
$env:JWT_SECRET='please_change'
$env:AURA_OTEL_ENABLE='true'
docker compose -f docker-compose.deploy.yml up -d
```

## 9) Marketing configuration

- Demo form → webhook/CRM
  - Frontend endpoint `/api/lead` forwards to `$AURA_DEMO_WEBHOOK_URL` if set (server env). Example:
  ```powershell
  $env:AURA_DEMO_WEBHOOK_URL = 'https://hooks.zapier.com/hooks/catch/xxxx/yyyy'
  npm run build; npm run preview
  ```
- Logos marquee
  - Replace files under `frontend/static/logos/*.svg` with your approved monochrome logos.
  - Update alt text in `src/lib/marketing/logos.ts`.
- Open Graph image
  - We generate `static/og-image.png` from `static/og-image.svg` at build time using `sharp`.
  - To update: edit `static/og-image.svg`, then run `npm run build`.

## 10) Support
- OpenAPI docs: /docs and /openapi.json
- Example SDKs under sdks/node, sdks/python, sdks/go/aura
- Reach out to your AURA support contact for enterprise integration help.
