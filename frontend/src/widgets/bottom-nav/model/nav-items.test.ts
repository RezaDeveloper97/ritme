import { describe, expect, it } from 'vitest';

import { isNavRootPath } from '@/shared/config';

import { activeTabKey, navConfig, resolveNavMode, type NavMode } from './nav-items';

const READY = { postpartum: true, children: true, analysis: true, ivfMeds: false };
const keys = (mode: NavMode, childIds?: string[], ready: Partial<typeof READY> = READY) => {
  const c = navConfig(mode, { childIds, ready });
  return [...c.before.map((t) => `${t.key}:${t.href}`), c.fab ? 'FAB' : null, ...c.after.map((t) => t.key)].filter(
    Boolean,
  );
};

describe('resolveNavMode', () => {
  it('maps the API mode and the TTC flag', () => {
    expect(resolveNavMode({ mode: 'cycle' })).toBe('cycle');
    expect(resolveNavMode({ mode: 'cycle', isTtc: true })).toBe('ttc');
    expect(resolveNavMode({ mode: 'pregnancy', isTtc: true })).toBe('pregnancy');
    expect(resolveNavMode({ mode: 'postpartum' })).toBe('postpartum');
    expect(resolveNavMode({ mode: 'menopause' })).toBe('menopause');
    expect(resolveNavMode({ mode: 'weird' })).toBe('cycle');
    expect(resolveNavMode({})).toBe('cycle');
  });

  it('prefers the life-stage mode (B-N2-03)', () => {
    expect(resolveNavMode({ lifeMode: 'menopause', mode: 'cycle', isTtc: true })).toBe('menopause');
    expect(resolveNavMode({ lifeMode: 'teen', mode: 'cycle' })).toBe('teen');
    expect(resolveNavMode({ lifeMode: 'ttc' })).toBe('ttc');
    expect(resolveNavMode({ lifeMode: 'nope', mode: 'pregnancy' })).toBe('pregnancy');
    expect(resolveNavMode({ lifeMode: null, mode: 'cycle', isTtc: true })).toBe('ttc');
  });
});

describe('navConfig (nav.md per-mode table)', () => {
  it('orders امروز · mode tab · FAB · خدمات · من', () => {
    expect(keys('cycle')).toEqual(['today:/home', 'calendar:/calendar', 'FAB', 'services', 'me']);
    expect(keys('teen')).toEqual(['today:/home', 'calendar:/calendar', 'FAB', 'services', 'me']);
    expect(keys('ttc')).toEqual(['today:/home', 'fertility:/calendar', 'FAB', 'services', 'me']);
    expect(keys('pregnancy')).toEqual(['today:/pregnancy', 'pregnancy:/pregnancy/weeks', 'FAB', 'services', 'me']);
    expect(keys('menopause')).toEqual(['today:/home', 'symptoms:/menopause/score', 'FAB', 'services', 'me']);
  });

  it('postpartum «کودک» targets the only child, else the list (gaps.md #15)', () => {
    expect(keys('postpartum', ['7'])[1]).toBe('child:/children/7');
    expect(keys('postpartum', [])[1]).toBe('child:/children');
    expect(keys('postpartum', ['1', '2'])[1]).toBe('child:/children');
  });

  it('uses interim targets until the postpartum and analysis screens exist (QUESTIONS #60)', () => {
    expect(keys('postpartum', ['7'], { postpartum: false, children: false })).toEqual([
      'today:/home',
      'calendar:/calendar',
      'FAB',
      'services',
      'me',
    ]);
    // B-N5-05: /children is live — «کودک» opens the only child, else the list.
    expect(keys('postpartum', ['7'], {})).toEqual(['today:/postpartum', 'child:/children/7', 'FAB', 'services', 'me']);
    // CB-MENO-08: «علائم» always opens the monthly score; the symptom report keeps the tab lit.
    expect(keys('menopause', undefined, { analysis: false })[1]).toBe('symptoms:/menopause/score');
    expect(activeTabKey(navConfig('menopause'), '/menopause/score')).toBe('symptoms');
    expect(activeTabKey(navConfig('menopause'), '/analysis/symptoms')).toBe('symptoms');
  });

  it('IVF sub-mode (CB-IVF-02): ttc + ivf → امروز /ivf and «درمان»; off → plain ttc', () => {
    const ivf = navConfig('ttc', { ivf: true });
    expect([...ivf.before, ...ivf.after].map((t) => `${t.key}:${t.href}`)).toEqual([
      'today:/ivf',
      'treatment:/ivf/meds', // CB-IVF-03 flipped NAV_READY.ivfMeds
      'services:/services',
      'me:/profile',
    ]);
    expect(ivf.before[1]?.icon).toBe('treatment');
    expect(navConfig('ttc', { ivf: true, ready: { ivfMeds: false } }).before[1]?.href).toBe('/ivf#ivf-doses');
    expect(keys('ttc')).toEqual(['today:/home', 'fertility:/calendar', 'FAB', 'services', 'me']);
    // The flag only means something for ttc.
    expect(navConfig('cycle', { ivf: true }).before.map((t) => t.key)).toEqual(['today', 'calendar']);
    expect(activeTabKey(ivf, '/ivf')).toBe('today');
    expect(activeTabKey(navConfig('ttc', { ivf: true, ready: { ivfMeds: true } }), '/ivf/meds')).toBe('treatment');
    expect(activeTabKey(ivf, '/ivf/scan')).toBe('treatment');
  });

  it('companion has no mode tab and no FAB', () => {
    expect(keys('companion')).toEqual(['today:/companion', 'services', 'me']);
  });
});

describe('activeTabKey', () => {
  it('lights the exact tab, services sub-hubs and the mode tab for history/analysis', () => {
    expect(activeTabKey(navConfig('cycle'), '/home')).toBe('today');
    expect(activeTabKey(navConfig('cycle'), '/services')).toBe('services');
    expect(activeTabKey(navConfig('cycle'), '/services/doctors')).toBe('services');
    expect(activeTabKey(navConfig('cycle'), '/profile')).toBe('me');
    expect(activeTabKey(navConfig('cycle'), '/cycle')).toBe('calendar');
    expect(activeTabKey(navConfig('ttc'), '/calendar')).toBe('fertility');
    expect(activeTabKey(navConfig('pregnancy'), '/pregnancy/weeks/12')).toBe('pregnancy');
    expect(activeTabKey(navConfig('pregnancy'), '/pregnancy')).toBe('today');
    expect(activeTabKey(navConfig('cycle'), '/log')).toBeNull();
  });
});

describe('isNavRootPath', () => {
  it('shows the nav on tab roots and hubs only', () => {
    for (const p of ['/home', '/calendar', '/services', '/profile', '/pregnancy', '/pregnancy/weeks/3', '/home/']) {
      expect(isNavRootPath(p)).toBe(true);
    }
    for (const p of ['/pregnancy/alerts', '/pregnancy/log', '/reminders', '/checkups', '/fertility/log', '/onboarding/name']) {
      expect(isNavRootPath(p)).toBe(false);
    }
  });
});
