import http from 'k6/http';
import { check, sleep } from 'k6';

export let options = {
  stages: [
    { duration: '30s', target: 10 },   // Ramp up to 10 virtual users over 30s
    { duration: '1m', target: 50 },    // Then go up to 50 users over 1m
    { duration: '30s', target: 0 },    // Ramp down back to 0 users over 30s
  ],
  thresholds: {
    http_req_duration: ['p(95)<500'], // 95% of requests should be faster than 500ms
  },
};

export default function () {
  let res = http.get('http://a26ffdd05c35d483cb5b2be479257d99-77102527.us-east-1.elb.amazonaws.com');
  check(res, {
    'status is 200': (r) => r.status === 200,
  });
  sleep(1);
}
