import { http, HttpResponse } from 'msw';
import { API_BASE } from '$lib/api';

const api = (p: string) => `${API_BASE.replace(/\/$/, '')}${p}`;

export const handlers = [
  // Auth login
  http.post(api('/auth/login'), async ({ request }) => {
    const body = await request.json() as any;
    if (body?.email && body?.password) {
      return HttpResponse.json({ token: 'test_token' });
    }
    return HttpResponse.json({ error: 'invalid' }, { status: 400 });
  }),
  // Organizations mine
  http.get(api('/organizations/mine'), () => {
    return HttpResponse.json([{ id: 'org_1', name: 'Test Org' }]);
  }),
  // Webhooks DLQ list
  http.get(api('/admin/webhooks/dlq'), ({ request }) => {
    const url = new URL(request.url);
    const count = parseInt(url.searchParams.get('count') || '50', 10);
    const items = Array.from({ length: Math.min(count, 3) }).map((_, i) => ({
      id: `x-1${i}`, endpoint: 'ep_1', url: 'https://example.com', event: 'test', attempts: 1, last_code: 500, at: Math.floor(Date.now()/1000)
    }));
    return HttpResponse.json({ items, count: items.length, total: 3 });
  }),
];
