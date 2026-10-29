// test/load/smoke.js
// Minimal smoke test: are the services up and healthy? Run:
//   k6 run -e REDIRECT_URL=http://localhost:9090 -e API_URL=http://localhost:9091 test/load/smoke.js
import http from 'k6/http';
import { check } from 'k6';

const REDIRECT = __ENV.REDIRECT_URL || 'http://localhost:9090';
const API = __ENV.API_URL || 'http://localhost:9091';

export const options = {
  vus: 1,
  iterations: 1,
  thresholds: { checks: ['rate==1.0'] },
};

export default function () {
  check(http.get(`${REDIRECT}/healthz`), { 'redirect live': (r) => r.status === 200 });
  check(http.get(`${API}/healthz`), { 'api live': (r) => r.status === 200 });
}
