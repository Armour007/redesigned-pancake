/**
 * AURA Node.js SDK
 */
const DEFAULT_BASE = process.env.AURA_API_BASE || 'http://localhost:8081';

export class AuraClient {
  /**
   * @param {object} opts
   * @param {string} opts.apiKey aura_sk_...
   * @param {string} [opts.baseURL]
   * @param {string} [opts.version] e.g. '2025-10-01'
   */
  constructor({ apiKey, baseURL = DEFAULT_BASE, version } = {}) {
    if (!apiKey) throw new Error('apiKey is required');
    this.apiKey = apiKey;
    this.baseURL = baseURL.replace(/\/$/, '');
    this.version = version;
  }

  /**
   * Verify a decision
   * @param {string} agentId UUID
   * @param {object} requestContext JSON-serializable context
   * @returns {Promise<{decision: 'ALLOWED'|'DENIED', reason: string}>}
   */
  async verify(agentId, requestContext) {
    const url = `${this.baseURL}/v1/verify`;
    const headers = {
      'Content-Type': 'application/json',
      'X-API-Key': this.apiKey,
    };
    if (this.version) headers['AURA-Version'] = this.version;
    const res = await fetch(url, {
      method: 'POST',
      headers,
      body: JSON.stringify({ agent_id: agentId, request_context: requestContext }),
    });
    if (!res.ok) {
      const text = await res.text().catch(() => '');
      throw new Error(`AURA verify failed: ${res.status} ${text}`);
    }
    return res.json();
  }
}

export { protect, protectExpress } from './protect.js'
export { buildRequestSigningHeaders } from './signing.js'
export { verifyTrustTokenOffline, fetchRevocations, fetchJWKS } from './trust.js'
export { TrustCache } from './trustCache.js'
export { auraTrustMiddleware } from './middleware.js'
