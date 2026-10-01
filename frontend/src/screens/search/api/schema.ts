import { z } from 'zod';

import { diffInDays, fromApiDate } from '@/shared/lib/date';

import {
  SEARCH_GROUP_KEYS,
  SEARCH_HIT_TYPES,
  SEARCH_SCOPES,
  type SearchHit,
  type SearchHitMeta,
  type SearchResults,
} from '../model/types';

/*
 * Boundary parser for `GET /api/v1/search` (CLAUDE.md §10 — zod at the edge).
 * Unknown hit types and group keys (a source added server-side before the
 * app knows how to show it) are dropped rather than failing the whole answer.
 */

const num = z.number().nullable().optional().transform((v) => v ?? null);

const metaSchema = z
  .object({
    days: num,
    window: z
      .object({ kind: z.string(), from: z.string().optional(), to: z.string().optional() })
      .nullable()
      .optional(),
    peak: z
      .object({ score: z.number(), cycle_day: num })
      .nullable()
      .optional(),
    kind: z.string().nullable().optional(),
    category: z.string().nullable().optional(),
    read_time_minutes: num,
  })
  .passthrough()
  .transform((m): SearchHitMeta => {
    const windowKind = m.window?.kind === 'cycle' || m.window?.kind === 'days' ? m.window.kind : null;
    return {
      category: m.category ?? null,
      days: m.days,
      windowKind,
      windowDays: windowKind === 'days' && m.window?.from && m.window.to ? spanDays(m.window.from, m.window.to) : null,
      peak: m.peak ? { score: m.peak.score, cycleDay: m.peak.cycle_day } : null,
      kind: m.kind ?? null,
      readTimeMinutes: m.read_time_minutes,
    };
  });

/** Inclusive day count between two `YYYY-MM-DD` API dates. */
function spanDays(from: string, to: string): number | null {
  const days = diffInDays(fromApiDate(to), fromApiDate(from));
  return Number.isFinite(days) && days >= 0 ? days + 1 : null;
}

const hitSchema = z.object({
  type: z.string(),
  id: z.string(),
  title: z.string(),
  subtitle: z.string().nullable().optional(),
  route: z.string(),
  meta: metaSchema.optional(),
});

const groupSchema = z.object({
  key: z.string(),
  total: z.number().int().nonnegative(),
  items: z.array(hitSchema),
});

const isHitType = (v: string): v is SearchHit['type'] => (SEARCH_HIT_TYPES as readonly string[]).includes(v);
const isGroupKey = (v: string): v is SearchResults['groups'][number]['key'] =>
  (SEARCH_GROUP_KEYS as readonly string[]).includes(v);

const EMPTY_META: SearchHitMeta = {
  category: null,
  days: null,
  windowKind: null,
  windowDays: null,
  peak: null,
  kind: null,
  readTimeMinutes: null,
};

export const searchResultsSchema = z
  .object({
    query: z.string(),
    scope: z.enum(SEARCH_SCOPES),
    groups: z.array(groupSchema),
  })
  .transform(
    (d): SearchResults => ({
      query: d.query,
      scope: d.scope,
      groups: d.groups.flatMap((g) => {
        if (!isGroupKey(g.key)) return [];
        const items = g.items.flatMap((h): SearchHit[] =>
          isHitType(h.type)
            ? [
                {
                  type: h.type,
                  id: h.id,
                  title: h.title,
                  subtitle: h.subtitle ?? null,
                  route: h.route,
                  meta: h.meta ?? EMPTY_META,
                },
              ]
            : [],
        );
        return [{ key: g.key, total: g.total, items }];
      }),
    }),
  );
