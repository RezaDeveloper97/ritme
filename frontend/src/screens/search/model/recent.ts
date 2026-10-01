import { onSessionEnd } from '@/shared/session';

/**
 * Recent searches (Nav_Search «جست‌وجوهای اخیر»), on this device only.
 *
 * Privacy (CLAUDE.md §11): a query can be health data, so the list never
 * leaves the device and is clearable from the screen. It is also stamped with
 * the account that made it: a session-end wipe registered here only runs when
 * this module happens to be loaded (security audit M3-M7 #1), so a list read
 * by a different account is dropped instead of shown. Storage can be missing
 * or throw (private mode); then there is simply no history.
 */
const KEY = 'ritme_search_recent';
export const RECENT_SEARCH_LIMIT = 6;

interface Stored {
  owner: string;
  items: string[];
}

function storage(): Storage | null {
  try {
    return typeof window === 'undefined' ? null : window.localStorage;
  } catch {
    return null;
  }
}

function load(): Stored | null {
  try {
    const raw = storage()?.getItem(KEY);
    if (!raw) return null;
    const parsed = JSON.parse(raw) as Partial<Stored> | null;
    if (!parsed || typeof parsed.owner !== 'string' || !Array.isArray(parsed.items)) return null;
    return {
      owner: parsed.owner,
      items: parsed.items
        .filter((v): v is string => typeof v === 'string' && v.trim() !== '')
        .slice(0, RECENT_SEARCH_LIMIT),
    };
  } catch {
    return null;
  }
}

function save(owner: string, items: readonly string[]): void {
  try {
    const s = storage();
    if (!s) return;
    if (items.length === 0) s.removeItem(KEY);
    else s.setItem(KEY, JSON.stringify({ owner, items: [...items] } satisfies Stored));
  } catch {
    // no storage: nothing to remember
  }
}

/** `owner`'s recent searches, newest first; another account's list is wiped. */
export function readRecentSearches(owner: string): string[] {
  const stored = load();
  if (!stored) return [];
  if (stored.owner !== owner) {
    clearRecentSearches();
    return [];
  }
  return stored.items;
}

/** Puts `query` first (deduplicated) and returns the new list. */
export function addRecentSearch(owner: string, query: string): string[] {
  const q = query.trim();
  const current = readRecentSearches(owner);
  if (!q) return current;
  const next = [q, ...current.filter((v) => v !== q)].slice(0, RECENT_SEARCH_LIMIT);
  save(owner, next);
  return next;
}

export function clearRecentSearches(): void {
  try {
    storage()?.removeItem(KEY);
  } catch {
    // nothing persisted
  }
}

onSessionEnd(clearRecentSearches);
