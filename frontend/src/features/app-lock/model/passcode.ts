/**
 * Passcode hashing for the app lock (B-N1-12). The passcode never leaves the
 * device and is never stored: only a PBKDF2-SHA-256 hash with a random
 * per-device salt is kept (in localStorage, see `config.ts`). WebCrypto only —
 * no dependency, and the same code runs under Node for the unit tests.
 */

export const PASSCODE_MIN = 4;
export const PASSCODE_MAX = 6;

/** OWASP 2023 guidance for PBKDF2-HMAC-SHA256. ~100–250 ms on a mid phone. */
export const PBKDF2_ITERATIONS = 210_000;

export function isValidPasscode(value: string): boolean {
  return new RegExp(`^\\d{${PASSCODE_MIN},${PASSCODE_MAX}}$`).test(value);
}

export function toBase64(bytes: Uint8Array): string {
  let s = '';
  for (const b of bytes) s += String.fromCharCode(b);
  return btoa(s);
}

export function fromBase64(value: string): Uint8Array {
  const s = atob(value);
  const out = new Uint8Array(s.length);
  for (let i = 0; i < s.length; i++) out[i] = s.charCodeAt(i);
  return out;
}

export function randomSalt(length = 16): Uint8Array {
  const salt = new Uint8Array(length);
  crypto.getRandomValues(salt);
  return salt;
}

/** PBKDF2-SHA-256(passcode, salt, iterations) → 32 bytes, base64. */
export async function hashPasscode(passcode: string, salt: Uint8Array, iterations = PBKDF2_ITERATIONS): Promise<string> {
  const key = await crypto.subtle.importKey('raw', new TextEncoder().encode(passcode), 'PBKDF2', false, ['deriveBits']);
  const bits = await crypto.subtle.deriveBits(
    { name: 'PBKDF2', hash: 'SHA-256', salt: salt as BufferSource, iterations },
    key,
    256,
  );
  return toBase64(new Uint8Array(bits));
}

/** Compares two base64 digests without an early exit on the first difference. */
export function digestEquals(a: string, b: string): boolean {
  if (a.length !== b.length) return false;
  let diff = 0;
  for (let i = 0; i < a.length; i++) diff |= a.charCodeAt(i) ^ b.charCodeAt(i);
  return diff === 0;
}
