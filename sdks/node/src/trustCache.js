import { fetchJWKS, fetchRevocations } from './trust.js'

export class TrustCache {
  constructor({ jwksTTL = 300_000, revTTL = 60_000 } = {}) {
    this.jwksTTL = jwksTTL
    this.revTTL = revTTL
    this._jwks = new Map() // key: base|org -> { at, data }
    this._rev = new Map()  // key: base|org -> { at, etag, set }
  }

  _key(baseURL, orgId) { return `${String(baseURL).replace(/\/$/, '')}|${orgId || ''}` }

  async getJWKS(baseURL, orgId) {
    const k = this._key(baseURL, orgId)
    const now = Date.now()
    const cur = this._jwks.get(k)
    if (cur && (now - cur.at) < this.jwksTTL) return cur.data
    const fresh = await fetchJWKS(baseURL, orgId)
    this._jwks.set(k, { at: now, data: fresh })
    return fresh
  }

  async getRevocations(baseURL, orgId) {
    const k = this._key(baseURL, orgId)
    const now = Date.now()
    const cur = this._rev.get(k) || { at: 0, etag: undefined, set: new Set() }
    if ((now - cur.at) < this.revTTL) {
      // best-effort conditional refresh
      try {
        const { items, etag, notModified } = await fetchRevocations({ baseURL, orgId, etag: cur.etag })
        if (!notModified && items && items.length) {
          cur.set = new Set(items.map(i => i.jti).filter(Boolean))
          cur.etag = etag
          cur.at = now
          this._rev.set(k, cur)
        }
      } catch {}
      return cur.set
    }
    // hard refresh
    const { items, etag, notModified } = await fetchRevocations({ baseURL, orgId, etag: cur.etag })
    if (!notModified && items) {
      cur.set = new Set(items.map(i => i.jti).filter(Boolean))
      cur.etag = etag
    }
    cur.at = now
    this._rev.set(k, cur)
    return cur.set
  }
}
