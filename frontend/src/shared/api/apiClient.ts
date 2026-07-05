import axios, { AxiosError } from 'axios';

import { getAuthToken, clearAuthToken } from '@/shared/session';
import { env } from '@/shared/config';

/**
 * The single shared HTTP client for the whole app (CLAUDE.md §2). Slices import
 * `apiClient` from `@/shared/api`; they never create their own axios instances.
 *
 * Privacy (§11): never put health data — cycle dates, symptoms, pregnancy
 * status — into URLs, query strings, headers, or logs.
 */
export const apiClient = axios.create({
  baseURL: env.apiBaseUrl,
  timeout: 15_000,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Attach the JWT bearer token from the session store (§8.1). Centralized here
// so auth is never re-implemented per slice.
apiClient.interceptors.request.use((config) => {
  const token = getAuthToken();
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

// On 401 the token is stale/revoked — drop it so the app falls back to the
// login flow. Route redirection is handled by the middleware/UI, not here.
apiClient.interceptors.response.use(
  (response) => response,
  (error: AxiosError) => {
    if (error.response?.status === 401) {
      clearAuthToken();
    }
    return Promise.reject(error);
  },
);
