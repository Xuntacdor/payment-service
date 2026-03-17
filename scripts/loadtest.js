import http from 'k6/http';
import { check, sleep } from 'k6';
import { randomString } from 'k6/crypto';

// Command to run:
// k6 run scripts/loadtest.js

export const options = {
    // Ramp up to 1000 requests per second
    stages: [
        { duration: '30s', target: 50 },  // simulate ramp-up of traffic from 1 to 50 users
        { duration: '1m', target: 1000 }, // stay at 1000 users for 1 minute
        { duration: '30s', target: 0 },   // ramp-down to 0 users
    ],
    thresholds: {
        http_req_duration: ['p(95)<500'], // 95% of requests should be below 500ms
        http_req_failed: ['rate<0.01'],   // Error rate should be less than 1%
    },
};

const API_KEY = 'my-secret-api-key';
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080/api/v1';

export default function () {
    const payload = JSON.stringify({
        order_id: `ord_${randomString(10)}`,
        amount: Math.floor(Math.random() * 1000) + 10,
        currency: 'VND',
        payment_method: 'BANK_TRANSFER'
    });

    const params = {
        headers: {
            'Content-Type': 'application/json',
            'X-API-Key': API_KEY, // Inject API key for auth middleware
        },
    };

    const res = http.post(`${BASE_URL}/payments`, payload, params);

    check(res, {
        'status is 201 or 429': (r) => r.status === 201 || r.status === 429,
        // Since rate limiting is active locally with a tiny 10req/s bucket, most of 
        // these 1000 user requests will hit 429, which is the expected loadtest behavior.
    });

    sleep(1);
}
