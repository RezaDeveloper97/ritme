// web_home (T-M12-30, P-05): replays the cold open of the WEB home (frontend/src/screens/home/ui/HomePage.tsx) for a
// cycle-mode user — not the Android-only /home that app_open measures. The list and its source lines are in
// docs/quality-2026-11/perf.md §1. The browser fires wave 1 (who is she: life stage + legacy mode) first, then mounts
// CycleHome and fires everything else at once; each wave is an http.batch. RPS is requests/s; one iteration is
// FANOUT.length requests, so iterations/s = RPS / FANOUT.length.
import http from 'k6/http';
import { check } from 'k6';
import { Trend } from 'k6/metrics';
import { DURATION, RPS, headers, monthsBack, pick, scenarios, thresholds, today, API } from './lib.js';

// Wave 1: HomePage → useLifeStage, useUserMode.
export const WAVE1 = ['/profile/life-stage', '/messages/mode'];

// Wave 2: CycleHome (cycle mode). `{m0}` / `{m1}` = this / next Gregorian month, `{today}` = the fixture day.
export const WAVE2 = [
  '/modules',
  '/cycle/today',
  '/messages/feed?slots=urgent_modal,status_banner,smart_tip,daily_advice,daily_challenge,modal',
  '/profile',
  '/fertility/today',
  '/home/cycle-overview',
  '/cycle/month/{m0}?view=calendar',
  '/cycle/month/{m1}?view=calendar',
  '/banners',
  '/banners/ads?mode=cycle',
  '/home/sections/articles',
  '/auth/user',
  '/todo',
  '/care/today',
  '/checkups/home',
  '/learning/courses',
  '/partners/lab-services',
  '/home/sections/challenge',
  '/logs/days/{today}',
  '/logs/taxonomy',
  '/logs/preferences',
  '/home/notifications?per_page=50',
  '/plus/status',
  '/conditions',
  '/messages/nudges',
  '/loss',
  '/companions',
  '/teen/linked',
];

const FANOUT = WAVE1.length + WAVE2.length;
const openMs = new Trend('web_home_open', true);

export const options = {
  scenarios: scenarios('webHome', {
    executor: 'constant-arrival-rate',
    rate: Math.max(1, Math.round(RPS / FANOUT)),
    timeUnit: '1s',
    duration: DURATION,
  }),
  thresholds: thresholds.web_home,
  summaryTrendStats: ['avg', 'min', 'med', 'p(90)', 'p(95)', 'p(99)', 'max'],
};

function wave(paths, u, phase) {
  const reqs = paths.map((p) => ['GET', API + p, null, { headers: headers(u), tags: { endpoint: 'web_home', phase: phase } }]);
  const res = http.batch(reqs);
  res.forEach((r, i) => check(r, { ['web_home ' + paths[i].split('?')[0] + ' 200']: (x) => x.status === 200 }));
}

export function webHome() {
  const u = pick();
  const phase = __ENV.PHASE;
  const m0 = monthsBack(0);
  const m1 = monthsBack(-1);
  const sub = (p) =>
    p
      .replace('{m0}', m0.year + '/' + m0.month)
      .replace('{m1}', m1.year + '/' + m1.month)
      .replace('{today}', today());
  const t0 = Date.now();
  wave(WAVE1, u, phase);
  wave(WAVE2.map(sub), u, phase);
  openMs.add(Date.now() - t0, { phase: phase });
}
