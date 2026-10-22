import { z } from 'zod';

/**
 * The consent ledger (T-M10-23, GET /consents/history): every grant and
 * withdrawal the user made, newest first, with the title of the consent in the
 * request language. `source` says where it happened; codes this build has no
 * label for read as `other`.
 */

export const CONSENT_SOURCES = [
  'signup',
  'settings',
  'feature',
  'family',
  'telemed',
  'sharelink',
  'backfill',
  'admin',
] as const;

export type ConsentSource = (typeof CONSENT_SOURCES)[number] | 'other';

export interface ConsentEvent {
  id: number;
  consent: string;
  /** Server-localized title, or null for a code it no longer knows. */
  title: string | null;
  action: 'grant' | 'revoke';
  source: ConsentSource;
  /** ISO 8601 (+03:30), or null. */
  createdAt: string | null;
}

const isSource = (s: string): s is (typeof CONSENT_SOURCES)[number] =>
  (CONSENT_SOURCES as readonly string[]).includes(s);

export const consentHistorySchema = z
  .object({
    events: z.array(
      z.object({
        id: z.number(),
        consent: z.string(),
        title: z.string().nullable(),
        action: z.enum(['grant', 'revoke']),
        source: z.string(),
        created_at: z.string().nullable(),
      }),
    ),
  })
  .transform(({ events }): ConsentEvent[] =>
    events.map((e) => ({
      id: e.id,
      consent: e.consent,
      title: e.title,
      action: e.action,
      source: isSource(e.source) ? e.source : 'other',
      createdAt: e.created_at,
    })),
  );
