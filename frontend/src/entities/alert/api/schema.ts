import { z } from 'zod';

import { ALERT_LEVELS, ALERT_STATES, type AlertItem, type AlertPage } from '../model/types';

const text = z.string().nullish().transform((v) => (v && v.trim() ? v : null));

const evidenceSchema = z
  .object({ seen: text, confidence: text, contact: text })
  .nullish()
  .transform((e) => (e && (e.seen || e.confidence || e.contact) ? { seen: e.seen ?? '', confidence: e.confidence ?? '', contact: e.contact } : null));

/** Boundary parser for one inbox row: unknown level/state fall back so a new server value never blanks the screen. */
export const alertItemSchema = z
  .object({
    id: z.number().int(),
    level: z.enum(ALERT_LEVELS).catch('info'),
    state: z.enum(ALERT_STATES).catch('active'),
    title: z.string().catch(''),
    body: z.string().catch(''),
    cta_text: text,
    cta_deeplink: text,
    evidence: evidenceSchema,
    hit_count: z.number().catch(1),
    dismissible: z.boolean().catch(true),
    last_at: z.string().nullish(),
    acked_at: z.string().nullish(),
  })
  .transform(
    (r): AlertItem => ({
      id: r.id,
      level: r.level,
      state: r.state,
      title: r.title,
      body: r.body,
      ctaText: r.cta_text,
      ctaDeeplink: r.cta_deeplink,
      evidence: r.evidence,
      hitCount: r.hit_count,
      // An urgent item that is not acknowledged is never closable, whatever the flag says (SPEC §5.4.1).
      dismissible: r.level === 'urgent' && r.state === 'active' ? false : r.dismissible,
      lastAt: r.last_at ?? null,
      ackedAt: r.acked_at ?? null,
    }),
  );

/** `data` of GET /alerts. Rows that fail to parse (no id) are dropped, not fatal. */
export function parseAlertPage(raw: unknown): AlertPage {
  const o = (typeof raw === 'object' && raw !== null ? raw : {}) as { items?: unknown; next_cursor?: unknown };
  const items: AlertItem[] = [];
  for (const row of Array.isArray(o.items) ? o.items : []) {
    const parsed = alertItemSchema.safeParse(row);
    if (parsed.success) items.push(parsed.data);
  }
  return { items, nextCursor: typeof o.next_cursor === 'string' && o.next_cursor ? o.next_cursor : null };
}
