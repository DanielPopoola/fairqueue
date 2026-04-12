import http from 'k6/http';
import { check } from 'k6';

// Single VU, single iteration — just check correctness
export const options = {
    vus: 1,
    iterations: 1,
};

const BASE_URL = __ENV.BASE_URL || 'https://loadtest-api.tulcanvcm.com';

export default function () {
    // This hits a stats endpoint you expose on the load test server
    // showing claimed count vs total inventory
    const resp = http.get(`${BASE_URL}/loadtest/stats?event_id=${__ENV.EVENT_ID}`);

    const stats = resp.json();

    check(stats, {
        'no overselling': s => s.claimed_count <= s.total_inventory,
        'has claims': s => s.claimed_count > 0,
        'confirmed plus claimed equals total activity':
            s => s.confirmed_count + s.claimed_count + s.released_count <= s.total_inventory,
    });

    console.log(`Total inventory: ${stats.total_inventory}`);
    console.log(`Claimed: ${stats.claimed_count}`);
    console.log(`Confirmed: ${stats.confirmed_count}`);
    console.log(`Released: ${stats.released_count}`);
}