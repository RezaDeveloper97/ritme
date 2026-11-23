// log_burst: save a day the way the app does (PUT /logs/days/:date, the log sheet's route — T-M12-31, P-06; it
// measured the legacy POST /health-logs before), then read it back at once. Asserts read-your-write (zero stale
// reads): the saved note is returned by GET /logs/days/:date and the engine's /cycle/today still answers.
// A unique marker goes into the note, so a stale (cached) read cannot pass by accident. The body is a 10-param
// change like querybudget.LogSaveBody (no flow: no period bridge).
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
    categories: {
      bleeding: { spotting: false },
      pain: { relief: ['heat', 'rest'] },
      mood: { moods: [['calm'], ['calm', 'energetic'], ['happy']][__ITER % 3] },
      sleep: { duration: '6_9', quality: ['great', 'good', 'fair'][__ITER % 3] },
      appetite_energy: { energy: ['low', 'medium', 'high'][__ITER % 3] },
      measurements: { weight: 55 + (__ITER % 300) / 10 },
      discharge: { amount: 'light' },
      activity: { duration: 30 + (__ITER % 60) },
      note: { text: marker },
    },
  });
  const h = Object.assign({ 'Content-Type': 'application/json' }, headers(u));
  const save = http.put(`${API}/logs/days/${day}`, body, { headers: h, tags: { endpoint: 'log_save', phase: 'warm' } });
  check(save, { 'log_save 200': (r) => r.status === 200 }, { endpoint: 'log_save' });

  const read = http.get(`${API}/logs/days/${day}`, { headers: headers(u), tags: { endpoint: 'log_read', phase: 'warm' } });
  let fresh = false;
  try {
    fresh = read.status === 200 && read.json('data.categories.note.text') === marker;
  } catch (e) {
    fresh = false;
  }
  if (!fresh) staleReads.add(1);
  check(read, { 'log_read sees the write': () => fresh }, { endpoint: 'log_read' });

  const t = http.get(`${API}/cycle/today`, { headers: headers(u), tags: { endpoint: 'cycle_today', phase: 'warm' } });
  check(t, { 'cycle_today 200 after write': (r) => r.status === 200 }, { endpoint: 'cycle_today' });
}
