import http from 'k6/http';
import { check} from 'k6';

export const options = {
    vus: 300,
    duration: '1m',
};

export default function () {
    const baseUrl = __ENV.SERVER_ADDRESS || 'http://localhost:8080';
    
    const url = `https://example.com/search?q=${__VU}-${__ITER}`;
    const payload = JSON.stringify({
        url: url,
    });

    const params = {
        headers: { 'Content-Type': 'application/json' },
    };

    let shortenRes = http.post(`${baseUrl}/api/shorten`, payload, params);

    if (shortenRes.status !== 201) {
    console.warn(`Error: ${shortenRes.status}, Body: ${shortenRes.body}`);
    }

    check(shortenRes, {
        '201 Created OR 409 Conflict Its OK': (r) => r.status === 201 || r.status === 409,
    });
}