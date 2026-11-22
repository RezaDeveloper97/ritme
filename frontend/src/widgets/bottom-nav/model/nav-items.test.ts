import { describe, expect, it } from 'vitest';

import { isNavRootPath } from '@/shared/config';

import { activeTabKey, isRootRetap, navConfig, resolveNavMode, type NavMode } from './nav-items';

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
  it('orders امروز · mode tab · FAB · تحلیل · من', () => {
    expect(keys('cycle')).toEqual(['today:/home', 'calendar:/calendar', 'FAB', 'analysis', 'me']);
    expect(keys('teen')).toEqual(['today:/home', 'calendar:/calendar', 'FAB', 'analysis', 'me']);
    expect(keys('ttc')).toEqual(['today:/home', 'fertility:/calendar', 'FAB', 'analysis', 'me']);
    expect(keys('pregnancy')).toEqual(['today:/pregnancy', 'pregnancy:/pregnancy/weeks', 'FAB', 'analysis', 'me']);
    expect(keys('menopause')).toEqual(['today:/home', 'symptoms:/menopause/score', 'FAB', 'analysis', 'me']);
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
      'analysis',
      'me',
    ]);
    // B-N5-05: /children is live — «کودک» opens the only child, else the list.
    expect(keys('postpartum', ['7'], {})).toEqual(['today:/postpartum', 'child:/children/7', 'FAB', 'analysis', 'me']);
    // CB-MENO-08: «علائم» always opens the monthly score; the symptom report lives under «تحلیل».
    expect(keys('menopause', undefined, { analysis: false })[1]).toBe('symptoms:/menopause/score');
    expect(activeTabKey(navConfig('menopause'), '/menopause/score')).toBe('symptoms');
    expect(activeTabKey(navConfig('menopause'), '/analysis/symptoms')).toBe('analysis');
  });

  it('IVF sub-mode (CB-IVF-02): ttc + ivf → امروز /ivf and «درمان»; off → plain ttc', () => {
    const ivf = navConfig('ttc', { ivf: true });
    expect([...ivf.before, ...ivf.after].map((t) => `${t.key}:${t.href}`)).toEqual([
      'today:/ivf',
      'treatment:/ivf/meds', // CB-IVF-03 flipped NAV_READY.ivfMeds
      'analysis:/analysis',
      'me:/profile',
    ]);
    expect(ivf.before[1]?.icon).toBe('treatment');
    expect(navConfig('ttc', { ivf: true, ready: { ivfMeds: false } }).before[1]?.href).toBe('/ivf#ivf-doses');
    expect(keys('ttc')).toEqual(['today:/home', 'fertility:/calendar', 'FAB', 'analysis', 'me']);
    // The flag only means something for ttc.
    expect(navConfig('cycle', { ivf: true }).before.map((t) => t.key)).toEqual(['today', 'calendar']);
    expect(activeTabKey(ivf, '/ivf')).toBe('today');
    expect(activeTabKey(navConfig('ttc', { ivf: true, ready: { ivfMeds: true } }), '/ivf/meds')).toBe('treatment');
    expect(activeTabKey(ivf, '/ivf/scan')).toBe('treatment');
  });

  it('companion v2: امروز · تقویم · یادگیری · اقدام‌ها · پروفایل, no FAB', () => {
    const c = navConfig('companion');
    expect([...c.before, ...c.after].map((tab) => `${tab.key}:${tab.href}`)).toEqual([
      'today:/companion',
      'calendar:/companion/calendar',
      'learn:/companion/learn',
      'actions:/companion/actions',
      'profile:/profile',
    ]);
    expect(c.fab).toBe(false);
    expect(activeTabKey(c, '/companion/calendar')).toBe('calendar');
    expect(activeTabKey(c, '/companion/learn')).toBe('learn');
    expect(activeTabKey(c, '/companion/actions')).toBe('actions');
    expect(activeTabKey(c, '/profile')).toBe('profile');
  });
});

describe('activeTabKey', () => {
  it('lights the exact tab, «من» for services, «تحلیل» for analysis and history', () => {
    expect(activeTabKey(navConfig('cycle'), '/home')).toBe('today');
    expect(activeTabKey(navConfig('cycle'), '/services')).toBe('me');
    expect(activeTabKey(navConfig('cycle'), '/services/doctors')).toBe('me');
    expect(activeTabKey(navConfig('cycle'), '/vitals')).toBe('me');
    expect(activeTabKey(navConfig('cycle'), '/profile')).toBe('me');
    expect(activeTabKey(navConfig('cycle'), '/analysis')).toBe('analysis');
    expect(activeTabKey(navConfig('ttc'), '/analysis/fertility')).toBe('analysis');
    expect(activeTabKey(navConfig('pregnancy'), '/analysis')).toBe('analysis');
    expect(activeTabKey(navConfig('cycle'), '/cycle')).toBe('analysis');
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

describe('CB-NAV-03 IA_Nav rules', () => {
  it('hides the nav on the log screen and the pregnancy calendar (back headers)', () => {
    expect(isNavRootPath('/log')).toBe(false);
    expect(isNavRootPath('/pregnancy/calendar')).toBe(false);
    for (const p of ['/loss', '/loss/care', '/plus/checkout', '/reminders/appointment/new', '/teen/onboarding']) {
      expect(isNavRootPath(p)).toBe(false);
    }
  });

  it('re-tap on the active tab root scrolls; from a sub-screen the link goes to the root', () => {
    const cal = navConfig('cycle').before[1]!;
    expect(isRootRetap(cal, '/calendar')).toBe(true);
    expect(isRootRetap(cal, '/calendar/')).toBe(true);
    expect(isRootRetap(cal, '/analysis/symptoms')).toBe(false);
    expect(activeTabKey(navConfig('cycle'), '/analysis/symptoms')).toBe('analysis');
    expect(cal.href).toBe('/calendar');
  });

  it('keeps the stage tab in the second slot with the mode label/icon (IVF «درمان», menopause «علائم»)', () => {
    const second = (mode: NavMode, ivf = false) => {
      const t = navConfig(mode, { ivf }).before[1]!;
      return `${t.key}/${t.icon}`;
    };
    expect(second('cycle')).toBe('calendar/calendar');
    expect(second('teen')).toBe('calendar/calendar');
    expect(second('ttc')).toBe('fertility/fertility');
    expect(second('ttc', true)).toBe('treatment/treatment');
    expect(second('pregnancy')).toBe('pregnancy/pregnancy');
    expect(second('postpartum')).toBe('child/child');
    expect(second('menopause')).toBe('symptoms/symptoms');
  });

  it('the FAB is present in every owner mode (it always opens the log sheet)', () => {
    for (const m of ['cycle', 'ttc', 'pregnancy', 'postpartum', 'menopause', 'teen'] as const) {
      expect(navConfig(m).fab).toBe(true);
    }
  });
});
