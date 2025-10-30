import crypto from 'node:crypto'

// Build X-Aura-* signing headers for request authentication
// secret: shared HMAC secret
// method: HTTP method (GET/POST/...)
// path: request path (e.g., /v1/verify) without domain
// body: Buffer or string of request body
export function buildRequestSigningHeaders(secret, method, path, body = '') {
  const ts = Math.floor(Date.now() / 1000).toString()
  const nonce = crypto.randomBytes(12).toString('hex')
  const b = typeof body === 'string' ? body : Buffer.from(body || '').toString()
  const unsigned = [method.toUpperCase(), path, ts, nonce, b].join('\n')
  const h = crypto.createHmac('sha256', secret).update(unsigned).digest('hex')
  return {
    'X-Aura-Timestamp': ts,
    'X-Aura-Nonce': nonce,
    'X-Aura-Signature': h,
  }
}
