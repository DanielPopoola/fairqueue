import http from 'k6/http';
import { BASE_URL, DEFAULT_HEADERS } from './scripts/lib/config.js';
import { setupEvent } from './scripts/lib/setup.js';
import { getToken } from './scripts/lib/tokens.js';

export const options = {
  vus: 5000,
  duration: '5m',
};

export function setup() {
  return setupEvent({ name: 'Polling Bottleneck Test' });
}

export default function (data) {
  const token = getToken(__VU);

  http.get(`${BASE_URL}/events/${data.eventID}/queue/position`, {
    headers: DEFAULT_HEADERS(token),
    tags: { name: 'poll_position' },
  });
}