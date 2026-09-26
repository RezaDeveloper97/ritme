/**
 * One queued write. `key` identifies *what* is being written (e.g. one day's
 * log), so a later write to the same key replaces the earlier one — replaying
 * the queue therefore sends each resource's latest body exactly once, and the
 * PUTs it carries are idempotent on the server.
 */
export interface OutboxEntry {
  key: string;
  method: "put" | "post" | "delete";
  url: string;
  body: unknown;
  /** Epoch ms of the (latest) enqueue — replay order. */
  queuedAt: number;
}

export interface OutboxStore {
  put(entry: OutboxEntry): Promise<void>;
  all(): Promise<OutboxEntry[]>;
  delete(key: string): Promise<void>;
}

/** In-memory store: tests, and browsers without IndexedDB (private mode). */
export function createMemoryStore(): OutboxStore {
  const map = new Map<string, OutboxEntry>();
  return {
    put: async (entry) => {
      map.set(entry.key, entry);
    },
    all: async () => [...map.values()],
    delete: async (key) => {
      map.delete(key);
    },
  };
}

const DB_NAME = "ritme-outbox";
const STORE = "entries";

function request<T>(req: IDBRequest<T>): Promise<T> {
  return new Promise((resolve, reject) => {
    req.onsuccess = () => resolve(req.result);
    req.onerror = () => reject(req.error);
  });
}

/**
 * IndexedDB store (`ritme-outbox` / `entries`, keyPath `key`). Survives a
 * reload or a closed tab, so a log saved offline is still sent later.
 * Privacy (§11): bodies are health data — they stay on this device until
 * they are sent to the API, and are deleted as soon as the server accepts them.
 */
export function createIdbStore(factory: IDBFactory): OutboxStore {
  let dbPromise: Promise<IDBDatabase> | null = null;
  const db = () => {
    dbPromise ??= new Promise<IDBDatabase>((resolve, reject) => {
      const open = factory.open(DB_NAME, 1);
      open.onupgradeneeded = () => {
        if (!open.result.objectStoreNames.contains(STORE)) {
          open.result.createObjectStore(STORE, { keyPath: "key" });
        }
      };
      open.onsuccess = () => resolve(open.result);
      open.onerror = () => {
        dbPromise = null;
        reject(open.error);
      };
    });
    return dbPromise;
  };
  const tx = async (mode: IDBTransactionMode) =>
    (await db()).transaction(STORE, mode).objectStore(STORE);
  return {
    put: async (entry) => {
      await request((await tx("readwrite")).put(entry));
    },
    all: async () =>
      request((await tx("readonly")).getAll() as IDBRequest<OutboxEntry[]>),
    delete: async (key) => {
      await request((await tx("readwrite")).delete(key));
    },
  };
}
