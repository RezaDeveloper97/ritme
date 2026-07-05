'use client';

import { useTranslations } from 'next-intl';
import { useEffect, useRef, useState } from 'react';

import { useRouter } from '@/shared/i18n';
import { Icon, NavBack } from '@/shared/ui';
import { toAsciiDigits, toPersianDigits } from '@/shared/lib/phone';
import { useOnboardingStore } from '@/entities/user';
import { authErrorKey, useSendOtp, useVerifyOtp } from '@/features/auth';

export function OtpPage() {
  const t = useTranslations('auth');
  const router = useRouter();
  const phone = useOnboardingStore(s => s.phone);

  const verifyOtp = useVerifyOtp();
  const sendOtp = useSendOtp();

  const [digits, setDigits] = useState(['', '', '', '']);
  const [seconds, setSeconds] = useState(59);
  const [canResend, setCanResend] = useState(false);
  const inputRefs = [
    useRef<HTMLInputElement>(null),
    useRef<HTMLInputElement>(null),
    useRef<HTMLInputElement>(null),
    useRef<HTMLInputElement>(null),
  ];

  const code = digits.join('');
  const isComplete = code.length === 4;

  useEffect(() => {
    inputRefs[0].current?.focus();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    if (seconds <= 0) { setCanResend(true); return; }
    const id = setTimeout(() => setSeconds(s => s - 1), 1000);
    return () => clearTimeout(id);
  }, [seconds]);

  const handleInput = (i: number, v: string) => {
    // accept Persian digits too, then keep a single ASCII digit
    const cleaned = toAsciiDigits(v).replace(/[^0-9]/g, '').slice(0, 1);
    const next = [...digits];
    next[i] = cleaned;
    setDigits(next);
    if (cleaned && i < 3) inputRefs[i + 1].current?.focus();
  };

  const handleKeyDown = (i: number, e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Backspace' && !digits[i] && i > 0) inputRefs[i - 1].current?.focus();
  };

  const handleVerify = () => {
    if (!isComplete || verifyOtp.isPending) return;
    verifyOtp.mutate(
      { mobile: phone, code },
      {
        onSuccess: ({ newUser }) => {
          // New accounts continue into onboarding; returning users go home.
          router.replace(newUser ? '/onboarding/name' : '/home');
        },
        onError: () => {
          setDigits(['', '', '', '']);
          inputRefs[0].current?.focus();
        },
      },
    );
  };

  const handleResend = () => {
    if (sendOtp.isPending) return;
    sendOtp.mutate(phone, {
      onSuccess: () => {
        setSeconds(59);
        setCanResend(false);
        setDigits(['', '', '', '']);
        inputRefs[0].current?.focus();
      },
    });
  };

  const pad = (n: number) => String(n).padStart(2, '0');

  return (
    <div className="view" style={{ background: '#fff' }}>

      <div className="hdr">
        <NavBack onClick={() => router.back()} />
        <span />
      </div>

      <div className="scroll" style={{ padding: '18px 22px', display: 'flex', flexDirection: 'column', alignItems: 'center' }}>
        <div style={{ width: '100%', textAlign: 'start' }}>
          <div className="titr">{t('otp.title')}</div>
          <p className="sub" style={{ margin: '12px 0 0' }}>
            {t('otp.subtitle', { phone: toPersianDigits(phone || '۰۹۱۲۳۴۵۶۷۸۹') })}
          </p>
        </div>

        <button
          className="btn-soft"
          style={{ width: 'auto', padding: '0 14px', margin: '16px 0 30px', display: 'flex', alignItems: 'center', gap: 6 }}
          onClick={() => router.back()}
        >
          <Icon name="pencil" size={15} />
          {t('otp.editPhone')}
        </button>

        {/* OTP inputs — LTR so digit order is left-to-right */}
        <div dir="ltr" style={{ display: 'flex', gap: 10, justifyContent: 'center' }}>
          {digits.map((d, i) => (
            <input
              key={i}
              ref={inputRefs[i]}
              className={`otp-box${d ? ' filled' : ''}`}
              inputMode="numeric"
              maxLength={1}
              value={d}
              onChange={e => handleInput(i, e.target.value)}
              onKeyDown={e => handleKeyDown(i, e)}
            />
          ))}
        </div>

        {verifyOtp.isError && (
          <p className="sub" style={{ color: '#E5484D', marginTop: 18 }}>
            {t(`errors.${authErrorKey(verifyOtp.error)}`)}
          </p>
        )}

        <div className="sub" style={{ marginTop: 26, display: 'flex', alignItems: 'center', gap: 6 }}>
          {canResend ? (
            <b style={{ color: 'var(--brand)', cursor: 'pointer' }} onClick={handleResend}>
              {t('otp.resend')}
            </b>
          ) : (
            <span>
              {t('otp.resendTimer', { time: `۰۰:${toPersianDigits(pad(seconds))}` })}
            </span>
          )}
        </div>
      </div>

      <div style={{ padding: '14px 16px 8px' }}>
        <button className="btn btn-primary" disabled={!isComplete || verifyOtp.isPending} onClick={handleVerify}>
          {verifyOtp.isPending ? t('otp.verifying') : t('otp.submit')}
        </button>
      </div>

    </div>
  );
}
