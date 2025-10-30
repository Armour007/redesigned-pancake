import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  vus: Number(__ENV.VUS || 20),
  duration: __ENV.DURATION || '1m',
  thresholds: {
    http_req_failed: ['rate<0.01'],
    http_req_duration: ['p(95)<500'],
  },
};

const BASE = __ENV.AURA_API_BASE || 'http://localhost:8081';
const API_KEY = __ENV.AURA_API_KEY || 'aura_sk_test';
const AGENT_ID = __ENV.AGENT_ID || '00000000-0000-0000-0000-000000000000';

export default function () {
  const url = `${BASE}/v1/verify`;
  const payload = JSON.stringify({
    agent_id: AGENT_ID,
    request_context: { action: 'read', resource: 'doc:1' },
  });
  const res = http.post(url, payload, {
    headers: {
      'Content-Type': 'application/json',
      'X-API-Key': API_KEY,
      'AURA-Version': '2025-10-01',
    },
  });
  check(res, { 'status is 200': (r) => r.status === 200 });
  sleep(0.2);
}
