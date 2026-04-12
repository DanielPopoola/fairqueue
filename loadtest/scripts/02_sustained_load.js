import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
    stages: [
        { duration: '2m', target: 1000 },   // ramp up
        { duration: '25m', target: 1000 },  // hold — watching for leaks
        { duration: '3m', target: 0 },      // ramp down
    ],
    thresholds: {
        'http_req_duration{name:join_queue}': ['p(99)<500'],
        'http_req_duration{name:claim_ticket}': ['p(99)<500'],
        'http_req_failed': ['rate<0.01'],
    },
};

const BASE_URL = __ENV.BASE_URL || 'https://loadtest-api.tulcanvcm.com';

export function setup() {
    // Same setup as scenario 1 — create organizer and event
    // Using 2000 tickets so claims don't exhaust inventory
    const orgResp = http.post(`${BASE_URL}/auth/organizer/login`,
        JSON.stringify({
            email: __ENV.ORGANIZER_EMAIL,
            password: __ENV.ORGANIZER_PASSWORD,
        }),
        { headers: { 'Content-Type': 'application/json' } }
    );
    const orgToken = orgResp.json('token');

    const eventResp = http.post(`${BASE_URL}/events`, JSON.stringify({
        name: 'Sustained Load Test Event',
        total_inventory: 999999, // effectively unlimited for stability test
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
    http.put(`${BASE_URL}/events/${eventID}/activate`, null, {
        headers: { 'Authorization': `Bearer ${orgToken}` },
    });

    return { eventID };
}

export default function (data) {
    const { eventID } = data;
    const email = `sustained-${__VU}-${__ITER}@test.com`;

    // Full flow — every VU completes the entire journey
    http.post(`${BASE_URL}/auth/customer/otp/request`,
        JSON.stringify({ email }),
        { headers: { 'Content-Type': 'application/json' } }
    );

    const otpResp = http.get(`${BASE_URL}/loadtest/otp?email=${email}`);
    const otp = otpResp.json('otp');

    const authResp = http.post(`${BASE_URL}/auth/customer/otp/verify`,
        JSON.stringify({ email, otp }),
        { headers: { 'Content-Type': 'application/json' } }
    );
    const token = authResp.json('token');

    const headers = {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`,
    };

    // Join queue
    const joinResp = http.post(
        `${BASE_URL}/events/${eventID}/queue`,
        null,
        { headers, tags: { name: 'join_queue' } }
    );
    check(joinResp, { 'joined queue': r => r.status === 201 });

    // Poll until admitted — max 60 attempts with 5s sleep
    let admitted = false;
    let admissionToken = null;

    for (let i = 0; i < 60; i++) {
        sleep(5);
        const posResp = http.get(
            `${BASE_URL}/events/${eventID}/queue/position`,
            { headers, tags: { name: 'poll_position' } }
        );

        if (posResp.json('status') === 'ADMITTED') {
            admitted = true;
            admissionToken = posResp.json('admission_token');
            break;
        }
    }

    if (!admitted) return; // timed out waiting — skip claim

    // Claim ticket
    const claimResp = http.post(
        `${BASE_URL}/events/${eventID}/claims`,
        JSON.stringify({ admission_token: admissionToken }),
        { headers, tags: { name: 'claim_ticket' } }
    );

    const claimOK = check(claimResp, {
        'claimed ticket': r => r.status === 201,
    });

    if (!claimOK) return;

    const claimID = claimResp.json('claim_id');

    // Initialize payment (hits mock gateway — instant response)
    const payResp = http.post(
        `${BASE_URL}/claims/${claimID}/payments`,
        null,
        { headers, tags: { name: 'init_payment' } }
    );

    check(payResp, { 'payment initialized': r => r.status === 201 });

    // Think time — real users don't hammer continuously
    sleep(Math.random() * 3 + 1);
}