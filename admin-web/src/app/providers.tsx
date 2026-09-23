'use client';

import { QueryClientProvider } from '@tanstack/react-query';
import { useEffect, useState, type ReactNode } from 'react';

import { createQueryClient, setUnauthorizedHandler } from '@/shared/api';
import { ConfirmDialog, Toaster } from '@/shared/ui';

/**
 * Client-side app init: one QueryClient, the global 401 → /login redirect,
 * and the toast / confirm hosts.
 */
export function Providers({ children }: { children: ReactNode }) {
  const [queryClient] = useState(createQueryClient);

  useEffect(() => {
    setUnauthorizedHandler(() => {
      const { pathname, search } = window.location;
      if (pathname === '/login') return;
      queryClient.clear();
      // Full document replace: nothing of the dead session survives in memory.
      window.location.replace(`/login?next=${encodeURIComponent(pathname + search)}`);
    });
    return () => setUnauthorizedHandler(null);
  }, [queryClient]);

  return (
    <QueryClientProvider client={queryClient}>
      {children}
      <Toaster />
      <ConfirmDialog />
    </QueryClientProvider>
  );
}
