// fertility_tab: the TTC tiles — today, BBT chart, insights. TTC users only (mode ttc in the tokens file).
import { DURATION, RPS, get, scenarios, thresholds, usersOf } from './lib.js';

const ttc = usersOf('ttc');

export const options = {
  scenarios: scenarios('tab', {
    executor: 'constant-arrival-rate',
    rate: Math.max(1, Math.round(RPS / 3)),
    timeUnit: '1s',
    duration: DURATION,
  }),
  thresholds: thresholds.fertility_tab,
  summaryTrendStats: ['avg', 'min', 'med', 'p(90)', 'p(95)', 'p(99)', 'max'],
};

// scenarios() sizes the cold phase by the whole pool; the TTC subset is walked modulo its own size.
export function tab() {
  const u = __ENV.PHASE === 'cold' ? ttc[__ITER % ttc.length] : ttc[Math.floor(Math.random() * ttc.length)];
  const p = __ENV.PHASE;
  get('/fertility/today', u, 'fertility_today', p);
  get('/fertility/bbt', u, 'fertility_bbt', p);
  get('/fertility/insights', u, 'fertility_insights', p);
}
