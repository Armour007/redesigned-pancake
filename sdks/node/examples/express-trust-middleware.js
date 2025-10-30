import express from 'express'
import { auraTrustMiddleware, TrustCache } from '../src/index.js'

const app = express()
const baseURL = process.env.AURA_API_BASE_URL || 'http://localhost:8081'
const orgId = process.env.ORG_ID || '00000000-0000-0000-0000-000000000001'

const cache = new TrustCache({ jwksTTL: 300_000, revTTL: 60_000 })

app.get('/secure', auraTrustMiddleware({ baseURL, orgId, cache, graceSeconds: 30 }), (req, res) => {
  res.json({ ok: true, claims: req.auraClaims })
})

app.listen(3000, () => console.log('listening on :3000'))
