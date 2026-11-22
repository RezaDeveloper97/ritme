import 'fake-indexeddb/auto';

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

const session = vi.hoisted(() => ({ token: null as string | null }));
vi.mock('@/shared/session/token', () => ({ getAuthToken: () => session.token }));

const jwtFor = (sub: string) => `h.${btoa(JSON.stringify({ sub })).replace(/=+$/, '')}.s`;

import {
  CHUNK_BYTES,
  MAX_DOWNLOAD_BYTES,
  OfflineError,
  assertRoom,
  deleteOffline,
  getOffline,
  listOffline,
  offlineUsage,
  purgeOffline,
  readOfflineBlob,
  resetOfflineForTests,
  saveOffline,
  subscribeOffline,
  wipeOffline,
  type OfflineMeta,
} from './offline-media';

const meta = (lessonId: number, courseId = 1): OfflineMeta => ({ lessonId, courseId, courseTitle: 'C', title: `L${lessonId}`, kind: 'video' });

function bytes(n: number, seed = 7): Uint8Array {
  const out = new Uint8Array(n);
  for (let i = 0; i < n; i++) out[i] = (i * 31 + seed) % 251;
  return out;
}

/** Byte compare (toEqual on megabytes is very slow). */
function same(a: Uint8Array, b: Uint8Array): boolean {
  if (a.length !== b.length) return false;
  for (let i = 0; i < a.length; i++) if (a[i] !== b[i]) return false;
  return true;
}

/** A response whose body arrives in uneven pieces (like the network). */
function response(data: Uint8Array, { length = true, piece = 1_000_003, failAt = -1 } = {}): Response {
  let at = 0;
  const body = new ReadableStream<Uint8Array>({
    pull(controller) {
      if (failAt >= 0 && at >= failAt) {
        controller.error(new TypeError('network'));
        return;
      }
      if (at >= data.length) {
        controller.close();
        return;
      }
      controller.enqueue(data.slice(at, at + piece));
      at += piece;
    },
  });
  const headers: Record<string, string> = { 'content-type': 'video/mp4' };
  if (length) headers['content-length'] = String(data.length);
  return new Response(body, { status: 200, headers });
}

async function rawChunk(key: string): Promise<{ iv: Uint8Array; data: ArrayBuffer } | undefined> {
  const db = await new Promise<IDBDatabase>((resolve, reject) => {
    const r = indexedDB.open('ritme-offline-media');
    r.onsuccess = () => resolve(r.result);
    r.onerror = () => reject(r.error);
  });
  const row = await new Promise<{ iv: Uint8Array; data: ArrayBuffer } | undefined>((resolve) => {
    const req = db.transaction('chunks').objectStore('chunks').get(key);
    req.onsuccess = () => resolve(req.result);
  });
  db.close();
  return row;
}

beforeEach(() => {
  session.token = jwtFor('1');
  resetOfflineForTests();
});

afterEach(async () => {
  for (const i of await listOffline()) await deleteOffline(i.lessonId);
  await purgeOffline(() => true);
  vi.unstubAllGlobals();
});

describe('offline media store', () => {
  it('round-trips a multi-chunk file encrypted at rest', async () => {
    const data = bytes(CHUNK_BYTES * 2 + 12345);
    const progress: number[] = [];
    const item = await saveOffline(meta(10), response(data), { onProgress: (r) => progress.push(r) });
    expect(item).toMatchObject({ lessonId: 10, size: data.length, chunks: 3, mime: 'video/mp4', complete: true });
    expect(progress.at(-1)).toBe(data.length);

    // At rest the chunk is ciphertext (+16-byte GCM tag), not the plain bytes.
    const row = await rawChunk('10:0');
    expect(row?.data.byteLength).toBe(CHUNK_BYTES + 16);
    expect(same(new Uint8Array(row!.data).slice(0, 64), data.slice(0, 64))).toBe(false);

    const blob = await readOfflineBlob(10);
    expect(blob?.type).toBe('video/mp4');
    expect(same(new Uint8Array(await blob!.arrayBuffer()), data)).toBe(true);
  }, 30_000);

  it('keeps a non-extractable key', async () => {
    await saveOffline(meta(11), response(bytes(100)));
    const db = await new Promise<IDBDatabase>((resolve) => {
      const r = indexedDB.open('ritme-offline-media');
      r.onsuccess = () => resolve(r.result);
    });
    const key = await new Promise<CryptoKey>((resolve) => {
      const req = db.transaction('keys').objectStore('keys').get('master');
      req.onsuccess = () => resolve(req.result);
    });
    db.close();
    expect(key.extractable).toBe(false);
    await expect(crypto.subtle.exportKey('raw', key)).rejects.toBeTruthy();
  });

  it('detects tampering (chunks bound to lesson + index)', async () => {
    await saveOffline(meta(12), response(bytes(CHUNK_BYTES + 10)));
    const db = await new Promise<IDBDatabase>((resolve) => {
      const r = indexedDB.open('ritme-offline-media');
      r.onsuccess = () => resolve(r.result);
    });
    // Swap chunk 0 and 1: GCM additional data no longer matches.
    const store = db.transaction('chunks', 'readwrite').objectStore('chunks');
    const a = await new Promise<unknown>((r) => (store.get('12:0').onsuccess = (e) => r((e.target as IDBRequest).result)));
    const b = await new Promise<unknown>((r) => (store.get('12:1').onsuccess = (e) => r((e.target as IDBRequest).result)));
    store.put(b, '12:0');
    store.put(a, '12:1');
    await new Promise((r) => (store.transaction.oncomplete = r));
    db.close();
    await expect(readOfflineBlob(12)).rejects.toMatchObject({ code: 'corrupt' });
  });

  it('works without content-length and lists newest first', async () => {
    await saveOffline(meta(1), response(bytes(10), { length: false }));
    await new Promise((r) => setTimeout(r, 5));
    await saveOffline(meta(2), response(bytes(20)));
    expect((await listOffline()).map((i) => i.lessonId)).toEqual([2, 1]);
    expect((await offlineUsage()).used).toBe(30);
  });

  it('leaves nothing behind when the network fails mid-way', async () => {
    await expect(saveOffline(meta(20), response(bytes(CHUNK_BYTES * 2), { failAt: CHUNK_BYTES + 1 }))).rejects.toMatchObject({ code: 'network' });
    expect(await getOffline(20)).toBeNull();
    expect(await rawChunk('20:0')).toBeUndefined();
  });

  it('aborts on signal', async () => {
    const ctrl = new AbortController();
    ctrl.abort();
    await expect(saveOffline(meta(21), response(bytes(100)), { signal: ctrl.signal })).rejects.toBeInstanceOf(OfflineError);
    expect(await getOffline(21)).toBeNull();
  });

  it('rejects oversize and over-quota files before writing', async () => {
    await expect(assertRoom(MAX_DOWNLOAD_BYTES + 1)).rejects.toMatchObject({ code: 'tooLarge' });
    vi.stubGlobal('navigator', { storage: { estimate: async () => ({ quota: 1000, usage: 900 }) } });
    await expect(assertRoom(500)).rejects.toMatchObject({ code: 'quota' });
    await expect(saveOffline(meta(22), response(bytes(500)))).rejects.toMatchObject({ code: 'quota' });
    expect(await getOffline(22)).toBeNull();
    await expect(assertRoom(50)).resolves.toBeUndefined();
  });

  it('purges what access no longer covers and notifies', async () => {
    await saveOffline(meta(30, 5), response(bytes(10)));
    await saveOffline(meta(31, 6), response(bytes(10)));
    const seen = vi.fn();
    const off = subscribeOffline(seen);
    expect(await purgeOffline((i) => i.courseId === 5)).toBe(1);
    off();
    expect(seen).toHaveBeenCalled();
    expect((await listOffline()).map((i) => i.lessonId)).toEqual([30]);
    expect(await rawChunk('31:0')).toBeUndefined();
  });

  it('replaces an older copy of the same lesson', async () => {
    await saveOffline(meta(40), response(bytes(CHUNK_BYTES + 5)));
    await saveOffline(meta(40), response(bytes(3, 1)));
    expect(await getOffline(40)).toMatchObject({ size: 3, chunks: 1 });
    expect(await rawChunk('40:1')).toBeUndefined();
  });

  it('never returns an item tagged with another user (list, get, read)', async () => {
    await saveOffline(meta(50), response(bytes(10)));
    session.token = jwtFor('2');
    expect(await listOffline()).toEqual([]);
    expect(await getOffline(50)).toBeNull();
    expect(await readOfflineBlob(50)).toBeNull();
    session.token = jwtFor('1');
    expect((await listOffline()).map((i) => i.lessonId)).toEqual([50]);
  });

  it('returns nothing without a session, and refuses to save', async () => {
    await saveOffline(meta(51), response(bytes(10)));
    session.token = null;
    expect(await listOffline()).toEqual([]);
    await expect(saveOffline(meta(52), response(bytes(10)))).rejects.toMatchObject({ code: 'unsupported' });
  });

  it('wipeOffline empties the store and the key; the next user starts clean', async () => {
    await saveOffline(meta(60), response(bytes(10)));
    const postMessage = vi.fn();
    vi.stubGlobal('navigator', { serviceWorker: { controller: { postMessage } } });
    await wipeOffline();
    expect(postMessage).toHaveBeenCalledWith({ type: 'CLEAR_OFFLINE_PAGES' });
    expect((await indexedDB.databases()).some((d) => d.name === 'ritme-offline-media')).toBe(false);
    vi.unstubAllGlobals();
    expect(await listOffline()).toEqual([]);
    // Same user id logging in again still sees nothing: the data is really gone.
    expect(await getOffline(60)).toBeNull();
  });
});
