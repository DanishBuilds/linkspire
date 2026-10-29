// test/load/redirect_hotpath.js
// Load test for the redirect hot path (Phase 3). Encodes the origin SLO:
// p99 < 30 ms and < 0.1% errors. Run:
//   k6 run -e BASE_URL=http://localhost:8080 -e CODES=NoRuDiB,abc123 test/load/redirect_hotpath.js
import http from 'k6/http';
import { check } from 'k6';

const BASE = __ENV.BASE_URL || 'http://localhost:8080';
const CODES = (__ENV.CODES || 'NoRuDiB').split(',').map((c) => c.trim()).filter(Boolean);

export const options = {
  discardResponseBodies: true,
  scenarios: {
    hotpath: {
      executor: 'ramping-arrival-rate',
      startRate: 200,
      timeUnit: '1s',
      preAllocatedVUs: 300,
      maxVUs: 3000,
      stages: [
        { target: 1000, duration: '1m' },
        { target: 5000, duration: '2m' },
        { target: 5000, duration: '3m' },
        { target: 0, duration: '1m' },
      ],
    },
  },
  thresholds: {
    // Origin redirect SLO from docs/architecture/01 and 02.
    http_req_duration: ['p(99)<30', 'p(95)<15'],
    http_req_failed: ['rate<0.001'],
    checks: ['rate>0.999'],
  },
};

export default function () {
  const code = CODES[Math.floor(Math.random() * CODES.length)];
  const res = http.get(`${BASE}/${code}`, { redirects: 0 });
  check(res, {
    'status is 3xx redirect': (r) => r.status >= 300 && r.status < 400,
    'has location header': (r) => !!r.headers['Location'],
  });
}
