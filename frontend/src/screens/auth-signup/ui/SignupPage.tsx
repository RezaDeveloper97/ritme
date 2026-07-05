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
    <div className="view" style={{ background: '#fff' }}>

      {/*
       * RTL (fa): first child → RIGHT, second child → LEFT.
       * NavBack goes first so it sits on the RIGHT side in RTL. ✓
       */}
      <div className="hdr">
        <NavBack onClick={() => router.replace('/splash')} />
        <span />
      </div>

      <div className="scroll" style={{ padding: '18px 22px' }}>
        {/* Figma «Titr»: 16px bold + sparkle, right-aligned in RTL */}
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'flex-end', gap: 8, marginTop: 8 }}>
          <span className="titr" style={{ fontSize: 16 }}>{t('signup.title')}</span>
          <span style={{ color: 'var(--ritme-pink)' }}>
            <Icon name="sparkle" size={22} fill="currentColor" strokeWidth={0} />
          </span>
        </div>

        <p className="sub" style={{ fontSize: 14, textAlign: 'start', margin: '12px 0 32px' }}>{t('signup.subtitle')}</p>

        <label className="lbl">{t('signup.phoneLabel')}</label>
        <div className="field">
          <input
            inputMode="numeric"
            maxLength={11}
            placeholder={t('signup.phonePlaceholder')}
            value={toPersianDigits(phone)}
            onChange={e => handlePhone(e.target.value)}
            dir="ltr"
            style={{ textAlign: 'start' }}
          />
          <span style={{ color: '#A9B2BC' }}>
            <Icon name="user" size={18} />
          </span>
        </div>

        {/* Terms checkbox */}
        <div
          role="button"
          tabIndex={0}
          style={{
            display: 'flex', alignItems: 'flex-start', gap: 10,
            marginTop: 26, cursor: 'pointer', justifyContent: 'flex-end',
          }}
          onClick={() => setTerms(v => !v)}
          onKeyDown={e => e.key === 'Enter' && setTerms(v => !v)}
        >
          <span className="sub" style={{ textAlign: 'start', flex: 1 }}>
            ثبت‌نام به منزله‌ی قبول{' '}
            <b style={{ color: 'var(--brand)', fontWeight: 700 }}>{t('signup.termsLink')}</b>{' '}
            و حریم خصوصی است.
          </span>
          <span className={`cbx pink${terms ? ' on' : ''}`}>
            <Icon name="check" size={13} stroke="#fff" />
          </span>
        </div>

        {sendOtp.isError && (
          <p className="sub" style={{ color: '#E5484D', textAlign: 'start', marginTop: 16 }}>
            {t(`errors.${authErrorKey(sendOtp.error)}`)}
          </p>
        )}
      </div>

      <div style={{ padding: '14px 16px 8px' }}>
        <button className="btn btn-primary" disabled={!isValid || sendOtp.isPending} onClick={handleSubmit}>
          {sendOtp.isPending ? t('signup.sending') : t('signup.submit')}
        </button>
      </div>

    </div>
  );
}
