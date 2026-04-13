import http from 'k6/http';
import { sleep } from 'k6';
import { BASE_URL, DEFAULT_HEADERS } from './lib/config.js';
import { setupEvent } from './lib/setup.js';
import { getToken } from './lib/tokens.js';

export const options = {
  stages: [
    { duration: '5m', target: 500 },
    { duration: '30m', target: 500 },
    { duration: '5m', target: 0 },
  ],
  thresholds: {
    http_req_failed: ['rate<0.01'],
    'http_req_duration{name:join_queue}': ['p(99)<500'],
  },
};

export function setup() {
  return setupEvent({ name: 'Sustained Queue Test' });
}

export default function (data) {
  const token = getToken(__VU);

  http.post(`${BASE_URL}/events/${data.eventID}/queue`, null, {
    headers: DEFAULT_HEADERS(token),
    tags: { name: 'join_queue' },
  });

  sleep(Math.random() * 3 + 1);
}