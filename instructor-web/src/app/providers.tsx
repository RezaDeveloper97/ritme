'use client';

import { QueryClientProvider } from '@tanstack/react-query';
import { useEffect, useState, type ReactNode } from 'react';

import { instructorKeys } from '@/entities/instructor';
import { createQueryClient, setForbiddenHandler, setUnauthorizedHandler } from '@/shared/api';

/**
 * Client-side app init: one QueryClient, the global 401 → /login redirect and
 * the 403 instructor_* → re-read /me hook (the gate then routes to /apply or
 * /pending).
 */
export function Providers({ children }: { children: ReactNode }) {
  const [queryClient] = useState(createQueryClient);

  useEffect(() => {
    setUnauthorizedHandler(() => {
      if (window.location.pathname === '/login') return;
      queryClient.clear();
      const next = encodeURIComponent(window.location.pathname + window.location.search);
      window.location.replace(`/login?next=${next}`);
    });
    setForbiddenHandler(() => {
      void queryClient.invalidateQueries({ queryKey: instructorKeys.me() });
    });
    return () => {
      setUnauthorizedHandler(null);
      setForbiddenHandler(null);
    };
  }, [queryClient]);

  return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
}
