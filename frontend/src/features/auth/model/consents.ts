import { z } from 'zod';

/**
 * Signup consents (T-M10-23, D-174). The phone screen shows two consent texts
 * (`auth.phone.terms`, `auth.phone.healthData`); verify-otp sends the version
 * of each text shown, and the API records them in the consent ledger. A new
 * account without them is refused (422 `consent_required`).
 *
 * Bump a version here together with the copy in `auth.phone` and the backend
 * catalog (`internal/consent`), never alone: the server accepts only the
 * version in force.
 */
export const SIGNUP_CONSENT_VERSIONS: Readonly<Record<string, number>> = Object.freeze({
  terms: 1,
  health_data_processing: 1,
});

/** The verify-otp body: the consents ride along only when the phone screen recorded them. */
export function verifyOtpBody(
  mobile: string,
  code: string,
  consents: Record<string, number> | null,
): { mobile: string; code: string; consents?: Record<string, number> } {
  return consents ? { mobile, code, consents: { ...consents } } : { mobile, code };
}

/** One mandatory consent still to accept (GET /consents/pending), its text as the server has it in force. */
export interface PendingConsent {
  code: string;
  version: number;
  title: string;
  body: string;
  points: string[];
}

export const pendingConsentsSchema = z
  .object({
    consents: z.array(
      z.object({
        code: z.string(),
        version: z.number().int().positive(),
        title: z.string().nullable().optional(),
        body: z.string().nullable().optional(),
        points: z.array(z.string()).nullable().optional(),
        needs_consent: z.boolean().optional(),
      }),
    ),
  })
  .transform(({ consents }): PendingConsent[] =>
    consents
      .filter((c) => c.needs_consent !== false)
      .map((c) => ({
        code: c.code,
        version: c.version,
        title: c.title ?? '',
        body: c.body ?? '',
        points: c.points ?? [],
      })),
  );

/**
 * Whether the one-time sheet shows (DECISIONS.md H-18): something is pending,
 * the user did not put it off during this app open, and the app is not locked.
 * Nothing is blocked server-side either way.
 */
export function shouldShowPending(pending: readonly PendingConsent[] | undefined, dismissed: boolean): boolean {
  return !dismissed && (pending?.length ?? 0) > 0;
}
