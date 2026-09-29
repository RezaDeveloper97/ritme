/**
 * One-time navigation handoff: data a screen needs from the one before it that
 * must NOT ride in the URL — a checkup's (possibly user-typed) title, a
 * pregnancy care-plan topic or date. App Router fetches the RSC payload for the
 * full URL, so a query string reaches the server, proxy logs and the browser
 * history (CLAUDE.md §11; security audit M3-M7 #3).
 *
 * The sender stashes the fields under a random id and navigates with only
 * `?<param>=<id>`; the receiver reads them back by that id. One slot per tab
 * (`sessionStorage`, with an in-memory fallback where storage is blocked): a
 * new stash replaces the old one, a reload of the target keeps working, and
 * `clearHandoff()` — on save and on session end — drops it.
 */

export type HandoffData = Record<string, string>;

const STORAGE_KEY = 'ritme_handoff';
const ID_PATTERN = /^[a-z0-9]{8,40}$/;

interface Slot {
  id: string;
  data: HandoffData;
}

let memory: Slot | null = null;

function storage(): Storage | null {
  try {
    return typeof sessionStorage === 'undefined' ? null : sessionStorage;
  } catch {
    return null;
  }
}

function randomId(): string {
  const bytes = new Uint8Array(12);
  globalThis.crypto.getRandomValues(bytes);
  return Array.from(bytes, (b) => b.toString(36).padStart(2, '0')).join('');
}

/** Only string fields survive; everything else is dropped. */
function clean(data: unknown): HandoffData | null {
  if (!data || typeof data !== 'object' || Array.isArray(data)) return null;
  const out: HandoffData = {};
  for (const [key, value] of Object.entries(data)) if (typeof value === 'string') out[key] = value;
  return out;
}

/** Keeps `data` for the next screen and returns the id to put in its URL. */
export function stashHandoff(data: HandoffData): string {
  const slot: Slot = { id: randomId(), data: clean(data) ?? {} };
  memory = slot;
  try {
    storage()?.setItem(STORAGE_KEY, JSON.stringify(slot));
  } catch {
    // Quota / blocked storage: the in-memory copy still covers a client-side navigation.
  }
  return slot.id;
}

/** The data stashed under `id`, or null (unknown, replaced, cleared, malformed). */
export function readHandoff(id: string | null | undefined): HandoffData | null {
  if (!id || !ID_PATTERN.test(id)) return null;
  if (memory?.id === id) return { ...memory.data };
  try {
    const raw = storage()?.getItem(STORAGE_KEY);
    if (!raw) return null;
    const slot = JSON.parse(raw) as Partial<Slot>;
    return slot.id === id ? clean(slot.data) : null;
  } catch {
    return null;
  }
}

/** Drops the stashed data (after it was used, and on session end). */
export function clearHandoff(): void {
  memory = null;
  try {
    storage()?.removeItem(STORAGE_KEY);
  } catch {
    // Nothing to clean if storage is unreachable.
  }
}

/** `href` with `param=<id>` added — `href` itself must not carry the data. */
export function withHandoff(href: string, data: HandoffData, param = 'prefill'): string {
  const id = stashHandoff(data);
  return `${href}${href.includes('?') ? '&' : '?'}${encodeURIComponent(param)}=${id}`;
}
