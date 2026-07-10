'use client';

import { useTranslations } from 'next-intl';
import { useEffect, useRef, useState } from 'react';

import { useOnboardingStore, type PregnancyBasis } from '@/entities/user';
import { useActivatePregnancy, useCompleteOnboarding, type OnboardingInput } from '@/entities/pregnancy';
import { jalaliPartsToApiDate, onboardingToProfileInput, useUpdateProfile } from '@/features/edit-profile';
import { useRouter } from '@/shared/i18n';

const FA = ['۰','۱','۲','۳','۴','۵','۶','۷','۸','۹'];
const faNum = (n: string | number) => String(n).replace(/[0-9]/g, d => FA[Number(d)]);

const CIRCUMFERENCE = 553;

/**
 * Turn the collected dating basis into the pregnancy onboarding body. Returns
 * null when no source was chosen (the pregnancy screen's activation gate will
 * then prompt for it later). Dates cross the boundary as Gregorian (§7).
 */
function buildPregnancyOnboarding(basis: PregnancyBasis): OnboardingInput | null {
  if (!basis.source) return null;
  const input: OnboardingInput = { age_source: basis.source };
  if (basis.source === 'lmp' && basis.lmp) {
    input.lmp_date = jalaliPartsToApiDate(basis.lmp);
  } else if (basis.source === 'ultrasound') {
    if (basis.ultrasoundDate) input.ultrasound_date = jalaliPartsToApiDate(basis.ultrasoundDate);
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
  const router = useRouter();
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
          const pregnancy = buildPregnancyOnboarding(onboarding.pregnancyBasis);
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
  useEffect(() => {
    if (!ringDone || !saveDone) return;
    document.cookie = 'ritme_onboarded=1; path=/; max-age=31536000; SameSite=Lax';
    router.replace(isPregnant ? '/pregnancy' : '/home');
  }, [ringDone, saveDone, isPregnant, router]);

  return (
    <div className="view" style={{ background: '#fff' }}>
      <div className="scroll" style={{ padding: '80px 22px 0', display: 'flex', flexDirection: 'column', alignItems: 'center', textAlign: 'center' }}>
        <div className="titr" style={{ fontSize: 19, lineHeight: 1.7 }}>{t('title')}</div>
        <p className="sub" style={{ margin: '12px 0 40px' }}>{t('subtitle')}</p>

        <div style={{ position: 'relative', width: 200, height: 200 }}>
          <svg width="200" height="200" viewBox="0 0 200 200" style={{ transform: 'rotate(-90deg)' }}>
            <circle cx="100" cy="100" r="88" fill="none" stroke="#E6EAF0" strokeWidth="13" />
            <circle
              ref={ringRef}
              cx="100" cy="100" r="88" fill="none"
              stroke="var(--ritme-pink)" strokeWidth="13" strokeLinecap="round"
              strokeDasharray={CIRCUMFERENCE} strokeDashoffset={CIRCUMFERENCE}
            />
          </svg>
          <div style={{ position: 'absolute', inset: 0, display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 40, fontWeight: 800, color: 'var(--slate)' }}>
            {faNum(pct)}٪
          </div>
        </div>

        <p className="sub" style={{ margin: '40px 0 0', fontSize: 12, color: 'var(--muted, #8A94A0)', lineHeight: 1.8, maxWidth: 340 }}>
          {t('disclaimer')}
        </p>
      </div>
      <div style={{ padding: 20, textAlign: 'center' }}>
        <span className="sub">{t('wait')}</span>
      </div>
    </div>
  );
}
