# AURA Node SDK

Minimal client and webhook verifier for AURA.

## Install

This repo vendors the SDK in `sdks/node`. To use locally:
- Import from the path directly in your monorepo tooling, or publish to your registry.

## Usage

```js
import { AuraClient } from '@auraai/sdk-node';

const client = new AuraClient({
  apiKey: process.env.AURA_API_KEY,               // aura_sk_...
  baseURL: process.env.AURA_API_BASE,             // e.g. https://api.example.com
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
- AURA_API_BASE: Backend base URL
- AURA_VERSION: Optional API version header (default `2025-10-01`)
- AURA_WEBHOOK_SECRET: For verifying webhook signatures
