// web_home (T-M12-30, P-05): replays the cold open of the WEB home (frontend/src/screens/home/ui/HomePage.tsx) for a
// cycle-mode user — not the Android-only /home that app_open measures. The list and its source lines are in
// docs/quality-2026-11/perf.md §1. The browser fires wave 1 (who is she: life stage + legacy mode) first, then mounts
// CycleHome and fires everything else at once; each wave is an http.batch. RPS is requests/s; one iteration is
// FANOUT.length requests, so iterations/s = RPS / FANOUT.length.
import http from 'k6/http';
import { check } from 'k6';
import { Counter, Rate, Trend } from 'k6/metrics';
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

// T-M12-34 (P-10): the slow-changing GETs answer ETag / 304 (etag.ByBody). From a user's second open on, this VU
// sends the ETag it got back as If-None-Match — what the browser's HTTP cache does. Recorded: the 304 ratio of these
// routes and the body bytes a 304 did not resend (the size of the 200 it revalidated).
export const REVALIDATED = [
  '/modules',
  '/banners',
  '/banners/ads',
  '/logs/taxonomy',
  '/health-logs/enums',
  '/learning/courses',
  '/checkups/home',
];
const notModified = new Rate('web_home_304');
const bytesSaved = new Counter('web_home_304_bytes_saved');
const seen = {}; // per VU: token + path → { tag, bytes } of the last 200
const revalidated = (p) => REVALIDATED.indexOf(p.split('?')[0]) >= 0;

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
  const reqs = paths.map((p) => {
    const h = headers(u);
    const prev = revalidated(p) ? seen[u.token + p] : undefined;
    if (prev) h['If-None-Match'] = prev.tag;
    return ['GET', API + p, null, { headers: h, tags: { endpoint: 'web_home', phase: phase } }];
  });
  const res = http.batch(reqs);
  res.forEach((r, i) => {
    const p = paths[i];
    const key = u.token + p;
    const prev = revalidated(p) ? seen[key] : undefined;
    check(r, {
      ['web_home ' + p.split('?')[0] + ' 200']: (x) => x.status === 200 || (prev !== undefined && x.status === 304),
    });
    if (!revalidated(p)) return;
    if (prev) {
      notModified.add(r.status === 304, { phase: phase });
      if (r.status === 304) bytesSaved.add(prev.bytes, { phase: phase });
    }
    const tag = r.headers['Etag'] || r.headers['ETag'];
    if (r.status === 200 && tag) seen[key] = { tag: tag, bytes: r.body ? r.body.length : 0 };
  });
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
