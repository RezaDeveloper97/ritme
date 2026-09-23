import { QueryClient } from '@tanstack/react-query';

import { isApiError } from './errors';

/** One QueryClient per browser tab (created in app/providers). */
export function createQueryClient(): QueryClient {
  return new QueryClient({
    defaultOptions: {
      queries: {
        staleTime: 30_000,
        refetchOnWindowFocus: false,
        // 4xx answers are final; retry only network errors and 5xx, once.
        retry: (count, error) =>
          count < 1 && (!isApiError(error) || error.status === 0 || error.status >= 500),
      },
      mutations: { retry: false },
    },
  });
}
