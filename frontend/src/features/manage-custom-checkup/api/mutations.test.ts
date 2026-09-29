import { QueryClient } from '@tanstack/react-query';
import { describe, expect, it, vi } from 'vitest';

import { checkupKeys } from '@/entities/checkup';

import { afterCustomCheckupDeleted } from './mutations';

describe('afterCustomCheckupDeleted', () => {
  it('drops the detail, invalidates records and prunes on-device reports', async () => {
    const queryClient = new QueryClient();
    queryClient.setQueryData(checkupKeys.detail(7), { id: 7 });
    queryClient.setQueryData(checkupKeys.detail(8), { id: 8 });
    const invalidate = vi.spyOn(queryClient, 'invalidateQueries');
    const prune = vi.fn(() => Promise.resolve(2));

    await afterCustomCheckupDeleted(queryClient, 7, prune);

    expect(prune).toHaveBeenCalledTimes(1);
    expect(queryClient.getQueryData(checkupKeys.detail(7))).toBeUndefined();
    expect(queryClient.getQueryData(checkupKeys.detail(8))).toEqual({ id: 8 });
    expect(invalidate).toHaveBeenCalledWith({ queryKey: checkupKeys.recordsAll() });
    expect(invalidate).toHaveBeenCalledWith({ queryKey: checkupKeys.home() });
  });

  it('never rejects when the sweep fails (offline, 5xx)', async () => {
    const queryClient = new QueryClient();
    const prune = vi.fn(() => Promise.reject(new Error('offline')));
    await expect(afterCustomCheckupDeleted(queryClient, 3, prune)).resolves.toBeUndefined();
    expect(prune).toHaveBeenCalledTimes(1);
  });
});
