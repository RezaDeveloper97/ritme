/*
 * Offline lesson downloads (B-N8-04), encrypted at rest by the browser.
 *
 * - The media is fetched once from a fresh signed URL and split into 4 MiB
 *   chunks; each chunk is sealed with AES-GCM (random 96-bit IV, the lesson id
 *   and chunk index as additional data, so chunks can't be swapped).
 * - The AES key is generated in the browser as NON-extractable and kept in
 *   IndexedDB: script can use it but never read its bytes, and the files are
 *   useless outside this origin.
 * - Playback only inside the app: `readOfflineBlob` decrypts into a Blob the
 *   page turns into a `blob:` URL. No file URL is ever exposed.
 * - Nothing here is in Cache Storage, so a service-worker update can't drop it.
 *
 * Framework-free (tested with fake-indexeddb); React glue is in useOfflineMedia.ts.
 */

export type OfflineKind = 'video' | 'audio' | 'pdf';

export interface OfflineMeta {
  lessonId: number;
  courseId: number;
  courseTitle: string;
  title: string;
  kind: OfflineKind;
}

export interface OfflineItem extends OfflineMeta {
  mime: string;
  /** Plain size in bytes. */
  size: number;
  chunks: number;
  savedAt: string;
  complete: boolean;
}

export type OfflineErrorCode = 'unsupported' | 'tooLarge' | 'quota' | 'network' | 'aborted' | 'corrupt';

export class OfflineError extends Error {
  constructor(readonly code: OfflineErrorCode) {
    super(`offline media: ${code}`);
    this.name = 'OfflineError';
  }
}

export const CHUNK_BYTES = 4 * 1024 * 1024;
/** One lesson may take at most this much (a 2 GB upload stays online-only). */
export const MAX_DOWNLOAD_BYTES = 1024 * 1024 * 1024;
/** Keep this share of the free quota for the rest of the app. */
const QUOTA_HEADROOM = 0.9;

const DB_NAME = 'ritme-offline-media';
const DB_VERSION = 1;
const KEYS = 'keys';
const ITEMS = 'items';
const CHUNKS = 'chunks';
const MASTER = 'master';

interface ChunkRow {
  iv: Uint8Array<ArrayBuffer>;
  data: ArrayBuffer;
}

// ── change notifications (screens re-read the list) ────────────────────────
type Listener = () => void;
const listeners = new Set<Listener>();
export function subscribeOffline(listener: Listener): () => void {
  listeners.add(listener);
  return () => listeners.delete(listener);
}
function notify() {
  listeners.forEach((l) => l());
}

export function offlineSupported(): boolean {
  return typeof indexedDB !== 'undefined' && typeof crypto !== 'undefined' && !!crypto.subtle;
}

// ── IndexedDB helpers ───────────────────────────────────────────────────────
let dbPromise: Promise<IDBDatabase> | null = null;

function openDb(): Promise<IDBDatabase> {
  if (!offlineSupported()) return Promise.reject(new OfflineError('unsupported'));
  dbPromise ??= new Promise<IDBDatabase>((resolve, reject) => {
    const req = indexedDB.open(DB_NAME, DB_VERSION);
    req.onupgradeneeded = () => {
      const db = req.result;
      if (!db.objectStoreNames.contains(KEYS)) db.createObjectStore(KEYS);
      if (!db.objectStoreNames.contains(ITEMS)) db.createObjectStore(ITEMS, { keyPath: 'lessonId' });
      if (!db.objectStoreNames.contains(CHUNKS)) db.createObjectStore(CHUNKS);
    };
    req.onsuccess = () => resolve(req.result);
    req.onerror = () => {
      dbPromise = null;
      reject(new OfflineError('unsupported'));
    };
  });
  return dbPromise;
}

/** Test hook: forget the cached connection (and key) after deleting the DB. */
export function resetOfflineForTests() {
  dbPromise = null;
  keyPromise = null;
}

function isQuotaError(e: unknown): boolean {
  return e instanceof DOMException && (e.name === 'QuotaExceededError' || e.name === 'NS_ERROR_DOM_QUOTA_REACHED');
}

async function tx<T>(stores: string[], mode: IDBTransactionMode, run: (t: IDBTransaction) => IDBRequest<T> | void): Promise<T> {
  const db = await openDb();
  return new Promise<T>((resolve, reject) => {
    const t = db.transaction(stores, mode);
    const req = run(t);
    let value: T;
    if (req) req.onsuccess = () => (value = req.result);
    t.oncomplete = () => resolve(value);
    t.onabort = t.onerror = () => {
      const err = t.error;
      reject(isQuotaError(err) ? new OfflineError('quota') : err ?? new Error('idb'));
    };
  });
}

const chunkKey = (lessonId: number, index: number) => `${lessonId}:${index}`;

// ── key ─────────────────────────────────────────────────────────────────────
let keyPromise: Promise<CryptoKey> | null = null;

function masterKey(): Promise<CryptoKey> {
  keyPromise ??= (async () => {
    const existing = await tx<CryptoKey | undefined>([KEYS], 'readonly', (t) => t.objectStore(KEYS).get(MASTER));
    if (existing) return existing;
    const key = await crypto.subtle.generateKey({ name: 'AES-GCM', length: 256 }, false, ['encrypt', 'decrypt']);
    await tx([KEYS], 'readwrite', (t) => {
      t.objectStore(KEYS).put(key, MASTER);
    });
    return key;
  })().catch((e) => {
    keyPromise = null;
    throw e;
  });
  return keyPromise;
}

const aad = (lessonId: number, index: number) => new TextEncoder().encode(chunkKey(lessonId, index));

// ── queries ─────────────────────────────────────────────────────────────────
async function allItems(): Promise<OfflineItem[]> {
  return tx<OfflineItem[]>([ITEMS], 'readonly', (t) => t.objectStore(ITEMS).getAll());
}

/** Finished downloads, newest first. */
export async function listOffline(): Promise<OfflineItem[]> {
  const items = await allItems();
  return items.filter((i) => i.complete).sort((a, b) => b.savedAt.localeCompare(a.savedAt));
}

export async function getOffline(lessonId: number): Promise<OfflineItem | null> {
  const item = await tx<OfflineItem | undefined>([ITEMS], 'readonly', (t) => t.objectStore(ITEMS).get(lessonId));
  return item?.complete ? item : null;
}

/** Bytes used by downloads, plus the browser's estimate of the origin quota. */
export async function offlineUsage(): Promise<{ used: number; quota: number | null }> {
  const items = await listOffline();
  const used = items.reduce((sum, i) => sum + i.size, 0);
  const est = await estimate();
  return { used, quota: est?.quota ?? null };
}

async function estimate(): Promise<StorageEstimate | null> {
  try {
    return (await navigator.storage?.estimate?.()) ?? null;
  } catch {
    return null;
  }
}

// ── write ───────────────────────────────────────────────────────────────────
export interface SaveOptions {
  onProgress?: (received: number, total: number | null) => void;
  signal?: AbortSignal;
}

/** Throws `tooLarge` / `quota` before anything is written when the size is known. */
export async function assertRoom(size: number | null): Promise<void> {
  if (size == null) return;
  if (size > MAX_DOWNLOAD_BYTES) throw new OfflineError('tooLarge');
  const est = await estimate();
  if (est?.quota != null && est.usage != null && size > (est.quota - est.usage) * QUOTA_HEADROOM) {
    throw new OfflineError('quota');
  }
}

/**
 * Stores the body of a successful media response (from a fresh signed URL)
 * encrypted. A failed / aborted download leaves nothing behind.
 */
export async function saveOffline(meta: OfflineMeta, response: Response, opts: SaveOptions = {}): Promise<OfflineItem> {
  if (!response.ok || !response.body) throw new OfflineError('network');
  const header = Number(response.headers.get('content-length'));
  const total = Number.isFinite(header) && header > 0 ? header : null;
  await assertRoom(total);
  const key = await masterKey();
  const mime = response.headers.get('content-type')?.split(';')[0]?.trim() || 'application/octet-stream';

  // Remove any older copy / leftover, then mark this one as in progress.
  await deleteOffline(meta.lessonId, { silent: true });
  const item: OfflineItem = { ...meta, mime, size: 0, chunks: 0, savedAt: new Date().toISOString(), complete: false };
  await tx([ITEMS], 'readwrite', (t) => {
    t.objectStore(ITEMS).put(item);
  });

  const reader = response.body.getReader();
  const buf = new Uint8Array(CHUNK_BYTES);
  let fill = 0;
  let received = 0;

  const flush = async () => {
    if (fill === 0) return;
    const index = item.chunks;
    const iv = crypto.getRandomValues(new Uint8Array(12));
    const data = await crypto.subtle.encrypt({ name: 'AES-GCM', iv, additionalData: aad(meta.lessonId, index) }, key, buf.slice(0, fill));
    const row: ChunkRow = { iv, data };
    await tx([CHUNKS], 'readwrite', (t) => {
      t.objectStore(CHUNKS).put(row, chunkKey(meta.lessonId, index));
    });
    item.chunks += 1;
    fill = 0;
  };

  try {
    for (;;) {
      if (opts.signal?.aborted) throw new OfflineError('aborted');
      let part: ReadableStreamReadResult<Uint8Array>;
      try {
        part = await reader.read();
      } catch {
        throw new OfflineError(opts.signal?.aborted ? 'aborted' : 'network');
      }
      if (part.done) break;
      let bytes = part.value;
      received += bytes.byteLength;
      if (received > MAX_DOWNLOAD_BYTES) throw new OfflineError('tooLarge');
      while (bytes.byteLength > 0) {
        const n = Math.min(CHUNK_BYTES - fill, bytes.byteLength);
        buf.set(bytes.subarray(0, n), fill);
        fill += n;
        bytes = bytes.subarray(n);
        if (fill === CHUNK_BYTES) await flush();
      }
      opts.onProgress?.(received, total);
    }
    await flush();
    if (total != null && received !== total) throw new OfflineError('network');
    item.size = received;
    item.complete = true;
    item.savedAt = new Date().toISOString();
    await tx([ITEMS], 'readwrite', (t) => {
      t.objectStore(ITEMS).put(item);
    });
  } catch (e) {
    reader.cancel().catch(() => undefined);
    await deleteOffline(meta.lessonId, { silent: true }).catch(() => undefined);
    if (e instanceof OfflineError) throw e;
    if (isQuotaError(e)) throw new OfflineError('quota');
    throw e;
  }
  notify();
  return item;
}

// ── read ────────────────────────────────────────────────────────────────────
/** Decrypts a finished download into a Blob (for a `blob:` URL), or null. */
export async function readOfflineBlob(lessonId: number): Promise<Blob | null> {
  const item = await getOffline(lessonId);
  if (!item) return null;
  const key = await masterKey();
  const parts: ArrayBuffer[] = [];
  for (let i = 0; i < item.chunks; i++) {
    const row = await tx<ChunkRow | undefined>([CHUNKS], 'readonly', (t) => t.objectStore(CHUNKS).get(chunkKey(lessonId, i)));
    if (!row) throw new OfflineError('corrupt');
    try {
      parts.push(await crypto.subtle.decrypt({ name: 'AES-GCM', iv: row.iv, additionalData: aad(lessonId, i) }, key, row.data));
    } catch {
      throw new OfflineError('corrupt');
    }
  }
  return new Blob(parts, { type: item.mime });
}

// ── delete / purge ──────────────────────────────────────────────────────────
export async function deleteOffline(lessonId: number, opts: { silent?: boolean } = {}): Promise<void> {
  await tx([ITEMS, CHUNKS], 'readwrite', (t) => {
    t.objectStore(ITEMS).delete(lessonId);
    t.objectStore(CHUNKS).delete(IDBKeyRange.bound(`${lessonId}:`, `${lessonId}:￿`));
  });
  if (!opts.silent) notify();
}

/**
 * Drops every download `keep` rejects (access over / course gone) and every
 * unfinished leftover. Returns how many were removed.
 */
export async function purgeOffline(keep: (item: OfflineItem) => boolean): Promise<number> {
  const items = await allItems();
  const drop = items.filter((i) => !i.complete || !keep(i));
  for (const i of drop) await deleteOffline(i.lessonId, { silent: true });
  if (drop.length) notify();
  return drop.length;
}
