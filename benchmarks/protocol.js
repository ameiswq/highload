
import http from 'k6/http';
import { check } from 'k6';

const BASE_URL =
    __ENV.BASE_URL || 'https://localhost:8443';

const EXPECTED =
    __ENV.EXPECTED_PROTO || 'HTTP/2.0';

export default function () {
    const response = http.get(
        `${BASE_URL}/health/protocol`
    );

    console.log('Protocol:', response.proto);
    console.log('Response:', response.body);

    check(response, {
        'correct HTTP protocol': (r) =>
            r.proto === EXPECTED,
    });
}
