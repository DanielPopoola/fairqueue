import http from 'k6/http';
import { check, sleep } from 'k6';
import { SharedArray } from 'k6/data';

// Test configuration
export const options = {
    // Ramp up to 50,000 VUs over 60 seconds, hold briefly, ramp down
    stages: [
        { duration: '60s', target: 50000 },
        { duration: '30s', target: 50000 },
        { duration: '30s', target: 0 },
    ],
    thresholds: {
        // 99% of queue join requests must complete under 500ms
       'http_req_duration{name:join_queue}': ['p(99)<500'],
        // Less than 1% of requests should fail
        'http_req_failed': ['rate<0.01'],
    },
};

const BASE_URL = __ENV.BASE_URL || 'https://loadtest-api.tulcanvcm.com';
const EVENT_ID = __ENV.EVENT_ID;

export function setup() {
    // Create organizer and event once before all VUs start
    // Returns data that every VU receives
    const orgResp = http.post(`${BASE_URL}/auth/organizer/login`, JSON.stringify({
        email: __ENV.ORGANIZER_EMAIL,
        password: __ENV.ORGANIZER_PASSWORD,
    }), { headers: { 'Content-Type': 'application/json' } });

    const orgToken = orgResp.json('token');

    const eventResp = http.post(`${BASE_URL}/events`, JSON.stringify({
        name: 'Load Test Event',
        total_inventory: 5000,
        price: 25000,
        sale_start: new Date(Date.now() - 3600000).toISOString(),
        sale_end: new Date(Date.now() + 86400000).toISOString(),
    }), {
        headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${orgToken}`,
        },
    });

    const eventID = eventResp.json('data.id');

    // Activate the event
    http.put(`${BASE_URL}/events/${eventID}/activate`, null, {
        headers: { 'Authorization': `Bearer ${orgToken}` },
    });

    return { eventID };
}

export default function (data) {
    const { eventID } = data;

    // Each VU is a unique customer — use VU ID to generate unique email
    const email = `loadtest-user-${__VU}-${__ITER}@test.com`;

    // Step 1: Request OTP
    http.post(`${BASE_URL}/auth/customer/otp/request`,
        JSON.stringify({ email }),
        { headers: { 'Content-Type': 'application/json' } }
    );

    // Step 2: Get OTP from load test helper endpoint
    // We add a special endpoint to the load test server for this
    const otpResp = http.get(`${BASE_URL}/loadtest/otp?email=${email}`);
    const otp = otpResp.json('otp');

    // Step 3: Verify OTP
    const authResp = http.post(`${BASE_URL}/auth/customer/otp/verify`,
        JSON.stringify({ email, otp }),
        { headers: { 'Content-Type': 'application/json' } }
    );
    const token = authResp.json('token');

    // Step 4: Join queue — this is what we're stress testing
    const joinResp = http.post(
        `${BASE_URL}/events/${eventID}/queue`,
        null,
        {
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${token}`,
            },
            tags: { name: 'join_queue' }, // tag for threshold targeting
        }
    );

    check(joinResp, {
        'joined queue successfully': r => r.status === 201,
        'has position': r => r.json('position') > 0,
    });

    // No sleep — we want maximum pressure on the join endpoint
}