import { afterEach, describe, expect, it, vi } from 'vitest';

async function load() {
  vi.resetModules();
  const cleanup = await import('./cleanup');
  const outbox = await import('@/shared/lib/outbox');
  return { ...cleanup, ...outbox };
}

describe('session cleanups', () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('runs every registered wipe even when one throws', async () => {
    const { onSessionEnd, runSessionCleanups } = await load();
    const first = vi.fn(() => {
      throw new Error('storage blocked');
    });
    const second = vi.fn();
    onSessionEnd(first);
    onSessionEnd(second);

    runSessionCleanups();

    expect(first).toHaveBeenCalledOnce();
    expect(second).toHaveBeenCalledOnce();
  });

  it('an unregistered wipe no longer runs', async () => {
    const { onSessionEnd, runSessionCleanups } = await load();
    const wipe = vi.fn();
    const off = onSessionEnd(wipe);
    off();

    runSessionCleanups();

    expect(wipe).not.toHaveBeenCalled();
  });

  it('drops the offline outbox so queued writes never replay into the next account', async () => {
    const { getOutbox, runSessionCleanups } = await load();
    const outbox = getOutbox();
    await outbox.enqueue({ key: 'pregnancy-day:2026-09-28', method: 'put', url: '/x', body: { water: 4 } });
    expect(await outbox.pending()).toHaveLength(1);

    runSessionCleanups();

    await vi.waitFor(async () => expect(await outbox.pending()).toHaveLength(0));
  });

  it('wipes on-device files (checkup reports) of every namespace (audit M3-M7 #1)', async () => {
    const { runSessionCleanups } = await load();
    const files = await import('@/shared/lib/local-files');
    const backend = files.createMemoryBackend();
    const reports = files.createLocalFileStore({ namespace: 'checkup-reports', maxFileBytes: 10, backend });
    const other = files.createLocalFileStore({ namespace: 'other', maxFileBytes: 10, backend });
    await reports.put(7, new Blob(['x'], { type: 'application/pdf' }));
    await other.put(1, new Blob(['y']));

    runSessionCleanups();

    await vi.waitFor(async () => {
      expect(await reports.list()).toEqual([]);
      expect(await other.list()).toEqual([]);
    });
  });

  it('drops a pending navigation prefill', async () => {
    const { runSessionCleanups } = await load();
    const { stashHandoff, readHandoff } = await import('@/shared/lib/handoff');
    const id = stashHandoff({ title: 'NT scan' });

    runSessionCleanups();

    expect(readHandoff(id)).toBeNull();
  });
});
