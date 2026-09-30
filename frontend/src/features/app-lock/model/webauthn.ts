'use client';

/**
 * Optional biometric unlock through the platform authenticator (Face ID,
 * fingerprint, Windows Hello) — WebAuthn with `userVerification: 'required'`.
 *
 * It is a *local presence check*, like the passcode: the credential never
 * leaves the device and nothing is verified server-side (the lock guards this
 * device's session, not the account). A random challenge per call; the
 * credential id is kept in the lock config.
 */

function b64url(bytes: ArrayBuffer): string {
  let s = '';
  for (const b of new Uint8Array(bytes)) s += String.fromCharCode(b);
  return btoa(s).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
}

function fromB64url(value: string): Uint8Array {
  const s = atob(value.replace(/-/g, '+').replace(/_/g, '/') + '==='.slice((value.length + 3) % 4));
  const out = new Uint8Array(s.length);
  for (let i = 0; i < s.length; i++) out[i] = s.charCodeAt(i);
  return out;
}

function random(n: number): Uint8Array {
  const b = new Uint8Array(n);
  crypto.getRandomValues(b);
  return b;
}

/** Whether this device offers a user-verifying platform authenticator. */
export async function isBiometricAvailable(): Promise<boolean> {
  try {
    if (typeof window === 'undefined' || !window.PublicKeyCredential || !window.isSecureContext) return false;
    return await PublicKeyCredential.isUserVerifyingPlatformAuthenticatorAvailable();
  } catch {
    return false;
  }
}

/** Create a platform credential for the lock; resolves to its id, or null when cancelled/failed. */
export async function registerBiometric(displayName: string): Promise<string | null> {
  try {
    const cred = (await navigator.credentials.create({
      publicKey: {
        rp: { name: displayName },
        user: { id: random(16) as BufferSource, name: 'ritme-app-lock', displayName },
        challenge: random(32) as BufferSource,
        pubKeyCredParams: [
          { type: 'public-key', alg: -7 },
          { type: 'public-key', alg: -257 },
        ],
        authenticatorSelection: {
          authenticatorAttachment: 'platform',
          userVerification: 'required',
          residentKey: 'discouraged',
        },
        attestation: 'none',
        timeout: 60_000,
      },
    })) as PublicKeyCredential | null;
    return cred ? b64url(cred.rawId) : null;
  } catch {
    return null;
  }
}

/** Ask the platform authenticator to verify the user with the lock's credential. */
export async function verifyBiometric(credentialId: string): Promise<boolean> {
  try {
    const cred = await navigator.credentials.get({
      publicKey: {
        challenge: random(32) as BufferSource,
        allowCredentials: [{ type: 'public-key', id: fromB64url(credentialId) as BufferSource }],
        userVerification: 'required',
        timeout: 60_000,
      },
    });
    return cred !== null;
  } catch {
    return false;
  }
}
