
import http from 'k6/http';
import { check } from 'k6';
import { Trend } from 'k6/metrics';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const POLL_ID = Number(__ENV.POLL_ID);
const RPS = Number(__ENV.RPS || 25);

const responseBytes = new Trend('json_response_bytes');

export const options = {
    scenarios: {
        results: {
            executor: 'constant-arrival-rate',
            rate: RPS,
            timeUnit: '1s',
            duration: '60s',
            preAllocatedVUs: 100,
        },
    },

    thresholds: {
        http_req_duration: ['p(99)<200'],
        http_req_failed: ['rate<0.01'],
        checks: ['rate>0.99'],
        dropped_iterations: ['count==0'],
    },

    summaryTrendStats: [
        'avg', 'med', 'p(95)', 'p(99)', 'min', 'max'
    ],
};

export default function () {
    const response = http.get(
        `${BASE_URL}/v1/polls/${POLL_ID}/results`,
        {
            timeout: '4s',
        }
    );

    const success = check(response, {
        'results received': r =>
            r.status === 200 &&
            r.json('poll_id') === POLL_ID,
    });

    if (success) {
        const encoder = new TextEncoder();

        responseBytes.add(
            encoder.encode(response.body || '').length
        );
    }
}
