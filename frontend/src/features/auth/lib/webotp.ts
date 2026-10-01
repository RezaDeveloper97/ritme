'use client';

import { useEffect, useRef } from 'react';

import { toAsciiDigits } from '@/shared/lib/phone';

/**
 * Digits in the SMS code. The design draws 5 boxes, but `POST /auth/verify-otp`
 * validates `size:4` (backend-go/internal/auth) — the boxes follow the API.
 */
export const OTP_LENGTH = 4;

/** Keep only the digits of a pasted / autofilled value, Persian digits included. */
export function otpDigits(value: string, length = OTP_LENGTH): string {
  return toAsciiDigits(value).replace(/\D/g, '').slice(0, length);
}

interface OtpCredential {
  code?: string;
}

/**
 * WebOTP autofill: on browsers that support it (Chrome on Android) the code is
 * read from the incoming SMS once the user approves the system prompt. The SMS
 * needs the `@<host> #<code>` origin line for this to fire; the
 * `autocomplete="one-time-code"` input covers iOS. Aborted on unmount.
 */
export function useWebOtp(onCode: (code: string) => void, enabled = true): void {
  const cb = useRef(onCode);
  cb.current = onCode;

  useEffect(() => {
    if (!enabled || typeof window === 'undefined' || !('OTPCredential' in window)) return;
    const controller = new AbortController();
    const request = { otp: { transport: ['sms'] }, signal: controller.signal } as unknown as CredentialRequestOptions;
    navigator.credentials
      .get(request)
      .then((credential) => {
        const code = otpDigits((credential as OtpCredential | null)?.code ?? '');
        if (code) cb.current(code);
      })
      .catch(() => undefined);
    return () => controller.abort();
  }, [enabled]);
}
