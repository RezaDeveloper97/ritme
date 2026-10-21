// app_open: what the app fires when opened — today + month + home + messages + banners (README §8 mixed load,
// 200 RPS over 2 000 users). RPS is requests/s; each iteration is 5 requests, so iterations/s = RPS / 5.
import { API, DURATION, RPS, get, monthsBack, pick, scenarios, thresholds } from './lib.js';

const perIter = 5;

export const options = {
  scenarios: scenarios('appOpen', {
    executor: 'constant-arrival-rate',
    rate: Math.max(1, Math.round(RPS / perIter)),
    timeUnit: '1s',
    duration: DURATION,
  }),
  thresholds: thresholds.app_open,
  summaryTrendStats: ['avg', 'min', 'med', 'p(90)', 'p(95)', 'p(99)', 'max'],
};

export function appOpen() {
  const u = pick();
  const phase = __ENV.PHASE;
  const m = monthsBack(0);
  get('/cycle/today', u, 'cycle_today', phase);
  get(`/cycle/month/${m.year}/${m.month}?view=calendar`, u, 'cycle_month', phase);
  get('/home', u, 'home', phase);
  get('/messages/selected?placement=status_banner', u, 'messages_selected', phase);
  get('/banners', u, 'banners', phase);
}
