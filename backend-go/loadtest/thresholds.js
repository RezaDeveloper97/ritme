// README §8 (docs/engine-v6/README.md) as k6 thresholds. Keys are `http_req_duration{endpoint:<name>,phase:<p>}`;
// `phase` is cold (first touch of a user) or warm. A scenario script spreads the entries it measures.
// Mixed app_open at 200 RPS: p95 < 100 ms, errors < 0.1 %.
const ms = (warm, cold) => ({ warm: warm, cold: cold });

export const BUDGET = {
  cycle_today: ms(15, 60), // /cycle/today, /cycle/date/:d
  cycle_month: ms(20, 80), // /cycle/month/:y/:m (calendar)
  home: ms(40, 120),
  messages_selected: ms(20, 60),
  messages_daily: ms(20, 60),
  banners: ms(20, 60),
  fertility_insights: ms(25, 80),
  analysis_summary: ms(20, 150),
  // every request of the web-home fan-out (web_home.js); the heaviest single read is /cycle/today + /home/cycle-overview
  web_home: ms(60, 200),
};

export function endpointThresholds(names) {
  const t = {};
  for (const n of names) {
    const b = BUDGET[n];
    if (!b) continue;
    t[`http_req_duration{endpoint:${n},phase:warm}`] = [`p(95)<${b.warm}`];
    t[`http_req_duration{endpoint:${n},phase:cold}`] = [`p(95)<${b.cold}`];
  }
  return t;
}

export const thresholds = {
  // app_open at RPS=200 (the mixed-load SLO)
  app_open: Object.assign(
    {
      'http_req_duration{phase:warm}': ['p(95)<100'],
      http_req_failed: ['rate<0.001'],
    },
    endpointThresholds(['cycle_today', 'cycle_month', 'home', 'messages_selected', 'banners']),
  ),
  calendar_browse: Object.assign({ http_req_failed: ['rate<0.001'] }, endpointThresholds(['cycle_month'])),
  log_burst: {
    http_req_failed: ['rate<0.001'],
    stale_reads: ['count==0'], // zero stale reads after a write
    'http_req_duration{endpoint:log_save}': ['p(95)<100'],
  },
  analysis_tab: Object.assign({ http_req_failed: ['rate<0.001'] }, endpointThresholds(['analysis_summary'])),
  fertility_tab: Object.assign({ http_req_failed: ['rate<0.001'] }, endpointThresholds(['fertility_insights'])),
  // web_home (T-M12-30): per request p95 (warm / cold) and the whole two-wave open as the browser sees it
  web_home: Object.assign(
    {
      http_req_failed: ['rate<0.001'],
      'web_home_open{phase:warm}': ['p(95)<400'],
      'web_home_open{phase:cold}': ['p(95)<1200'],
    },
    endpointThresholds(['web_home']),
  ),
  soak: {
    'http_req_duration{phase:warm}': ['p(95)<100'],
    http_req_failed: ['rate<0.001'],
    stale_reads: ['count==0'],
  },
};
