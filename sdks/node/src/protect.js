// Lightweight AURA adapter for Node.js
// Usage:
//   import { AuraClient, protect, protectExpress } from '@aura/sdk'
//   const client = new AuraClient({ apiKey: process.env.AURA_API_KEY, baseURL: process.env.AURA_API_BASE_URL })
//   const secured = protect({ client, agentId: process.env.AURA_AGENT_ID })(async function dangerous() { /* ... */ })
//   app.post('/op', protectExpress({ client, agentId, contextFromReq: (req) => ({ path: req.path, method: req.method }) }), handler)

/**
 * Wrap any async function and verify with AURA before executing it.
 * @param {object} opts
 * @param {import('./index.js').AuraClient} opts.client
 * @param {string} opts.agentId
 * @param {(fn:Function, args:any[])=>any} [opts.context]
 * @param {(reason:string, fn:Function, args:any[])=>any} [opts.onDeny]
 */
export function protect({ client, agentId, context, onDeny }) {
  if (!client) throw new Error('protect: client required')
  if (!agentId) throw new Error('protect: agentId required')
  const ctxBuilder = context || ((fn, args) => ({ function: fn.name || 'anonymous', args }))
  return function wrap(fn) {
    if (typeof fn !== 'function') throw new Error('protect: expected a function')
    return async function protectedFn(...args) {
      const ctx = ctxBuilder(fn, args)
      const res = await client.verify(agentId, ctx)
      if (!res || res.decision !== 'ALLOWED') {
        const reason = (res && res.reason) || 'access denied'
        if (typeof onDeny === 'function') return onDeny(reason, fn, args)
        throw new Error('AURA denied: ' + reason)
      }
      return fn.apply(this, args)
    }
  }
}

/**
 * Express middleware factory that verifies with AURA using request-derived context.
 * @param {object} opts
 * @param {import('./index.js').AuraClient} opts.client
 * @param {string} opts.agentId
 * @param {(req:import('http').IncomingMessage & { body?:any })=>any} [opts.contextFromReq]
 * @returns {import('express').RequestHandler}
 */
export function protectExpress({ client, agentId, contextFromReq }) {
  const ctx = contextFromReq || ((req) => ({ path: req.path, method: req.method, body: req.body }))
  return async function (req, res, next) {
    try {
      const rc = ctx(req)
      const out = await client.verify(agentId, rc)
      if (!out || out.decision !== 'ALLOWED') {
        const reason = (out && out.reason) || 'access denied'
        res.status(403).json({ error: 'AURA denied', reason })
        return
      }
      next()
    } catch (e) {
      next(e)
    }
  }
}
