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
  const [terms, setTerms] = useState(false);

  const isValid = isValidMobile(phone) && terms;

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

        {/* Terms checkbox */}
        <div
          role="button"
          tabIndex={0}
          className="signup-terms"
          onClick={() => setTerms(v => !v)}
          onKeyDown={e => e.key === 'Enter' && setTerms(v => !v)}
        >
          <span className="sub signup-terms-t">
            ثبت‌نام به منزله‌ی قبول{' '}
            <b className="signup-terms-link">{t('signup.termsLink')}</b>{' '}
            و حریم خصوصی است.
          </span>
          <span className={`cbx pink${terms ? ' on' : ''}`}>
            <Icon name="check" size={13} stroke="var(--on-accent)" />
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
