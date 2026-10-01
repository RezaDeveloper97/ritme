/**
 * Night & Bloom bottom nav per life-stage mode (B-N1-04, docs/night-bloom/nav.md):
 * امروز · <mode tab> · FAB(+) · خدمات · من.
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

export type NavKey = 'today' | 'calendar' | 'fertility' | 'pregnancy' | 'child' | 'symptoms' | 'services' | 'me';

export type NavIconName = 'home' | 'calendar' | 'fertility' | 'pregnancy' | 'child' | 'symptoms' | 'services' | 'me';

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
  /** Override {@link NAV_READY} (tests). */
  ready?: Partial<NavReady>;
}

export interface NavReady {
  /** `/postpartum` + `/children` exist (B-N5-01/05). */
  postpartum: boolean;
  /** `/analysis/symptoms` exists (B-N3-09 — on). */
  analysis: boolean;
}

/**
 * Which nav targets of the per-mode table already have screens. Until they do,
 * a mode whose tab would 404 gets the interim target (B-N2-03, QUESTIONS #60):
 * postpartum → امروز `/home` (cycle home + a "coming soon" note) and the
 * «تقویم» tab; menopause «علائم» → the symptom pattern `/cycle/symptoms`.
 * The owning tasks flip their flag — nothing else changes.
 */
export const NAV_READY: NavReady = { postpartum: false, analysis: true };

const SERVICES: NavTab = { key: 'services', href: '/services', icon: 'services' };
const ME: NavTab = { key: 'me', href: '/profile', icon: 'me' };

/** The second tab, whose label and target follow the mode. */
export function modeTab(mode: NavMode, options: NavOptions = {}): NavTab | null {
  const ready = { ...NAV_READY, ...options.ready };
  if (mode === 'postpartum' && !ready.postpartum) return modeTab('cycle', options);
  switch (mode) {
    case 'cycle':
    case 'teen':
      // /cycle (history) and the future /analysis live under the mode tab (nav.md).
      return { key: 'calendar', href: '/calendar', icon: 'calendar', alsoActive: ['/cycle', '/analysis', '/analysis/'] };
    case 'ttc':
      return {
        key: 'fertility',
        href: '/calendar',
        icon: 'fertility',
        alsoActive: ['/cycle', '/analysis', '/analysis/', '/fertility/'],
      };
    case 'pregnancy':
      return {
        key: 'pregnancy',
        href: '/pregnancy/weeks',
        icon: 'pregnancy',
        alsoActive: ['/pregnancy/weeks/', '/pregnancy/calendar', '/analysis', '/analysis/'],
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
      return {
        key: 'symptoms',
        href: ready.analysis ? '/analysis/symptoms' : '/cycle/symptoms',
        icon: 'symptoms',
        alsoActive: ['/analysis', '/analysis/', '/cycle/symptoms'],
      };
    case 'companion':
      return null;
  }
}

/** «امروز» target per mode. */
export function todayHref(mode: NavMode, ready: Partial<NavReady> = {}): string {
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

/** Every tab + FAB presence for `mode`. */
export function navConfig(mode: NavMode, options: NavOptions = {}): NavConfig {
  const today: NavTab = { key: 'today', href: todayHref(mode, options.ready), icon: 'home' };
  const second = modeTab(mode, options);
  return {
    before: second ? [today, second] : [today],
    after: [SERVICES, ME],
    fab: mode !== 'companion',
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
 * Services sub-hubs (`/services/*`) keep «خدمات» lit (nav.md).
 */
export function activeTabKey(config: NavConfig, pathname: string): NavKey | null {
  const path = stripSlash(pathname);
  const tabs = [...config.before, ...config.after];
  const exact = tabs.find((tab) => tab.href === path);
  if (exact) return exact.key;
  if (path.startsWith('/services/')) return 'services';
  const also = tabs.find((tab) => tab.alsoActive?.some((p) => matches(p, path)));
  return also?.key ?? null;
}
