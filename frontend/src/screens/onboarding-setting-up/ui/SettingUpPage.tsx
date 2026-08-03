'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useRef, useState } from 'react';

import { useOnboardingStore, type PregnancyBasis } from '@/entities/user';
import { useActivatePregnancy, useCompleteOnboarding, type OnboardingInput } from '@/entities/pregnancy';
import { datePartsToApiDate, onboardingToProfileInput, useUpdateProfile } from '@/features/edit-profile';
import { type Locale } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { getAuthToken, setAuthToken } from '@/shared/session';

const CIRCUMFERENCE = 553;

/**
 * Turn the collected dating basis into the pregnancy onboarding body. Returns
 * null when no source was chosen (the pregnancy screen's activation gate will
 * then prompt for it later). The stored parts are in `locale`'s calendar; they
 * cross the boundary as Gregorian (§7).
 */
function buildPregnancyOnboarding(basis: PregnancyBasis, locale: Locale): OnboardingInput | null {
  if (!basis.source) return null;
  const input: OnboardingInput = { age_source: basis.source };
  if (basis.source === 'lmp' && basis.lmp) {
    input.lmp_date = datePartsToApiDate(basis.lmp, locale);
  } else if (basis.source === 'ultrasound') {
    if (basis.ultrasoundDate) input.ultrasound_date = datePartsToApiDate(basis.ultrasoundDate, locale);
    if (basis.ultrasoundWeeks != null) input.ultrasound_weeks = basis.ultrasoundWeeks;
    input.ultrasound_days = basis.ultrasoundDays ?? 0;
  } else if (basis.source === 'manual') {
    if (basis.manualWeeks != null) input.manual_weeks = basis.manualWeeks;
    input.manual_days = basis.manualDays ?? 0;
  }
  return input;
}

export function SettingUpPage() {
  const t = useTranslations('onboarding.settingUp');
  const locale = useLocale() as Locale;
  const update = useUpdateProfile();
  const activate = useActivatePregnancy();
  const completeOnboarding = useCompleteOnboarding();
  const onboarding = useOnboardingStore();
  const isPregnant = onboarding.intention === 'pregnant';

  const [pct, setPct] = useState(0);
  const [ringDone, setRingDone] = useState(false);
  const [saveDone, setSaveDone] = useState(false);
  const ringRef = useRef<SVGCircleElement>(null);
  const savedRef = useRef(false);

  // Persist the collected answers exactly once. The guard keeps React 18
  // StrictMode's double-invoke from firing two POSTs. Pregnant users also get
  // pregnancy mode activated + dated inline; everyone else just saves cycle data.
  useEffect(() => {
    if (savedRef.current) return;
    savedRef.current = true;

    void (async () => {
      try {
        await update.mutateAsync(onboardingToProfileInput(onboarding));
        if (onboarding.intention === 'pregnant') {
          await activate.mutateAsync();
          const pregnancy = buildPregnancyOnboarding(onboarding.pregnancyBasis, onboarding.locale);
          if (pregnancy) await completeOnboarding.mutateAsync(pregnancy);
        }
      } catch {
        // Best-effort: proceed to the app regardless — the user can adjust
        // everything later, and the pregnancy screen re-prompts if needed.
      } finally {
        setSaveDone(true);
      }
    })();
    // Run once on mount; the persisted store values are stable by then.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // The progress ring is a reassurance animation, independent of the request.
  useEffect(() => {
    let p = 0;
    const id = setInterval(() => {
      p += 2;
      if (p > 100) {
        clearInterval(id);
        setRingDone(true);
        return;
      }
      setPct(p);
      if (ringRef.current) {
        ringRef.current.style.strokeDashoffset = String(CIRCUMFERENCE * (1 - p / 100));
      }
    }, 45);
    return () => clearInterval(id);
  }, []);

  // Leave only once the ring has finished AND the save has settled, so the
  // profile is written before the next screen refetches it. Pregnant users land
  // in pregnancy mode; everyone else on home.
  //
  // This last hop is a **full document navigation**, not `router.replace`: the
  // target is behind the auth middleware, and a client-side transition can be
  // answered from the App Router's cache — including a redirect-to-signup
  // cached from before the session cookie existed — which dumped freshly
  // registered users back on the phone-number screen even though reloading
  // took them straight in. The auth cookie is also re-asserted first, so the
  // request the middleware sees always carries it.
  useEffect(() => {
    if (!ringDone || !saveDone) return;
    document.cookie = 'ritme_onboarded=1; path=/; max-age=31536000; SameSite=Lax';
    const token = getAuthToken();
    if (token) setAuthToken(token);
    window.location.replace(`/${locale}${isPregnant ? '/pregnancy' : '/home'}`);
  }, [ringDone, saveDone, isPregnant, locale]);

  return (
    <div className="view onb-page">
      <div className="scroll setup-body">
        <div className="titr setup-titr">{t('title')}</div>
        <p className="sub setup-sub">{t('subtitle')}</p>

        <div className="setup-ring">
          <svg width="200" height="200" viewBox="0 0 200 200">
            <circle cx="100" cy="100" r="88" fill="none" stroke="var(--field-border)" strokeWidth="13" />
            <circle
              ref={ringRef}
              cx="100" cy="100" r="88" fill="none"
              stroke="var(--pink)" strokeWidth="13" strokeLinecap="round"
              strokeDasharray={CIRCUMFERENCE} strokeDashoffset={CIRCUMFERENCE}
            />
          </svg>
          <div className="setup-ring-pct">
            {formatNumber(pct, locale)}٪
          </div>
        </div>

        <p className="sub setup-note">
          {t('disclaimer')}
        </p>
      </div>
      <div className="setup-footer">
        <span className="sub">{t('wait')}</span>
      </div>
    </div>
  );
}
