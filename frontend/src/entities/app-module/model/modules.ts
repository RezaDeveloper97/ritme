/**
 * Admin kill-switches for whole app modules (GET /modules, backend-go internal/modules). An off module leaves
 * onboarding and the app; its API answers 403 `module_disabled`. Unknown or not-yet-loaded = on (fail open, like
 * the API), so a slow or failed read never hides a live feature.
 */
export const APP_MODULES = [
  'companion',
  'ttc',
  'ivf',
  'menopause',
  'teen',
  'postpartum',
  'children',
  'contraception',
  'programs',
  'mother_child',
  'learning',
  'my_courses', // the «دوره‌های من» row only (no route of its own)
  'shop',
  'insurance',
  'record',
  'devices',
  'doctors', // پزشک و ماما
  'todo',
  'reminders', // the «یادآورها» hub + «یادآورهای امروز» card; the visit/medication forms stay (other modules use them)
  'home_articles', // home's «بر اساس سیکل فعلی‌ات» strip (no route of its own)
] as const;
export type AppModule = (typeof APP_MODULES)[number];

export type AppModules = Record<AppModule, boolean>;

export const ALL_MODULES_ON: AppModules = Object.fromEntries(APP_MODULES.map((m) => [m, true])) as AppModules;

/** The module that owns an app route (locale prefix stripped), or null. */
export function moduleOfRoute(pathname: string): AppModule | null {
  const path = pathname.replace(/\/+$/, '') || '/';
  const under = (prefix: string) => path === prefix || path.startsWith(`${prefix}/`);
  if (under('/companion') || under('/companions') || under('/onboarding/partner')) return 'companion';
  if (under('/ivf')) return 'ivf';
  if (under('/fertility') || under('/analysis/fertility')) return 'ttc';
  if (under('/menopause')) return 'menopause';
  if (under('/teen')) return 'teen';
  if (under('/postpartum')) return 'postpartum';
  if (under('/children')) return 'children';
  if (under('/contraception')) return 'contraception';
  if (under('/programs')) return 'programs';
  // The public web pages of the city services (places, venue sign-up) belong to the same module.
  if (under('/city-services') || under('/places') || under('/business') || under('/profile/bookings')) return 'mother_child';
  if (under('/learn')) return 'learning';
  if (under('/shop') || under('/store') || under('/profile/orders')) return 'shop';
  if (under('/insurance')) return 'insurance';
  // The emergency card is safety information: it stays reachable with the rest of the record switched off.
  if (under('/record/emergency')) return null;
  if (under('/record')) return 'record';
  if (under('/profile/devices')) return 'devices';
  if (under('/services/doctors') || under('/services/bookings')) return 'doctors';
  if (/^\/assistant\/chat\/[^/]+\/handoff$/.test(path)) return 'doctors'; // «ارسال خلاصه به پزشک» needs a booking
  if (under('/todo')) return 'todo';
  // Only the hub: pregnancy visits, IVF, checkups and the companion book through /reminders/{appointment,medication}.
  if (path === '/reminders') return 'reminders';
  return null;
}

/** The life modes an admin switch can hide (cycle and pregnancy are always on). */
export function modeAvailable(modules: AppModules, mode: string): boolean {
  switch (mode) {
    case 'ttc':
    case 'menopause':
    case 'teen':
    case 'postpartum':
      return modules[mode];
    default:
      return true;
  }
}
