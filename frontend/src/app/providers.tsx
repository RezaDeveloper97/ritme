'use client';

import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { type ReactNode, useEffect, useState } from 'react';

import { track } from '@/shared/analytics';

/**
 * Client-side providers. Server state lives in TanStack Query — never copied
 * into Zustand or useState (CLAUDE.md §8). The QueryClient is created once per
 * browser session (lazy initializer) so it stays stable across re-renders.
 */
export function AppProviders({ children }: { children: ReactNode }) {
  const [queryClient] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: {
            staleTime: 60_000,
            refetchOnWindowFocus: false,
          },
        },
      }),
  );

  useEffect(() => {
    track({ name: 'app_open' });
  }, []);

  return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
}
