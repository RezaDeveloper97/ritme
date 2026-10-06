/**
 * Where the floating bottom nav is shown (Night & Bloom, B-N1-04,
 * docs/night-bloom/nav.md «Where the nav is shown").
 *
 * The nav belongs to **tab roots and first-level hubs** only. Screens with a
 * back-button header, onboarding, checkout, forms and sheets never show it.
 * Paths are locale-agnostic (no `/fa` prefix). A screen task that adds a new
 * hub appends it here — the widget itself never changes for that.
 */

/** Exact paths that show the nav. */
export const NAV_ROOT_PATHS: readonly string[] = [
  '/home',
  '/calendar',
  '/services',
  '/profile',
  '/pregnancy',
  '/pregnancy/weeks',
  '/postpartum',
  '/children',
  '/companion',
  '/analysis',
  '/menopause/score', // CB-MENO-08: menopause mode tab «علائم» (nbl_Meno_Score)
  '/ivf', // CB-IVF-02: IVF «امروز» (nbl_IVF_Home, TTC + «IVF/IUI»)
  '/ivf/meds', // CB-IVF-03: IVF stage tab «درمان» (nbl_IVF_Meds)
  '/vitals', // B-N6-02: «علائم حیاتی» hub (nbl_Vitals_Hub, خدمات tab)
  // Transitional (B-N1-04): these screens have no back button yet, so hiding
  // the nav would strand the user. Their restyle tasks add a ScreenHeader and
  // drop them from this list: /log (B-N3-03),
  // /pregnancy/calendar (B-N1-14).
  '/log',
  '/pregnancy/calendar',
];

/** Prefixes whose sub-paths also show the nav (`/pregnancy/weeks/12`, `/analysis/symptoms`). */
export const NAV_ROOT_PREFIXES: readonly string[] = ['/pregnancy/weeks/', '/analysis/'];

/**
 * Path patterns that show the nav: the child home `/children/<id>` (v16_ChildHome, the «کودک» tab, B-N5-05) —
 * but not `/children/new`, `/children/<id>/edit` or the growth/vaccines/… sub-screens (back headers).
 */
export const NAV_ROOT_PATTERNS: readonly RegExp[] = [/^\/children\/\d+$/];

/** Should the bottom nav render on `pathname` (locale already stripped)? */
export function isNavRootPath(pathname: string): boolean {
  const path = pathname.length > 1 && pathname.endsWith('/') ? pathname.slice(0, -1) : pathname;
  return (
    NAV_ROOT_PATHS.includes(path) ||
    NAV_ROOT_PREFIXES.some((p) => path.startsWith(p)) ||
    NAV_ROOT_PATTERNS.some((re) => re.test(path))
  );
}
