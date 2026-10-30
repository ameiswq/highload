
import http from 'k6/http';
import { check } from 'k6';
import { SharedArray } from 'k6/data';
import exec from 'k6/execution';
import { Trend } from 'k6/metrics';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const POLL_ID = Number(__ENV.POLL_ID);
const OPTION_ID = Number(__ENV.OPTION_ID);
const RPS = Number(__ENV.RPS || 12);
const USER_OFFSET = Number(__ENV.USER_OFFSET || 0);

// Load real user IDs from CSV
const voters = new SharedArray('voters', function () {
    const csv = open('./voters.csv')
        .replace(/^\uFEFF/, '')
        .trim();

    return csv
        .split(/\r?\n/)
        .slice(1)
        .map(line => Number(
            line.trim().replace(/^"|"$/g, '')
        ));
});

// Custom metrics for JSON body sizes
const requestBytes = new Trend('json_request_bytes');
const responseBytes = new Trend('json_response_bytes');

export const options = {
    scenarios: {
        voting: {
            executor: 'constant-arrival-rate',
            rate: RPS,
            timeUnit: '1s',
            duration: '60s',
            preAllocatedVUs: 100,
        },
    },

    thresholds: {
        http_req_duration: ['p(99)<300'],
        http_req_failed: ['rate<0.01'],
        checks: ['rate>0.99'],
        dropped_iterations: ['count==0'],
    },

    summaryTrendStats: [
        'avg', 'med', 'p(95)', 'p(99)', 'min', 'max'
    ],
};

export default function () {
    // Unique user for every test iteration
    const index = USER_OFFSET +
        exec.scenario.iterationInTest;

    if (index >= voters.length) {
        throw new Error(
            'Not enough test users in voters.csv'
        );
    }

    const userID = voters[index];

    const payload = JSON.stringify({
        user_id: userID,
        option_id: OPTION_ID,
    });

    // New UUID for each logical vote
    const key = crypto.randomUUID();

    const response = http.post(
        `${BASE_URL}/v1/polls/${POLL_ID}/votes`,
        payload,
        {
            headers: {
                'Content-Type': 'application/json',
                'Idempotency-Key': key,
            },
            timeout: '6s',
        }
    );

    const success = check(response, {
        'vote accepted': r =>
            r.status === 201 &&
            r.json('status') === 'accepted',
    });

    if (success) {
        const encoder = new TextEncoder();

        requestBytes.add(
            encoder.encode(payload).length
        );

        responseBytes.add(
            encoder.encode(response.body || '').length
        );
    } else {
        if (exec.scenario.iterationInTest < 10) {
            console.error(
                `Vote failed: ${response.status} ` +
                response.body
            );
        }
    }
}
