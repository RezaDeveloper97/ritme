'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useCallback, useEffect, useRef, useState } from 'react';

import { type Locale, useRouter } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { Icon } from '@/shared/ui';
import { getOnboardingPending } from '@/shared/session';
import { isOnboardingStep, onboardingRoute, useOnboardingStore } from '@/entities/user';
import { authErrorKey, OnbFrame, OTP_LENGTH, otpDigits, useSendOtp, useVerifyOtp, useWebOtp } from '@/features/auth';

const RESEND_SECONDS = 59;
const empty = () => Array.from({ length: OTP_LENGTH }, () => '');

/**
 * What the user put into a box: typing into a filled box replaces its digit
 * (the old one is dropped from either end), a paste / autofill keeps them all.
 */
function typedDigits(value: string, previous: string): string {
  const raw = otpDigits(value);
  if (!previous || raw.length < 2) return raw;
  if (raw.length === 2 && raw.startsWith(previous)) return raw.slice(1);
  if (raw.length === 2 && raw.endsWith(previous)) return raw.slice(0, 1);
  return raw;
}

/** `09123456789` → `0912 345 6789`. */
const groupMobile = (m: string) => [m.slice(0, 4), m.slice(4, 7), m.slice(7)].filter(Boolean).join(' ');

/**
 * nbl_Onb_OTP: one box per digit, WebOTP autofill (+ `one-time-code` for iOS),
 * paste of the whole code into any box, and a resend countdown.
 */
export function OtpPage() {
  const t = useTranslations('auth.code');
  const te = useTranslations('auth.errors');
  const router = useRouter();
  const locale = useLocale() as Locale;
  const phone = useOnboardingStore((s) => s.phone);

  const verifyOtp = useVerifyOtp();
  const sendOtp = useSendOtp();

  const [digits, setDigits] = useState<string[]>(empty);
  const [seconds, setSeconds] = useState(RESEND_SECONDS);
  const inputRefs = useRef<(HTMLInputElement | null)[]>([]);
  const focusBox = (i: number) => inputRefs.current[Math.max(0, Math.min(OTP_LENGTH - 1, i))]?.focus();

  const code = digits.join('');
  const isComplete = code.length === OTP_LENGTH;

  // No number to verify (reload after the store was cleared) → back to the phone screen.
  // Decided once the persisted store has hydrated (it holds the number).
  useEffect(() => {
    const check = () => {
      if (!useOnboardingStore.getState().phone) router.replace('/signup');
      else focusBox(0);
    };
    const persist = useOnboardingStore.persist;
    if (persist.hasHydrated()) check();
    else return persist.onFinishHydration(check);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    if (seconds <= 0) return;
    const id = setTimeout(() => setSeconds((s) => s - 1), 1000);
    return () => clearTimeout(id);
  }, [seconds]);

  const verify = useCallback(
    (value: string) => {
      if (value.length !== OTP_LENGTH || verifyOtp.isPending || !phone) return;
      verifyOtp.mutate(
        { mobile: phone, code: value },
        {
          onSuccess: ({ profileCompleted }) => {
            // Gate on whether registration was ever *finished* (an existing
            // user is never sent through onboarding again). Full document
            // navigation across the auth boundary, so no App Router cache from
            // the signed-out session can bounce the visitor back here.
            const pending = getOnboardingPending();
            const next =
              profileCompleted || !pending || !isOnboardingStep(pending) ? '/home' : onboardingRoute(pending);
            window.location.replace(`/${locale}${next}`);
          },
          onError: () => {
            setDigits(empty());
            focusBox(0);
          },
        },
      );
    },
    [verifyOtp, phone, locale],
  );

  const fill = (value: string, at: number) => {
    // A whole code (paste, autofill) always lands from the first box.
    const from = value.length === OTP_LENGTH ? 0 : at;
    const incoming = value.slice(0, OTP_LENGTH - from);
    const next = [...digits];
    if (!incoming) {
      next[from] = '';
      setDigits(next);
      return;
    }
    for (let k = 0; k < incoming.length; k += 1) next[from + k] = incoming[k]!;
    setDigits(next);
    focusBox(from + incoming.length);
    const joined = next.join('');
    if (joined.length === OTP_LENGTH) verify(joined);
  };

  useWebOtp((received) => {
    setDigits(received.padEnd(OTP_LENGTH, ' ').split('').map((d) => d.trim()));
    verify(received);
  }, Boolean(phone));

  const handleKeyDown = (i: number, e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Backspace' && !digits[i] && i > 0) focusBox(i - 1);
    if (e.key === 'Enter') verify(code);
  };

  const handleResend = () => {
    if (sendOtp.isPending || !phone) return;
    sendOtp.mutate(phone, {
      onSuccess: () => {
        setSeconds(RESEND_SECONDS);
        setDigits(empty());
        focusBox(0);
      },
    });
  };

  const clock = `${formatNumber(Math.floor(seconds / 60), locale)}:${formatNumber(String(seconds % 60).padStart(2, '0'), locale)}`;
  const error = verifyOtp.isError ? verifyOtp.error : sendOtp.isError ? sendOtp.error : null;

  return (
    <OnbFrame
      progress={{ current: 2, total: 8 }}
      onBack={() => router.replace('/signup')}
      title={t('title')}
      subtitle={t.rich('subtitle', {
        count: OTP_LENGTH,
        number: groupMobile(phone),
        phone: (chunks) => (
          <bdi dir="ltr" className="onb2-strong">
            {chunks}
          </bdi>
        ),
      })}
      primary={{
        label: verifyOtp.isPending ? t('verifying') : t('submit'),
        onClick: () => verify(code),
        disabled: !isComplete || verifyOtp.isPending,
        loading: verifyOtp.isPending,
      }}
    >
      <div dir="ltr" className="onb2-otp" role="group" aria-label={t('boxesLabel')}>
        {digits.map((d, i) => (
          <input
            key={i}
            ref={(el) => {
              inputRefs.current[i] = el;
            }}
            className={`onb2-otp-box${d ? ' is-filled' : ''}`}
            inputMode="numeric"
            autoComplete={i === 0 ? 'one-time-code' : 'off'}
            aria-label={t('digitLabel', { n: i + 1 })}
            value={d ? formatNumber(d, locale) : ''}
            onChange={(e) => fill(typedDigits(e.target.value, d), i)}
            onPaste={(e) => {
              e.preventDefault();
              fill(otpDigits(e.clipboardData.getData('text')), i);
            }}
            onKeyDown={(e) => handleKeyDown(i, e)}
          />
        ))}
      </div>

      <div className="onb2-otp-meta">
        <button type="button" className="onb2-link" onClick={() => router.replace('/signup')}>
          {t('editPhone')}
        </button>
        {seconds > 0 ? (
          <span className="onb2-timer" aria-live="off">
            {t('resendIn')} <b>{clock}</b>
          </span>
        ) : (
          <button type="button" className="onb2-link" onClick={handleResend} disabled={sendOtp.isPending}>
            {t('resend')}
          </button>
        )}
      </div>

      {error ? (
        <p className="onb2-error" role="alert">
          {te(authErrorKey(error))}
        </p>
      ) : null}

      <div className="nb-card onb2-hint">
        <Icon name="lock" size={18} className="onb2-hint-icon" />
        <span>{t('autoRead')}</span>
      </div>
    </OnbFrame>
  );
}
