import { z } from 'zod';

import {
  FEED_SLOTS,
  type FeedChallenge,
  type FeedEvidence,
  type FeedItem,
  type FeedSlot,
  type MessageFeed,
} from '../model/feed';

const str = z.string().catch('');
const optStr = z
  .string()
  .nullish()
  .transform((v) => (v && v.trim() ? v : null));

const evidenceSchema = z
  .object({ seen: str.default(''), confidence: str.default(''), contact: str.default('') })
  .nullish()
  .transform((e): FeedEvidence | null => (e ? { seen: e.seen, confidence: e.confidence, contact: e.contact } : null));

const itemSchema = z.object({
  id: z.number().int().positive(),
  rule_key: optStr,
  version: z.number().int().catch(1),
  variant: z.string().min(1).catch('a'),
  kind: str,
  level: z.string().catch('info'),
  title: str,
  body: str,
  cta_text: optStr,
  cta_deeplink: optStr,
  dismissible: z.boolean().catch(true),
  evidence: evidenceSchema,
});

const challengeSchema = z.object({
  id: z.number().int().positive(),
  title: z.string().min(1),
  description: optStr,
  category: optStr,
  duration_days: z.number().catch(0),
  day: z.number().nullish(),
  reason: str,
});

const alertsSchema = z.object({ open: z.number().catch(0), urgent: z.number().catch(0) });

/**
 * Boundary parser for GET /messages/feed (CLAUDE.md §10). Tolerant by design: unknown slots and fields are
 * ignored, an item that does not parse is dropped (the rest of the slot survives), and an item with no text at
 * all is not worth a card.
 */
export function parseFeed(raw: unknown): MessageFeed | null {
  const root = z.object({ slots: z.record(z.string(), z.unknown()).catch({}) }).safeParse(raw);
  if (!root.success || typeof raw !== 'object' || raw === null) return null;
  const data = raw as Record<string, unknown>;

  const slots = Object.fromEntries(FEED_SLOTS.map((s) => [s, [] as FeedItem[]])) as Record<FeedSlot, FeedItem[]>;
  for (const slot of FEED_SLOTS) {
    const list = root.data.slots[slot];
    if (!Array.isArray(list)) continue;
    for (const entry of list) {
      const p = itemSchema.safeParse(entry);
      if (!p.success) continue;
      const i = p.data;
      if (!i.title && !i.body) continue;
      slots[slot].push({
        slot,
        id: i.id,
        ruleKey: i.rule_key,
        version: i.version,
        variant: i.variant,
        kind: i.kind,
        level: i.level,
        title: i.title,
        body: i.body,
        ctaText: i.cta_text,
        ctaDeeplink: i.cta_deeplink,
        dismissible: i.dismissible,
        evidence: i.evidence,
      });
    }
  }

  const alerts = alertsSchema.safeParse(data.alerts);
  const ch = challengeSchema.safeParse(data.daily_challenge);
  const dailyChallenge: FeedChallenge | null = ch.success
    ? {
        id: ch.data.id,
        title: ch.data.title,
        description: ch.data.description,
        category: ch.data.category,
        durationDays: ch.data.duration_days,
        day: ch.data.day ?? null,
        reason: ch.data.reason,
      }
    : null;

  return {
    slots,
    alerts: alerts.success ? alerts.data : { open: 0, urgent: 0 },
    rulesRev: typeof data.rules_rev === 'number' ? data.rules_rev : 0,
    dailyChallenge,
  };
}
