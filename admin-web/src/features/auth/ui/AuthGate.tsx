'use client';

import type { ReactNode } from 'react';

import { isApiError } from '@/shared/api';
import { ErrorState, Spinner } from '@/shared/ui';

import { useMe } from '../api/queries';

/**
 * Renders its children only for a signed-in admin. A 401 is handled globally
 * (redirect to /login); any other failure shows a retry.
 */
export function AuthGate({ children }: { children: ReactNode }) {
  const me = useMe();
  if (me.data) return <>{children}</>;
  if (me.error && !(isApiError(me.error) && me.error.status === 401)) {
    return (
      <div className="grid min-h-dvh place-items-center">
        <ErrorState error={me.error} onRetry={() => me.refetch()} />
      </div>
    );
  }
  return (
    <div className="grid min-h-dvh place-items-center text-brand" aria-busy="true">
      <Spinner className="size-6" />
    </div>
  );
}
