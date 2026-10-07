import { diffInDays, fromApiDate } from '@/shared/lib/date';
import { toAsciiDigits } from '@/shared/lib/phone';

import {
  DATE_FIELDS,
  type DocumentKind,
  EXTRACT_FIELDS,
  type Extraction,
  ITEM_FIELDS,
  type ItemField,
  NUMBER_FIELDS,
  type LinkState,
  type TimelineLink,
} from './documents';

/* Pure helpers of the record documents (CB-REC-04). No React, no date library. */

/** Below this the review form marks a read value «مطمئن نیستم — بررسی کن». */
export const LOW_CONFIDENCE = 0.6;

/** One editable value of the review form: always a string in the input; `''` = cleared. */
export type ReviewDraft = Record<string, string>;
export type ReviewItemDraft = Record<ItemField, string>;

function asText(v: unknown): string {
  if (v === null || v === undefined) return '';
  if (typeof v === 'boolean') return v ? '1' : '';
  return String(v);
}

/** The review form's starting values: what the user confirmed, else what the AI read, per the kind's schema. */
export function reviewDraftOf(kind: DocumentKind, e: Extraction | null): ReviewDraft {
  const out: ReviewDraft = {};
  for (const key of EXTRACT_FIELDS[kind]) {
    const confirmed = e?.reviewed && key in e.reviewed ? e.reviewed[key] : undefined;
    out[key] = confirmed !== undefined ? asText(confirmed) : asText(e?.fields[key]?.value);
  }
  return out;
}

/** The prescription rows of the review form. */
export function reviewItemsOf(e: Extraction | null): ReviewItemDraft[] {
  if (!e) return [];
  if (e.reviewedItems) {
    return e.reviewedItems.map((r) => Object.fromEntries(ITEM_FIELDS.map((k) => [k, asText(r[k])])) as ReviewItemDraft);
  }
  return e.items.map((r) => Object.fromEntries(ITEM_FIELDS.map((k) => [k, asText(r[k]?.value)])) as ReviewItemDraft);
}

/**
 * The `fields` of POST …/review: numbers for the gestational age, `null` for a cleared value, trimmed strings
 * otherwise. Only the kind's schema keys are sent (an unknown key is a 422).
 */
export function reviewFieldsBody(kind: DocumentKind, draft: ReviewDraft): Record<string, string | number | null> {
  const out: Record<string, string | number | null> = {};
  for (const key of EXTRACT_FIELDS[kind]) {
    const raw = (draft[key] ?? '').trim();
    if (raw === '') out[key] = null;
    else if (NUMBER_FIELDS.includes(key)) {
      const n = Number(toAsciiDigits(raw));
      out[key] = Number.isFinite(n) ? Math.trunc(n) : null;
    } else out[key] = raw;
  }
  return out;
}

/** The `items` of POST …/review: empty rows dropped, cleared cells null. */
export function reviewItemsBody(items: readonly ReviewItemDraft[]): Record<ItemField, string | null>[] {
  return items
    .filter((r) => ITEM_FIELDS.some((k) => r[k].trim() !== ''))
    .map((r) => Object.fromEntries(ITEM_FIELDS.map((k) => [k, r[k].trim() || null])) as Record<ItemField, string | null>);
}

export const isDateField = (key: string): boolean => DATE_FIELDS.includes(key);
export const isNumberField = (key: string): boolean => NUMBER_FIELDS.includes(key);

/** A timeline row's claim badge (board: «خسارت: منتظر» / «پیوست خسارت»), from its links only. */
export function claimBadge(links: readonly TimelineLink[]): Extract<LinkState, 'waiting' | 'attached'> | null {
  if (links.some((l) => l.type === 'claim' && l.state === 'waiting')) return 'waiting';
  if (links.some((l) => l.type === 'claim' && l.state === 'attached')) return 'attached';
  return null;
}

/** Hospital stay in days, admission and discharge inclusive (`Y-m-d`); null without both dates. */
export function stayDays(date: string | null, endedOn: string | null): number | null {
  if (!date || !endedOn) return null;
  const days = diffInDays(fromApiDate(endedOn), fromApiDate(date));
  return Number.isNaN(days) || days < 0 ? null : days + 1;
}

export const isImageMime = (mime: string): boolean => mime.startsWith('image/');

/**
 * A signed file link is used only when it points at this API's own `/files/` route over https (http only while the
 * API base itself is http, i.e. local dev); anything else becomes null (security audit CB-REC-04 L1).
 */
export function safeFileUrl(url: string | null, apiBase: string): string | null {
  if (!url) return null;
  try {
    const base = new URL(apiBase);
    const u = new URL(url);
    if (u.protocol !== 'https:' && !(u.protocol === 'http:' && base.protocol === 'http:')) return null;
    if (u.origin !== base.origin) return null;
    if (!u.pathname.startsWith(`${base.pathname.replace(/\/+$/, '')}/files/`)) return null;
    return u.toString();
  } catch {
    return null;
  }
}
