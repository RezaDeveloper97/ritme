'use client';

import { useTranslations } from 'next-intl';
import { useEffect, useRef, useState } from 'react';

import { useRouter } from '@/shared/i18n';
import { StatusBar } from '@/shared/ui';

const FA = ['۰','۱','۲','۳','۴','۵','۶','۷','۸','۹'];
const faNum = (n: string | number) => String(n).replace(/[0-9]/g, d => FA[Number(d)]);

const CIRCUMFERENCE = 553;

export function SettingUpPage() {
  const t = useTranslations('onboarding.settingUp');
  const router = useRouter();
  const [pct, setPct] = useState(0);
  const ringRef = useRef<SVGCircleElement>(null);

  useEffect(() => {
    let p = 0;
    const id = setInterval(() => {
      p += 2;
      if (p > 100) {
        clearInterval(id);
        // Mark onboarded via cookie
        document.cookie = 'ritme_onboarded=1; path=/; max-age=31536000; SameSite=Lax';
        router.replace('/home');
        return;
      }
      setPct(p);
      if (ringRef.current) {
        ringRef.current.style.strokeDashoffset = String(CIRCUMFERENCE * (1 - p / 100));
      }
    }, 45);
    return () => clearInterval(id);
  }, [router]);

  return (
    <div className="view" style={{ background: '#fff' }}>
      <StatusBar />
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
