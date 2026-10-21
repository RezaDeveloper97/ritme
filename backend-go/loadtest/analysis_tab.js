// analysis_tab: the Analysis tab — summary (budgeted) plus cycle, period and symptoms cards.
import { DURATION, RPS, get, pick, scenarios, thresholds } from './lib.js';

export const options = {
  scenarios: scenarios('tab', {
    executor: 'constant-arrival-rate',
    rate: Math.max(1, Math.round(RPS / 4)),
    timeUnit: '1s',
    duration: DURATION,
  }),
  thresholds: thresholds.analysis_tab,
  summaryTrendStats: ['avg', 'min', 'med', 'p(90)', 'p(95)', 'p(99)', 'max'],
};

export function tab() {
  const u = pick();
  const p = __ENV.PHASE;
  get('/analysis/summary', u, 'analysis_summary', p);
  get('/analysis/cycle', u, 'analysis_cycle', p);
  get('/analysis/period', u, 'analysis_period', p);
  get('/analysis/symptoms', u, 'analysis_symptoms', p);
}
