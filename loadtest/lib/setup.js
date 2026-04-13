import http from 'k6/http';
import { check } from 'k6';
import { BASE_URL, requireEnv } from './config.js';

function isoFromNow(ms) {
  return new Date(Date.now() + ms).toISOString();
}

export function setupEvent({
  name = 'Load Test Event',
  totalInventory = 1000000,
  price = 1000,
} = {}) {
  if (__ENV.EVENT_ID) {
    return { eventID: __ENV.EVENT_ID };
  }

  const organizerEmail = requireEnv('ORGANIZER_EMAIL');
  const organizerPassword = requireEnv('ORGANIZER_PASSWORD');

  const loginRes = http.post(
    `${BASE_URL}/auth/organizer/login`,
    JSON.stringify({ email: organizerEmail, password: organizerPassword }),
    { headers: { 'Content-Type': 'application/json' }, tags: { name: 'organizer_login' } }
  );

  check(loginRes, {
    'organizer login succeeded': (r) => r.status === 200,
  });

  const organizerToken = loginRes.json('token');
  if (!organizerToken) {
    throw new Error('Organizer login did not return a token');
  }

  const eventRes = http.post(
    `${BASE_URL}/events`,
    JSON.stringify({
      name,
      total_inventory: totalInventory,
      price,
      sale_start: isoFromNow(-3600000),
      sale_end: isoFromNow(86400000),
    }),
    {
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${organizerToken}`,
      },
      tags: { name: 'create_event' },
    }
  );

  check(eventRes, {
    'event created': (r) => r.status === 201,
  });

  const eventID = eventRes.json('data.id');
  if (!eventID) {
    throw new Error('Event creation did not return data.id');
  }

  const activateRes = http.put(`${BASE_URL}/events/${eventID}/activate`, null, {
    headers: { Authorization: `Bearer ${organizerToken}` },
    tags: { name: 'activate_event' },
  });

  check(activateRes, {
    'event activated': (r) => r.status === 200,
  });

  return { eventID };
}