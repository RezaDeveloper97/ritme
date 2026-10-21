// log_burst: POST a health log, then read it back at once. Asserts read-your-write (zero stale reads): the
// saved log is returned by GET /health-logs/:date and the engine's /cycle/today still answers.
// A unique marker goes into `notes`, so a stale (cached) read cannot pass by accident.
import http from 'k6/http';
import { check } from 'k6';
import { Counter } from 'k6/metrics';
import { API, DURATION, headers, pick, today, scenarios, thresholds } from './lib.js';

const staleReads = new Counter('stale_reads');

export const options = {
  scenarios: {
    burst: {
      executor: 'constant-arrival-rate',
      rate: Math.max(1, Math.round(Number(__ENV.RPS || 100) / 3)), // 3 requests per iteration
      timeUnit: '1s',
      duration: DURATION,
      preAllocatedVUs: 50,
      maxVUs: 400,
      exec: 'burst',
      env: { PHASE: 'warm' },
    },
  },
  thresholds: thresholds.log_burst,
  summaryTrendStats: ['avg', 'min', 'med', 'p(90)', 'p(95)', 'p(99)', 'max'],
};

export function burst() {
  const u = pick();
  const day = today();
  const marker = `lt-${__VU}-${__ITER}-${Date.now()}`;
  const body = JSON.stringify({
    log_date: day,
    notes: marker,
    energy_level: ['low', 'medium', 'high'][__ITER % 3],
    weight: 55 + (__ITER % 300) / 10,
  });
  const h = Object.assign({ 'Content-Type': 'application/json', 'Idempotency-Key': marker }, headers(u));
  const save = http.post(`${API}/health-logs`, body, { headers: h, tags: { endpoint: 'log_save', phase: 'warm' } });
  check(save, { 'log_save 200/201': (r) => r.status === 200 || r.status === 201 }, { endpoint: 'log_save' });

  const read = http.get(`${API}/health-logs/${day}`, { headers: headers(u), tags: { endpoint: 'log_read', phase: 'warm' } });
  let fresh = false;
  try {
    fresh = read.status === 200 && read.json('data.notes') === marker;
  } catch (e) {
    fresh = false;
  }
  if (!fresh) staleReads.add(1);
  check(read, { 'log_read sees the write': () => fresh }, { endpoint: 'log_read' });

  const t = http.get(`${API}/cycle/today`, { headers: headers(u), tags: { endpoint: 'cycle_today', phase: 'warm' } });
  check(t, { 'cycle_today 200 after write': (r) => r.status === 200 }, { endpoint: 'cycle_today' });
}
