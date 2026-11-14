/**
 * Night & Bloom bottom nav per life-stage mode (B-N1-04, docs/night-bloom/nav.md):
 * امروز · <mode tab> · FAB(+) · تحلیل · من. The services hub (`/services`) left
 * the nav for «تحلیل» (analysis + cycle history) and is now reached from «من».
 *
 * DOM order is the RTL visual order: «امروز» sits on the right in `fa`, «من»
 * on the left, the FAB always in the middle; in LTR it simply mirrors.
 */

/**
 * The life-stage a user's nav is built for. `cycle` and `pregnancy` come from
 * the API's `mode` today, `ttc` from its `isTtc` flag; the rest arrive with N2
 * (onboarding v2 modes) / N4 (companion) and already have their tabs here.
 */
export type NavMode = 'cycle' | 'ttc' | 'pregnancy' | 'postpartum' | 'menopause' | 'teen' | 'companion';

export type NavKey =
  | 'today'
  | 'calendar'
  | 'fertility'
  | 'treatment'
  | 'pregnancy'
  | 'child'
  | 'symptoms'
  | 'analysis'
  | 'services'
  | 'me'
  // Companion nav (companion v2 tabbarM): یادگیری · اقدام‌ها · پروفایل.
  | 'learn'
  | 'actions'
  | 'profile';

export type NavIconName =
  | 'home'
  | 'calendar'
  | 'fertility'
  | 'treatment'
  | 'pregnancy'
  | 'child'
  | 'symptoms'
  | 'analysis'
  | 'services'
  | 'me'
  | 'learn'
  | 'actions';

export interface NavTab {
  /** Stable key; also the label's message key under the `nav` i18n namespace. */
  key: NavKey;
  /** Locale-agnostic path — the locale prefix is added by the i18n `Link`. */
  href: string;
  icon: NavIconName;
  /** Other paths that light this tab up (exact, or a prefix when it ends in `/`). */
  alsoActive?: readonly string[];
}

export interface NavConfig {
  /** Tabs in DOM order; the FAB sits between `before` and `after`. */
  before: NavTab[];
  after: NavTab[];
  /** Companion nav has no FAB (nav.md «male companion»). */
  fab: boolean;
}

export interface NavModeInput {
  /**
   * The effective life-stage mode of `GET /profile/life-stage` (B-N2-01/03) —
   * the source of truth when present (it already folds in the TTC goal and an
   * active pregnancy profile).
   */
  lifeMode?: string | null;
  /** `mode` from `/messages/mode` (`cycle` | `pregnancy` | `postpartum`): the fallback where life-stage is missing. */
  mode?: string | null;
  isTtc?: boolean;
}

const KNOWN: readonly NavMode[] = ['cycle', 'ttc', 'pregnancy', 'postpartum', 'menopause', 'teen', 'companion'];

/** Map the life-stage mode (else the API's message mode + TTC flag) onto a nav mode; unknown or missing → `cycle`. */
export function resolveNavMode({ lifeMode, mode, isTtc }: NavModeInput): NavMode {
  if (lifeMode && (KNOWN as readonly string[]).includes(lifeMode)) return lifeMode as NavMode;
  if (mode === 'pregnancy') return 'pregnancy';
  if (mode && mode !== 'cycle' && (KNOWN as readonly string[]).includes(mode)) return mode as NavMode;
  return isTtc ? 'ttc' : 'cycle';
}

export interface NavOptions {
  /**
   * Postpartum «کودک» target (gaps.md #15): exactly one child → that child's
   * page, zero or several → the children list. Unknown until N5 → the list.
   */
  childIds?: readonly (string | number)[];
  /**
   * CB-IVF-02 (CB-CORE-01 C3): IVF is a TTC sub-mode — bloom's «IVF/IUI»
   * switch (`ivf_iui`) on a `ttc` user gives امروز → `/ivf` and the stage tab
   * «درمان». Ignored for every other mode; off → the plain TTC nav.
   */
  ivf?: boolean;
  /** Override {@link NAV_READY} (tests). */
  ready?: Partial<NavReady>;
}

export interface NavReady {
  /** `/postpartum` (the postpartum «امروز», B-N5-04) exists. */
  postpartum: boolean;
  /** `/children` (the «کودک» tab, B-N5-05) exists; until then postpartum keeps the «تقویم» tab. */
  children: boolean;
  /** `/analysis/*` exists (B-N3-09 — on). The menopause tab no longer depends on it (CB-MENO-08). */
  analysis: boolean;
  /**
   * `/ivf/meds` (the injection schedule, CB-IVF-03) exists. Until then «درمان»
   * opens the IVF home at its «تزریق‌های امروز» section ({@link IVF_TREATMENT_FALLBACK}).
   */
  ivfMeds: boolean;
}

/**
 * Which nav targets of the per-mode table already have screens. Until they do,
 * a mode whose tab would 404 gets the interim target (B-N2-03, QUESTIONS #60):
 * postpartum → امروز `/home` until `/postpartum` (B-N5-04, on) and the «تقویم»
 * tab until `/children` (B-N5-05).
 * The owning tasks flip their flag — nothing else changes.
 */
export const NAV_READY: NavReady = { postpartum: true, children: true, analysis: true, ivfMeds: true };

/** «درمان» before CB-IVF-03: the IVF home's today-injections section (`id="ivf-doses"`), never a 404. */
export const IVF_TREATMENT_FALLBACK = '/ivf#ivf-doses';

// «تحلیل»: the analysis hub + the cycle history (`/cycle` redirects to `/analysis?tab=history`).
const ANALYSIS: NavTab = { key: 'analysis', href: '/analysis', icon: 'analysis', alsoActive: ['/analysis/', '/cycle', '/cycle/'] };
// B-N6-08: «کارهای من» lives under «من» (nav.md: Todo_Home → من). The services hub and its sub-hubs
// (B-N6-02 vitals, programs, insurance, city services) moved under «من» when «تحلیل» took the «خدمات» slot.
const SERVICES_PATHS: readonly string[] = [
  '/services',
  '/vitals',
  '/programs',
  '/programs/pelvic',
  '/insurance',
  '/assistant',
  '/city-services',
  '/city-services/',
];
const ME: NavTab = { key: 'me', href: '/profile', icon: 'me', alsoActive: ['/todo', '/learn', ...SERVICES_PATHS] };

/** The second tab, whose label and target follow the mode. */
export function modeTab(mode: NavMode, options: NavOptions = {}): NavTab | null {
  const ready = { ...NAV_READY, ...options.ready };
  if (mode === 'postpartum' && !ready.children) return modeTab('cycle', options);
  if (mode === 'ttc' && options.ivf) {
    // CB-IVF-02: the IVF stage tab «درمان» (syringe) → the injection schedule.
    return {
      key: 'treatment',
      href: ready.ivfMeds ? '/ivf/meds' : IVF_TREATMENT_FALLBACK,
      icon: 'treatment',
      alsoActive: ['/ivf/'],
    };
  }
  switch (mode) {
    case 'cycle':
    case 'teen':
      return { key: 'calendar', href: '/calendar', icon: 'calendar' };
    case 'ttc':
      return {
        key: 'fertility',
        href: '/calendar',
        icon: 'fertility',
        alsoActive: ['/fertility/'],
      };
    case 'pregnancy':
      return {
        key: 'pregnancy',
        href: '/pregnancy/weeks',
        icon: 'pregnancy',
        alsoActive: ['/pregnancy/weeks/', '/pregnancy/calendar'],
      };
    case 'postpartum': {
      const ids = options.childIds ?? [];
      return {
        key: 'child',
        href: ids.length === 1 ? `/children/${ids[0]}` : '/children',
        icon: 'child',
        alsoActive: ['/children', '/children/'],
      };
    }
    case 'menopause':
      // CB-MENO-08 (CB-CORE-01 C3): «علائم» = the monthly score (nbl_Meno_Score).
      return { key: 'symptoms', href: '/menopause/score', icon: 'symptoms' };
    case 'companion':
      return null;
  }
}

/** «امروز» target per mode (`ivf` = the TTC IVF sub-mode, CB-IVF-02). */
export function todayHref(mode: NavMode, ready: Partial<NavReady> = {}, ivf = false): string {
  if (mode === 'ttc' && ivf) return '/ivf';
  switch (mode) {
    case 'pregnancy':
      return '/pregnancy';
    case 'postpartum':
      return { ...NAV_READY, ...ready }.postpartum ? '/postpartum' : '/home';
    case 'companion':
      return '/companion';
    default:
      return '/home';
  }
}

/**
 * Companion v2 (tabbarM): امروز · یادگیری · اقدام‌ها · پروفایل, no FAB — a man logs nothing of hers, and his own
 * value is the learn content and the action ideas.
 */
const COMPANION_NAV: NavConfig = {
  before: [
    { key: 'today', href: '/companion', icon: 'home' },
    { key: 'learn', href: '/companion/learn', icon: 'learn' },
  ],
  after: [
    { key: 'actions', href: '/companion/actions', icon: 'actions' },
    { key: 'profile', href: '/profile', icon: 'me' },
  ],
  fab: false,
};

/** Every tab + FAB presence for `mode`. */
export function navConfig(mode: NavMode, options: NavOptions = {}): NavConfig {
  if (mode === 'companion') return COMPANION_NAV;
  const today: NavTab = { key: 'today', href: todayHref(mode, options.ready, options.ivf), icon: 'home' };
  const second = modeTab(mode, options);
  return {
    before: second ? [today, second] : [today],
    after: [ANALYSIS, ME],
    fab: true,
  };
}

function stripSlash(path: string): string {
  return path.length > 1 && path.endsWith('/') ? path.slice(0, -1) : path;
}

function matches(pattern: string, path: string): boolean {
  return pattern.endsWith('/') ? path.startsWith(pattern) : path === pattern;
}

/**
 * Which tab is active on `pathname` (locale stripped), or `null`. The FAB is
 * never "active" — it opens a sheet over the current screen.
 * Services sub-screens (`/services/*`) keep «من» lit (the services hub lives under «من»).
 */
export function activeTabKey(config: NavConfig, pathname: string): NavKey | null {
  const path = stripSlash(pathname);
  const tabs = [...config.before, ...config.after];
  const exact = tabs.find((tab) => tab.href === path);
  if (exact) return exact.key;
  if (path.startsWith('/services/')) return 'me';
  const also = tabs.find((tab) => tab.alsoActive?.some((p) => matches(p, path)));
  return also?.key ?? null;
}

/**
 * CB-NAV-03 (IA_Nav «بازگشت»): re-tapping the active tab returns to that tab's root. From a sub-screen
 * the tab's link already goes there; on the root itself the tap has nowhere to go, so the screen
 * scrolls back to the top instead. `true` = the tap is on the root of the active tab (scroll to top).
 */
export function isRootRetap(tab: NavTab, pathname: string): boolean {
  return stripSlash(pathname) === tab.href.split('#')[0];
}
