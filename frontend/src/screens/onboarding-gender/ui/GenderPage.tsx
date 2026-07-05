'use client';

import { useTranslations } from 'next-intl';
import type { ReactNode } from 'react';

import { useRouter } from '@/shared/i18n';
import { NavBack } from '@/shared/ui';
import { useOnboardingStore, type Gender } from '@/entities/user';

const FA = ['۰','۱','۲','۳','۴','۵','۶','۷','۸','۹'];
const faNum = (n: string | number) => String(n).replace(/[0-9]/g, d => FA[Number(d)]);

interface GenderCardProps {
  label: string;
  color: string;
  bg: string;
  selected: boolean;
  onClick: () => void;
  children: ReactNode;
}

function GenderCard({ label, color, bg, selected, onClick, children }: GenderCardProps) {
  return (
    <button
      onClick={onClick}
      style={{
        width: 108, height: 140, borderRadius: 20, cursor: 'pointer',
        display: 'flex', flexDirection: 'column', alignItems: 'center',
        justifyContent: 'center', gap: 12,
        border: `2px solid ${selected ? color : '#EBEEF2'}`,
        background: selected ? bg : '#fff',
        transition: '.18s',
      }}
    >
      <span style={{
        width: 60, height: 60, borderRadius: '50%', background: bg,
        display: 'flex', alignItems: 'center', justifyContent: 'center', color,
      }}>
        {children}
      </span>
      <span style={{ fontWeight: 700, fontSize: 15, color: selected ? color : 'var(--ink)' }}>
        {label}
      </span>
    </button>
  );
}

export function GenderPage() {
  const t = useTranslations('onboarding');
  const router = useRouter();
  const { gender, setGender } = useOnboardingStore();

  const select = (g: Gender) => {
    setGender(g);
    router.push('/onboarding/birthday');
  };

  return (
    <div className="view" style={{ background: '#fff' }}>
      <div className="hdr">
        <NavBack onClick={() => router.back()} />
        <span className="stepcount">{faNum(2)}<span style={{ opacity: .5 }}> / ۷</span></span>
      </div>

      <div className="scroll" style={{ padding: '8px 22px 0', display: 'flex', flexDirection: 'column' }}>
        <div style={{ textAlign: 'start', margin: '6px 0' }}>
          <div className="titr">{t('gender.title')}</div>
          <p className="sub" style={{ margin: '10px 0 0' }}>{t('gender.subtitle')}</p>
        </div>
        <div style={{ flex: 1, display: 'flex', flexDirection: 'column', justifyContent: 'center', padding: '8px 0' }}>
          <div style={{ display: 'flex', gap: 18, justifyContent: 'center', marginTop: 6 }}>
            <GenderCard
              label={t('gender.female')}
              color="#FB64B6" bg="#FFF0F7"
              selected={gender === 'female'} onClick={() => select('female')}
            >
              <svg viewBox="0 0 24 24" width="30" height="30" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <path d="M12 3a6 6 0 1 0 0 12 6 6 0 0 0 0-12zM12 15v6M9 19h6" />
              </svg>
            </GenderCard>
            <GenderCard
              label={t('gender.male')}
              color="#3B9EF0" bg="#EEF6FF"
              selected={gender === 'male'} onClick={() => select('male')}
            >
              <svg viewBox="0 0 24 24" width="30" height="30" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <circle cx="10" cy="14" r="6" />
                <path d="M14.5 9.5L20 4M20 4h-5M20 4v5" />
              </svg>
            </GenderCard>
          </div>
          <div style={{ display: 'flex', justifyContent: 'center', marginTop: 26 }}>
            <button className="btn-soft" style={{ width: 'auto', padding: '0 20px' }} onClick={() => select('none')}>
              {t('gender.skip')}
            </button>
          </div>
        </div>
      </div>

      <div style={{ padding: '14px 16px 8px' }}>
        <button className="btn btn-primary" onClick={() => router.push('/onboarding/birthday')}>
          {t('continue')}
        </button>
      </div>
    </div>
  );
}
