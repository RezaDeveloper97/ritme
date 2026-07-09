'use client';

import { useTranslations } from 'next-intl';
import { useEffect, useRef, useState } from 'react';

import { useOnboardingStore } from '@/entities/user';
import { onboardingToProfileInput, useUpdateProfile } from '@/features/edit-profile';
import { useRouter } from '@/shared/i18n';

const FA = ['۰','۱','۲','۳','۴','۵','۶','۷','۸','۹'];
const faNum = (n: string | number) => String(n).replace(/[0-9]/g, d => FA[Number(d)]);

const CIRCUMFERENCE = 553;

export function SettingUpPage() {
  const t = useTranslations('onboarding.settingUp');
  const router = useRouter();
  const update = useUpdateProfile();
  const onboarding = useOnboardingStore();
  const [pct, setPct] = useState(0);
  const [ringDone, setRingDone] = useState(false);
  const ringRef = useRef<SVGCircleElement>(null);
  const savedRef = useRef(false);

  // Persist the collected answers to the profile exactly once. The guard keeps
  // React 18 StrictMode's double-invoke from firing two POSTs.
  useEffect(() => {
    if (savedRef.current) return;
    savedRef.current = true;
    update.mutate(onboardingToProfileInput(onboarding));
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

  // Leave for home only once the ring has finished AND the save has settled, so
  // the profile is written before the home screen refetches it. On error we
  // still proceed — the user can adjust everything later in their profile.
  useEffect(() => {
    if (!ringDone || update.isPending) return;
    document.cookie = 'ritme_onboarded=1; path=/; max-age=31536000; SameSite=Lax';
    router.replace('/home');
  }, [ringDone, update.isPending, router]);

  return (
    <div className="view" style={{ background: '#fff' }}>
      <div className="scroll" style={{ padding: '80px 22px 0', display: 'flex', flexDirection: 'column', alignItems: 'center', textAlign: 'center' }}>
        <div className="titr" style={{ fontSize: 19, lineHeight: 1.7 }}>{t('title')}</div>
        <p className="sub" style={{ margin: '12px 0 50px' }}>{t('subtitle')}</p>

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
      </div>
      <div style={{ padding: 20, textAlign: 'center' }}>
        <span className="sub">{t('wait')}</span>
      </div>
    </div>
  );
}
