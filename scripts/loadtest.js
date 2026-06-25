import http from 'k6/http';
import { check, sleep } from 'k6';

const BASE_URL = __ENV.BASE_URL || 'http://127.0.0.1:8080';

export const options = {
  scenarios: {
    health: {
      executor: 'constant-vus',
      vus: 20,
      duration: '30s',
      exec: 'healthCheck',
    },
    chat: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '10s', target: 20 },
        { duration: '20s', target: 20 },
        { duration: '10s', target: 0 },
      ],
      exec: 'chatCompletion',
      startTime: '5s',
    },
  },
  thresholds: {
    http_req_failed: ['rate<0.01'],
    http_req_duration: ['p(99)<500'],
  },
};

export function healthCheck() {
  const res = http.get(`${BASE_URL}/health`);
  check(res, {
    'health status 200': (r) => r.status === 200,
  });
}

export function chatCompletion() {
  const payload = JSON.stringify({
    model: 'mock-model',
    messages: [{ role: 'user', content: 'load test ping' }],
  });

  const res = http.post(`${BASE_URL}/v1/chat/completions`, payload, {
    headers: { 'Content-Type': 'application/json' },
  });

  check(res, {
    'chat status 200': (r) => r.status === 200,
  });
  sleep(0.1);
}
