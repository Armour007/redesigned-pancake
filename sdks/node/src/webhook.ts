import crypto from 'crypto';

export interface VerifyOptions {
  toleranceSec?: number; // default 300s
}

function parseSignatureHeader(header: string): { t: string; v1: string } | null {
  if (!header) return null;
  const parts = header.split(',').map(p => p.trim());
  let t = '', v1 = '';
  for (const p of parts) {
    const [k, v] = p.split('=');
    if (k === 't') t = v;
    if (k === 'v1') v1 = v;
  }
  if (!t || !v1) return null;
  return { t, v1 };
}

export function verifyAuraSignature(rawBody: string | Buffer, signatureHeader: string, secret: string, opts: VerifyOptions = {}): boolean {
  const parsed = parseSignatureHeader(signatureHeader);
  if (!parsed) return false;
  const { t, v1 } = parsed;
  const payload = `${t}.${typeof rawBody === 'string' ? rawBody : rawBody.toString('utf8')}`;
  const hmac = crypto.createHmac('sha256', secret).update(payload, 'utf8').digest('hex');

  // constant-time compare
  const a = Buffer.from(hmac, 'hex');
  const b = Buffer.from(v1, 'hex');
  if (a.length !== b.length) return false;
  const equal = crypto.timingSafeEqual(a, b);
  if (!equal) return false;

  const tol = opts.toleranceSec ?? 300;
  const ts = Number(t);
  if (!Number.isFinite(ts)) return false;
  const now = Math.floor(Date.now() / 1000);
  return Math.abs(now - ts) <= tol;
}
