import { createPublicKey, verify as nodeVerify } from 'node:crypto'

export async function fetchJWKS(baseURL, orgId) {
  const url = orgId
    ? `${baseURL.replace(/\/$/, '')}/.well-known/aura/${orgId}/jwks.json`
    : `${baseURL.replace(/\/$/, '')}/.well-known/aura-jwks.json`;
  const res = await fetch(url);
  if (!res.ok) throw new Error(`JWKS fetch failed: ${res.status}`);
  return res.json();
}

function b64urlToBuf(s) {
  s = s.replace(/-/g, '+').replace(/_/g, '/');
  while (s.length % 4) s += '=';
  return Buffer.from(s, 'base64');
}

// Convert JOSE r||s (64 bytes) to DER ECDSA
function joseToDer(rs) {
  const r = rs.slice(0, 32);
  const s = rs.slice(32);
  const trim = (b) => {
    let i = 0;
    while (i < b.length - 1 && b[i] === 0) i++;
    // if high bit set, prefix 0x00
    if (b[i] & 0x80) return Buffer.concat([Buffer.from([0]), b.slice(i)]);
    return b.slice(i);
  };
  const rT = trim(r);
  const sT = trim(s);
  const len = 2 + rT.length + 2 + sT.length;
  return Buffer.concat([
    Buffer.from([0x30, len, 0x02, rT.length]), rT,
    Buffer.from([0x02, sT.length]), sT,
  ]);
}

export async function verifyTrustTokenOffline({ baseURL, token, orgId, graceSeconds = 0, revokedJTIs = new Set() }) {
  const parts = String(token || '').split('.');
  if (parts.length !== 3) return { valid: false, reason: 'bad_format' };
  let header, claims;
  try {
    header = JSON.parse(Buffer.from(parts[0], 'base64url').toString('utf8'));
    claims = JSON.parse(Buffer.from(parts[1], 'base64url').toString('utf8'));
  } catch {
    return { valid: false, reason: 'bad_json' };
  }
  const { alg, kid } = header || {};
  const now = Math.floor(Date.now() / 1000);
  const exp = Number(claims?.exp || 0);
  if (!exp) return { valid: false, reason: 'missing_exp' };
  if (now > exp + Number(graceSeconds || 0)) return { valid: false, reason: 'expired' };
  const jti = String(claims?.jti || '');
  if (jti && revokedJTIs && revokedJTIs.has(jti)) return { valid: false, reason: 'revoked' };

  const unsigned = Buffer.from(`${parts[0]}.${parts[1]}`, 'utf8');
  const sig = Buffer.from(parts[2], 'base64url');

  if (alg === 'HS256') {
    // Offline verify not supported without shared secret
    return { valid: false, reason: 'hs256_offline_unsupported' };
  }

  const jwks = await fetchJWKS(baseURL, orgId);
  const key = (jwks?.keys || []).find((k) => !kid || k.kid === kid);
  if (!key) return { valid: false, reason: 'kid_not_found' };

  if (alg === 'EdDSA' && key.kty === 'OKP' && key.crv === 'Ed25519') {
    try {
      const pub = createPublicKey({
        key: Buffer.concat([
          Buffer.from('302a300506032b6570032100', 'hex'),
          b64urlToBuf(key.x),
        ]), // ASN.1 SPKI for Ed25519
        format: 'der', type: 'spki'
      });
      const ok = nodeVerify(null, unsigned, pub, sig);
      return ok ? { valid: true, claims } : { valid: false, reason: 'bad_sig' };
    } catch (e) {
      return { valid: false, reason: 'verify_error' };
    }
  }
  if (alg === 'ES256' && key.kty === 'EC' && key.crv === 'P-256') {
    try {
      // Build SPKI for P-256 public key: 0x3059... per RFC5480
      const x = b64urlToBuf(key.x); const y = b64urlToBuf(key.y);
      const uncompressed = Buffer.concat([Buffer.from([0x04]), x, y]);
      const spki = Buffer.concat([
        Buffer.from('3059301306072a8648ce3d020106082a8648ce3d030107034200', 'hex'),
        uncompressed,
      ]);
      const pub = createPublicKey({ key: spki, format: 'der', type: 'spki' });
      const der = joseToDer(sig);
      const ok = nodeVerify('sha256', unsigned, pub, der);
      return ok ? { valid: true, claims } : { valid: false, reason: 'bad_sig' };
    } catch {
      return { valid: false, reason: 'verify_error' };
    }
  }
  return { valid: false, reason: 'unsupported_alg' };
}

export async function fetchRevocations({ baseURL, orgId, etag }) {
  const url = `${baseURL.replace(/\/$/, '')}/organizations/${orgId}/trust-tokens/revocations`;
  const headers = etag ? { 'If-None-Match': etag } : {};
  const res = await fetch(url, { headers });
  if (res.status === 304) return { notModified: true, etag };
  if (!res.ok) throw new Error(`revocations fetch failed: ${res.status}`);
  const body = await res.json();
  return { items: body.items || [], etag: res.headers.get('etag') || undefined };
}
