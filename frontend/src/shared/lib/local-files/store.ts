/**
 * On-device file store (IndexedDB) for files that must never leave the phone —
 * e.g. checkup report photos and PDFs (docs/checkups/README.md: "The attachment
 * file never leaves the phone"; the server only learns `has_attachment`).
 *
 * Domain-agnostic: a slice creates its own store with a `namespace` (one
 * logical bucket inside the shared database) and keys files by whatever id it
 * owns (a record id). Nothing here uploads, logs or reports file contents or
 * names (CLAUDE.md §11).
 *
 * Graceful by contract: when storage is unavailable (SSR, a private window
 * that refuses IndexedDB, a blocked database) reads resolve empty — `get` →
 * null, `list` → [], `delete` → no-op — and only `put` rejects, with a
 * {@link LocalFilesError} whose `code` the UI can translate.
 */

export type LocalFilesErrorCode =
  /** IndexedDB is missing or refused to open. */
  | 'unavailable'
  /** The file is over the store's per-file cap. */
  | 'too_large'
  /** Storing it would take the namespace over its total cap. */
  | 'store_full'
  /** The MIME type is not in the store's accept list. */
  | 'unsupported_type'
  /** The browser's quota refused the write. */
  | 'quota'
  | 'failed';

export class LocalFilesError extends Error {
  readonly code: LocalFilesErrorCode;

  constructor(code: LocalFilesErrorCode) {
    // Deliberately no file name / size in the message: it may reach an error UI or a report.
    super(`local-files: ${code}`);
    this.name = 'LocalFilesError';
    this.code = code;
  }
}

/** What callers see about a stored file, without loading the bytes. */
export interface LocalFileMeta {
  key: string;
  name: string;
  /** MIME type (`image/jpeg`, `application/pdf`, …); '' when the browser didn't know. */
  type: string;
  size: number;
  /** Epoch ms of the last `put`. */
  savedAt: number;
}

export interface LocalFile extends LocalFileMeta {
  blob: Blob;
}

/** A row as persisted: namespaced id + the file. Exported for backend implementations. */
export interface LocalFileRow extends LocalFile {
  id: string;
  namespace: string;
}

/**
 * The persistence the store sits on. The default is IndexedDB; tests (Node has
 * no IndexedDB) pass the in-memory one. Every method may reject — the store
 * turns a rejection into the graceful behaviour described above.
 */
export interface LocalFilesBackend {
  get(id: string): Promise<LocalFileRow | undefined>;
  put(row: LocalFileRow): Promise<void>;
  delete(id: string): Promise<void>;
  /** Every row whose `namespace` matches. */
  list(namespace: string): Promise<LocalFileRow[]>;
  /** Every row of every namespace — the session-end wipe. Optional for test doubles. */
  clearAll?(): Promise<void>;
}

export interface LocalFileStoreOptions {
  /** Bucket name inside the shared database, e.g. `checkup-reports`. */
  namespace: string;
  /** Per-file cap in bytes. */
  maxFileBytes: number;
  /** Optional cap on the whole namespace, in bytes (a replaced file doesn't count twice). */
  maxTotalBytes?: number;
  /**
   * Accepted MIME types, exact (`image/jpeg`, `application/pdf`). Omitted =
   * anything. Wildcards like `image/*` are deliberately NOT supported: they
   * admit `image/svg+xml`, which can carry script (security audit M3-M7 #2).
   * A file with an empty type is rejected when a list is given.
   */
  accept?: readonly string[];
  /** Defaults to IndexedDB (or "unavailable" where it doesn't exist). */
  backend?: LocalFilesBackend | null;
  /** Clock, for tests. */
  now?: () => number;
}

export interface LocalFileStore {
  /** True when a write can be attempted (IndexedDB exists and opens). */
  isAvailable(): Promise<boolean>;
  /**
   * The synchronous part of `put`'s validation (type, per-file size), so a
   * caller can refuse a file before doing anything else with it. Null = OK.
   */
  check(file: Blob): LocalFilesErrorCode | null;
  /** Stores (or replaces) the file under `key`. Rejects with {@link LocalFilesError}. */
  put(key: string | number, file: Blob, name?: string): Promise<LocalFileMeta>;
  get(key: string | number): Promise<LocalFile | null>;
  /** Existence check that doesn't hand the bytes to the caller. */
  has(key: string | number): Promise<boolean>;
  delete(key: string | number): Promise<void>;
  /** Metadata of every file in this namespace, newest first. */
  list(): Promise<LocalFileMeta[]>;
  /** Deletes every file in this namespace. Never rejects. */
  clear(): Promise<void>;
  /** Deletes the files of these keys. Never rejects. */
  deleteMany(keys: Iterable<string | number>): Promise<void>;
  /** Accept list + caps, for `<input accept>` and pre-checks in the UI. */
  readonly limits: { maxFileBytes: number; maxTotalBytes: number | null; accept: readonly string[] | null };
}

/** `Image/JPEG; foo=bar` → `image/jpeg`; '' when there is no usable type. */
export function mimeEssence(type: string): string {
  return (type.split(';')[0] ?? '').trim().toLowerCase();
}

/**
 * Exact allow-list match on the MIME essence. A `*` pattern matches nothing:
 * see {@link LocalFileStoreOptions.accept}.
 */
export function matchesAccept(type: string, accept: readonly string[]): boolean {
  const t = mimeEssence(type);
  if (!t) return false;
  return accept.some((pattern) => !pattern.includes('*') && mimeEssence(pattern) === t);
}

function toMeta(row: LocalFileRow): LocalFileMeta {
  return { key: row.key, name: row.name, type: row.type, size: row.size, savedAt: row.savedAt };
}

function isQuotaError(error: unknown): boolean {
  return (
    typeof error === 'object' &&
    error !== null &&
    'name' in error &&
    (error as { name: unknown }).name === 'QuotaExceededError'
  );
}

/**
 * Every store created in this page, so a session-end wipe also reaches stores
 * on a non-default backend. The default IndexedDB database is wiped as a whole
 * by {@link clearAllLocalFiles} even when a store's module was never loaded.
 */
const liveStores = new Set<LocalFileStore>();

/**
 * Wipes every on-device file of every namespace: the whole IndexedDB database
 * plus any store on another backend. Files kept here are per-user health data
 * (checkup reports), so `shared/session` runs this on every session end.
 * Never rejects.
 */
export async function clearAllLocalFiles(): Promise<void> {
  const tasks: Promise<unknown>[] = [...liveStores].map((store) => store.clear());
  const idb = createIndexedDbBackend();
  if (idb?.clearAll) tasks.push(idb.clearAll().catch(() => undefined));
  await Promise.all(tasks);
}

export function createLocalFileStore(options: LocalFileStoreOptions): LocalFileStore {
  const { namespace, maxFileBytes, maxTotalBytes, accept } = options;
  const now = options.now ?? Date.now;
  let resolvedBackend: LocalFilesBackend | null | undefined = options.backend;

  const backend = (): LocalFilesBackend | null => {
    if (resolvedBackend === undefined) resolvedBackend = createIndexedDbBackend();
    return resolvedBackend;
  };
  const idOf = (key: string | number): string => `${namespace}/${String(key)}`;

  const check = (file: Blob): LocalFilesErrorCode | null => {
    if (accept && !matchesAccept(file.type, accept)) return 'unsupported_type';
    if (file.size > maxFileBytes) return 'too_large';
    return null;
  };

  async function listRows(): Promise<LocalFileRow[]> {
    const b = backend();
    if (!b) return [];
    try {
      return await b.list(namespace);
    } catch {
      return [];
    }
  }

  const store: LocalFileStore = {
    limits: {
      maxFileBytes,
      maxTotalBytes: maxTotalBytes ?? null,
      accept: accept ?? null,
    },

    async isAvailable() {
      const b = backend();
      if (!b) return false;
      try {
        await b.list(namespace);
        return true;
      } catch {
        return false;
      }
    },

    check,

    async put(key, file, name) {
      const invalid = check(file);
      if (invalid) throw new LocalFilesError(invalid);
      const b = backend();
      if (!b) throw new LocalFilesError('unavailable');

      const id = idOf(key);
      if (maxTotalBytes !== undefined) {
        let others: LocalFileRow[];
        try {
          others = await b.list(namespace);
        } catch {
          throw new LocalFilesError('unavailable');
        }
        const used = others.filter((r) => r.id !== id).reduce((sum, r) => sum + r.size, 0);
        if (used + file.size > maxTotalBytes) throw new LocalFilesError('store_full');
      }

      const fileName = name ?? (typeof File !== 'undefined' && file instanceof File ? file.name : '');
      const row: LocalFileRow = {
        id,
        namespace,
        key: String(key),
        name: fileName,
        type: file.type,
        size: file.size,
        savedAt: now(),
        blob: file,
      };
      try {
        await b.put(row);
      } catch (error) {
        if (error instanceof LocalFilesError) throw error;
        throw new LocalFilesError(isQuotaError(error) ? 'quota' : 'failed');
      }
      return toMeta(row);
    },

    async get(key) {
      const b = backend();
      if (!b) return null;
      try {
        const row = await b.get(idOf(key));
        if (!row) return null;
        return { ...toMeta(row), blob: row.blob };
      } catch {
        return null;
      }
    },

    async has(key) {
      const b = backend();
      if (!b) return false;
      try {
        return (await b.get(idOf(key))) !== undefined;
      } catch {
        return false;
      }
    },

    async delete(key) {
      const b = backend();
      if (!b) return;
      try {
        await b.delete(idOf(key));
      } catch {
        // Nothing to clean up if the database can't be reached.
      }
    },

    async list() {
      const rows = await listRows();
      return rows.map(toMeta).sort((a, b) => b.savedAt - a.savedAt);
    },

    async clear() {
      await store.deleteMany((await listRows()).map((row) => row.key));
    },

    async deleteMany(keys) {
      await Promise.all([...keys].map((key) => store.delete(key)));
    },
  };
  liveStores.add(store);
  return store;
}

// ── Backends ─────────────────────────────────────────────────────

/** A Map-backed backend — tests, and a sane stand-in for environments without IndexedDB. */
export function createMemoryBackend(): LocalFilesBackend {
  const rows = new Map<string, LocalFileRow>();
  return {
    async get(id) {
      return rows.get(id);
    },
    async put(row) {
      rows.set(row.id, row);
    },
    async delete(id) {
      rows.delete(id);
    },
    async list(namespace) {
      return [...rows.values()].filter((r) => r.namespace === namespace);
    },
    async clearAll() {
      rows.clear();
    },
  };
}

const DB_NAME = 'ritme-local-files';
const DB_VERSION = 1;
const STORE = 'files';

function promisify<T>(request: IDBRequest<T>): Promise<T> {
  return new Promise((resolve, reject) => {
    request.onsuccess = () => resolve(request.result);
    request.onerror = () => reject(request.error);
  });
}

/**
 * IndexedDB backend: one database for the whole app, one object store keyed by
 * `<namespace>/<key>`, with a `namespace` index. Returns null where IndexedDB
 * doesn't exist (SSR, Node). The connection opens lazily on first use; a failed
 * open is retried on the next call rather than cached forever.
 */
export function createIndexedDbBackend(factory?: IDBFactory): LocalFilesBackend | null {
  const idb = factory ?? (typeof indexedDB === 'undefined' ? undefined : indexedDB);
  if (!idb) return null;

  let dbPromise: Promise<IDBDatabase> | null = null;
  const open = (): Promise<IDBDatabase> => {
    if (!dbPromise) {
      dbPromise = new Promise<IDBDatabase>((resolve, reject) => {
        let request: IDBOpenDBRequest;
        try {
          request = idb.open(DB_NAME, DB_VERSION);
        } catch (error) {
          reject(error);
          return;
        }
        request.onupgradeneeded = () => {
          const db = request.result;
          if (!db.objectStoreNames.contains(STORE)) {
            const store = db.createObjectStore(STORE, { keyPath: 'id' });
            store.createIndex('namespace', 'namespace', { unique: false });
          }
        };
        request.onsuccess = () => {
          const db = request.result;
          // Another tab upgrading the schema: let go so it isn't blocked.
          db.onversionchange = () => {
            db.close();
            dbPromise = null;
          };
          resolve(db);
        };
        request.onerror = () => reject(request.error);
        request.onblocked = () => reject(new LocalFilesError('unavailable'));
      }).catch((error: unknown) => {
        dbPromise = null;
        throw error;
      });
    }
    return dbPromise;
  };

  const run = async <T>(
    mode: IDBTransactionMode,
    body: (store: IDBObjectStore) => IDBRequest<T>,
  ): Promise<T> => {
    const db = await open();
    const tx = db.transaction(STORE, mode);
    const result = promisify(body(tx.objectStore(STORE)));
    if (mode === 'readonly') return result;
    // The transaction's own error is the one reported; don't leave this rejection unhandled.
    result.catch(() => undefined);
    // A write only counts once the transaction commits (quota errors surface here).
    await new Promise<void>((resolve, reject) => {
      tx.oncomplete = () => resolve();
      tx.onerror = () => reject(tx.error);
      tx.onabort = () => reject(tx.error ?? new LocalFilesError('failed'));
    });
    return result;
  };

  return {
    get: (id) => run('readonly', (s) => s.get(id) as IDBRequest<LocalFileRow | undefined>),
    put: async (row) => {
      await run('readwrite', (s) => s.put(row));
    },
    delete: async (id) => {
      await run('readwrite', (s) => s.delete(id));
    },
    list: (namespace) =>
      run('readonly', (s) => s.index('namespace').getAll(namespace) as IDBRequest<LocalFileRow[]>),
    clearAll: async () => {
      await run('readwrite', (s) => s.clear());
    },
  };
}
