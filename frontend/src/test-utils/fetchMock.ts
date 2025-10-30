import { API_BASE as API_BASE_DEFAULT, authHeaders } from '$lib/api';

export type RestoreFn = () => void;

export function installFetchMock(apiBase: string = API_BASE_DEFAULT): RestoreFn {
  const orig = globalThis.fetch.bind(globalThis);
  const api = (p: string) => `${apiBase.replace(/\/$/, '')}${p}`;

  globalThis.fetch = (async (input: RequestInfo | URL, init?: RequestInit) => {
    const url = typeof input === 'string' ? input : input instanceof URL ? input.toString() : input.url;
    const method = (init?.method || 'GET').toUpperCase();

    // Login
    if (url === api('/auth/login') && method === 'POST') {
      try {
        const body = init?.body ? JSON.parse(String(init.body)) : {};
        if (body?.email && body?.password && body?.password !== 'bad') {
          return new Response(JSON.stringify({ token: 'test_token' }), {
            status: 200,
            headers: { 'Content-Type': 'application/json' }
          });
        } else {
          return new Response(JSON.stringify({ error: 'invalid' }), {
            status: 400,
            headers: { 'Content-Type': 'application/json' }
          });
        }
      } catch {
        return new Response(JSON.stringify({ error: 'invalid' }), { status: 400, headers: { 'Content-Type': 'application/json' } });
      }
    }

    // Orgs
    if (url === api('/organizations/mine') && method === 'GET') {
      return new Response(JSON.stringify([{ id: 'org_1', name: 'Test Org' }]), {
        status: 200,
        headers: { 'Content-Type': 'application/json' }
      });
    }

    // Audit export schedule
    const auditGet = api('/organizations/org_1/regulator/audit-export/schedule');
    if (url === auditGet && method === 'GET') {
      return new Response(JSON.stringify({ schedule: { cron: '0 2 * * *', destType: 'file', dest: '', format: 'json', lookback: '720h' } }), {
        status: 200, headers: { 'Content-Type': 'application/json' }
      });
    }
    if (url === auditGet && method === 'POST') {
      try {
        const body = init?.body ? JSON.parse(String(init.body)) : {};
        if (body?.dest === 'fail') {
          return new Response(JSON.stringify({ error: 'bad dest' }), { status: 500, headers: { 'Content-Type': 'application/json' } });
        }
        return new Response(JSON.stringify({ ok: true }), { status: 200, headers: { 'Content-Type': 'application/json' } });
      } catch {
        return new Response(JSON.stringify({ error: 'invalid' }), { status: 400, headers: { 'Content-Type': 'application/json' } });
      }
    }

    // Webhooks DLQ list
    if (url.startsWith(api('/admin/webhooks/dlq')) && method === 'GET') {
      const items = Array.from({ length: 3 }).map((_, i) => ({
        id: `x-1${i}`, endpoint: 'ep_1', url: 'https://example.com', event: 'test', attempts: 1, last_code: 500, at: Math.floor(Date.now()/1000)
      }));
      return new Response(JSON.stringify({ items, count: items.length, total: 3 }), {
        status: 200, headers: { 'Content-Type': 'application/json' }
      });
    }

    // Webhooks DLQ actions
    if (url === api('/admin/webhooks/dlq/requeue') && method === 'POST') {
      return new Response(JSON.stringify({ ok: true }), { status: 200, headers: { 'Content-Type': 'application/json' } });
    }
    if (url === api('/admin/webhooks/dlq/delete') && method === 'POST') {
      return new Response(JSON.stringify({ ok: true }), { status: 200, headers: { 'Content-Type': 'application/json' } });
    }

    return orig(input as any, init);
  }) as typeof fetch;

  return () => {
    globalThis.fetch = orig;
  };
}
