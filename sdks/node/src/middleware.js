import { verifyTrustTokenOffline } from './trust.js'
import { TrustCache } from './trustCache.js'

/**
 * Express middleware that verifies a trust token offline and attaches claims to req.auraClaims.
 * 401 with { error, reason } on failure.
 */
export function auraTrustMiddleware({ baseURL, orgId, cache = new TrustCache(), graceSeconds = 0, header = 'authorization' } = {}) {
  if (!baseURL) throw new Error('baseURL is required')
  if (!orgId) throw new Error('orgId is required')
  return async function (req, res, next) {
    try {
      const raw = req.headers[header]
      if (!raw) return res.status(401).json({ error: 'missing_token', reason: 'No Authorization header' })
      const m = String(raw).match(/Bearer\s+(.+)/i)
      const token = m ? m[1] : String(raw)
      const revoked = await cache.getRevocations(baseURL, orgId)
      const result = await verifyTrustTokenOffline({ baseURL, token, orgId, graceSeconds, revokedJTIs: revoked })
      if (!result.valid) return res.status(401).json({ error: 'invalid_token', reason: result.reason })
      req.auraClaims = result.claims
      return next()
    } catch (e) {
      return res.status(401).json({ error: 'verify_error' })
    }
  }
}
