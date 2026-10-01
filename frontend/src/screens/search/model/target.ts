/**
 * Where a search hit opens. The API names routes by the product map, and some
 * of those screens don't exist in the app yet (CB-NAV-01 open item: the pelvic
 * program CB-PELV-02, article pages). A hit opens only a screen that exists —
 * an article opens the article sheet — and anything else is hidden, so search
 * never links to a 404. A new screen is added here when it ships.
 */
export type SearchTarget =
  | { kind: 'route'; href: string }
  | { kind: 'sheet'; id: 'article'; arg: string };

/** Exact paths that exist (locale-free). */
const EXACT_ROUTES: ReadonlySet<string> = new Set([
  '/contraception',
  '/checkups',
  '/checkups/self-exam',
  '/reminders',
  '/analysis',
  '/analysis/period',
  '/analysis/body',
  '/analysis/symptoms',
  '/analysis/cycle',
  '/analysis/labs',
  '/analysis/correlations',
]);

/** Parameterised paths that exist. */
const ROUTE_PATTERNS: readonly RegExp[] = [
  /^\/checkups\/\d+$/,
  /^\/reminders\/(medication|appointment)\/\d+$/,
];

const ARTICLE = /^\/articles\/([^/?#]+)$/;

export function searchTarget(route: string): SearchTarget | null {
  const [path] = route.split(/[?#]/, 1);
  if (!path || !path.startsWith('/') || path.startsWith('//')) return null;
  const article = ARTICLE.exec(path);
  if (article) {
    try {
      return { kind: 'sheet', id: 'article', arg: decodeURIComponent(article[1]) };
    } catch {
      return null;
    }
  }
  if (EXACT_ROUTES.has(path) || ROUTE_PATTERNS.some((p) => p.test(path))) {
    return { kind: 'route', href: route };
  }
  return null;
}
