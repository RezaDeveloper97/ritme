import { afterEach, describe, expect, it, vi } from 'vitest';

import { ApiError, apiClient } from '@/shared/api';

import { fetchDailyMessage, isNoDailyMessage } from './queries';

const fail = (status: number) =>
  new ApiError(`Request failed with status code ${status}`, {
    response: { status, data: { success: false, message: 'x' }, headers: {} },
    sentToken: 't',
  });

describe('daily message', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('treats the API 400 (no period on record) as "no message yet"', async () => {
    vi.spyOn(apiClient, 'get').mockRejectedValue(fail(400));
    await expect(fetchDailyMessage()).resolves.toBeNull();
  });

  it('still fails on anything else', async () => {
    vi.spyOn(apiClient, 'get').mockRejectedValue(fail(422));
    await expect(fetchDailyMessage('2026-13-01')).rejects.toBeInstanceOf(ApiError);

    vi.spyOn(apiClient, 'get').mockRejectedValue(fail(500));
    await expect(fetchDailyMessage()).rejects.toBeInstanceOf(ApiError);

    vi.spyOn(apiClient, 'get').mockRejectedValue(new ApiError('offline', { code: 'network', sentToken: 't' }));
    await expect(fetchDailyMessage()).rejects.toBeInstanceOf(ApiError);
  });

  it('only a 400 ApiError counts as empty', () => {
    expect(isNoDailyMessage(fail(400))).toBe(true);
    expect(isNoDailyMessage(fail(401))).toBe(false);
    expect(isNoDailyMessage(new Error('400'))).toBe(false);
    expect(isNoDailyMessage(null)).toBe(false);
  });
});
