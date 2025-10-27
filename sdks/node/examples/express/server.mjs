import express from 'express';
import { AuraClient } from '../../src/index.js';
import { expressVerifier } from '../../src/webhooks.js';

const app = express();
// use raw body for webhook verification
app.post('/webhooks/aura', express.raw({ type: 'application/json' }), expressVerifier(process.env.AURA_WEBHOOK_SECRET), (req, res) => {
  const event = JSON.parse(req.body.toString());
  console.log('AURA event:', event.type, event.data);
  res.sendStatus(200);
});

app.use(express.json());

app.get('/verify', async (req, res) => {
  const client = new AuraClient({ apiKey: process.env.AURA_API_KEY, baseURL: process.env.AURA_API_BASE, version: process.env.AURA_VERSION });
  const agentId = process.env.AURA_AGENT_ID;
  const ctx = { action: 'deploy:prod', branch: 'main' };
  try {
    const decision = await client.verify(agentId, ctx);
    res.json(decision);
  } catch (e) {
    res.status(500).json({ error: String(e) });
  }
});

const port = process.env.PORT || 3001;
app.listen(port, () => console.log(`Example listening on ${port}`));
