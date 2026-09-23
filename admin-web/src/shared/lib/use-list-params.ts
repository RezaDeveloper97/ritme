'use client';

import { usePathname, useRouter, useSearchParams } from 'next/navigation';
import { useCallback, useMemo } from 'react';

import { nextSearch, parseListParams, toApiQuery, type ListParams } from './list-params';

/**
 * URL-synced list state for a server-paginated table:
 *   const list = useListParams({ status: 'all' });
 *   useQuery({ queryKey: userKeys.list(list.query), queryFn: () => fetchUsers(list.query) })
 */
export function useListParams(filterDefaults: Record<string, string> = {}) {
  const search = useSearchParams();
  const router = useRouter();
  const pathname = usePathname();
  // Stable identity for the defaults object literal callers pass inline.
  const defaultsKey = JSON.stringify(filterDefaults);

  const params: ListParams = useMemo(
    () => parseListParams(new URLSearchParams(search.toString()), JSON.parse(defaultsKey)),
    [search, defaultsKey],
  );
  const query = useMemo(() => toApiQuery(params), [params]);

  const update = useCallback(
    (patch: Record<string, string | number | null>) => {
      const qs = nextSearch(new URLSearchParams(search.toString()), patch, JSON.parse(defaultsKey));
      router.replace(`${pathname}${qs}`, { scroll: false });
    },
    [search, router, pathname, defaultsKey],
  );

  return {
    params,
    query,
    setPage: (page: number) => update({ page }),
    setSearch: (q: string) => update({ q }),
    setFilter: (key: string, value: string) => update({ [key]: value }),
  };
}
