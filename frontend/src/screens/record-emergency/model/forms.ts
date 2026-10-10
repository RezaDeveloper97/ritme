import { toAsciiDigits } from '@/shared/lib/phone';

/*
 * Client-side checks mirroring PUT /health-record/emergency-card (D-72): phone 4–15 digits (Persian digits
 * accepted, separators dropped), insurance takes exactly the last 4 digits. The server stays the authority.
 */

/** Digits only (Persian / Arabic-Indic digits converted), a leading `+` kept. */
export function normalizePhone(input: string): string {
  const ascii = toAsciiDigits(input.trim());
  const plus = ascii.startsWith('+') ? '+' : '';
  return plus + ascii.replace(/\D/g, '');
}

export function isValidPhone(input: string): boolean {
  const digits = normalizePhone(input).replace(/^\+/, '');
  return digits.length >= 4 && digits.length <= 15;
}

export function normalizeLast4(input: string): string {
  return toAsciiDigits(input).replace(/\D/g, '');
}

export function isValidLast4(input: string): boolean {
  return /^\d{4}$/.test(normalizeLast4(input));
}

/** The public link's absolute URL from the server's `/{locale}/emergency/{token}` path. */
export function emergencyLinkUrl(origin: string, path: string): string {
  return `${origin.replace(/\/+$/, '')}${path}`;
}
