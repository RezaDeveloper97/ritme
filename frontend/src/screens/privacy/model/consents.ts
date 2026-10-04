import { z } from 'zod';

/**
 * Privacy consents (B-N1-12) — GET/PUT /profile/consents. Opt-in: a consent
 * the user never answered is `granted: false`. The server owns the list and
 * order; codes this build has no copy for are hidden.
 */

export type ConsentCode = 'ai_lab_analysis' | 'assistant_profile' | 'anonymous_stats';

export const KNOWN_CONSENTS: readonly ConsentCode[] = ['ai_lab_analysis', 'assistant_profile', 'anonymous_stats'];

export interface Consent {
  code: ConsentCode;
  granted: boolean;
  /** ISO 8601 (+03:30) of the last grant, or null. */
  grantedAt: string | null;
  /** Version of the consent text in force; a grant sends it back (B-N6-05b). */
  version: number;
}

const isKnown = (code: string): code is ConsentCode => (KNOWN_CONSENTS as readonly string[]).includes(code);

export const consentsSchema = z
  .object({
    consents: z.array(
      z.object({
        code: z.string(),
        granted: z.boolean(),
        granted_at: z.string().nullable(),
        revoked_at: z.string().nullable().optional(),
        version: z.number().int().positive().optional(),
      }),
    ),
  })
  .transform((v): Consent[] =>
    v.consents.flatMap((c) =>
      isKnown(c.code) ? [{ code: c.code, granted: c.granted, grantedAt: c.granted_at, version: c.version ?? 1 }] : [],
    ),
  );

/**
 * PUT /profile/consents body for one switch. A grant names the version of the
 * text shown (the server refuses AI grants without it, B-N6-05b).
 */
export function consentChangeBody(code: ConsentCode, granted: boolean, version: number) {
  return granted ? { consents: { [code]: true }, versions: { [code]: version } } : { consents: { [code]: false } };
}

/** The optimistic copy after flipping one consent (the server stamps the real time). */
export function applyConsent(list: readonly Consent[], code: ConsentCode, granted: boolean, nowIso: string): Consent[] {
  return list.map((c) => (c.code === code ? { ...c, granted, grantedAt: granted ? nowIso : c.grantedAt } : c));
}
