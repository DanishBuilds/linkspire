// test/load/api_load.js
// Load test for the management API (Phase 4): register once, then create and
// list links under sustained load. Run:
//   k6 run -e API_URL=http://localhost:8081 test/load/api_load.js
import http from 'k6/http';
import { check } from 'k6';

const API = __ENV.API_URL || 'http://localhost:8081';

export const options = {
  scenarios: {
    api: {
      executor: 'ramping-vus',
      startVUs: 5,
      stages: [
        { target: 50, duration: '1m' },
        { target: 50, duration: '2m' },
        { target: 0, duration: '30s' },
      ],
    },
  },
  thresholds: {
    http_req_duration: ['p(95)<200'],
    http_req_failed: ['rate<0.01'],
    checks: ['rate>0.99'],
  },
};

export function setup() {
  const email = `load+${Date.now()}@example.com`;
  const res = http.post(`${API}/v1/auth/register`, JSON.stringify({
    email, password: 'load-test-pass-123', home_region: 'us',
  }), { headers: { 'Content-Type': 'application/json' } });
  check(res, { 'registered': (r) => r.status === 201 });
  return { token: res.json('token') };
}

export default function (data) {
  const headers = { 'Content-Type': 'application/json', Authorization: `Bearer ${data.token}` };
  const create = http.post(`${API}/v1/links`, JSON.stringify({
    destination: `https://example.com/${Math.random().toString(36).slice(2)}`,
  }), { headers });
  check(create, { 'link created (201)': (r) => r.status === 201 });

  const list = http.get(`${API}/v1/links?limit=20`, { headers });
  check(list, { 'list ok (200)': (r) => r.status === 200 });
}
