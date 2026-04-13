import http from 'k6/http';
import { check, sleep } from 'k6';
import { BASE_URL, DEFAULT_HEADERS } from './scripts/lib/config.js';
import { setupEvent } from './scripts/lib/setup.js';
import { getToken } from './scripts/lib/tokens.js';

export const options = {
  vus: 20000,
  duration: '5m',
};

export function setup() {
  return setupEvent({ name: 'Contention Queue Test', totalInventory: 20000 });
}

export default function (data) {
  const token = getToken(__VU);
  const headers = DEFAULT_HEADERS(token);

  http.post(`${BASE_URL}/events/${data.eventID}/queue`, null, { headers });

  let admitted = false;
  let admissionToken;

  for (let i = 0; i < 30; i++) {
    const res = http.get(`${BASE_URL}/events/${data.eventID}/queue/position`, { headers });

    if (res.json('status') === 'ADMITTED') {
      admitted = true;
      admissionToken = res.json('admission_token');
      break;
    }

    sleep(1);
  }

  if (!admitted) return;

  const claimRes = http.post(
    `${BASE_URL}/events/${data.eventID}/claims`,
    JSON.stringify({ admission_token: admissionToken }),
    { headers }
  );

  check(claimRes, {
    'claimed ticket': (r) => r.status === 201,
  });
}