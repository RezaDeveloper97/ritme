import { MutationObserver, QueryClient } from '@tanstack/react-query';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { cycleKeys, type CycleTodayUi } from '@/entities/cycle';
import { ApiError, apiClient } from '@/shared/api';

import { canOfferSkip } from '../model/visibility';

import { skipErrorKind, skipMutationOptions, undoMutationOptions } from './mutations';

function failed(status: number, data: unknown): ApiError {
  return new ApiError(`Request failed with status code ${status}`, { response: { status, data, headers: {} }, sentToken: null });
}
const noQueryRetry = () => new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });

afterEach(() => vi.restoreAllMocks());

describe('skipErrorKind', () => {
  it('maps each 422 code, a network failure and the rest', () => {
    for (const code of ['not_expected_yet', 'period_started', 'prediction_mismatch']) {
      expect(skipErrorKind(failed(422, { success: false, message: 'x', code }))).toBe(code);
    }
    expect(skipErrorKind(failed(422, { success: false, message: 'x' }))).toBe('other');
    expect(skipErrorKind(failed(500, { code: 'period_started' }))).toBe('other');
    expect(skipErrorKind(new ApiError('Network error', { code: 'network', sentToken: null }))).toBe('network');
    expect(skipErrorKind(new Error('boom'))).toBe('other');
  });
});

describe('skip mutations', () => {
  it('POSTs the expected date and refetches the cycle on success', async () => {
    const qc = noQueryRetry();
    const post = vi.spyOn(apiClient, 'post').mockResolvedValue({ status: 201, headers: {}, data: { success: true, data: { expected_start: '2026-10-20' } } });
    const invalidate = vi.spyOn(qc, 'invalidateQueries');
    const out = await new MutationObserver(qc, skipMutationOptions(qc)).mutate('2026-10-20');
    expect(out).toBe('2026-10-20');
    expect(post).toHaveBeenCalledWith('/cycle/skips', { expected_start: '2026-10-20' });
    expect(invalidate).toHaveBeenCalledWith({ queryKey: [...cycleKeys.all] });
  });

  it.each([
    ['not_expected_yet', false],
    ['period_started', true],
    ['prediction_mismatch', true],
  ])('422 %s rejects (refetch=%s)', async (code, refetch) => {
    const qc = noQueryRetry();
    vi.spyOn(apiClient, 'post').mockRejectedValue(failed(422, { success: false, message: 'm', code }));
    const invalidate = vi.spyOn(qc, 'invalidateQueries');
    await expect(new MutationObserver(qc, skipMutationOptions(qc)).mutate('2026-10-20')).rejects.toBeInstanceOf(ApiError);
    expect(invalidate.mock.calls.length > 0).toBe(refetch);
  });

  it('a network error rejects without refetching', async () => {
    const qc = noQueryRetry();
    vi.spyOn(apiClient, 'post').mockRejectedValue(new ApiError('Network error', { code: 'network', sentToken: null }));
    const invalidate = vi.spyOn(qc, 'invalidateQueries');
    await expect(new MutationObserver(qc, skipMutationOptions(qc)).mutate('2026-10-20')).rejects.toBeInstanceOf(ApiError);
    expect(invalidate).not.toHaveBeenCalled();
  });

  it('undo DELETEs by date and treats 404 as already undone', async () => {
    const qc = noQueryRetry();
    const del = vi.spyOn(apiClient, 'delete').mockResolvedValueOnce({ status: 204, headers: {}, data: undefined });
    await new MutationObserver(qc, undoMutationOptions(qc)).mutate('2026-10-20');
    expect(del).toHaveBeenCalledWith('/cycle/skips/2026-10-20');
    del.mockRejectedValueOnce(failed(404, { success: false, message: 'none' }));
    await expect(new MutationObserver(qc, undoMutationOptions(qc)).mutate('2026-10-20')).resolves.toBeUndefined();
    del.mockRejectedValueOnce(failed(500, {}));
    await expect(new MutationObserver(qc, undoMutationOptions(qc)).mutate('2026-10-20')).rejects.toBeInstanceOf(ApiError);
  });
});

describe('overdue → skip → banner gone → undo → overdue again', () => {
  it('follows the server state through a real query client', async () => {
    const qc = noQueryRetry();
    let skipped = false;
    // The fake server: a skip moves the prediction, so the engine drops the overdue banner and the start prompt.
    const todayUi = (): CycleTodayUi =>
      ({ ringAction: skipped ? 'edit' : 'start', banner: skipped ? null : 'overdue', showPregnancyTestButton: false }) as CycleTodayUi;
    vi.spyOn(apiClient, 'post').mockImplementation(async () => {
      skipped = true;
      return { status: 201, headers: {}, data: { success: true, data: { expected_start: '2026-10-20' } } };
    });
    vi.spyOn(apiClient, 'delete').mockImplementation(async () => {
      skipped = false;
      return { status: 204, headers: {}, data: undefined };
    });
    const load = () => qc.fetchQuery({ queryKey: [...cycleKeys.all, 'today'], queryFn: async () => todayUi(), staleTime: 0 });
    const offered = (ui: CycleTodayUi) =>
      canOfferSkip({ isToday: true, todayUi: ui, predictedNext: '2026-10-20', todayIso: '2026-10-24', inPeriod: false });

    expect(offered(await load())).toBe(true);
    await new MutationObserver(qc, skipMutationOptions(qc)).mutate('2026-10-20');
    const after = await load();
    expect(after.banner).toBeNull();
    expect(offered(after)).toBe(false);
    await new MutationObserver(qc, undoMutationOptions(qc)).mutate('2026-10-20');
    const back = await load();
    expect(back.banner).toBe('overdue');
    expect(offered(back)).toBe(true);
  });
});
