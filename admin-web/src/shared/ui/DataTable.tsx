'use client';

import { useTranslations } from 'next-intl';
import type { ReactNode } from 'react';

import { cn } from '@/shared/lib';

import { ErrorState } from './ErrorState';
import { Skeleton } from './Skeleton';

export interface Column<T> {
  key: string;
  header: ReactNode;
  cell: (row: T) => ReactNode;
  /** `num` right-aligns digits in their own column (tabular). */
  className?: string;
}

/**
 * Presentational table for one page of a server list. Paging/filtering state
 * belongs to the screen (useListParams) — pass `items` from the query and put
 * <Pagination meta={…}/> under it.
 */
export function DataTable<T>({
  columns,
  items,
  rowKey,
  loading = false,
  error,
  onRetry,
  emptyText,
  onRowClick,
  skeletonRows = 6,
  caption,
}: {
  columns: ReadonlyArray<Column<T>>;
  items: readonly T[] | undefined;
  rowKey: (row: T) => string | number;
  loading?: boolean;
  error?: unknown;
  onRetry?: () => void;
  emptyText?: ReactNode;
  onRowClick?: (row: T) => void;
  skeletonRows?: number;
  caption?: string;
}) {
  const t = useTranslations('table');
  if (error && !items) return <ErrorState error={error} onRetry={onRetry} />;

  return (
    <div className="table-wrap">
      <table className="data-table" aria-busy={loading || undefined}>
        {caption ? <caption className="sr-only">{caption}</caption> : null}
        <thead>
          <tr>
            {columns.map((c) => (
              <th key={c.key} scope="col" className={c.className}>
                {c.header}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {loading && !items
            ? Array.from({ length: skeletonRows }, (_, i) => (
                <tr key={`sk-${i}`}>
                  {columns.map((c) => (
                    <td key={c.key}>
                      <Skeleton className="w-3/4" />
                    </td>
                  ))}
                </tr>
              ))
            : null}
          {items && items.length === 0 ? (
            <tr>
              <td className="cell-empty" colSpan={columns.length}>
                {emptyText ?? t('empty')}
              </td>
            </tr>
          ) : null}
          {items?.map((row) => (
            <tr
              key={rowKey(row)}
              data-clickable={onRowClick ? 'true' : undefined}
              onClick={onRowClick ? () => onRowClick(row) : undefined}
            >
              {columns.map((c) => (
                <td key={c.key} className={cn(c.className)}>
                  {c.cell(row)}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
