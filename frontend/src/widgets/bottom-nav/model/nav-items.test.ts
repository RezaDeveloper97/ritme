import { describe, expect, it } from 'vitest';

import { isNavRootPath } from '@/shared/config';

import { activeTabKey, navConfig, resolveNavMode, type NavMode } from './nav-items';

const keys = (mode: NavMode, childIds?: string[]) => {
  const c = navConfig(mode, { childIds });
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
});

describe('navConfig (nav.md per-mode table)', () => {
  it('orders امروز · mode tab · FAB · خدمات · من', () => {
    expect(keys('cycle')).toEqual(['today:/home', 'calendar:/calendar', 'FAB', 'services', 'me']);
    expect(keys('teen')).toEqual(['today:/home', 'calendar:/calendar', 'FAB', 'services', 'me']);
    expect(keys('ttc')).toEqual(['today:/home', 'fertility:/calendar', 'FAB', 'services', 'me']);
    expect(keys('pregnancy')).toEqual(['today:/pregnancy', 'pregnancy:/pregnancy/weeks', 'FAB', 'services', 'me']);
    expect(keys('menopause')).toEqual(['today:/home', 'symptoms:/analysis/symptoms', 'FAB', 'services', 'me']);
  });

  it('postpartum «کودک» targets the only child, else the list (gaps.md #15)', () => {
    expect(keys('postpartum', ['7'])[1]).toBe('child:/children/7');
    expect(keys('postpartum', [])[1]).toBe('child:/children');
    expect(keys('postpartum', ['1', '2'])[1]).toBe('child:/children');
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
