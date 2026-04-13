import http from 'k6/http';
import { check } from 'k6';
import { BASE_URL, DEFAULT_HEADERS } from './lib/config.js';
import { setupEvent } from './lib/setup.js';
import { getToken } from './lib/tokens.js';

export const options = {
  vus: 50,
  duration: '2m',
  thresholds: {
    http_req_failed: ['rate<0.01'],
    'http_req_duration{name:join_queue}': ['p(99)<500'],
  },
};

export function setup() {
  return setupEvent({ name: 'Baseline Queue Test' });
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