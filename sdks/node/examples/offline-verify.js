import { verifyTrustTokenOffline, fetchRevocations } from '../src/trust.js'

async function main() {
  const baseURL = process.env.AURA_API_BASE_URL || 'http://localhost:8081';
  const orgId = process.env.AURA_ORG_ID || 'org_00000000000000000000000000000000';
  const token = process.env.AURA_TRUST_TOKEN;
  if (!token) {
    console.error('Set AURA_TRUST_TOKEN to a JWT to verify');
    process.exit(1);
  }
  let revoked = new Set();
  try {
    const { items } = await fetchRevocations({ baseURL, orgId });
    revoked = new Set(items.map(i => i.jti));
  } catch {}
  const res = await verifyTrustTokenOffline({ baseURL, token, orgId, graceSeconds: 30, revokedJTIs: revoked });
  console.log(res);
}

main().catch((e) => { console.error(e); process.exit(1); })
