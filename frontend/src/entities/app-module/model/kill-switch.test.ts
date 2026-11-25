import { describe, expect, it } from 'vitest';

import { ALL_MODULES_ON, APP_MODULES, type AppModule, modeAvailable, moduleOfRoute, servicesHubHref } from './modules';

// T-M12-18: with a module off, every entry point that resolves to it is hidden (the same predicate the home feed
// CTAs, quick-access shortcuts and the route gate use); with it on, nothing is hidden. Mirrors backend modules.Keys.
const ENTRY_POINTS: Record<string, string[]> = {
  companion: ['/companion', '/companions/3'],
  ttc: ['/fertility/bbt', '/analysis/fertility'],
  ivf: ['/ivf/meds'],
  menopause: ['/menopause/hot-flash'],
  teen: ['/teen/parent'],
  postpartum: ['/postpartum/mood'],
  children: ['/children/2'],
  contraception: ['/contraception'],
  programs: ['/programs/pain'],
  mother_child: ['/city-services/1'],
  learning: ['/learn'],
  shop: ['/shop/cart'],
  insurance: ['/insurance'],
  record: ['/record'],
  doctors: ['/services/doctors/1/book'],
  todo: ['/todo'],
  devices: ['/profile/devices'],
  reminders: ['/reminders'],
  // «من» rows (user request 2026-11-25).
  services: ['/services'],
  vitals: ['/vitals', '/vitals/bp/new'],
  backup: ['/profile/backup'],
  support: ['/profile/support'],
};
const NO_ROUTE: AppModule[] = ['my_courses', 'home_articles'];

const hidden = (modules: Record<AppModule, boolean>, href: string) => {
  const owner = moduleOfRoute(href);
  return owner !== null && !modules[owner];
};

describe('module kill-switch hides entry points', () => {
  it('covers every module', () => {
    for (const m of APP_MODULES) expect(m in ENTRY_POINTS || NO_ROUTE.includes(m), m).toBe(true);
  });

  for (const [mod, hrefs] of Object.entries(ENTRY_POINTS)) {
    it(`${mod}: hidden only while off`, () => {
      const off = { ...ALL_MODULES_ON, [mod]: false } as Record<AppModule, boolean>;
      for (const h of hrefs) {
        expect(moduleOfRoute(h), h).toBe(mod);
        expect(hidden(off, h), h).toBe(true);
        expect(hidden(ALL_MODULES_ON, h), h).toBe(false);
        // boundary: another module off leaves it visible
        const other = { ...ALL_MODULES_ON, todo: mod === 'todo' ? true : false } as Record<AppModule, boolean>;
        expect(hidden(other, h), h).toBe(false);
      }
    });
  }

  it('keeps the emergency card and unowned routes visible with everything off', () => {
    const allOff = Object.fromEntries(APP_MODULES.map((m) => [m, false])) as Record<AppModule, boolean>;
    for (const h of ['/record/emergency', '/home', '/pregnancy']) expect(hidden(allOff, h), h).toBe(false);
    expect(modeAvailable(allOff, 'ttc')).toBe(false);
    expect(modeAvailable(allOff, 'cycle')).toBe(true);
  });
});

describe('services hub switch', () => {
  it('owns only the hub: its sub-sections keep their own switches', () => {
    expect(moduleOfRoute('/services')).toBe('services');
    expect(moduleOfRoute('/services/doctors')).toBe('doctors');
    expect(moduleOfRoute('/services/home-lab')).toBeNull();
  });
  it('sends «back» to «من» while the hub is off', () => {
    expect(servicesHubHref(ALL_MODULES_ON)).toBe('/services');
    expect(servicesHubHref({ ...ALL_MODULES_ON, services: false })).toBe('/profile');
  });
});
