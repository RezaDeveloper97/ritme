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
  // Transitional (B-N1-04): these screens have no back button yet, so hiding
  // the nav would strand the user. Their restyle tasks add a ScreenHeader and
  // drop them from this list: /log (B-N3-03),
  // /pregnancy/calendar (B-N1-14).
  '/log',
  '/pregnancy/calendar',
];

/** Prefixes whose sub-paths also show the nav (`/pregnancy/weeks/12`, `/analysis/symptoms`). */
export const NAV_ROOT_PREFIXES: readonly string[] = ['/pregnancy/weeks/', '/analysis/', '/children/'];

/** Should the bottom nav render on `pathname` (locale already stripped)? */
export function isNavRootPath(pathname: string): boolean {
  const path = pathname.length > 1 && pathname.endsWith('/') ? pathname.slice(0, -1) : pathname;
  return NAV_ROOT_PATHS.includes(path) || NAV_ROOT_PREFIXES.some((p) => path.startsWith(p));
}
