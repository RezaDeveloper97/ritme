// calendar_browse: one user paging the calendar 12 months back (and forward to today).
import { API, DURATION, RPS, get, monthsBack, pick, scenarios, thresholds } from './lib.js';

export const options = {
  scenarios: scenarios('browse', {
    executor: 'constant-arrival-rate',
    rate: Math.max(1, Math.round(RPS / 12)),
    timeUnit: '1s',
    duration: DURATION,
  }),
  thresholds: thresholds.calendar_browse,
  summaryTrendStats: ['avg', 'min', 'med', 'p(90)', 'p(95)', 'p(99)', 'max'],
};

export function browse() {
  const u = pick();
  for (let back = 0; back < 12; back++) {
    const m = monthsBack(back);
    get(`/cycle/month/${m.year}/${m.month}?view=calendar`, u, 'cycle_month', __ENV.PHASE);
  }
}
