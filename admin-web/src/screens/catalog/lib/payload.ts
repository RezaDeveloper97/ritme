/**
 * Pure helpers for the catalog editor (docs/canvas-build/catalog.md §2–§4): code checks,
 * audiences / meta text ↔ API values and write bodies.
 */

/** Lowercase snake case starting with a letter — groups, item codes and audience codes (backend codePattern). */
export const CODE_PATTERN = '[a-z][a-z0-9_]*';
const CODE_RE = new RegExp(`^${CODE_PATTERN}$`);

export const MAX_CODE_LEN = 64;
export const MAX_AUDIENCES = 20;
export const MAX_AUDIENCE_LEN = 32;
export const MAX_TITLE_LEN = 255;
export const MAX_BODY_LEN = 5000;
export const MAX_META_BYTES = 16384;

export function isValidCode(value: string, max = MAX_CODE_LEN): boolean {
  return value.length > 0 && value.length <= max && CODE_RE.test(value);
}

/** Drop empty translations; `null` when nothing is left (the API clears the column). */
export function cleanTranslations(value: Record<string, string>): Record<string, string> | null {
  const out: Record<string, string> = {};
  for (const [code, text] of Object.entries(value)) if (text.trim()) out[code] = text.trim();
  return Object.keys(out).length ? out : null;
}

/** `"teen, menopause teen"` → `['teen', 'menopause']` (comma / space / «،» separated, distinct, in order). */
export function parseAudiences(text: string): string[] {
  const out: string[] = [];
  for (const part of text.split(/[\s,،]+/)) {
    const code = part.trim().toLowerCase();
    if (code && !out.includes(code)) out.push(code);
  }
  return out;
}

/** Audience codes that the API would reject (bad shape or too long). */
export function invalidAudiences(codes: readonly string[]): string[] {
  return codes.filter((c) => !isValidCode(c, MAX_AUDIENCE_LEN));
}

export type MetaParse =
  | { ok: true; value: Record<string, unknown> | null }
  | { ok: false; reason: 'json' | 'object' | 'size' };

/**
 * The meta textarea → the API value. Blank (or `{}`) is `null` (clears the column); anything else
 * must be a JSON object (catalog.md §4: flat object, per-group shape) of at most 16 KB encoded.
 */
export function parseMeta(text: string): MetaParse {
  const trimmed = text.trim();
  if (!trimmed) return { ok: true, value: null };
  let value: unknown;
  try {
    value = JSON.parse(trimmed);
  } catch {
    return { ok: false, reason: 'json' };
  }
  if (value === null) return { ok: true, value: null };
  if (typeof value !== 'object' || Array.isArray(value)) return { ok: false, reason: 'object' };
  if (new TextEncoder().encode(JSON.stringify(value)).length > MAX_META_BYTES) return { ok: false, reason: 'size' };
  return { ok: true, value: Object.keys(value).length ? (value as Record<string, unknown>) : null };
}

/** Stored meta → the textarea (pretty, raw UTF-8). */
export function formatMeta(meta: unknown): string {
  if (meta === null || meta === undefined) return '';
  if (typeof meta === 'object' && !Array.isArray(meta) && Object.keys(meta).length === 0) return '';
  return JSON.stringify(meta, null, 2);
}

/** The fields a catalog row carries into a PUT (title is required on every write). */
export interface RowForWrite {
  id: number;
  title: Record<string, string>;
  sort_order: number;
  is_active: boolean;
}

/**
 * A partial PUT: the API keeps every absent optional field (catalog.md §3), but `title` is
 * required on each write, so list-level edits (the active switch) send it along.
 */
export function partialUpdate(row: RowForWrite, patch: { sort_order?: number; is_active?: boolean }) {
  return { title: row.title, ...patch };
}

/** Move `from` to `to` (a new array); out-of-range moves return a copy. */
export function moveItem<T>(items: readonly T[], from: number, to: number): T[] {
  const next = [...items];
  if (from < 0 || from >= next.length || to < 0 || to >= next.length) return next;
  const [item] = next.splice(from, 1);
  next.splice(to, 0, item as T);
  return next;
}

/** First message for `audiences` or any `audiences.N` in a 422 bag. */
export function audiencesError(errors: Record<string, readonly string[]> | undefined): string | undefined {
  if (!errors) return undefined;
  const key = Object.keys(errors).find((k) => k === 'audiences' || k.startsWith('audiences.'));
  return key ? errors[key]?.[0] : undefined;
}
