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
  /** `mode` from `/messages/mode` (`cycle` | `pregnancy` today; future modes pass through). */
  mode?: string | null;
  isTtc?: boolean;
}

const KNOWN: readonly NavMode[] = ['cycle', 'ttc', 'pregnancy', 'postpartum', 'menopause', 'teen', 'companion'];

/** Map the API's mode + TTC flag onto a nav mode; unknown or missing → `cycle`. */
export function resolveNavMode({ mode, isTtc }: NavModeInput): NavMode {
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
}

const SERVICES: NavTab = { key: 'services', href: '/services', icon: 'services' };
const ME: NavTab = { key: 'me', href: '/profile', icon: 'me' };

/** The second tab, whose label and target follow the mode. */
export function modeTab(mode: NavMode, options: NavOptions = {}): NavTab | null {
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
      return { key: 'symptoms', href: '/analysis/symptoms', icon: 'symptoms', alsoActive: ['/analysis', '/analysis/'] };
    case 'companion':
      return null;
  }
}

/** «امروز» target per mode. */
export function todayHref(mode: NavMode): string {
  switch (mode) {
    case 'pregnancy':
      return '/pregnancy';
    case 'postpartum':
      return '/postpartum';
    case 'companion':
      return '/companion';
    default:
      return '/home';
  }
}

/** Every tab + FAB presence for `mode`. */
export function navConfig(mode: NavMode, options: NavOptions = {}): NavConfig {
  const today: NavTab = { key: 'today', href: todayHref(mode), icon: 'home' };
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
