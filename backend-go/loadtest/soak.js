// soak: 30 minutes of mixed traffic at a modest rate — app_open reads, calendar paging and ~5 % log writes with
// the read-your-write check — to surface leaks (memory, goroutines, pool exhaustion) and slow drifts.
// Compare /metrics before/after (make loadtest wraps it) and watch process_resident_memory_bytes.
import http from 'k6/http';
import { check } from 'k6';
import { Counter } from 'k6/metrics';
import { API, headers, get, monthsBack, pick, today, thresholds } from './lib.js';

const staleReads = new Counter('stale_reads');

export const options = {
  scenarios: {
    soak: {
      executor: 'constant-arrival-rate',
      rate: Number(__ENV.SOAK_RATE || 20), // iterations/s, ~100 req/s
      timeUnit: '1s',
      duration: __ENV.SOAK_DURATION || '30m',
      preAllocatedVUs: 50,
      maxVUs: 300,
      env: { PHASE: 'warm' },
    },
  },
  thresholds: thresholds.soak,
  summaryTrendStats: ['avg', 'min', 'med', 'p(90)', 'p(95)', 'p(99)', 'max'],
};

export default function () {
  const u = pick();
  const m = monthsBack(__ITER % 12);
  get('/cycle/today', u, 'cycle_today', 'warm');
  get(`/cycle/month/${m.year}/${m.month}?view=calendar`, u, 'cycle_month', 'warm');
  get('/home', u, 'home', 'warm');
  get('/messages/selected?placement=status_banner', u, 'messages_selected', 'warm');
  get('/banners', u, 'banners', 'warm');

  if (__ITER % 20 === 0) {
    const day = today();
    const marker = `soak-${__VU}-${__ITER}-${Date.now()}`;
    const h = Object.assign({ 'Content-Type': 'application/json', 'Idempotency-Key': marker }, headers(u));
    const save = http.post(`${API}/health-logs`, JSON.stringify({ log_date: day, notes: marker }), { headers: h, tags: { endpoint: 'log_save', phase: 'warm' } });
    check(save, { 'log_save ok': (r) => r.status === 200 || r.status === 201 });
    const read = http.get(`${API}/health-logs/${day}`, { headers: headers(u), tags: { endpoint: 'log_read', phase: 'warm' } });
    let fresh = false;
    try {
      fresh = read.json('data.notes') === marker;
    } catch (e) {}
    if (!fresh) staleReads.add(1);
  }
}
