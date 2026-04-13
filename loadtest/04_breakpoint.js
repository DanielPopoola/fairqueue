import http from 'k6/http';
import { check } from 'k6';
import { BASE_URL, DEFAULT_HEADERS } from './scripts/lib/config.js';
import { setupEvent } from './scripts/lib/setup.js';
import { getToken } from './scripts/lib/tokens.js';

export const options = {
  stages: [
    { duration: '2m', target: 1000 },
    { duration: '2m', target: 5000 },
    { duration: '2m', target: 10000 },
    { duration: '2m', target: 20000 },
    { duration: '2m', target: 40000 },
  ],
  thresholds: {
    http_req_failed: ['rate<0.05'],
  },
};

export function setup() {
  return setupEvent({ name: 'Breakpoint Queue Test' });
}

export default function (data) {
  const token = getToken(__VU);

  const res = http.post(`${BASE_URL}/events/${data.eventID}/queue`, null, {
    headers: DEFAULT_HEADERS(token),
    tags: { name: 'join_queue' },
  });

  check(res, {
    'joined queue': (r) => r.status === 201,
  });
}