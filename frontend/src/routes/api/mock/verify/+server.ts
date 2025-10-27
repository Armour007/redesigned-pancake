import type { RequestHandler } from '@sveltejs/kit';

export const POST: RequestHandler = async ({ request }) => {
  try {
    const body = await request.json();
    const start = Date.now();
    await new Promise((r) => setTimeout(r, 200 + Math.random() * 300));
    const allow = Math.random() > 0.2;
    const resp = {
      decision: allow ? 'allow' : 'deny',
      reason: allow ? 'rule: demo-default' : 'risk: velocity high',
      request_id: 'req_' + Math.random().toString(36).slice(2, 10),
      latency_ms: Date.now() - start,
      echo: body
    };
    return new Response(JSON.stringify(resp), { status: 200, headers: { 'content-type': 'application/json' } });
  } catch (e) {
    return new Response(JSON.stringify({ error: 'bad_request' }), { status: 400, headers: { 'content-type': 'application/json' } });
  }
};
