'use client';

import type { ReactNode } from 'react';

import { ErrorState } from './ErrorState';
import { Panel } from './Panel';
import { Skeleton } from './Skeleton';

type Query = { data: unknown; error: unknown; refetch: () => unknown; fetchStatus?: string; isPending?: boolean };

/**
 * Loading / error frame for a form or detail screen that needs one or more
 * queries before it can render. `children` is a function so the form mounts
 * only once its data is there (its useState seeds from it).
 */
export function LoadGate({
  queries,
  header,
  children,
}: {
  /** Disabled queries (e.g. the detail query on a create form) are ignored. */
  queries: readonly Query[];
  header?: ReactNode;
  children: () => ReactNode;
}) {
  const needed = queries.filter((q) => !(q.isPending && q.fetchStatus === 'idle' && q.data === undefined));
  const failed = needed.find((q) => q.error && q.data === undefined);
  if (failed) {
    return (
      <div className="flex flex-col gap-4">
        {header}
        <Panel>
          <ErrorState error={failed.error} onRetry={() => needed.forEach((q) => q.error && q.refetch())} />
        </Panel>
      </div>
    );
  }
  if (needed.some((q) => q.data === undefined)) {
    return (
      <div className="flex flex-col gap-4" aria-busy="true">
        {header}
        <Panel>
          <div className="flex flex-col gap-4">
            <Skeleton className="h-9 w-1/2" />
            <Skeleton className="h-9 w-full" />
            <Skeleton className="h-28 w-full" />
            <Skeleton className="h-9 w-1/3" />
          </div>
        </Panel>
      </div>
    );
  }
  return <>{children()}</>;
}
