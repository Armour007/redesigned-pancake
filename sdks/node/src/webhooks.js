/**
 * Webhook signature verification utilities (Node)
 */
import crypto from 'node:crypto';

/**
 * Compute signature using HMAC SHA256 over `${timestamp}.${payload}` (hex)
 * @param {string} secret
 * @param {number} timestamp seconds since epoch
 * @param {Uint8Array|string} payload raw body
 */
export function computeSignature(secret, timestamp, payload) {
  const h = crypto.createHmac('sha256', secret);
  const msg = Buffer.concat([
    Buffer.from(String(timestamp)),
    Buffer.from('.')
  ]);
  h.update(msg);
  h.update(Buffer.isBuffer(payload) ? payload : Buffer.from(payload));
  return h.digest('hex');
}

/**
 * Verify AURA-Signature header of form: t=<unix>, v1=<hex>
 * @param {Object} opts
 * @param {string} opts.secret
 * @param {string} opts.header signature header value
 * @param {Uint8Array|string} opts.payload raw body
 * @param {number} [opts.tolerance] seconds (default 300)
 */
export function verifySignature({ secret, header, payload, tolerance = 300 }) {
  if (!header) return false;
  const parts = Object.fromEntries(header.split(',').map(p => p.trim().split('=')));
  const ts = Number(parts.t);
  const sig = parts.v1;
  if (!ts || !sig) return false;
  // timestamp tolerance
  const now = Math.floor(Date.now() / 1000);
  if (Math.abs(now - ts) > tolerance) return false;
  const expected = computeSignature(secret, ts, payload);
  const a = Buffer.from(expected, 'hex');
  const b = Buffer.from(sig, 'hex');
  if (a.length !== b.length) return false;
  return crypto.timingSafeEqual(a, b);
}

/**
 * Express middleware example helper
 * @param {string} secret
 * @returns express middleware (req,res,next)
 */
export function expressVerifier(secret) {
  return (req, res, next) => {
    // ensure body is raw buffer: use express.raw({ type: 'application/json' }) before this
    const header = req.header('AURA-Signature');
    const ok = verifySignature({ secret, header, payload: req.body });
    if (!ok) return res.status(401).send('invalid signature');
    next();
  };
}
