'use client';

import { QueryClientProvider } from '@tanstack/react-query';
import { useEffect, useState, type ReactNode } from 'react';

import { createQueryClient, setUnauthorizedHandler } from '@/shared/api';
import { stripBasePath, withBasePath } from '@/shared/config';
import { ConfirmDialog, Toaster } from '@/shared/ui';

/**
 * Client-side app init: one QueryClient, the global 401 → /login redirect,
 * and the toast / confirm hosts.
 */
export function Providers({ children }: { children: ReactNode }) {
  const [queryClient] = useState(createQueryClient);

  useEffect(() => {
    setUnauthorizedHandler(() => {
      // window.location carries the base path (`/panel/...` on staging); `next`
      // is an app path, because the login form hands it to router.replace().
      const pathname = stripBasePath(window.location.pathname);
      if (pathname === '/login') return;
      queryClient.clear();
      // Full document replace: nothing of the dead session survives in memory.
      const next = encodeURIComponent(pathname + window.location.search);
      window.location.replace(withBasePath(`/login?next=${next}`));
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
