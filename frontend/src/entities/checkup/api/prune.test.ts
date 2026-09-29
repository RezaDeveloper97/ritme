import { describe, expect, it, vi } from 'vitest';

import { createLocalFileStore, createMemoryBackend } from '@/shared/lib/local-files';

import type { CheckupRecord, CheckupRecordPage } from '../model/types';
import { pruneCheckupAttachments } from './prune';

const pdf = () => new Blob(['x'], { type: 'application/pdf' });
const rec = (id: number) => ({ id }) as CheckupRecord;
const page = (ids: number[], p = 1, last = 1): CheckupRecordPage => ({
  records: ids.map(rec),
  page: p,
  lastPage: last,
  total: ids.length,
});

function store() {
  return createLocalFileStore({ namespace: 'checkup-reports', maxFileBytes: 10, backend: createMemoryBackend() });
}

describe('pruneCheckupAttachments', () => {
  it('deletes files whose record is gone (custom type deleted → records cascaded)', async () => {
    const s = store();
    await s.put(1, pdf());
    await s.put(2, pdf());
    await s.put(3, pdf());
    const fetchPage = vi.fn(async (p: number) => (p === 1 ? page([1], 1, 2) : page([3], 2, 2)));

    expect(await pruneCheckupAttachments(s, fetchPage)).toBe(1);
    expect((await s.list()).map((m) => m.key).sort()).toEqual(['1', '3']);
    expect(fetchPage).toHaveBeenCalledTimes(2);
  });

  it('does not touch the network when there are no files', async () => {
    const fetchPage = vi.fn();
    expect(await pruneCheckupAttachments(store(), fetchPage)).toBe(0);
    expect(fetchPage).not.toHaveBeenCalled();
  });

  it('keeps every file when a page fails', async () => {
    const s = store();
    await s.put(1, pdf());
    await s.put(2, pdf());
    const fetchPage = vi.fn(async (p: number) => {
      if (p === 2) throw new Error('offline');
      return page([], 1, 2);
    });

    await expect(pruneCheckupAttachments(s, fetchPage)).rejects.toThrow();
    expect(await s.list()).toHaveLength(2);
  });
});
