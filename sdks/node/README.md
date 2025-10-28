# AURA Node SDK

Minimal client, function wrapper, Express middleware, and webhook verifier for AURA.

## Install

This repo vendors the SDK in `sdks/node`. To use locally:
- Import from the path in your monorepo tooling, or publish to your registry.

## 1‑minute plug‑in: function wrapper

Protect any async function with a tiny wrapper. The adapter calls AURA and only executes when allowed.

```js
import { AuraClient, protect } from '@auraai/sdk-node';

const client = new AuraClient({
  apiKey: process.env.AURA_API_KEY,               // aura_sk_...
  baseURL: process.env.AURA_API_BASE_URL || process.env.AURA_API_BASE || 'http://localhost:8081',
  version: process.env.AURA_VERSION || '2025-10-01'
});

const secureDangerous = protect({ client, agentId: process.env.AURA_AGENT_ID })(async function dangerous(userId) {
  // ... sensitive code ...
  return `deleted ${userId}`
})

await secureDangerous('123')
```

### Express middleware

```js
import express from 'express'
import { AuraClient, protectExpress } from '@auraai/sdk-node'

const app = express()
app.use(express.json())

const client = new AuraClient({
  apiKey: process.env.AURA_API_KEY,
  baseURL: process.env.AURA_API_BASE_URL || 'http://localhost:8081'
})

app.post('/danger',
  protectExpress({ client, agentId: process.env.AURA_AGENT_ID, contextFromReq: req => ({ path: req.path, method: req.method, user: req.user?.id }) }),
  (req, res) => res.json({ ok: true })
)
```

## Client usage

```js
import { AuraClient } from '@auraai/sdk-node';

const client = new AuraClient({
  apiKey: process.env.AURA_API_KEY,
  baseURL: process.env.AURA_API_BASE_URL || process.env.AURA_API_BASE || 'http://localhost:8081',
  version: process.env.AURA_VERSION || '2025-10-01'
});

const res = await client.verify(process.env.AURA_AGENT_ID, {
  action: 'deploy:prod',
  branch: 'main'
});
console.log(res.decision, res.reason);
```

### Webhooks (Express)

```js
import express from 'express';
import { expressVerifier } from '@auraai/sdk-node/webhooks';

const app = express();
app.post('/webhooks/aura', express.raw({ type: 'application/json' }), expressVerifier(process.env.AURA_WEBHOOK_SECRET), (req, res) => {
  const event = JSON.parse(req.body.toString());
  console.log('AURA event', event.type);
  res.sendStatus(200);
});
```

## Env
- AURA_API_KEY: aura_sk_...
- AURA_API_BASE_URL (or AURA_API_BASE): Backend base URL (default http://localhost:8081)
- AURA_VERSION: Optional API version header (default `2025-10-01`)
- AURA_WEBHOOK_SECRET: For verifying webhook signatures
