'use client';

import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { useRouter, useSearchParams } from 'next/navigation';
import { useEffect, useState, type FormEvent } from 'react';

import { formatMobile, normalizeMobile, toPersianDigits } from '@/shared/lib';
import { hasAuthToken } from '@/shared/session';
import { Button, Icon, IconButton } from '@/shared/ui';

import { sendOtp, verifyOtp } from '../api/otp-api';
import { OTP_LENGTH, RESEND_AFTER_SECONDS, retryAfterSeconds, sendErrorKey, verifyErrorKey } from '../model/otp';
import { safeNext } from '../model/safe-next';
import { useCountdown } from '../model/use-countdown';
import { OtpBoxes } from './OtpBoxes';

/** Two steps on one screen: mobile → 4-digit code. */
export function LoginFlow() {
  const t = useTranslations('login');
  const router = useRouter();
  const params = useSearchParams();
  const queryClient = useQueryClient();
  const next = safeNext(params.get('next'));

  const [step, setStep] = useState<'phone' | 'code'>('phone');
  const [phone, setPhone] = useState('');
  const [mobile, setMobile] = useState('');
  const [code, setCode] = useState('');
  const [error, setError] = useState<string | null>(null);
  const countdown = useCountdown();

  // Already signed in: the gate decides where to go.
  useEffect(() => {
    if (hasAuthToken()) router.replace(next);
  }, [router, next]);

  const send = useMutation({
    mutationFn: (m: string) => sendOtp(m),
    onSuccess: (_d, m) => {
      setMobile(m);
      setCode('');
      setError(null);
      setStep('code');
      countdown.start(RESEND_AFTER_SECONDS);
    },
    onError: (err, m) => {
      const key = sendErrorKey(err);
      if (key === 'wait') {
        // A code is already on its way: go to the code step with the remaining wait.
        setMobile(m);
        setStep('code');
        countdown.start(retryAfterSeconds(err));
        setError(null);
        return;
      }
      setError(t(`errors.${key}`));
    },
  });

  const verify = useMutation({
    mutationFn: (c: string) => verifyOtp(mobile, c),
    onSuccess: () => {
      queryClient.clear();
      router.replace(next);
    },
    onError: (err) => {
      setCode('');
      setError(t(`errors.${verifyErrorKey(err)}`));
    },
  });

  const submitPhone = (e: FormEvent) => {
    e.preventDefault();
    const m = normalizeMobile(phone);
    if (!m) {
      setError(t('errors.invalidMobile'));
      return;
    }
    setError(null);
    send.mutate(m);
  };

  const onCode = (value: string) => {
    setCode(value);
    setError(null);
    if (value.length === OTP_LENGTH && !verify.isPending) verify.mutate(value);
  };

  const submitCode = (e: FormEvent) => {
    e.preventDefault();
    if (code.length === OTP_LENGTH) verify.mutate(code);
  };

  if (step === 'code') {
    return (
      <form className="auth-card" onSubmit={submitCode} noValidate>
        <div className="auth-card__top">
          <IconButton
            label={t('editNumber')}
            onClick={() => {
              setStep('phone');
              setError(null);
            }}
          >
            <Icon name="back" />
          </IconButton>
        </div>
        <h1 className="auth-card__title">{t('codeTitle')}</h1>
        <p className="auth-card__lead">
          {t.rich('codeLead', {
            length: toPersianDigits(OTP_LENGTH),
            mobile: () => (
              <bdi className="auth-card__mobile" dir="ltr">
                {formatMobile(mobile)}
              </bdi>
            ),
          })}
        </p>
        <OtpBoxes
          value={code}
          onChange={onCode}
          label={t('codeLabel')}
          invalid={Boolean(error)}
          disabled={verify.isPending}
        />
        <div className="auth-card__row">
          <button
            type="button"
            className="link-btn"
            onClick={() => {
              setStep('phone');
              setError(null);
            }}
          >
            {t('editNumber')}
          </button>
          {countdown.left > 0 ? (
            <span className="auth-card__timer">
              {t('resendIn', { time: toPersianDigits(`0:${String(countdown.left).padStart(2, '0')}`) })}
            </span>
          ) : (
            <button
              type="button"
              className="link-btn"
              disabled={send.isPending}
              onClick={() => send.mutate(mobile)}
            >
              {t('resend')}
            </button>
          )}
        </div>
        {error ? (
          <p className="form-error" role="alert">
            {error}
          </p>
        ) : null}
        <div className="auth-card__spacer" />
        <p className="auth-card__note">
          <Icon name="shield" size={18} />
          <span>{t('privacy')}</span>
        </p>
        <Button type="submit" block loading={verify.isPending} disabled={code.length !== OTP_LENGTH}>
          {t('verify')}
        </Button>
      </form>
    );
  }

  return (
    <form className="auth-card" onSubmit={submitPhone} noValidate>
      <div className="auth-card__brand">
        <span className="brand-mark" aria-hidden="true">
          <Icon name="sparkle" />
        </span>
        <span className="auth-card__eyebrow">{t('eyebrow')}</span>
      </div>
      <h1 className="auth-card__title">{t('phoneTitle')}</h1>
      <p className="auth-card__lead">{t('phoneLead')}</p>
      <div className={error ? 'phone-field is-invalid' : 'phone-field'} dir="ltr">
        <span className="phone-field__prefix" aria-hidden="true">
          +98
        </span>
        <input
          className="phone-field__input"
          type="tel"
          inputMode="numeric"
          autoComplete="tel-national"
          placeholder="0912 345 6789"
          aria-label={t('phoneLabel')}
          aria-invalid={error ? true : undefined}
          value={phone}
          onChange={(e) => {
            setPhone(e.target.value);
            setError(null);
          }}
          autoFocus
          maxLength={16}
        />
      </div>
      {error ? (
        <p className="form-error" role="alert">
          {error}
        </p>
      ) : (
        <p className="auth-card__hint">{t('phoneHint')}</p>
      )}
      <div className="auth-card__spacer" />
      <p className="auth-card__note">
        <Icon name="shield" size={18} />
        <span>{t('privacy')}</span>
      </p>
      <Button type="submit" block loading={send.isPending}>
        {t('sendCode')}
      </Button>
    </form>
  );
}
