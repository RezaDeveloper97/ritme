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
