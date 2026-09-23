/**
 * Server-list state lives in the URL (`?page=2&q=sara&status=blocked`), so a
 * list is shareable and survives reload. Pure helpers here; the React hook is
 * use-list-params.ts.
 */
export interface ListParams {
  page: number;
  perPage: number;
  q: string;
  filters: Record<string, string>;
}

export const DEFAULT_PER_PAGE = 20;

export function parseListParams(
  search: URLSearchParams,
  filterDefaults: Record<string, string> = {},
  perPageDefault = DEFAULT_PER_PAGE,
): ListParams {
  const int = (raw: string | null, fallback: number, max: number): number => {
    const n = Number(raw);
    return Number.isInteger(n) && n >= 1 ? Math.min(n, max) : fallback;
  };
  const filters: Record<string, string> = {};
  for (const [key, fallback] of Object.entries(filterDefaults)) {
    filters[key] = search.get(key) || fallback;
  }
  return {
    page: int(search.get('page'), 1, Number.MAX_SAFE_INTEGER),
    perPage: int(search.get('per_page'), perPageDefault, 100),
    q: (search.get('q') ?? '').trim(),
    filters,
  };
}

/** Query object for the API (`page`, `per_page`, `q`, filters). */
export function toApiQuery(params: ListParams): Record<string, string | number> {
  const query: Record<string, string | number> = { page: params.page, per_page: params.perPage };
  if (params.q) query.q = params.q;
  for (const [key, value] of Object.entries(params.filters)) query[key] = value;
  return query;
}

/**
 * Next URL search string after a change. Changing anything but the page resets
 * to page 1; values equal to their default are dropped to keep URLs short.
 */
export function nextSearch(
  current: URLSearchParams,
  patch: Record<string, string | number | null>,
  filterDefaults: Record<string, string> = {},
): string {
  const next = new URLSearchParams(current);
  for (const [key, raw] of Object.entries(patch)) {
    const value = raw === null ? '' : String(raw);
    if (value === '' || value === filterDefaults[key] || (key === 'page' && value === '1')) {
      next.delete(key);
    } else {
      next.set(key, value);
    }
  }
  if (!('page' in patch)) next.delete('page');
  const qs = next.toString();
  return qs ? `?${qs}` : '';
}
