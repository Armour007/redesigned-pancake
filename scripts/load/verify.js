import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  scenarios: {
    ramp: {
      executor: 'ramping-arrival-rate',
      startRate: 1000,
      stages: [
        { target: 5000, duration: '1m' },
        { target: 10000, duration: '1m' },
      ],
      preAllocatedVUs: 200,
      maxVUs: 1000,
    },
  },
  thresholds: {
    http_req_failed: ['rate<0.01'],
    http_req_duration: ['p(95)<1000', 'p(99)<2000'],
  },
};

const BASE = __ENV.AURA_API_BASE_URL || 'http://localhost:8081';
const API_KEY = __ENV.AURA_API_KEY || '';
const AGENT_ID = __ENV.AURA_AGENT_ID || '00000000-0000-0000-0000-000000000000';

export default function () {
  const url = `${BASE}/v1/verify`;
  const payload = JSON.stringify({ agent_id: AGENT_ID, request_context: { action: 'test', n: Math.random() } });
  const params = { headers: { 'Content-Type': 'application/json', 'X-API-Key': API_KEY } };
  const res = http.post(url, payload, params);
  check(res, { 'status is 200': (r) => r.status === 200 });
  sleep(0.01);
}
