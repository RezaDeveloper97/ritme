// Shared helpers of the k6 scripts (T-M9-14). LOCAL ONLY: the base URL must be loopback / host.docker.internal.
import http from 'k6/http';
import { check } from 'k6';
import { SharedArray } from 'k6/data';

export const BASE = (__ENV.BASE_URL || 'http://127.0.0.1:8020').replace(/\/$/, '');
export const API = BASE + '/api/v1';

const host = BASE.replace(/^https?:\/\//, '').split(/[:/]/)[0];
if (!['127.0.0.1', 'localhost', 'host.docker.internal', '[::1]'].includes(host)) {
  throw new Error('loadtest refuses to run against ' + BASE + ': local APIs only (never stage, beta or production)');
}

// [{user_id, mobile, mode, token}] written by `make loadtest-seed` (loadtest/seed -tokens-out).
export const users = new SharedArray('users', function () {
  return JSON.parse(open(__ENV.TOKENS_FILE || './.tokens.json'));
});

export function usersOf(mode) {
  return users.filter((u) => u.mode === mode);
}

export function headers(u) {
  return { Authorization: 'Bearer ' + u.token, Accept: 'application/json', 'Accept-Language': 'fa' };
}

// ym is the "now" of the seeded data: the seed tool anchors history on its own run day, which is today.
export function today() {
  const d = new Date(Date.now() + 3.5 * 3600 * 1000); // Tehran wall clock, good enough for a date string
  return d.toISOString().slice(0, 10);
}

export function monthsBack(n) {
  const d = new Date(Date.now() + 3.5 * 3600 * 1000);
  d.setUTCDate(1);
  d.setUTCMonth(d.getUTCMonth() - n);
  return { year: d.getUTCFullYear(), month: d.getUTCMonth() + 1 };
}

// get issues one tagged GET. `endpoint` is the label the thresholds key on, `phase` is cold|warm.
export function get(path, u, endpoint, phase) {
  const res = http.get(API + path, { headers: headers(u), tags: { endpoint: endpoint, phase: phase } });
  check(res, { [endpoint + ' 200']: (r) => r.status === 200 }, { endpoint: endpoint });
  return res;
}

// scenarios gives the two-phase plan every read script uses: `cold` touches each user once (nothing cached yet:
// README §8 cold p95), `warm` then runs the real arrival-rate load after COLD_WAIT.
export function scenarios(exec, warm) {
  const nUsers = users.length;
  return {
    cold: {
      executor: 'shared-iterations',
      vus: Math.min(20, nUsers),
      iterations: nUsers,
      maxDuration: __ENV.COLD_WAIT || '45s',
      exec: exec,
      env: { PHASE: 'cold' },
    },
    warm: Object.assign(
      {
        startTime: __ENV.COLD_WAIT || '45s',
        exec: exec,
        env: { PHASE: 'warm' },
        preAllocatedVUs: 50,
        maxVUs: 400,
      },
      warm,
    ),
  };
}

// pick returns the user for this iteration: cold walks the pool in order, warm picks at random.
export function pick() {
  if (__ENV.PHASE === 'cold') return users[__ITER % users.length];
  return users[Math.floor(Math.random() * users.length)];
}

export const RPS = Number(__ENV.RPS || 200);
export const DURATION = __ENV.DURATION || '2m';

// Base thresholds = README §8 (warm p95 / cold p95 per endpoint, mixed app_open p95 < 100 ms, errors < 0.1 %).
export { thresholds } from './thresholds.js';
