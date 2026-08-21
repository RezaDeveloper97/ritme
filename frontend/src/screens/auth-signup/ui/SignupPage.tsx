'use client';

import { useTranslations } from 'next-intl';
import Image from 'next/image';
import { useState } from 'react';

import { useRouter } from '@/shared/i18n';
import { Icon, NavBack } from '@/shared/ui';
import { isValidMobile, normalizeMobile, toPersianDigits } from '@/shared/lib/phone';
import { useOnboardingStore } from '@/entities/user';
import { authErrorKey, useSendOtp } from '@/features/auth';

export function SignupPage() {
  const t = useTranslations('auth');
  const router = useRouter();
  const setPhone = useOnboardingStore(s => s.setPhone);
  const sendOtp = useSendOtp();

  // phone is always stored as ASCII digits for reliable validation
  const [phone, setPhoneLocal] = useState('');

  // Terms consent lives on the first onboarding step instead: at this point we
  // still don't know whether this number belongs to an existing account, and
  // returning users shouldn't have to re-accept on every sign-in.
  const isValid = isValidMobile(phone);

  const handlePhone = (v: string) => {
    setPhoneLocal(normalizeMobile(v));
  };

  const handleSubmit = () => {
    if (!isValid || sendOtp.isPending) return;
    sendOtp.mutate(phone, {
      onSuccess: () => {
        setPhone(phone);
        router.push('/otp');
      },
    });
  };

  return (
    <div className="view onb-page">

      {/*
       * RTL (fa): first child → RIGHT, second child → LEFT.
       * NavBack goes first so it sits on the RIGHT side in RTL. ✓
       */}
      <div className="hdr">
        <NavBack onClick={() => router.replace('/splash')} />
        <span />
      </div>

      <div className="scroll auth-body">
        {/* Brand mark: logo floating in a soft gradient halo, above the title */}
        <div className="signup-brand">
          <span aria-hidden className="signup-halo" />
          <Image
            src="/logo.webp"
            alt=""
            aria-hidden
            width={72}
            height={72}
            priority
            className="signup-logo"
          />
        </div>

        {/* Titr: right-aligned in RTL (text-align: start), sparkle trails the words */}
        <h1 className="signup-titr">
          {t('signup.title')}
          <span className="signup-spark">
            <Icon name="sparkle" size={20} fill="currentColor" strokeWidth={0} />
          </span>
        </h1>

        <p className="sub signup-sub">{t('signup.subtitle')}</p>

        <label className="lbl signup-lbl">{t('signup.phoneLabel')}</label>
        <div className="field signup-field">
          <input
            inputMode="numeric"
            maxLength={11}
            placeholder={t('signup.phonePlaceholder')}
            value={toPersianDigits(phone)}
            onChange={e => handlePhone(e.target.value)}
            dir="ltr"
            className="signup-input"
          />
          <span className="placeholder-soft">
            <Icon name="user" size={18} />
          </span>
        </div>

        {/* Trust cue: this is health data, so say plainly that it stays private (§11) */}
        <p className="signup-privacy">
          <Icon name="shield" size={16} />
          <span>{t('signup.privacyNote')}</span>
        </p>

        {sendOtp.isError && (
          <p className="sub auth-error">
            {t(`errors.${authErrorKey(sendOtp.error)}`)}
          </p>
        )}
      </div>

      <div className="onb-actions">
        <button className="btn btn-primary" disabled={!isValid || sendOtp.isPending} onClick={handleSubmit}>
          {sendOtp.isPending ? t('signup.sending') : t('signup.submit')}
        </button>
      </div>

    </div>
  );
}
