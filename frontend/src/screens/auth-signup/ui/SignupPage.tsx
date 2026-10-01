'use client';

import { useTranslations } from 'next-intl';
import { useState } from 'react';

import { useRouter } from '@/shared/i18n';
import { Checkbox, Icon } from '@/shared/ui';
import { isValidMobile, normalizeMobile } from '@/shared/lib/phone';
import { useOnboardingStore } from '@/entities/user';
import { authErrorKey, OnbFrame, OTP_LENGTH, useSendOtp } from '@/features/auth';

/** `09123456789` → `912 345 6789` (the +98 box carries the country code). */
function displayNational(mobile: string): string {
  const national = mobile.startsWith('0') ? mobile.slice(1) : mobile;
  return [national.slice(0, 3), national.slice(3, 6), national.slice(6)].filter(Boolean).join(' ');
}

/**
 * nbl_Onb_Phone: +98 mobile number and the two consents (terms + privacy,
 * processing of self-entered health data). Both consents are required to ask
 * for a code; the API has no consent codes for them yet, so they gate the
 * button only (bloom/QUESTIONS — see the B-N2-02 report).
 */
export function SignupPage() {
  const t = useTranslations('auth.phone');
  const te = useTranslations('auth.errors');
  const router = useRouter();
  const setPhone = useOnboardingStore((s) => s.setPhone);
  const sendOtp = useSendOtp();

  // ASCII digits in the local `0…` form, for reliable validation.
  const [phone, setPhoneLocal] = useState('');
  const [terms, setTerms] = useState(false);
  const [healthData, setHealthData] = useState(false);

  const isValid = isValidMobile(phone);
  const canSubmit = isValid && terms && healthData && !sendOtp.isPending;
  const showInvalid = phone.length >= 11 && !isValid;

  const handleSubmit = () => {
    if (!canSubmit) return;
    sendOtp.mutate(phone, {
      onSuccess: () => {
        setPhone(phone);
        router.push('/otp');
      },
    });
  };

  return (
    <OnbFrame
      progress={{ current: 1, total: 8 }}
      onBack={() => router.replace('/splash')}
      title={t('title')}
      subtitle={t('subtitle', { count: OTP_LENGTH })}
      primary={{
        label: sendOtp.isPending ? t('sending') : t('submit'),
        onClick: handleSubmit,
        disabled: !canSubmit,
        loading: sendOtp.isPending,
      }}
    >
      <div className="onb2-phone" dir="ltr">
        <span className="onb2-cc" aria-hidden>
          <span className="onb2-flag">🇮🇷</span>
          +98
        </span>
        <input
          className="onb2-input onb2-phone-input"
          type="tel"
          inputMode="numeric"
          autoComplete="tel-national"
          aria-label={t('label')}
          aria-invalid={showInvalid || undefined}
          placeholder="912 345 6789"
          value={displayNational(phone)}
          onChange={(e) => setPhoneLocal(normalizeMobile(e.target.value))}
          onKeyDown={(e) => e.key === 'Enter' && handleSubmit()}
        />
      </div>
      {showInvalid ? <p className="onb2-error" role="alert">{t('invalid')}</p> : null}

      <section className="nb-card onb2-consent" aria-labelledby="onb2-consent-title">
        <h2 id="onb2-consent-title" className="onb2-consent-title">
          <Icon name="shield" size={20} className="onb2-consent-icon" />
          {t('consentTitle')}
        </h2>
        <Checkbox
          checked={terms}
          onCheckedChange={setTerms}
          label={t.rich('terms', {
            terms: (chunks) => <span className="onb2-em">{chunks}</span>,
            privacy: (chunks) => <span className="onb2-em">{chunks}</span>,
          })}
        />
        <Checkbox checked={healthData} onCheckedChange={setHealthData} label={t('healthData')} />
      </section>

      {sendOtp.isError ? (
        <p className="onb2-error" role="alert">
          {authErrorKey(sendOtp.error) === 'invalidCode' ? t('invalid') : te(authErrorKey(sendOtp.error))}
        </p>
      ) : null}
    </OnbFrame>
  );
}
