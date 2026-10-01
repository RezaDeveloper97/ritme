import type { LogPreferences, LogPreferencesChanges } from '@/entities/health-log';

/**
 * The editable part of the log preferences (B-N3-04): category order, hidden categories and quick tile
 * keys. Pure — the screen keeps a `base` (what the server holds) and a `draft`, and saves `diff(base, draft)`
 * so only the lists she touched are sent (a list she never touched keeps following the mode/phase default).
 */
export interface CustomizeDraft {
  order: string[];
  hidden: string[];
  /** Quick tile keys in order: a category code or `category.param`. */
  pinned: string[];
}

export function draftFromPrefs(prefs: LogPreferences): CustomizeDraft {
  return {
    order: prefs.categories.map((c) => c.code),
    hidden: prefs.categories.filter((c) => c.hidden).map((c) => c.code),
    pinned: [...prefs.pinned],
  };
}

/** The category a tile key belongs to (`measurements.weight` → `measurements`). */
export function tileCategory(key: string): string {
  const dot = key.indexOf('.');
  return dot < 0 ? key : key.slice(0, dot);
}

/** The category's tile is on (the category itself or one of its params). */
export function isPinned(draft: CustomizeDraft, code: string): boolean {
  return draft.pinned.some((k) => tileCategory(k) === code);
}

export function isHidden(draft: CustomizeDraft, code: string): boolean {
  return draft.hidden.includes(code);
}

/** Moves the category at `from` to `to` (clamped); the same draft when nothing moves. */
export function moveCategory(draft: CustomizeDraft, from: number, to: number): CustomizeDraft {
  const last = draft.order.length - 1;
  const target = Math.max(0, Math.min(last, to));
  if (from < 0 || from > last || from === target) return draft;
  const order = [...draft.order];
  const [code] = order.splice(from, 1);
  order.splice(target, 0, code);
  return { ...draft, order };
}

export function toggleHidden(draft: CustomizeDraft, code: string): CustomizeDraft {
  const hidden = isHidden(draft, code) ? draft.hidden.filter((c) => c !== code) : [...draft.hidden, code];
  return { ...draft, hidden };
}

/** Tile keys follow the category order, so the quick tiles read in the order of the list she arranged. */
function sortByOrder(keys: string[], order: string[]): string[] {
  const rank = (k: string) => {
    const i = order.indexOf(tileCategory(k));
    return i < 0 ? order.length : i;
  };
  return keys
    .map((k, i) => ({ k, i }))
    .sort((a, b) => rank(a.k) - rank(b.k) || a.i - b.i)
    .map((x) => x.k);
}

/**
 * Pins or unpins a category's tile. Unpinning drops every key of the category; pinning brings back the
 * keys it had on the server (`measurements.weight` stays a weight tile) or else the category itself.
 * At `max` tiles a pin is refused (the same draft).
 */
export function togglePin(draft: CustomizeDraft, code: string, basePinned: string[], max: number): CustomizeDraft {
  if (isPinned(draft, code)) {
    return { ...draft, pinned: draft.pinned.filter((k) => tileCategory(k) !== code) };
  }
  const restored = basePinned.filter((k) => tileCategory(k) === code);
  const add = restored.length ? restored : [code];
  if (draft.pinned.length + add.length > max) {
    if (draft.pinned.length >= max) return draft;
    add.splice(max - draft.pinned.length);
  }
  return { ...draft, pinned: sortByOrder([...draft.pinned, ...add], draft.order) };
}

/** Pinning one more category is refused (the counter reads `max / max`). */
export function pinsFull(draft: CustomizeDraft, max: number): boolean {
  return draft.pinned.length >= max;
}

const sameList = (a: string[], b: string[]) => a.length === b.length && a.every((v, i) => v === b[i]);
const sameSet = (a: string[], b: string[]) => a.length === b.length && a.every((v) => b.includes(v));

/** The PUT body: only the lists that differ from `base` (hidden compared as a set). */
export function diffDraft(base: CustomizeDraft, draft: CustomizeDraft): LogPreferencesChanges {
  const out: LogPreferencesChanges = {};
  if (!sameList(base.order, draft.order)) out.order = draft.order;
  if (!sameSet(base.hidden, draft.hidden)) out.hidden = draft.order.filter((c) => draft.hidden.includes(c));
  if (!sameList(base.pinned, draft.pinned)) out.pinned = draft.pinned;
  return out;
}

export function isDirty(base: CustomizeDraft, draft: CustomizeDraft): boolean {
  return Object.keys(diffDraft(base, draft)).length > 0;
}

/**
 * The preferences as they will read after the save — for the optimistic cache entry the log sheet reads
 * while the PUT is in flight. Tiles of hidden categories are left out, as the server does.
 */
export function applyDraft(prefs: LogPreferences, draft: CustomizeDraft): LogPreferences {
  const byCode = new Map(prefs.categories.map((c) => [c.code, c]));
  const categories = draft.order.flatMap((code) => {
    const c = byCode.get(code);
    if (!c) return [];
    return [{ ...c, hidden: draft.hidden.includes(code), pinned: draft.pinned.includes(code) && !draft.hidden.includes(code) }];
  });
  return {
    ...prefs,
    isDefault: false,
    categories,
    pinned: draft.pinned.filter((k) => !draft.hidden.includes(tileCategory(k))),
  };
}

// ── Custom items ──────────────────────────────────────────────────────────────

export const CUSTOM_LABEL_MAX = 40;

/** The label as the server stores it: trimmed, inner whitespace collapsed. */
export function normalizeLabel(raw: string): string {
  return raw.replace(/\s+/g, ' ').trim();
}

export type LabelProblem = 'empty' | 'tooLong' | 'duplicate';

/**
 * Client-side check before POST/PATCH (the server repeats it and answers 422): 1–40 characters, unique
 * among her active items of the category, case-insensitive. `others` = the labels to compare against.
 */
export function labelProblem(raw: string, others: string[]): LabelProblem | null {
  const label = normalizeLabel(raw);
  if (!label) return 'empty';
  if (Array.from(label).length > CUSTOM_LABEL_MAX) return 'tooLong';
  const key = label.toLocaleLowerCase();
  if (others.some((o) => normalizeLabel(o).toLocaleLowerCase() === key)) return 'duplicate';
  return null;
}
