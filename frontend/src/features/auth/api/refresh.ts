'use client';

import { type ApiEnvelope, apiClient, getApiErrorStatus } from '@/shared/api';
import { getAuthToken, setAuthToken, tokenExpiresAt } from '@/shared/session';

import { createSessionRefresher, type RefreshOutcome } from '../lib/refresh';

async function requestRefresh(): Promise<RefreshOutcome> {
  try {
    const { data } = await apiClient.post<
      ApiEnvelope<{ refreshed?: boolean; access_token?: string }>
    >('/auth/refresh-session');
    const token = data.data?.access_token;
    return data.data?.refreshed && token ? { kind: 'refreshed', token } : { kind: 'unchanged' };
  } catch (error) {
    const status = getApiErrorStatus(error);
    if (status === 404 || status === 405) return { kind: 'unavailable' };
    throw error;
  }
}

/** The app-wide single-flight refresher (one per page, like the token itself). */
export const sessionRefresher = createSessionRefresher({
  getToken: getAuthToken,
  setToken: setAuthToken,
  expiresAt: tokenExpiresAt,
  request: requestRefresh,
  now: () => Date.now(),
});
