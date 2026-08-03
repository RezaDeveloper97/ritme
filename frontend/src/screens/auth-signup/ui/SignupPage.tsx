'use client';

import { useTranslations } from 'next-intl';
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
        {/* Figma «Titr»: 16px bold + sparkle, right-aligned in RTL */}
        <div className="signup-titr-row">
          <span className="titr signup-titr">{t('signup.title')}</span>
          <span className="signup-spark">
            <Icon name="sparkle" size={22} fill="currentColor" strokeWidth={0} />
          </span>
        </div>

        <p className="sub signup-sub">{t('signup.subtitle')}</p>

        <label className="lbl">{t('signup.phoneLabel')}</label>
        <div className="field">
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
