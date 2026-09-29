import { describe, expect, it } from 'vitest';

import {
  clearAllLocalFiles,
  createIndexedDbBackend,
  createLocalFileStore,
  createMemoryBackend,
  LocalFilesError,
  matchesAccept,
  type LocalFilesBackend,
} from './store';

const pdf = (bytes: number) => new Blob([new Uint8Array(bytes)], { type: 'application/pdf' });
const jpeg = (bytes: number) => new Blob([new Uint8Array(bytes)], { type: 'image/jpeg' });

function makeStore(overrides: Partial<Parameters<typeof createLocalFileStore>[0]> = {}) {
  let clock = 1_000;
  return createLocalFileStore({
    namespace: 'reports',
    maxFileBytes: 100,
    accept: ['image/jpeg', 'image/heic', 'application/pdf'],
    backend: createMemoryBackend(),
    now: () => clock++,
    ...overrides,
  });
}

async function codeOf(promise: Promise<unknown>): Promise<string | null> {
  try {
    await promise;
    return null;
  } catch (error) {
    return error instanceof LocalFilesError ? error.code : 'not-a-LocalFilesError';
  }
}

describe('createLocalFileStore', () => {
  it('puts, gets, checks and deletes a file keyed by record id', async () => {
    const store = makeStore();
    const meta = await store.put(42, pdf(10), 'report.pdf');
    expect(meta).toEqual({ key: '42', name: 'report.pdf', type: 'application/pdf', size: 10, savedAt: 1000 });

    const file = await store.get('42');
    expect(file?.size).toBe(10);
    expect(file?.blob.type).toBe('application/pdf');
    expect(await store.has(42)).toBe(true);

    await store.delete(42);
    expect(await store.get(42)).toBeNull();
    expect(await store.has(42)).toBe(false);
    // Deleting what isn't there is fine.
    await expect(store.delete(42)).resolves.toBeUndefined();
  });

  it('replaces a file stored under the same key', async () => {
    const store = makeStore();
    await store.put(1, pdf(10));
    await store.put(1, jpeg(20), 'photo.jpg');
    const list = await store.list();
    expect(list).toHaveLength(1);
    expect(list[0]).toMatchObject({ key: '1', type: 'image/jpeg', size: 20, name: 'photo.jpg' });
  });

  it('lists only its own namespace, newest first', async () => {
    const backend = createMemoryBackend();
    const a = makeStore({ backend });
    const b = makeStore({ backend, namespace: 'other' });
    await a.put(1, pdf(1));
    await a.put(2, pdf(1));
    await b.put(1, pdf(1));
    expect((await a.list()).map((m) => m.key)).toEqual(['2', '1']);
    expect((await b.list()).map((m) => m.key)).toEqual(['1']);
    // Same key, different namespace: independent.
    await a.delete(1);
    expect(await b.has(1)).toBe(true);
  });

  it('enforces the per-file cap', async () => {
    const store = makeStore();
    expect(await codeOf(store.put(1, pdf(101)))).toBe('too_large');
    expect(await codeOf(store.put(1, pdf(100)))).toBeNull();
  });

  it('enforces the total cap without double-counting a replaced file', async () => {
    const store = makeStore({ maxTotalBytes: 150 });
    await store.put(1, pdf(100));
    expect(await codeOf(store.put(2, pdf(60)))).toBe('store_full');
    // Replacing key 1 frees its old size first.
    expect(await codeOf(store.put(1, pdf(100)))).toBeNull();
    expect(await codeOf(store.put(2, pdf(50)))).toBeNull();
  });

  it('rejects types outside the accept list', async () => {
    const store = makeStore();
    expect(await codeOf(store.put(1, new Blob(['x'], { type: 'text/html' })))).toBe('unsupported_type');
    expect(await codeOf(store.put(1, new Blob(['x'])))).toBe('unsupported_type');
    expect(await codeOf(store.put(1, new Blob(['x'], { type: 'image/heic' })))).toBeNull();
    // Script-capable image types never get in (audit M3-M7 #2).
    expect(await codeOf(store.put(1, new Blob(['<svg/>'], { type: 'image/svg+xml' })))).toBe('unsupported_type');
    expect(await codeOf(store.put(1, new Blob(['x'], { type: 'IMAGE/JPEG; q=1' })))).toBeNull();
    // No accept list = anything goes.
    const open = makeStore({ accept: undefined });
    expect(await codeOf(open.put(1, new Blob(['x'])))).toBeNull();
  });

  it('check() runs the synchronous validation without writing', async () => {
    const store = makeStore();
    expect(store.check(pdf(101))).toBe('too_large');
    expect(store.check(new Blob(['x'], { type: 'text/plain' }))).toBe('unsupported_type');
    expect(store.check(jpeg(5))).toBeNull();
    expect(await store.list()).toEqual([]);
  });

  it('exposes its limits for the file input', () => {
    const store = makeStore({ maxTotalBytes: 500 });
    expect(store.limits).toEqual({
      maxFileBytes: 100,
      maxTotalBytes: 500,
      accept: ['image/jpeg', 'image/heic', 'application/pdf'],
    });
  });

  it('matchesAccept is exact and ignores wildcards', () => {
    expect(matchesAccept('image/png', ['image/*'])).toBe(false);
    expect(matchesAccept('image/svg+xml', ['image/*', 'image/png'])).toBe(false);
    expect(matchesAccept('image/png', ['image/png'])).toBe(true);
    expect(matchesAccept('', ['image/png'])).toBe(false);
  });

  it('clear() empties only its own namespace; deleteMany() removes the given keys', async () => {
    const backend = createMemoryBackend();
    const a = makeStore({ backend });
    const b = makeStore({ backend, namespace: 'other' });
    await a.put(1, pdf(1));
    await a.put(2, pdf(1));
    await a.put(3, pdf(1));
    await b.put(1, pdf(1));

    await a.deleteMany([1, '3', 99]);
    expect((await a.list()).map((m) => m.key)).toEqual(['2']);

    await a.clear();
    expect(await a.list()).toEqual([]);
    expect(await b.has(1)).toBe(true);
  });

  it('clearAllLocalFiles() empties every store (the session-end wipe)', async () => {
    const a = makeStore();
    const b = makeStore({ namespace: 'other' });
    await a.put(1, pdf(1));
    await b.put(2, jpeg(1));

    await clearAllLocalFiles();

    expect(await a.list()).toEqual([]);
    expect(await b.list()).toEqual([]);
  });

  describe('when storage is unavailable', () => {
    it('reads resolve empty and put rejects with "unavailable" (no backend)', async () => {
      const store = makeStore({ backend: null });
      expect(await store.isAvailable()).toBe(false);
      expect(await store.get(1)).toBeNull();
      expect(await store.has(1)).toBe(false);
      expect(await store.list()).toEqual([]);
      await expect(store.delete(1)).resolves.toBeUndefined();
      expect(await codeOf(store.put(1, pdf(1)))).toBe('unavailable');
    });

    it('a backend that throws degrades the same way', async () => {
      const broken: LocalFilesBackend = {
        get: () => Promise.reject(new Error('blocked')),
        put: () => Promise.reject(new Error('blocked')),
        delete: () => Promise.reject(new Error('blocked')),
        list: () => Promise.reject(new Error('blocked')),
      };
      const store = makeStore({ backend: broken });
      expect(await store.isAvailable()).toBe(false);
      expect(await store.get(1)).toBeNull();
      expect(await store.list()).toEqual([]);
      await expect(store.delete(1)).resolves.toBeUndefined();
      expect(await codeOf(store.put(1, pdf(1)))).toBe('failed');
      expect(await codeOf(makeStore({ backend: broken, maxTotalBytes: 10 }).put(1, pdf(1)))).toBe(
        'unavailable',
      );
    });

    it('maps a browser quota error to "quota"', async () => {
      const full = createMemoryBackend();
      full.put = () => Promise.reject(Object.assign(new Error('full'), { name: 'QuotaExceededError' }));
      expect(await codeOf(makeStore({ backend: full }).put(1, pdf(1)))).toBe('quota');
    });

    it('the default IndexedDB backend is null where IndexedDB does not exist (Node / SSR)', async () => {
      expect(typeof indexedDB).toBe('undefined');
      expect(createIndexedDbBackend()).toBeNull();
      const store = createLocalFileStore({ namespace: 'reports', maxFileBytes: 10 });
      expect(await store.isAvailable()).toBe(false);
      expect(await codeOf(store.put(1, pdf(1)))).toBe('unavailable');
    });
  });
});
