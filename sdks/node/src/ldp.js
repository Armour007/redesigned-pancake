import jsonld from 'jsonld';

const DEFAULT_BASE = process.env.AURA_API_BASE_URL || process.env.AURA_API_BASE || 'http://localhost:8081';

async function normalizeVC(vc) {
  const doc = { ...vc };
  delete doc.proof;
  const nquads = await jsonld.normalize(doc, {
    algorithm: 'URDNA2015',
    format: 'application/n-quads',
    processingMode: 'json-ld-1.1',
  });
  return nquads;
}

async function resolveDID(baseURL, did) {
  const base = (baseURL || DEFAULT_BASE).replace(/\/$/, '');
  const res = await fetch(`${base}/resolve?did=${encodeURIComponent(did)}`);
  if (!res.ok) throw new Error(`resolve ${res.status}`);
  return res.json();
}

export async function verifyVCLDP(vc, { baseURL, expectedOrgId, expectedOwner } = {}) {
  try {
    const proof = vc.proof || {};
    const jws = proof.jws;
    const vmid = proof.verificationMethod;
    if (!jws || !vmid) return { valid: false, reason: 'invalid_proof' };
    const parts = jws.split('.');
    if (parts.length !== 3 || parts[1] !== '') return { valid: false, reason: 'not_detached' };
    const protectedB64 = parts[0];
    const sigB64 = parts[2];
    const ph = JSON.parse(Buffer.from(protectedB64, 'base64url').toString('utf8'));
    const alg = ph.alg;
    if (ph.b64 !== false) return { valid: false, reason: 'b64_must_be_false' };
    const nquads = await normalizeVC(vc);
    const signingInput = Buffer.concat([
      Buffer.from(protectedB64 + '.', 'utf8'),
      Buffer.from(nquads, 'utf8'),
    ]);
    const sig = Buffer.from(sigB64, 'base64url');
    const did = vmid.includes('#') ? vmid.slice(0, vmid.indexOf('#')) : vmid;
    const doc = await resolveDID(baseURL, did);
    const vm = (doc.verificationMethod || []).find(v => v.id === vmid);
    if (!vm) return { valid: false, reason: 'vm_not_found' };
    const jwk = vm.publicKeyJwk || {};
    // Build KeyObject from JWK
    const crypto = await import('node:crypto');
    let key;
    try {
      key = crypto.createPublicKey({ key: jwk, format: 'jwk' });
    } catch {
      return { valid: false, reason: 'bad_jwk' };
    }
    let ok = false;
    if (alg === 'EdDSA' && jwk.kty === 'OKP' && jwk.crv === 'Ed25519') {
      ok = crypto.verify(null, signingInput, key, sig);
    } else if (alg === 'ES256' && jwk.kty === 'EC' && jwk.crv === 'P-256') {
      ok = crypto.verify('sha256', signingInput, { key, dsaEncoding: 'ieee-p1363' }, sig);
    } else {
      return { valid: false, reason: 'unsupported_alg' };
    }
    if (!ok) return { valid: false, reason: 'bad_sig' };
    if (expectedOrgId) {
      const want = `did:aura:org:${expectedOrgId}`;
      if (vc.issuer !== want) return { valid: false, reason: 'issuer_mismatch' };
    }
    if (expectedOwner) {
      const owner = (vc.credentialSubject && vc.credentialSubject.owner) || undefined;
      if (owner !== expectedOwner) return { valid: false, reason: 'owner_mismatch' };
    }
    return { valid: true };
  } catch (e) {
    return { valid: false, reason: 'verify_error' };
  }
}

export async function normalizeForSigning(vc) {
  return normalizeVC(vc);
}
