import { MutationObserver, QueryClient } from '@tanstack/react-query';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const apiDelete = vi.fn();
const clearAuthToken = vi.fn();

vi.mock('@/shared/api', () => ({ apiClient: { delete: apiDelete } }));
vi.mock('@/shared/session', () => ({ clearAuthToken }));

const { deleteAccountMutationOptions } = await import('./mutations');

function run(queryClient: QueryClient) {
  return new MutationObserver(queryClient, deleteAccountMutationOptions(queryClient)).mutate();
}

describe('deleteAccountMutationOptions', () => {
  beforeEach(() => {
    apiDelete.mockReset();
    clearAuthToken.mockReset();
  });

  it('deletes the account, then ends the session and drops every cached query', async () => {
    apiDelete.mockResolvedValue({ data: { success: true } });
    const queryClient = new QueryClient();
    queryClient.setQueryData(['profile'], { name: 'x' });

    await run(queryClient);

    expect(apiDelete).toHaveBeenCalledWith('/account');
    expect(clearAuthToken).toHaveBeenCalledTimes(1);
    expect(queryClient.getQueryData(['profile'])).toBeUndefined();
  });

  it('keeps the session when the server refuses the deletion', async () => {
    apiDelete.mockRejectedValue(new Error('500'));
    const queryClient = new QueryClient();
    queryClient.setQueryData(['profile'], { name: 'x' });

    await expect(run(queryClient)).rejects.toThrow('500');

    expect(clearAuthToken).not.toHaveBeenCalled();
    expect(queryClient.getQueryData(['profile'])).toEqual({ name: 'x' });
  });
});
